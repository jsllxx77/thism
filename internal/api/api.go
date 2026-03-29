package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/alerting"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/notify"
	"github.com/thism-dev/thism/internal/security"
	"github.com/thism-dev/thism/internal/store"
	sharedversion "github.com/thism-dev/thism/internal/version"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: sameOriginWebsocketRequest,
}

const adminSessionCookieName = "thism_admin"
const guestSessionCookieName = "thism_guest"
const uiLanguageCookieName = "thism-lang"

type accessRole string

const (
	accessRoleNone  accessRole = ""
	accessRoleGuest accessRole = "guest"
	accessRoleAdmin accessRole = "admin"
)

type accessRoleContextKey struct{}

type AuthConfig struct {
	AdminToken string
	Username   string
	Password   string
}

func (c AuthConfig) PasswordLoginEnabled() bool {
	return strings.TrimSpace(c.Username) != "" && c.Password != ""
}

func (c AuthConfig) ValidPasswordLogin(username, password string) bool {
	if !c.PasswordLoginEnabled() {
		return false
	}
	userMatch := constantTimeStringEqual(username, c.Username)
	passMatch := security.VerifyPassword(password, c.Password)
	return userMatch && passMatch
}

type authManager struct {
	mu         sync.RWMutex
	adminToken string
	username   string
	password   string
	sessions   *sessionManager
}

type sessionStore interface {
	CreateAdminSession(sessionID string, expiresAt int64) error
	HasAdminSession(sessionID string) (bool, error)
	DeleteAdminSession(sessionID string) error
	CleanupExpiredAdminSessions() error
}

type sessionManager struct {
	mu       sync.RWMutex
	sessions map[string]struct{}
	store    sessionStore
}

var (
	errPasswordLoginDisabled = errors.New("password login is not configured")
	errInvalidCurrentPass    = errors.New("invalid current password")
)

func newAuthManager(cfg AuthConfig, sessionBackend sessionStore) *authManager {
	return &authManager{
		adminToken: cfg.AdminToken,
		username:   strings.TrimSpace(cfg.Username),
		password:   cfg.Password,
		sessions:   newSessionManager(sessionBackend),
	}
}

func newSessionManager(sessionBackend sessionStore) *sessionManager {
	m := &sessionManager{sessions: make(map[string]struct{}), store: sessionBackend}
	if sessionBackend != nil {
		_ = sessionBackend.CleanupExpiredAdminSessions()
	}
	return m
}

func (m *sessionManager) Create() (string, error) {
	sessionID, err := generateHexBytes(32)
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Unix()
	if m.store != nil {
		if err := m.store.CreateAdminSession(sessionID, expiresAt); err != nil {
			return "", err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionID] = struct{}{}
	return sessionID, nil
}

func (m *sessionManager) Has(sessionID string) bool {
	if sessionID == "" {
		return false
	}

	m.mu.RLock()
	_, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if ok {
		return true
	}
	if m.store == nil {
		return false
	}
	ok, err := m.store.HasAdminSession(sessionID)
	if err != nil || !ok {
		return false
	}
	m.mu.Lock()
	m.sessions[sessionID] = struct{}{}
	m.mu.Unlock()
	return true
}

func (m *sessionManager) Delete(sessionID string) {
	if sessionID == "" {
		return
	}

	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	if m.store != nil {
		_ = m.store.DeleteAdminSession(sessionID)
	}
}

func (m *authManager) AdminToken() string {
	return m.adminToken
}

func (m *authManager) SetCredentials(username, password string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.username = strings.TrimSpace(username)
	m.password = password
}

func (m *authManager) Credentials() (username, password string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.username, m.password
}

func (m *authManager) PasswordLoginEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.username != "" && m.password != ""
}

func (m *authManager) ValidPasswordLogin(username, password string) bool {
	return m.AuthenticatePassword(username, password, nil)
}

func (m *authManager) AuthenticatePassword(username, password string, persistFn func(username, password string) error) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.username == "" || m.password == "" {
		return false
	}

	userMatch := constantTimeStringEqual(username, m.username)
	passMatch := security.VerifyPassword(password, m.password)
	if !userMatch || !passMatch {
		return false
	}

	if security.NeedsPasswordHashUpgrade(m.password) {
		hashedPassword, err := security.HashPassword(password)
		if err == nil {
			if persistFn == nil || persistFn(m.username, hashedPassword) == nil {
				m.password = hashedPassword
			}
		}
	}

	return true
}

func (m *authManager) ChangePassword(currentPassword, newPassword string, persistFn func(username, password string) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.username == "" || m.password == "" {
		return errPasswordLoginDisabled
	}
	if !security.VerifyPassword(currentPassword, m.password) {
		return errInvalidCurrentPass
	}

	hashedPassword, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if persistFn != nil {
		if err := persistFn(m.username, hashedPassword); err != nil {
			return err
		}
	}

	m.password = hashedPassword
	return nil
}

func (m *authManager) HasAdminAccess(r *http.Request) bool {
	if m == nil || m.adminToken == "" {
		return false
	}
	if bearerToken(r) == m.adminToken {
		return true
	}
	return m.sessions.Has(adminSessionID(r))
}

func (m *authManager) HasBootstrapQueryToken(r *http.Request) bool {
	if m == nil || m.adminToken == "" || r == nil {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
		return false
	}
	return r.URL.Query().Get("token") == m.adminToken
}

func (m *authManager) BootstrapSessionCookie(w http.ResponseWriter, r *http.Request) {
	if m == nil || m.sessions.Has(adminSessionID(r)) {
		return
	}

	if !m.HasBootstrapQueryToken(r) && bearerToken(r) != m.adminToken {
		return
	}

	sessionID, err := m.sessions.Create()
	if err != nil {
		return
	}
	writeAdminSessionCookie(w, r, sessionID)
}

func (m *authManager) IssueAdminSession(w http.ResponseWriter, r *http.Request) error {
	if m == nil {
		return errors.New("auth manager is nil")
	}

	m.sessions.Delete(adminSessionID(r))
	sessionID, err := m.sessions.Create()
	if err != nil {
		return err
	}
	writeAdminSessionCookie(w, r, sessionID)
	return nil
}

func (m *authManager) ClearAdminSession(w http.ResponseWriter, r *http.Request) {
	if m != nil {
		m.sessions.Delete(adminSessionID(r))
	}
	clearSessionCookie(w, r)
}

func constantTimeStringEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// NewRouter builds and returns the HTTP router.
// If frontendHandler is non-nil it is used as a fallback for unmatched routes;
// otherwise unmatched routes return 404.
func NewRouter(s *store.Store, h *hub.Hub, adminToken string, frontendHandler http.Handler) http.Handler {
	return NewRouterWithAuth(s, h, AuthConfig{AdminToken: adminToken}, frontendHandler)
}

// NewRouterWithAuth builds and returns the HTTP router with configurable
// admin login credentials.
func NewRouterWithAuth(s *store.Store, h *hub.Hub, auth AuthConfig, frontendHandler http.Handler) http.Handler {
	authState := newAuthManager(auth, s)

	// Load persisted credentials if present, otherwise bootstrap from startup
	// configuration when password login is enabled.
	if s != nil {
		if username, password, found, err := s.GetAdminAuth(); err == nil {
			if found {
				authState.SetCredentials(username, password)
				if security.NeedsPasswordHashUpgrade(password) {
					if hashedPassword, err := security.HashPassword(password); err == nil {
						if err := s.UpsertAdminAuth(username, hashedPassword); err == nil {
							authState.SetCredentials(username, hashedPassword)
						}
					}
				}
			} else if authState.PasswordLoginEnabled() {
				username, password := authState.Credentials()
				if hashedPassword, err := security.HashPassword(password); err == nil {
					if err := s.UpsertAdminAuth(username, hashedPassword); err == nil {
						authState.SetCredentials(username, hashedPassword)
					}
				}
			}
		}
	}

	r := chi.NewRouter()

	// ---------------------------------------------------------------
	// WebSocket endpoints (auth handled inside each handler)
	// ---------------------------------------------------------------

	// Agent WebSocket: authenticates via node token (?token=)
	r.Get("/ws/agent", func(w http.ResponseWriter, req *http.Request) {
		handleAgentWS(w, req, s, h)
	})

	// Dashboard WebSocket: requires admin or guest access
	r.Get("/ws/dashboard", func(w http.ResponseWriter, req *http.Request) {
		role := resolveAccessRole(req, authState)
		if role == accessRoleNone {
			http.Error(w, uiMessage(resolveUILanguage(req), "unauthorized"), http.StatusUnauthorized)
			return
		}
		if role == accessRoleAdmin {
			authState.BootstrapSessionCookie(w, req)
		}
		handleDashboardWS(w, req, s, h)
	})

	// ---------------------------------------------------------------
	// Viewer API (admin + guest)
	// ---------------------------------------------------------------
	r.Group(func(r chi.Router) {
		r.Use(viewerAuth(authState))

		r.Get("/api/auth/session", func(w http.ResponseWriter, req *http.Request) {
			handleSession(w, req)
		})

		r.Get("/api/nodes", func(w http.ResponseWriter, req *http.Request) {
			handleListNodes(w, req, s, h)
		})

		r.Get("/api/meta/version", func(w http.ResponseWriter, req *http.Request) {
			handleGetVersionMetadata(w, req)
		})

		r.Get("/api/settings/metrics-retention", func(w http.ResponseWriter, req *http.Request) {
			handleGetMetricsRetention(w, req, s)
		})

		r.Get("/api/settings/notifications", func(w http.ResponseWriter, req *http.Request) {
			handleGetNotificationSettings(w, req, s)
		})

		r.Get("/api/settings/dashboard", func(w http.ResponseWriter, req *http.Request) {
			handleGetDashboardSettings(w, req, s)
		})
	})

	// ---------------------------------------------------------------
	// Admin API
	// ---------------------------------------------------------------
	r.Group(func(r chi.Router) {
		r.Use(adminAuth(authState))

		r.Post("/api/auth/change-password", func(w http.ResponseWriter, req *http.Request) {
			handleChangePassword(w, req, s, authState)
		})

		r.Put("/api/settings/metrics-retention", func(w http.ResponseWriter, req *http.Request) {
			handleUpdateMetricsRetention(w, req, s)
		})

		r.Put("/api/settings/notifications", func(w http.ResponseWriter, req *http.Request) {
			handleUpdateNotificationSettings(w, req, s)
		})

		r.Put("/api/settings/dashboard", func(w http.ResponseWriter, req *http.Request) {
			handleUpdateDashboardSettings(w, req, s)
		})

		r.Post("/api/settings/notifications/test", func(w http.ResponseWriter, req *http.Request) {
			handleSendTestNotification(w, req, s)
		})

		r.Post("/api/nodes/register", func(w http.ResponseWriter, req *http.Request) {
			handleRegisterNode(w, req, s)
		})

		r.Patch("/api/nodes/{id}", func(w http.ResponseWriter, req *http.Request) {
			handleUpdateNode(w, req, s)
		})

		r.Delete("/api/nodes/{id}", func(w http.ResponseWriter, req *http.Request) {
			handleDeleteNode(w, req, s)
		})

		r.Get("/api/nodes/{id}/install-command", func(w http.ResponseWriter, req *http.Request) {
			handleGetInstallCommand(w, req, s)
		})

		r.Get("/api/nodes/{id}/metrics", func(w http.ResponseWriter, req *http.Request) {
			handleGetMetrics(w, req, s)
		})

		r.Get("/api/nodes/{id}/processes", func(w http.ResponseWriter, req *http.Request) {
			handleGetProcesses(w, req, s)
		})

		r.Get("/api/nodes/{id}/services", func(w http.ResponseWriter, req *http.Request) {
			handleGetServices(w, req, s)
		})

		r.Get("/api/nodes/{id}/docker", func(w http.ResponseWriter, req *http.Request) {
			handleGetDocker(w, req, s)
		})

		r.Post("/api/agent-updates", func(w http.ResponseWriter, req *http.Request) {
			handleCreateAgentUpdateJob(w, req, s, h)
		})

		r.Get("/api/agent-updates/{id}", func(w http.ResponseWriter, req *http.Request) {
			handleGetAgentUpdateJob(w, req, s)
		})
	})

	// ---------------------------------------------------------------
	// Unauthenticated endpoints for agent installation
	// ---------------------------------------------------------------
	r.Get("/install.sh", func(w http.ResponseWriter, req *http.Request) {
		handleInstallScript(w, req)
	})
	r.Get("/dl/{filename}", func(w http.ResponseWriter, req *http.Request) {
		handleDownload(w, req)
	})
	r.Get("/api/agent-release", func(w http.ResponseWriter, req *http.Request) {
		handleAgentRelease(w, req)
	})
	r.Get("/login", func(w http.ResponseWriter, req *http.Request) {
		handleLoginPage(w, req, authState)
	})
	r.Post("/api/auth/login", func(w http.ResponseWriter, req *http.Request) {
		handlePasswordLogin(w, req, s, authState)
	})
	r.Post("/api/auth/guest", func(w http.ResponseWriter, req *http.Request) {
		handleGuestLogin(w, req, authState)
	})
	r.Post("/api/auth/logout", func(w http.ResponseWriter, req *http.Request) {
		handleLogout(w, req, authState)
	})

	// ---------------------------------------------------------------
	// Fallback
	// ---------------------------------------------------------------
	if frontendHandler != nil {
		r.Group(func(r chi.Router) {
			r.Use(frontendAuth(authState))
			r.Handle("/*", frontendHandler)
		})
	} else {
		r.Handle("/*", http.NotFoundHandler())
	}

	return r
}

// -----------------------------------------------------------------------
// Auth helpers
// -----------------------------------------------------------------------

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
}

func adminSessionID(r *http.Request) string {
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func writeAdminSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
		Expires:  expiresAt,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func guestSessionActive(r *http.Request) bool {
	cookie, err := r.Cookie(guestSessionCookieName)
	return err == nil && cookie.Value == "1"
}

func setGuestSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     guestSessionCookieName,
		Value:    "1",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func clearGuestSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     guestSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func withAccessRole(r *http.Request, role accessRole) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), accessRoleContextKey{}, role))
}

func accessRoleFromRequest(r *http.Request) accessRole {
	if r == nil {
		return accessRoleNone
	}
	role, _ := r.Context().Value(accessRoleContextKey{}).(accessRole)
	return role
}

func resolveAccessRole(r *http.Request, auth *authManager) accessRole {
	if auth != nil && (auth.HasAdminAccess(r) || auth.HasBootstrapQueryToken(r)) {
		return accessRoleAdmin
	}
	if guestSessionActive(r) {
		return accessRoleGuest
	}
	return accessRoleNone
}

func redirectWithoutToken(w http.ResponseWriter, r *http.Request) {
	cleanURL := *r.URL
	query := cleanURL.Query()
	query.Del("token")
	cleanURL.RawQuery = query.Encode()

	target := cleanURL.Path
	if target == "" {
		target = "/"
	}
	if cleanURL.RawQuery != "" {
		target += "?" + cleanURL.RawQuery
	}

	http.Redirect(w, r, target, http.StatusFound)
}

// adminAuth returns a middleware that enforces administrator access.
func adminAuth(auth *authManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auth == nil || !auth.HasAdminAccess(r) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": uiMessage(resolveUILanguage(r), "unauthorized")})
				return
			}
			auth.BootstrapSessionCookie(w, r)
			next.ServeHTTP(w, withAccessRole(r, accessRoleAdmin))
		})
	}
}

func viewerAuth(auth *authManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := resolveAccessRole(r, auth)
			if role == accessRoleNone {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": uiMessage(resolveUILanguage(r), "unauthorized")})
				return
			}
			if role == accessRoleAdmin {
				auth.BootstrapSessionCookie(w, r)
			}
			next.ServeHTTP(w, withAccessRole(r, role))
		})
	}
}

// frontendAuth enforces authenticated access for the SPA and bootstraps an
// admin session cookie so subsequent static assets and API calls do not require query tokens.
func frontendAuth(auth *authManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			adminToken := auth.AdminToken()
			role := resolveAccessRole(r, auth)
			if role == accessRoleNone {
				if shouldRedirectToLogin(r, auth) {
					http.Redirect(w, r, "/login", http.StatusFound)
					return
				}
				http.Error(w, uiMessage(resolveUILanguage(r), "unauthorized"), http.StatusUnauthorized)
				return
			}

			if role == accessRoleAdmin {
				auth.BootstrapSessionCookie(w, r)

				if (r.Method == http.MethodGet || r.Method == http.MethodHead) && r.URL.Query().Get("token") == adminToken {
					redirectWithoutToken(w, r)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func shouldRedirectToLogin(r *http.Request, auth *authManager) bool {
	if !auth.PasswordLoginEnabled() {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/html")
}

type uiLanguage string

const (
	uiLanguageEnglish uiLanguage = "en"
	uiLanguageChinese uiLanguage = "zh-CN"
)

type loginPageData struct {
	Language             uiLanguage
	PageTitle            string
	BrandKicker          string
	BrandTitle           string
	ToggleLabel          string
	ToggleTarget         uiLanguage
	AccessEyebrow        string
	LoginTitle           string
	LoginDescription     string
	UsernameLabel        string
	PasswordLabel        string
	SubmitLabel          string
	GuestLabel           string
	GuestDescription     string
	MetaLabel            string
	InvalidCredentials   string
	LoginFailedMessage   string
	RequestFailedMessage string
	GuestFailedMessage   string
}

var loginPageTemplate = template.Must(template.New("login-page").Parse(`<!doctype html>
<html lang="{{.Language}}">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.PageTitle}}</title>
  <style>
    :root {
      color-scheme: light;
      --page-bg:
        radial-gradient(860px circle at 0% 0%, rgba(37, 99, 235, 0.1), transparent 50%),
        radial-gradient(820px circle at 100% 10%, rgba(51, 65, 85, 0.07), transparent 58%),
        linear-gradient(180deg, #f8f9fb 0%, #f3f4f6 100%);
      --panel-bg: rgba(255, 255, 255, 0.96);
      --panel-border: #e5e7eb;
      --panel-shadow: 0 8px 24px rgba(30, 64, 175, 0.05);
      --text-strong: #0f172a;
      --text-muted: #475569;
      --text-soft: #64748b;
      --input-bg: #ffffff;
      --input-border: #cbd5e1;
      --input-focus: rgba(37, 99, 235, 0.18);
      --button-bg: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
      --button-shadow: 0 14px 30px rgba(37, 99, 235, 0.18);
      --button-text: #ffffff;
      --badge-bg: rgba(37, 99, 235, 0.08);
      --badge-border: rgba(148, 163, 184, 0.22);
      --badge-text: #1e3a8a;
      --surface-line: rgba(148, 163, 184, 0.16);
      --error: #b91c1c;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        color-scheme: dark;
        --page-bg:
          radial-gradient(920px circle at 10% -5%, rgba(59, 130, 246, 0.2), transparent 50%),
          radial-gradient(880px circle at 90% 10%, rgba(56, 189, 248, 0.12), transparent 56%),
          linear-gradient(180deg, #020617 0%, #0b1220 100%);
        --panel-bg: rgba(15, 23, 42, 0.88);
        --panel-border: rgba(71, 85, 105, 0.45);
        --panel-shadow: 0 12px 28px rgba(2, 6, 23, 0.62);
        --text-strong: #e2e8f0;
        --text-muted: #cbd5e1;
        --text-soft: #94a3b8;
        --input-bg: rgba(15, 23, 42, 0.78);
        --input-border: #334155;
        --input-focus: rgba(59, 130, 246, 0.25);
        --button-bg: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
        --button-shadow: 0 16px 30px rgba(37, 99, 235, 0.28);
        --badge-bg: rgba(37, 99, 235, 0.14);
        --badge-border: rgba(96, 165, 250, 0.18);
        --badge-text: #bfdbfe;
        --surface-line: rgba(148, 163, 184, 0.18);
        --error: #fca5a5;
      }
      .toolbar-button {
        background: rgba(15, 23, 42, 0.88);
        color: #f8fafc;
        border-color: rgba(148, 163, 184, 0.4);
        box-shadow: 0 10px 26px rgba(2, 6, 23, 0.42);
        -webkit-text-fill-color: #f8fafc;
      }
      .toolbar-button:hover {
        background: rgba(30, 41, 59, 0.98);
        color: #ffffff;
        border-color: rgba(148, 163, 184, 0.6);
      }
    }
    * {
      box-sizing: border-box;
    }
    body {
      margin: 0;
      min-height: 100vh;
      min-height: 100svh;
      display: grid;
      place-items: center;
      padding: 24px 16px;
      background: var(--page-bg);
      color: var(--text-strong);
      font-family: "Outfit", "Geist", "Segoe UI", "Helvetica Neue", sans-serif;
      line-height: 1.55;
      -webkit-font-smoothing: antialiased;
      text-rendering: optimizeLegibility;
    }
    .shell {
      width: min(100%, 432px);
    }
    @media (min-width: 768px) {
      body {
        display: block;
        padding-top: clamp(40px, 9vh, 96px);
        padding-bottom: 56px;
      }
      .shell {
        margin: 0 auto;
      }
    }
    .page-toolbar {
      display: flex;
      justify-content: flex-end;
      margin-bottom: 12px;
    }
    .toolbar-button {
      border: 1px solid rgba(15, 23, 42, 0.14);
      border-radius: 999px;
      background: #ffffff;
      color: #0f172a;
      padding: 8px 12px;
      font: inherit;
      font-size: 0.82rem;
      font-weight: 800;
      letter-spacing: 0.01em;
      cursor: pointer;
      box-shadow: 0 10px 24px rgba(15, 23, 42, 0.12);
      -webkit-text-fill-color: #0f172a;
      opacity: 1;
      backdrop-filter: blur(14px);
      transition: background-color 0.18s ease, color 0.18s ease, border-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 16px;
    }
    .brand-mark {
      display: inline-flex;
      height: 44px;
      width: 44px;
      align-items: center;
      justify-content: center;
      border-radius: 14px;
      border: 1px solid var(--panel-border);
      background: rgba(255, 255, 255, 0.7);
      box-shadow: 0 8px 22px rgba(15, 23, 42, 0.08);
      backdrop-filter: blur(14px);
    }
    .brand-copy {
      min-width: 0;
    }
    .brand-kicker {
      display: block;
      font-size: 0.78rem;
      font-weight: 600;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--text-soft);
    }
    .brand-title {
      display: block;
      margin-top: 2px;
      font-size: 1rem;
      font-weight: 700;
      letter-spacing: -0.02em;
      color: var(--text-strong);
    }
    .card {
      width: min(100%, 432px);
      border: 1px solid var(--panel-border);
      border-radius: 24px;
      background: var(--panel-bg);
      box-shadow: var(--panel-shadow);
      backdrop-filter: blur(18px);
      overflow: hidden;
    }
    .card-header {
      padding: 24px 24px 18px;
      border-bottom: 1px solid var(--surface-line);
    }
    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 6px 10px;
      border-radius: 999px;
      border: 1px solid var(--badge-border);
      background: var(--badge-bg);
      color: var(--badge-text);
      font-size: 0.76rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    h1 {
      margin: 14px 0 6px;
      font-size: clamp(1.7rem, 3vw, 2rem);
      line-height: 1.1;
      letter-spacing: -0.03em;
    }
    p {
      margin: 0;
      color: var(--text-muted);
      font-size: 0.96rem;
    }
    .form {
      padding: 22px 24px 24px;
    }
    label {
      display: block;
      margin: 14px 0 8px;
      font-size: 0.82rem;
      font-weight: 600;
      letter-spacing: 0.02em;
      color: var(--text-muted);
    }
    input {
      width: 100%;
      border: 1px solid var(--input-border);
      border-radius: 14px;
      background: var(--input-bg);
      color: var(--text-strong);
      padding: 13px 14px;
      font: inherit;
      outline: none;
      transition: border-color 0.2s ease, box-shadow 0.2s ease, background-color 0.2s ease;
    }
    input::placeholder {
      color: var(--text-soft);
    }
    input:focus {
      border-color: #2563eb;
      box-shadow: 0 0 0 4px var(--input-focus);
    }
    .submit-button {
      width: 100%;
      margin-top: 18px;
      border: none;
      border-radius: 14px;
      padding: 13px 16px;
      font: inherit;
      font-weight: 700;
      letter-spacing: -0.01em;
      color: var(--button-text);
      background: var(--button-bg);
      box-shadow: var(--button-shadow);
      cursor: pointer;
      transition: transform 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
    }
    .submit-button:hover {
      transform: translateY(-1px);
    }
    .toolbar-button:hover {
      background: #f8fafc;
      color: #020617;
      border-color: #64748b;
      transform: translateY(-1px);
      box-shadow: 0 12px 28px rgba(15, 23, 42, 0.16);
    }
    .submit-button:active {
      transform: translateY(0);
    }
    .toolbar-button:active {
      transform: translateY(0);
    }
    .submit-button:focus-visible,
    .toolbar-button:focus-visible {
      outline: none;
      box-shadow: var(--button-shadow), 0 0 0 4px var(--input-focus);
    }
    .submit-button:disabled {
      opacity: 0.7;
      cursor: not-allowed;
      transform: none;
      box-shadow: none;
    }
    .secondary-button {
      width: 100%;
      margin-top: 10px;
      border: 1px solid var(--panel-border);
      border-radius: 16px;
      background: transparent;
      color: var(--text-strong);
      padding: 13px 16px;
      font: inherit;
      font-weight: 600;
      cursor: pointer;
      transition: border-color 0.18s ease, background 0.18s ease, color 0.18s ease;
    }
    .secondary-button:hover,
    .secondary-button:focus-visible {
      border-color: #94a3b8;
      background: rgba(148, 163, 184, 0.08);
      outline: none;
    }
    .secondary-button:disabled {
      opacity: 0.7;
      cursor: not-allowed;
    }
    .meta {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin-top: 14px;
      font-size: 0.82rem;
      color: var(--text-soft);
    }
    .meta strong {
      color: var(--text-muted);
      font-weight: 600;
    }
    .error {
      min-height: 20px;
      margin-top: 12px;
      color: var(--error);
      font-size: 0.84rem;
    }
    @media (max-width: 480px) {
      .card-header,
      .form {
        padding-left: 18px;
        padding-right: 18px;
      }
      .meta {
        flex-direction: column;
        align-items: flex-start;
      }
    }
  </style>
</head>
<body data-language="{{.Language}}" data-login-failed-message="{{.LoginFailedMessage}}" data-request-failed-message="{{.RequestFailedMessage}}" data-guest-failed-message="{{.GuestFailedMessage}}">
  <main class="shell">
    <div class="page-toolbar">
      <button id="lang-switch" class="toolbar-button" type="button" data-language-target="{{.ToggleTarget}}">{{.ToggleLabel}}</button>
    </div>
    <div class="brand">
      <div class="brand-mark" aria-hidden="true">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M5 17.5V11.5" stroke="#2563EB" stroke-width="2" stroke-linecap="round"/>
          <path d="M12 17.5V6.5" stroke="#2563EB" stroke-width="2" stroke-linecap="round"/>
          <path d="M19 17.5V9.5" stroke="#2563EB" stroke-width="2" stroke-linecap="round"/>
        </svg>
      </div>
      <div class="brand-copy">
        <span class="brand-kicker">{{.BrandKicker}}</span>
        <span class="brand-title">{{.BrandTitle}}</span>
      </div>
    </div>
    <section class="card" aria-labelledby="login-title">
      <div class="card-header">
        <span class="eyebrow">{{.AccessEyebrow}}</span>
        <h1 id="login-title">{{.LoginTitle}}</h1>
        <p>{{.LoginDescription}}</p>
      </div>
      <form id="login-form" class="form">
        <label for="username">{{.UsernameLabel}}</label>
        <input id="username" name="username" type="text" autocomplete="username" required />
        <label for="password">{{.PasswordLabel}}</label>
        <input id="password" name="password" type="password" autocomplete="current-password" required />
        <button id="submit-btn" class="submit-button" type="submit">{{.SubmitLabel}}</button>
        <button id="guest-btn" class="secondary-button" type="button" data-endpoint="/api/auth/guest">{{.GuestLabel}}</button>
        <p>{{.GuestDescription}}</p>
        <div class="meta" aria-hidden="true">
          <span>{{.MetaLabel}}</span>
          <strong>ThisM</strong>
        </div>
        <div class="error" id="error-msg" role="alert" aria-live="polite"></div>
      </form>
    </section>
  </main>
  <script>
    const form = document.getElementById("login-form");
    const errorMsg = document.getElementById("error-msg");
    const submitBtn = document.getElementById("submit-btn");
    const guestBtn = document.getElementById("guest-btn");
    const langSwitch = document.getElementById("lang-switch");
    const body = document.body;
    const currentLanguage = body.dataset.language || "en";
    const loginFailedMessage = body.dataset.loginFailedMessage || "Login failed. Please check credentials.";
    const requestFailedMessage = body.dataset.requestFailedMessage || "Request failed. Please retry.";
    const guestFailedMessage = body.dataset.guestFailedMessage || requestFailedMessage;

    function persistLanguage(language) {
      document.cookie = "thism-lang=" + encodeURIComponent(language) + "; Path=/; Max-Age=31536000; SameSite=Lax";
      try {
        window.localStorage.setItem("thism-language", language);
      } catch (error) {
      }
    }

    persistLanguage(currentLanguage);

    if (langSwitch) {
      langSwitch.addEventListener("click", () => {
        const nextLanguage = langSwitch.dataset.languageTarget;
        if (!nextLanguage) {
          return;
        }
        persistLanguage(nextLanguage);
        window.location.reload();
      });
    }

    if (guestBtn) {
      guestBtn.addEventListener("click", async () => {
        errorMsg.textContent = "";
        submitBtn.disabled = true;
        guestBtn.disabled = true;
        try {
          const endpoint = guestBtn.dataset.endpoint || "/api/auth/guest";
          const response = await fetch(endpoint, { method: "POST" });
          if (response.ok) {
            window.location.assign("/");
            return;
          }
          const data = await response.json().catch(() => ({}));
          errorMsg.textContent = data.error || guestFailedMessage;
        } catch (error) {
          errorMsg.textContent = requestFailedMessage;
        } finally {
          submitBtn.disabled = false;
          guestBtn.disabled = false;
        }
      });
    }

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      errorMsg.textContent = "";
      submitBtn.disabled = true;
      if (guestBtn) {
        guestBtn.disabled = true;
      }
      const payload = {
        username: document.getElementById("username").value,
        password: document.getElementById("password").value
      };
      try {
        const response = await fetch("/api/auth/login", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload)
        });
        if (response.ok) {
          window.location.assign("/");
          return;
        }
        const data = await response.json().catch(() => ({}));
        errorMsg.textContent = data.error || loginFailedMessage;
      } catch (error) {
        errorMsg.textContent = requestFailedMessage;
      } finally {
        submitBtn.disabled = false;
        if (guestBtn) {
          guestBtn.disabled = false;
        }
      }
    });
  </script>
</body>
</html>
`))

func normalizeUILanguage(value string) uiLanguage {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(trimmed, "zh"):
		return uiLanguageChinese
	case strings.HasPrefix(trimmed, "en"):
		return uiLanguageEnglish
	default:
		return ""
	}
}

func resolveUILanguage(r *http.Request) uiLanguage {
	if r == nil {
		return uiLanguageEnglish
	}
	if cookie, err := r.Cookie(uiLanguageCookieName); err == nil {
		if lang := normalizeUILanguage(cookie.Value); lang != "" {
			return lang
		}
	}
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		langTag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if lang := normalizeUILanguage(langTag); lang != "" {
			return lang
		}
	}
	return uiLanguageEnglish
}

func uiMessage(language uiLanguage, key string) string {
	if language == uiLanguageChinese {
		switch key {
		case "unauthorized":
			return "未授权"
		case "tokenRequired":
			return "缺少令牌"
		case "invalidToken":
			return "令牌无效"
		case "passwordLoginDisabled":
			return "未配置密码登录"
		case "invalidRequestBody":
			return "请求体无效"
		case "invalidCredentials":
			return "登录凭证错误"
		case "invalidCurrentPassword":
			return "当前密码错误"
		case "passwordChangeRequired":
			return "当前密码和新密码均为必填"
		case "newPasswordMustDiffer":
			return "新密码必须与当前密码不同"
		case "sessionStartFailed":
			return "会话创建失败"
		}
	}
	switch key {
	case "unauthorized":
		return "unauthorized"
	case "tokenRequired":
		return "token required"
	case "invalidToken":
		return "invalid token"
	case "passwordLoginDisabled":
		return "password login is not configured"
	case "invalidRequestBody":
		return "invalid request body"
	case "invalidCredentials":
		return "invalid credentials"
	case "invalidCurrentPassword":
		return "invalid current password"
	case "passwordChangeRequired":
		return "current_password and new_password are required"
	case "newPasswordMustDiffer":
		return "new password must be different"
	case "sessionStartFailed":
		return "failed to start session"
	default:
		return key
	}
}

func loginPageDataForLanguage(language uiLanguage) loginPageData {
	if language == uiLanguageChinese {
		return loginPageData{
			Language:             uiLanguageChinese,
			PageTitle:            "ThisM 登录",
			BrandKicker:          "安全访问",
			BrandTitle:           "ThisM 控制台",
			ToggleLabel:          "English",
			ToggleTarget:         uiLanguageEnglish,
			AccessEyebrow:        "管理员访问",
			LoginTitle:           "登录",
			LoginDescription:     "使用管理员凭据访问仪表盘与节点控制。",
			UsernameLabel:        "用户名",
			PasswordLabel:        "密码",
			SubmitLabel:          "登录",
			GuestLabel:           "游客模式",
			GuestDescription:     "以只读访客身份进入，仅查看展示卡片与节点基础信息。",
			MetaLabel:            "平台管理的受保护会话",
			InvalidCredentials:   uiMessage(uiLanguageChinese, "invalidCredentials"),
			LoginFailedMessage:   "登录失败，请检查凭据。",
			RequestFailedMessage: "请求失败，请重试。",
			GuestFailedMessage:   "进入游客模式失败，请重试。",
		}
	}
	return loginPageData{
		Language:             uiLanguageEnglish,
		PageTitle:            "ThisM Login",
		BrandKicker:          "Secure Access",
		BrandTitle:           "ThisM Console",
		ToggleLabel:          "中文",
		ToggleTarget:         uiLanguageChinese,
		AccessEyebrow:        "Administrator Access",
		LoginTitle:           "Sign in",
		LoginDescription:     "Use your administrator credentials to access the dashboard and node controls.",
		UsernameLabel:        "Username",
		PasswordLabel:        "Password",
		SubmitLabel:          "Sign in",
		GuestLabel:           "Continue as guest",
		GuestDescription:     "Enter a read-only guest view with dashboard cards and basic node details only.",
		MetaLabel:            "Protected session for platform administration",
		InvalidCredentials:   uiMessage(uiLanguageEnglish, "invalidCredentials"),
		LoginFailedMessage:   "Login failed. Please check credentials.",
		RequestFailedMessage: "Request failed. Please retry.",
		GuestFailedMessage:   "Guest access failed. Please retry.",
	}
}

func renderLoginPageHTML(language uiLanguage) string {
	var buf bytes.Buffer
	if err := loginPageTemplate.Execute(&buf, loginPageDataForLanguage(language)); err != nil {
		return ""
	}
	return buf.String()
}

func handleLoginPage(w http.ResponseWriter, r *http.Request, auth *authManager) {
	if resolveAccessRole(r, auth) == accessRoleAdmin {
		auth.BootstrapSessionCookie(w, r)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if !auth.PasswordLoginEnabled() {
		http.Error(w, uiMessage(resolveUILanguage(r), "passwordLoginDisabled"), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(renderLoginPageHTML(resolveUILanguage(r))))
}

func handlePasswordLogin(w http.ResponseWriter, r *http.Request, s *store.Store, auth *authManager) {
	if !auth.PasswordLoginEnabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": uiMessage(resolveUILanguage(r), "passwordLoginDisabled")})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidRequestBody")})
		return
	}

	if !auth.AuthenticatePassword(req.Username, req.Password, func(username, password string) error {
		if s == nil {
			return nil
		}
		return s.UpsertAdminAuth(username, password)
	}) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidCredentials")})
		return
	}

	clearGuestSessionCookie(w, r)
	if err := auth.IssueAdminSession(w, r); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": uiMessage(resolveUILanguage(r), "sessionStartFailed")})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleGuestLogin(w http.ResponseWriter, r *http.Request, auth *authManager) {
	auth.ClearAdminSession(w, r)
	setGuestSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	role := accessRoleFromRequest(r)
	if role == accessRoleNone {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": uiMessage(resolveUILanguage(r), "unauthorized")})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"role": string(role)})
}

func handleChangePassword(w http.ResponseWriter, r *http.Request, s *store.Store, auth *authManager) {
	if !auth.PasswordLoginEnabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": uiMessage(resolveUILanguage(r), "passwordLoginDisabled")})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidRequestBody")})
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "passwordChangeRequired")})
		return
	}
	if req.CurrentPassword == req.NewPassword {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "newPasswordMustDiffer")})
		return
	}

	err := auth.ChangePassword(req.CurrentPassword, req.NewPassword, func(username, password string) error {
		if s == nil {
			return nil
		}
		return s.UpsertAdminAuth(username, password)
	})
	if err != nil {
		if errors.Is(err, errPasswordLoginDisabled) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": uiMessage(resolveUILanguage(r), "passwordLoginDisabled")})
			return
		}
		if errors.Is(err, errInvalidCurrentPass) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidCurrentPassword")})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleGetMetricsRetention(w http.ResponseWriter, r *http.Request, s *store.Store) {
	if s == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	days, err := s.GetMetricsRetentionDays()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retention_days": days,
		"options":        store.MetricsRetentionOptions(),
	})
}

func handleGetNotificationSettings(w http.ResponseWriter, r *http.Request, s *store.Store) {
	if s == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	view, err := s.NotificationSettingsView(false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func handleGetDashboardSettings(w http.ResponseWriter, r *http.Request, s *store.Store) {
	if s == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	settings, err := s.GetDashboardSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func handleUpdateNotificationSettings(w http.ResponseWriter, r *http.Request, s *store.Store) {
	if s == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	current, err := s.GetNotificationSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var reqBody models.NotificationSettings
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidRequestBody")})
		return
	}
	if strings.TrimSpace(reqBody.TelegramBotToken) == "" {
		reqBody.TelegramBotToken = current.TelegramBotToken
	}
	if err := s.UpsertNotificationSettings(reqBody); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	view, err := s.NotificationSettingsView(false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func handleUpdateDashboardSettings(w http.ResponseWriter, r *http.Request, s *store.Store) {
	if s == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}

	var reqBody struct {
		ShowDashboardCardIP *bool `json:"show_dashboard_card_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidRequestBody")})
		return
	}
	if reqBody.ShowDashboardCardIP == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "show_dashboard_card_ip is required"})
		return
	}

	settings := models.DashboardSettings{
		ShowDashboardCardIP: *reqBody.ShowDashboardCardIP,
	}
	if err := s.UpsertDashboardSettings(settings); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func handleSendTestNotification(w http.ResponseWriter, r *http.Request, s *store.Store) {
	if s == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	stored, err := s.GetNotificationSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var reqBody struct {
		TelegramBotToken string                 `json:"telegram_bot_token"`
		Target           *models.TelegramTarget `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidRequestBody")})
		return
	}
	if strings.TrimSpace(reqBody.TelegramBotToken) != "" {
		stored.TelegramBotToken = reqBody.TelegramBotToken
	}
	if reqBody.Target != nil {
		stored.TelegramTargets = []models.TelegramTarget{reqBody.Target.Normalized()}
	}
	if strings.TrimSpace(stored.TelegramBotToken) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "telegram bot token is required"})
		return
	}
	if len(stored.TelegramTargets) == 0 || strings.TrimSpace(stored.TelegramTargets[0].ChatID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one telegram target is required"})
		return
	}
	sender := notify.NewTelegramSender(nil)
	err = sender.Send(stored, models.AlertEvent{
		NodeID:     "test-node",
		NodeName:   "Thism Control Plane",
		Metric:     models.ResourceMetricCPU,
		Severity:   models.AlertSeverityWarning,
		Value:      42,
		Threshold:  80,
		ObservedAt: time.Now().Unix(),
	})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleGetVersionMetadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":    sharedversion.Version,
		"commit":     sharedversion.Commit,
		"build_time": sharedversion.BuildTime,
	})
}

func handleUpdateMetricsRetention(w http.ResponseWriter, r *http.Request, s *store.Store) {
	if s == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	var req struct {
		RetentionDays int `json:"retention_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidRequestBody")})
		return
	}
	if !store.IsValidMetricsRetentionDays(req.RetentionDays) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid metrics retention days"})
		return
	}
	if err := s.SetMetricsRetentionDays(req.RetentionDays); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.PruneOldMetrics(req.RetentionDays); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retention_days": req.RetentionDays,
		"options":        store.MetricsRetentionOptions(),
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request, auth *authManager) {
	auth.ClearAdminSession(w, r)
	clearGuestSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// -----------------------------------------------------------------------
// Token / ID generation
// -----------------------------------------------------------------------

// generateHex returns a random 32-character hex string (16 random bytes).
func generateHex() (string, error) {
	return generateHexBytes(16)
}

func generateHexBytes(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// -----------------------------------------------------------------------
// JSON helpers
// -----------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// -----------------------------------------------------------------------
// REST handlers
// -----------------------------------------------------------------------

func handleListNodes(w http.ResponseWriter, r *http.Request, s *store.Store, h *hub.Hub) {
	nodes, err := s.ListNodes()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	nodeIDs := make([]string, 0, len(nodes))
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.ID)
	}

	latestMetrics, err := s.LatestMetricsByNodeIDs(nodeIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	onlineSet := make(map[string]struct{})
	for _, id := range h.OnlineNodeIDs() {
		onlineSet[id] = struct{}{}
	}

	role := accessRoleFromRequest(r)
	result := make([]*models.Node, 0, len(nodes))
	for _, n := range nodes {
		current := *n
		_, current.Online = onlineSet[current.ID]
		current.LatestMetrics = latestMetrics[current.ID]
		if role == accessRoleGuest {
			current.IP = ""
		}
		result = append(result, &current)
	}

	writeJSON(w, http.StatusOK, map[string]any{"nodes": result})
}

func handleRegisterNode(w http.ResponseWriter, r *http.Request, s *store.Store) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	id, err := generateHex()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
		return
	}
	token, err := generateHex()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
		return
	}

	node := &models.Node{
		ID:        id,
		Name:      req.Name,
		Token:     token,
		CreatedAt: time.Now().Unix(),
	}
	if err := s.UpsertNode(node); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": id, "token": token})
}

func handleUpdateNode(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")

	existing, err := s.GetNodeByID(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if err := s.RenameNode(nodeID, req.Name); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	updated, err := s.GetNodeByID(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if updated == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "node not found after update"})
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func handleDeleteNode(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	existing, err := s.GetNodeByID(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}

	if err := s.DeleteNode(nodeID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleGetInstallCommand(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	node, err := s.GetNodeByID(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if node == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"command": buildInstallCommand(r, node.Token, node.Name),
	})
}

type createAgentUpdateJobRequest struct {
	NodeIDs       []string `json:"node_ids"`
	TargetVersion string   `json:"target_version"`
	DownloadURL   string   `json:"download_url"`
	SHA256        string   `json:"sha256"`
}

type updateJobResponse struct {
	Job     *models.UpdateJob         `json:"job"`
	Targets []*models.UpdateJobTarget `json:"targets"`
}

func decodeWSPayload[T any](payload any) (T, error) {
	var out T
	raw, err := json.Marshal(payload)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(raw, &out)
	return out, err
}

func handleAgentCommandStatus(nodeID string, payload models.AgentCommandStatusPayload, s *store.Store, h *hub.Hub) {
	if s == nil || strings.TrimSpace(payload.JobID) == "" {
		return
	}
	_ = s.UpdateUpdateJobTargetStatus(payload.JobID, nodeID, payload.Status, payload.Message, payload.ReportedVersion)
	h.Broadcast(models.WSMessage{
		Type: "agent_update_status",
		Payload: map[string]any{
			"job_id":           payload.JobID,
			"node_id":          nodeID,
			"status":           payload.Status,
			"message":          payload.Message,
			"reported_version": payload.ReportedVersion,
		},
	})
}

func handleCreateAgentUpdateJob(w http.ResponseWriter, r *http.Request, s *store.Store, h *hub.Hub) {
	var req createAgentUpdateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidRequestBody")})
		return
	}
	if s == nil || len(req.NodeIDs) == 0 || strings.TrimSpace(req.TargetVersion) == "" || strings.TrimSpace(req.DownloadURL) == "" || strings.TrimSpace(req.SHA256) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": uiMessage(resolveUILanguage(r), "invalidRequestBody")})
		return
	}
	for _, nodeID := range req.NodeIDs {
		node, err := s.GetNodeByID(nodeID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if node == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
			return
		}
	}
	jobID, err := generateHex()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	now := time.Now().Unix()
	job := &models.UpdateJob{
		ID:            jobID,
		Kind:          models.AgentCommandKindSelfUpdate,
		TargetVersion: req.TargetVersion,
		DownloadURL:   req.DownloadURL,
		SHA256:        req.SHA256,
		CreatedAt:     now,
		UpdatedAt:     now,
		CreatedBy:     "admin",
		Status:        models.UpdateJobStatusPending,
	}
	if err := s.CreateUpdateJob(job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.CreateUpdateJobTargets(job.ID, req.NodeIDs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	for _, nodeID := range req.NodeIDs {
		if !h.IsOnline(nodeID) {
			_ = s.UpdateUpdateJobTargetStatus(job.ID, nodeID, models.UpdateJobTargetStatusOfflineSkipped, "agent offline", "")
			continue
		}
		cmd := models.AgentCommandPayload{
			JobID:         job.ID,
			Kind:          models.AgentCommandKindSelfUpdate,
			TargetVersion: req.TargetVersion,
			DownloadURL:   req.DownloadURL,
			SHA256:        req.SHA256,
		}
		if err := h.SendToAgent(nodeID, models.WSMessage{Type: "agent_command", Payload: cmd}); err != nil {
			_ = s.UpdateUpdateJobTargetStatus(job.ID, nodeID, models.UpdateJobTargetStatusFailed, err.Error(), "")
			continue
		}
		_ = s.UpdateUpdateJobTargetStatus(job.ID, nodeID, models.UpdateJobTargetStatusDispatched, "command dispatched", "")
	}
	storedJob, err := s.GetUpdateJob(job.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	targets, err := s.ListUpdateJobTargets(job.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updateJobResponse{Job: storedJob, Targets: targets})
}

func handleGetAgentUpdateJob(w http.ResponseWriter, r *http.Request, s *store.Store) {
	jobID := chi.URLParam(r, "id")
	job, err := s.GetUpdateJob(jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if job == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}
	targets, err := s.ListUpdateJobTargets(jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updateJobResponse{Job: job, Targets: targets})
}

func handleCreateAgentUpdates(w http.ResponseWriter, r *http.Request, s *store.Store, h *hub.Hub) {
	if s == nil || h == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store unavailable"})
		return
	}
	var req struct {
		NodeIDs       []string `json:"node_ids"`
		TargetVersion string   `json:"target_version"`
		DownloadURL   string   `json:"download_url"`
		SHA256        string   `json:"sha256"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	nodeIDs := make([]string, 0, len(req.NodeIDs))
	seen := map[string]struct{}{}
	for _, nodeID := range req.NodeIDs {
		nodeID = strings.TrimSpace(nodeID)
		if nodeID == "" {
			continue
		}
		if _, ok := seen[nodeID]; ok {
			continue
		}
		seen[nodeID] = struct{}{}
		nodeIDs = append(nodeIDs, nodeID)
	}
	if len(nodeIDs) == 0 || strings.TrimSpace(req.TargetVersion) == "" || strings.TrimSpace(req.DownloadURL) == "" || strings.TrimSpace(req.SHA256) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	jobID, err := generateHex()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	now := time.Now().Unix()
	job := &models.UpdateJob{ID: jobID, Kind: models.AgentCommandKindSelfUpdate, TargetVersion: strings.TrimSpace(req.TargetVersion), DownloadURL: strings.TrimSpace(req.DownloadURL), SHA256: strings.TrimSpace(req.SHA256), CreatedAt: now, UpdatedAt: now, CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if err := s.CreateUpdateJob(job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := s.CreateUpdateJobTargets(job.ID, nodeIDs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	for _, nodeID := range nodeIDs {
		node, err := s.GetNodeByID(nodeID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if node == nil || !h.IsOnline(nodeID) {
			_ = s.UpdateUpdateJobTargetStatus(job.ID, nodeID, models.UpdateJobTargetStatusOfflineSkipped, "node offline", "")
			continue
		}
		msg := models.WSMessage{Type: "agent_command", Payload: models.AgentCommandPayload{JobID: job.ID, Kind: models.AgentCommandKindSelfUpdate, TargetVersion: job.TargetVersion, DownloadURL: job.DownloadURL, SHA256: job.SHA256}}
		if err := h.SendToAgent(nodeID, msg); err != nil {
			_ = s.UpdateUpdateJobTargetStatus(job.ID, nodeID, models.UpdateJobTargetStatusOfflineSkipped, err.Error(), "")
			continue
		}
		_ = s.UpdateUpdateJobTargetStatus(job.ID, nodeID, models.UpdateJobTargetStatusDispatched, "sent", "")
	}
	storedJob, _ := s.GetUpdateJob(job.ID)
	targets, _ := s.ListUpdateJobTargets(job.ID)
	writeJSON(w, http.StatusOK, map[string]any{"job": storedJob, "targets": targets})
}

func handleGetAgentUpdate(w http.ResponseWriter, r *http.Request, s *store.Store) {
	jobID := chi.URLParam(r, "id")
	job, err := s.GetUpdateJob(jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if job == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	targets, err := s.ListUpdateJobTargets(jobID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job, "targets": targets})
}

func handleGetMetrics(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	now := time.Now().Unix()
	from := now - 3600 // default: last hour
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			from = parsed
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			to = parsed
		}
	}

	resolution := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("resolution")))
	if resolution == "" {
		resolution = "auto"
	}

	span := to - from
	use1m := false
	switch resolution {
	case "1m":
		use1m = true
	case "raw":
		use1m = false
	case "auto":
		// Switch to 1m when the range is large to avoid returning too many points.
		if span > int64((6 * time.Hour).Seconds()) {
			use1m = true
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid resolution"})
		return
	}

	var (
		rows []*store.MetricsRow
		err  error
	)
	metaResolution := "raw"
	if use1m {
		rows, err = s.QueryMetrics1m(nodeID, from, to)
		metaResolution = "1m"
	} else {
		rows, err = s.QueryMetrics(nodeID, from, to)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []*store.MetricsRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"metrics": rows,
		"meta": map[string]any{
			"resolution": metaResolution,
		},
	})
}

func handleGetProcesses(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	data, err := s.GetProcesses(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// data is already a JSON string; write it raw.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(data))
}

func handleGetDocker(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	available, data, err := s.GetDockerContainers(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"docker_available": available,
		"containers":       json.RawMessage(data),
	})
}

func handleGetServices(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	checks, err := s.GetServiceChecks(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if checks == nil {
		checks = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": checks})
}

func buildInstallCommand(r *http.Request, token, name string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	baseURL := scheme + "://" + r.Host

	query := url.Values{}
	query.Set("name", name)
	scriptURL := baseURL + "/install.sh?" + query.Encode()
	return `curl -fsSL -H "Authorization: Bearer ` + token + `" "` + scriptURL + `" | bash`
}

// -----------------------------------------------------------------------
// Agent installation handlers
// -----------------------------------------------------------------------

func handleInstallScript(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	name := r.URL.Query().Get("name")
	if token == "" || name == "" {
		http.Error(w, "authorization bearer token and name query param required", http.StatusBadRequest)
		return
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	host := r.Host
	baseURL := scheme + "://" + host
	targetVersion := sharedversion.Version
	if strings.TrimSpace(targetVersion) == "" {
		targetVersion = "dev"
	}

	script := "#!/bin/bash\nset -e\n\n" +
		"TOKEN=\"" + token + "\"\n" +
		"NAME=\"" + name + "\"\n" +
		"BASE=\"" + baseURL + "\"\n\n" +
		"TARGET_BIN=\"/usr/local/bin/thism-agent\"\n" +
		"VERSION_FILE=\"/usr/local/bin/.thism-agent.version\"\n" +
		"TMP_BIN=\"/usr/local/bin/.thism-agent.$$\"\n" +
		"trap 'rm -f \"${TMP_BIN}\"' EXIT\n\n" +
		"ARCH=$(uname -m)\n" +
		"case \"$ARCH\" in\n" +
		"  x86_64|amd64) ARCH=\"amd64\" ;;\n" +
		"  aarch64|arm64) ARCH=\"arm64\" ;;\n" +
		"  *) echo \"Unsupported architecture: $ARCH\"; exit 1 ;;\n" +
		"esac\n\n" +
		"OS=$(uname -s | tr '[:upper:]' '[:lower:]')\n" +
		"if [ \"$OS\" != \"linux\" ]; then\n" +
		"  echo \"Unsupported OS: $OS (only linux is supported)\"\n" +
		"  exit 1\n" +
		"fi\n\n" +
		"BINARY=\"thism-agent-${OS}-${ARCH}\"\n" +
		"echo \"Downloading ${BINARY}...\"\n" +
		"curl -fsSL \"${BASE}/dl/${BINARY}\" -o \"${TMP_BIN}\"\n" +
		"chmod +x \"${TMP_BIN}\"\n" +
		"mv -f \"${TMP_BIN}\" \"${TARGET_BIN}\"\n" +
		"TARGET_VERSION=\"" + targetVersion + "\"\n" +
		"printf \"%s\\n\" \"${TARGET_VERSION}\" > \"${VERSION_FILE}\"\n" +
		"trap - EXIT\n\n" +
		"WS_SCHEME=\"ws\"\n" +
		"case \"$BASE\" in\n" +
		"  https://*) WS_SCHEME=\"wss\" ;;\n" +
		"esac\n" +
		"WS_HOST=$(echo \"$BASE\" | sed 's|^https\\?://||')\n\n" +
		"cat > /etc/systemd/system/thism-agent.service <<UNIT\n" +
		"[Unit]\n" +
		"Description=ThisM Agent\n" +
		"After=network-online.target\n" +
		"Wants=network-online.target\n\n" +
		"[Service]\n" +
		"ExecStart=/usr/local/bin/thism-agent --server ${WS_SCHEME}://${WS_HOST} --token ${TOKEN} --name ${NAME}\n" +
		"Restart=always\n" +
		"RestartSec=5\n\n" +
		"[Install]\n" +
		"WantedBy=multi-user.target\n" +
		"UNIT\n\n" +
		"systemctl daemon-reload\n" +
		"systemctl enable thism-agent\n" +
		"systemctl restart thism-agent\n" +
		"echo \"thisM agent installed and started successfully.\"\n"

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(script))
}

func handleAgentRelease(w http.ResponseWriter, r *http.Request) {
	osName := strings.TrimSpace(r.URL.Query().Get("os"))
	arch := strings.TrimSpace(r.URL.Query().Get("arch"))
	filename, ok := resolveAgentBinaryFilename(osName, arch)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	filePath, err := resolveAgentBinaryPath(filename)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	digest := sha256.Sum256(raw)
	checksum := strings.ToLower(hex.EncodeToString(digest[:]))
	targetVersion := sharedversion.Version
	if strings.TrimSpace(targetVersion) == "" {
		targetVersion = "dev"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"target_version":         targetVersion,
		"download_url":           buildDownloadURL(r, filename),
		"sha256":                 checksum,
		"check_interval_seconds": int((30 * time.Minute).Seconds()),
	})
}

func resolveAgentBinaryFilename(osName, arch string) (string, bool) {
	if osName != "linux" {
		return "", false
	}
	switch arch {
	case "amd64":
		return "thism-agent-linux-amd64", true
	case "arm64":
		return "thism-agent-linux-arm64", true
	default:
		return "", false
	}
}

func resolveAgentBinaryPath(filename string) (string, error) {
	candidates := []string{
		"dist/" + filename,
		"../dist/" + filename,
		"../../dist/" + filename,
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", os.ErrNotExist
}

func buildBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	return scheme + "://" + r.Host
}

func buildDownloadURL(r *http.Request, filename string) string {
	return buildBaseURL(r) + "/dl/" + filename
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if _, ok := resolveAgentBinaryFilename("linux", strings.TrimPrefix(filename, "thism-agent-linux-")); !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	filepath := "dist/" + filename
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	http.ServeFile(w, r, filepath)
}

// -----------------------------------------------------------------------
// WebSocket handlers
// -----------------------------------------------------------------------

func handleAgentWS(w http.ResponseWriter, r *http.Request, s *store.Store, h *hub.Hub) {
	alertEvaluator := &alerting.Evaluator{Store: s, Sender: notify.NewTelegramSender(nil)}
	token := r.URL.Query().Get("token")
	if token == "" {
		// Also accept bearer token in header for agent connections.
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if token == "" {
		http.Error(w, uiMessage(resolveUILanguage(r), "tokenRequired"), http.StatusUnauthorized)
		return
	}

	node, err := s.GetNodeByToken(token)
	if err != nil || node == nil {
		http.Error(w, uiMessage(resolveUILanguage(r), "invalidToken"), http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.Register(node.ID, conn)
	_ = alertEvaluator.ProcessHeartbeat(node, true, time.Now().Unix())
	defer func() {
		_ = alertEvaluator.ProcessHeartbeat(node, false, time.Now().Unix())
		conn.Close()
		h.Unregister(node.ID)
	}()

	// Read loop: parse incoming agent messages and persist them.
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var envelope models.WSMessage
		if err := json.Unmarshal(msg, &envelope); err != nil {
			continue
		}

		if envelope.Type == "agent_command_status" {
			statusPayload, err := decodeWSPayload[models.AgentCommandStatusPayload](envelope.Payload)
			if err != nil {
				continue
			}
			handleAgentCommandStatus(node.ID, statusPayload, s, h)
			continue
		}

		var payload models.MetricsPayload
		if err := json.Unmarshal(msg, &payload); err != nil {
			continue
		}

		lastSeen := time.Now().Unix()

		// Persist metrics before broadcasting them so the dashboard does not
		// show data that failed to land in storage.
		if err := s.InsertMetrics(node.ID, &payload); err != nil {
			log.Printf("agent metrics: persist sample for node %s failed: %v", node.ID, err)
			continue
		}
		if err := s.UpdateNodeMetadata(node.ID, resolveNodeIP(r, payload.IP), payload.OS, payload.Arch, payload.AgentVersion, payload.Hardware, lastSeen); err != nil {
			log.Printf("agent metrics: update node metadata for node %s failed: %v", node.ID, err)
			continue
		}
		if strings.TrimSpace(payload.AgentVersion) != "" {
			if err := s.FinalizeUpdateJobsForNodeVersion(node.ID, payload.AgentVersion); err != nil {
				log.Printf("agent metrics: finalize update jobs for node %s failed: %v", node.ID, err)
			}
		}

		// Persist processes as a JSON string.
		if len(payload.Processes) > 0 {
			procJSON, err := json.Marshal(payload.Processes)
			if err == nil {
				if err := s.UpsertProcesses(node.ID, payload.TS, string(procJSON)); err != nil {
					log.Printf("agent metrics: persist processes for node %s failed: %v", node.ID, err)
				}
			} else {
				log.Printf("agent metrics: marshal processes for node %s failed: %v", node.ID, err)
			}
		}

		// Persist service checks.
		for _, svc := range payload.Services {
			if err := s.UpsertServiceCheck(node.ID, svc.Name, svc.Status); err != nil {
				log.Printf("agent metrics: persist service check %q for node %s failed: %v", svc.Name, node.ID, err)
			}
		}

		// Persist docker availability and container snapshot when the agent reports it.
		if payload.DockerAvailable != nil {
			available := *payload.DockerAvailable
			containers := payload.Containers
			if containers == nil || !available {
				containers = []models.DockerContainer{}
			}
			containersJSON, err := json.Marshal(containers)
			if err == nil {
				if err := s.UpsertDockerContainers(node.ID, payload.TS, available, string(containersJSON)); err != nil {
					log.Printf("agent metrics: persist docker snapshot for node %s failed: %v", node.ID, err)
				}
			} else {
				log.Printf("agent metrics: marshal docker snapshot for node %s failed: %v", node.ID, err)
			}
		}

		if err := alertEvaluator.Process(node, &payload); err != nil {
			log.Printf("agent metrics: evaluate alerts for node %s failed: %v", node.ID, err)
		}

		// Broadcast metrics to dashboard subscribers, wrapped with node_id.
		h.Broadcast(models.WSMessage{
			Type: "metrics",
			Payload: map[string]any{
				"node_id":   node.ID,
				"last_seen": lastSeen,
				"data":      payload,
			},
		})
	}
}

func resolveNodeIP(r *http.Request, payloadIP string) string {
	payload := strings.TrimSpace(payloadIP)
	forwarded := forwardedForIP(r.Header.Get("X-Forwarded-For"))
	remote := remoteAddrIP(r.RemoteAddr)

	publicIPv4 := []string{payload, forwarded, remote}
	for _, candidate := range publicIPv4 {
		if isPublicIPv4(candidate) {
			return candidate
		}
	}

	usableIPv4 := []string{payload, forwarded, remote}
	for _, candidate := range usableIPv4 {
		if isUsableIPv4(candidate) {
			return candidate
		}
	}

	publicAny := []string{forwarded, remote, payload}
	for _, candidate := range publicAny {
		if isPublicIP(candidate) {
			return candidate
		}
	}

	usableAny := []string{payload, forwarded, remote}
	for _, candidate := range usableAny {
		if isUsableIP(candidate) {
			return candidate
		}
	}

	return ""
}

func forwardedForIP(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func remoteAddrIP(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(trimmed)
	if err == nil && host != "" {
		return host
	}
	return trimmed
}

func sameOriginWebsocketRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return false
	}
	return strings.EqualFold(originURL.Host, r.Host)
}

func isPublicIP(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return false
	}
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || isCarrierGradeNAT(ip) {
		return false
	}
	return true
}

func isPublicIPv4(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && ip.To4() != nil && isPublicIP(value)
}

func isUsableIP(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return false
	}
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsMulticast() {
		return false
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	return true
}

func isUsableIPv4(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && ip.To4() != nil && isUsableIP(value)
}

func isCarrierGradeNAT(ip net.IP) bool {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return false
	}
	// RFC 6598: 100.64.0.0/10
	return ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127
}

func handleDashboardWS(w http.ResponseWriter, r *http.Request, s *store.Store, h *hub.Hub) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan models.WSMessage, 32)
	h.Subscribe(ch)
	defer h.Unsubscribe(ch)

	initialMessages, err := dashboardInitialMessages(s, h)
	if err != nil {
		return
	}

	for _, msg := range initialMessages {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}

	for msg := range ch {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
	}
}

func dashboardInitialMessages(s *store.Store, h *hub.Hub) ([]models.WSMessage, error) {
	nodes, err := s.ListNodes()
	if err != nil {
		return nil, err
	}

	nodeIDs := make([]string, 0, len(nodes))
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.ID)
	}

	latestMetrics, err := s.LatestMetricsByNodeIDs(nodeIDs)
	if err != nil {
		return nil, err
	}

	onlineSet := make(map[string]struct{}, len(h.OnlineNodeIDs()))
	for _, id := range h.OnlineNodeIDs() {
		onlineSet[id] = struct{}{}
	}

	messages := make([]models.WSMessage, 0, len(nodes)*2)
	for _, node := range nodes {
		_, online := onlineSet[node.ID]
		messages = append(messages, models.WSMessage{
			Type: "node_status",
			Payload: map[string]any{
				"node_id": node.ID,
				"online":  online,
			},
		})

		snapshot := latestMetrics[node.ID]
		if snapshot == nil {
			continue
		}

		messages = append(messages, models.WSMessage{
			Type: "metrics",
			Payload: map[string]any{
				"node_id":   node.ID,
				"last_seen": node.LastSeen,
				"data": map[string]any{
					"ts":  snapshot.TS,
					"cpu": snapshot.CPU,
					"mem": map[string]any{
						"used":  snapshot.MemUsed,
						"total": snapshot.MemTotal,
					},
					"net": map[string]any{
						"rx_bytes": snapshot.NetRx,
						"tx_bytes": snapshot.NetTx,
					},
					"disk_used":  snapshot.DiskUsed,
					"disk_total": snapshot.DiskTotal,
				},
			},
		})
	}

	return messages, nil
}
