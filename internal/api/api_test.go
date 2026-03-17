package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
	_ "modernc.org/sqlite"
)

func TestGetNodesEmpty(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "test-admin-token", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["nodes"] == nil {
		t.Error("expected 'nodes' key in response")
	}
}

func TestGetNodesUnauthorized(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "test-admin-token", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetNodesIncludesLatestMetricsSnapshot(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "test-admin-token", nil)

	node := &models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token",
		CreatedAt: time.Now().Unix(),
		LastSeen:  time.Now().Unix(),
	}
	if err := s.UpsertNode(node); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	if err := s.InsertMetrics("node-1", &models.MetricsPayload{
		TS:            1733011200,
		CPU:           37.5,
		UptimeSeconds: 3723,
		Mem:           models.MemStats{Used: 2048, Total: 4096},
		Net:           models.NetStats{RxBytes: 1234, TxBytes: 5678},
	}); err != nil {
		t.Fatalf("InsertMetrics: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Nodes []struct {
			ID            string            `json:"id"`
			LatestMetrics *store.MetricsRow `json:"latest_metrics"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(body.Nodes))
	}
	if body.Nodes[0].ID != "node-1" {
		t.Fatalf("expected node id node-1, got %q", body.Nodes[0].ID)
	}
	if body.Nodes[0].LatestMetrics == nil {
		t.Fatal("expected latest_metrics to be included")
	}
	if body.Nodes[0].LatestMetrics.CPU != 37.5 {
		t.Fatalf("expected latest CPU 37.5, got %v", body.Nodes[0].LatestMetrics.CPU)
	}
	if body.Nodes[0].LatestMetrics.MemUsed != 2048 || body.Nodes[0].LatestMetrics.MemTotal != 4096 {
		t.Fatalf("unexpected latest memory snapshot: %#v", body.Nodes[0].LatestMetrics)
	}
	if body.Nodes[0].LatestMetrics.UptimeSeconds != 3723 {
		t.Fatalf("expected latest uptime 3723, got %d", body.Nodes[0].LatestMetrics.UptimeSeconds)
	}
}

func TestGetNodesIncludesHardwareSnapshot(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "test-admin-token", nil)

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "hardware-node",
		Token:     "hardware-token",
		CreatedAt: time.Now().Unix(),
		Hardware: &models.NodeHardware{
			CPUModel:             "AMD EPYC 7B13",
			CPUCores:             8,
			CPUThreads:           16,
			MemoryTotal:          34359738368,
			DiskTotal:            322122547200,
			VirtualizationSystem: "kvm",
			VirtualizationRole:   "guest",
		},
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Nodes []struct {
			ID       string               `json:"id"`
			Hardware *models.NodeHardware `json:"hardware"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(body.Nodes))
	}
	if body.Nodes[0].Hardware == nil {
		t.Fatal("expected hardware snapshot to be included")
	}
	if body.Nodes[0].Hardware.CPUModel != "AMD EPYC 7B13" {
		t.Fatalf("expected cpu model AMD EPYC 7B13, got %q", body.Nodes[0].Hardware.CPUModel)
	}
}

func TestInstallScriptUsesTempBinarySwap(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "test-admin-token", nil)

	req := httptest.NewRequest(http.MethodGet, "/install.sh?token=node-token-1&name=Bitsflow", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	script := w.Body.String()
	if !strings.Contains(script, `TARGET_BIN="/usr/local/bin/thism-agent"`) {
		t.Fatalf("expected install script to define target binary path, got: %s", script)
	}
	if !strings.Contains(script, `VERSION_FILE="/usr/local/bin/.thism-agent.version"`) {
		t.Fatalf("expected install script to define version file path, got: %s", script)
	}
	if !strings.Contains(script, `TMP_BIN="/usr/local/bin/.thism-agent.$$"`) {
		t.Fatalf("expected install script to define temp binary path, got: %s", script)
	}
	if !strings.Contains(script, `curl -fsSL "${BASE}/dl/${BINARY}" -o "${TMP_BIN}"`) {
		t.Fatalf("expected install script to download to temp binary, got: %s", script)
	}
	if !strings.Contains(script, `mv -f "${TMP_BIN}" "${TARGET_BIN}"`) {
		t.Fatalf("expected install script to atomically move temp binary into place, got: %s", script)
	}
	if !strings.Contains(script, `TARGET_VERSION=$(sha256sum "${TARGET_BIN}" | awk '{print substr($1,1,12)}')`) {
		t.Fatalf("expected install script to derive target version from installed binary checksum, got: %s", script)
	}
	if !strings.Contains(script, `printf "%s\n" "${TARGET_VERSION}" > "${VERSION_FILE}"`) {
		t.Fatalf("expected install script to persist installed agent version, got: %s", script)
	}
	if strings.Contains(script, `curl -fsSL "${BASE}/dl/${BINARY}" -o /usr/local/bin/thism-agent`) {
		t.Fatalf("expected install script to avoid writing binary directly in place, got: %s", script)
	}
	if !strings.Contains(script, `systemctl enable thism-agent`) {
		t.Fatalf("expected install script to enable service explicitly, got: %s", script)
	}
	if !strings.Contains(script, `systemctl restart thism-agent`) {
		t.Fatalf("expected install script to restart existing service, got: %s", script)
	}
	if strings.Contains(script, `systemctl enable --now thism-agent`) {
		t.Fatalf("expected install script to avoid enable --now because it won't restart an already-running service, got: %s", script)
	}
}

func TestFrontendRequiresAdminAuth(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	frontend := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("frontend"))
	})
	router := api.NewRouter(s, h, "admin-token", frontend)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated frontend request, got %d", w.Code)
	}
}

func TestFrontendQueryTokenCreatesSessionCookie(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	frontend := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("frontend"))
	})
	router := api.NewRouter(s, h, "admin-token", frontend)

	firstReq := httptest.NewRequest(http.MethodGet, "/?token=admin-token", nil)
	firstResp := httptest.NewRecorder()
	router.ServeHTTP(firstResp, firstReq)

	if firstResp.Code != http.StatusFound {
		t.Fatalf("expected redirect when query token is provided, got %d", firstResp.Code)
	}
	if location := firstResp.Header().Get("Location"); location != "/" {
		t.Fatalf("expected redirect location '/', got %q", location)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range firstResp.Result().Cookies() {
		if cookie.Name == "thism_admin" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected thism_admin cookie to be set")
	}
	if sessionCookie.Value == "" {
		t.Fatal("expected thism_admin cookie to have an opaque session value")
	}
	if sessionCookie.Value == "admin-token" {
		t.Fatalf("expected thism_admin cookie to avoid exposing the admin token, got %q", sessionCookie.Value)
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/", nil)
	secondReq.AddCookie(sessionCookie)
	secondResp := httptest.NewRecorder()
	router.ServeHTTP(secondResp, secondReq)

	if secondResp.Code != http.StatusOK {
		t.Fatalf("expected authenticated frontend request to return 200, got %d", secondResp.Code)
	}
	if strings.TrimSpace(secondResp.Body.String()) != "frontend" {
		t.Fatalf("expected frontend handler body, got %q", secondResp.Body.String())
	}
}

func TestAdminQueryTokenDoesNotAuthorizeAPIRequests(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "admin-token", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/nodes?token=admin-token", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected query token to be rejected for admin API access, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestLoginPageAccessibleWhenPasswordAuthConfigured(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for login page, got %d", w.Code)
	}
	if !strings.Contains(strings.ToLower(w.Body.String()), "sign in") {
		t.Fatalf("expected login page body to contain sign in text, got %q", w.Body.String())
	}
}

func TestLoginPageUsesStableViewportHeightForCenteredCard(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for login page, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "min-height: 100svh;") {
		t.Fatalf("expected login page to use stable viewport height to avoid layout jumps, got %q", w.Body.String())
	}
}

func TestLoginPageUsesConsoleVisualLanguage(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for login page, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "ThisM Console") {
		t.Fatalf("expected login page to contain console brand text, got %q", body)
	}
	if !strings.Contains(body, "font-family: \"Outfit\", \"Geist\"") {
		t.Fatalf("expected login page to use app font stack, got %q", body)
	}
	if !strings.Contains(body, "@media (prefers-color-scheme: dark)") {
		t.Fatalf("expected login page to include dark mode styles, got %q", body)
	}
}

func TestLoginPageUsesSelectedLanguage(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{Name: "thism-lang", Value: "zh-CN"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for login page, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "登录") {
		t.Fatalf("expected login page to render Chinese copy, got %q", body)
	}
	if !strings.Contains(body, "English") {
		t.Fatalf("expected login page to render language toggle target label, got %q", body)
	}
}

func TestLoginPageRendersChineseWhenLanguageCookieSet(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{Name: "thism-lang", Value: "zh-CN"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for login page, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "登录") {
		t.Fatalf("expected chinese login page copy, got %q", body)
	}
	if !strings.Contains(body, "用户名") {
		t.Fatalf("expected chinese username label, got %q", body)
	}
}

func TestPasswordLoginCreatesSessionCookie(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"username":"admin","password":"secret-pass"}`),
	)
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)

	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for password login, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var sessionCookie *http.Cookie
	for _, cookie := range loginResp.Result().Cookies() {
		if cookie.Name == "thism_admin" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected thism_admin cookie to be set on successful password login")
	}
	if sessionCookie.Value == "" {
		t.Fatal("expected thism_admin cookie to have an opaque session value")
	}
	if sessionCookie.Value == "admin-token" {
		t.Fatalf("expected thism_admin cookie to avoid exposing the admin token, got %q", sessionCookie.Value)
	}

	frontendReq := httptest.NewRequest(http.MethodGet, "/", nil)
	frontendReq.AddCookie(sessionCookie)
	frontendResp := httptest.NewRecorder()
	router.ServeHTTP(frontendResp, frontendReq)

	if frontendResp.Code != http.StatusOK {
		t.Fatalf("expected authenticated frontend request to return 200, got %d", frontendResp.Code)
	}
}

func TestPasswordLoginRejectsInvalidCredentials(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"username":"admin","password":"wrong"}`),
	)
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)

	if loginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid credentials, got %d", loginResp.Code)
	}
}

func TestPasswordLoginRejectsInvalidCredentialsInChinese(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"username":"admin","password":"wrong"}`),
	)
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.AddCookie(&http.Cookie{Name: "thism-lang", Value: "zh-CN"})
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)

	if loginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid credentials, got %d", loginResp.Code)
	}
	if !strings.Contains(loginResp.Body.String(), "凭证") {
		t.Fatalf("expected Chinese invalid credentials error, got %q", loginResp.Body.String())
	}
}

func TestFrontendUnauthorizedHTMLRedirectsToLogin(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusFound {
		t.Fatalf("expected 302 for unauthenticated HTML frontend request, got %d", resp.Code)
	}
	if location := resp.Header().Get("Location"); location != "/login" {
		t.Fatalf("expected redirect to /login, got %q", location)
	}
}

func TestLoginPageIncludesGuestModeEntry(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for login page, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "/api/auth/guest") {
		t.Fatalf("expected login page to wire guest login action, got %q", body)
	}
	if !strings.Contains(strings.ToLower(body), "guest") {
		t.Fatalf("expected login page to expose guest mode copy, got %q", body)
	}
}

func TestGuestSessionCanAccessFrontendAndGetsRedactedNodeData(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "edge-1",
		Token:     "node-token-1",
		IP:        "10.0.0.9",
		OS:        "linux",
		Arch:      "amd64",
		CreatedAt: time.Now().Unix(),
		LastSeen:  time.Now().Unix(),
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	guestReq := httptest.NewRequest(http.MethodPost, "/api/auth/guest", nil)
	guestResp := httptest.NewRecorder()
	router.ServeHTTP(guestResp, guestReq)

	if guestResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for guest login, got %d: %s", guestResp.Code, guestResp.Body.String())
	}

	var guestCookie *http.Cookie
	for _, cookie := range guestResp.Result().Cookies() {
		if cookie.Name == "thism_guest" {
			guestCookie = cookie
			break
		}
	}
	if guestCookie == nil {
		t.Fatal("expected thism_guest cookie to be set on guest login")
	}

	frontendReq := httptest.NewRequest(http.MethodGet, "/", nil)
	frontendReq.Header.Set("Accept", "text/html")
	frontendReq.AddCookie(guestCookie)
	frontendResp := httptest.NewRecorder()
	router.ServeHTTP(frontendResp, frontendReq)

	if frontendResp.Code != http.StatusOK {
		t.Fatalf("expected guest frontend request to return 200, got %d", frontendResp.Code)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sessionReq.AddCookie(guestCookie)
	sessionResp := httptest.NewRecorder()
	router.ServeHTTP(sessionResp, sessionReq)

	if sessionResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for guest session status, got %d: %s", sessionResp.Code, sessionResp.Body.String())
	}
	if !strings.Contains(sessionResp.Body.String(), `"role":"guest"`) {
		t.Fatalf("expected guest session response, got %q", sessionResp.Body.String())
	}

	nodesReq := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	nodesReq.AddCookie(guestCookie)
	nodesResp := httptest.NewRecorder()
	router.ServeHTTP(nodesResp, nodesReq)

	if nodesResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for guest nodes request, got %d: %s", nodesResp.Code, nodesResp.Body.String())
	}

	var body struct {
		Nodes []struct {
			Name string `json:"name"`
			IP   string `json:"ip"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(nodesResp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal guest nodes: %v", err)
	}
	if len(body.Nodes) != 1 {
		t.Fatalf("expected 1 guest-visible node, got %d", len(body.Nodes))
	}
	if body.Nodes[0].IP != "" {
		t.Fatalf("expected guest node IP to be redacted, got %q", body.Nodes[0].IP)
	}
}

func TestGuestSessionCannotAccessProcesses(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "edge-1",
		Token:     "node-token-1",
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	guestReq := httptest.NewRequest(http.MethodPost, "/api/auth/guest", nil)
	guestResp := httptest.NewRecorder()
	router.ServeHTTP(guestResp, guestReq)

	var guestCookie *http.Cookie
	for _, cookie := range guestResp.Result().Cookies() {
		if cookie.Name == "thism_guest" {
			guestCookie = cookie
			break
		}
	}
	if guestCookie == nil {
		t.Fatal("expected thism_guest cookie to be set on guest login")
	}

	processReq := httptest.NewRequest(http.MethodGet, "/api/nodes/node-1/processes", nil)
	processReq.AddCookie(guestCookie)
	processResp := httptest.NewRecorder()
	router.ServeHTTP(processResp, processReq)

	if processResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for guest process access, got %d: %s", processResp.Code, processResp.Body.String())
	}
}

func TestChangePasswordUpdatesLoginCredentials(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret-pass"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for initial login, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var sessionCookie *http.Cookie
	for _, cookie := range loginResp.Result().Cookies() {
		if cookie.Name == "thism_admin" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected thism_admin cookie to be set on login")
	}

	changeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/change-password",
		strings.NewReader(`{"current_password":"secret-pass","new_password":"new-pass-123"}`),
	)
	changeReq.Header.Set("Content-Type", "application/json")
	changeReq.AddCookie(sessionCookie)
	changeResp := httptest.NewRecorder()
	router.ServeHTTP(changeResp, changeReq)
	if changeResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for password change, got %d: %s", changeResp.Code, changeResp.Body.String())
	}

	oldLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret-pass"}`))
	oldLoginReq.Header.Set("Content-Type", "application/json")
	oldLoginResp := httptest.NewRecorder()
	router.ServeHTTP(oldLoginResp, oldLoginReq)
	if oldLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected old password to be rejected after change, got %d", oldLoginResp.Code)
	}

	newLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"new-pass-123"}`))
	newLoginReq.Header.Set("Content-Type", "application/json")
	newLoginResp := httptest.NewRecorder()
	router.ServeHTTP(newLoginResp, newLoginReq)
	if newLoginResp.Code != http.StatusOK {
		t.Fatalf("expected new password to be accepted, got %d: %s", newLoginResp.Code, newLoginResp.Body.String())
	}
}

func TestChangePasswordRejectsInvalidCurrentPassword(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	changeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/change-password",
		strings.NewReader(`{"current_password":"wrong","new_password":"new-pass-123"}`),
	)
	changeReq.Header.Set("Authorization", "Bearer admin-token")
	changeReq.Header.Set("Content-Type", "application/json")
	changeResp := httptest.NewRecorder()
	router.ServeHTTP(changeResp, changeReq)

	if changeResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when current password is invalid, got %d", changeResp.Code)
	}
}

func TestChangePasswordPersistsAcrossRouterRebuild(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	changeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/change-password",
		strings.NewReader(`{"current_password":"secret-pass","new_password":"persisted-pass"}`),
	)
	changeReq.Header.Set("Authorization", "Bearer admin-token")
	changeReq.Header.Set("Content-Type", "application/json")
	changeResp := httptest.NewRecorder()
	router.ServeHTTP(changeResp, changeReq)
	if changeResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for password change, got %d: %s", changeResp.Code, changeResp.Body.String())
	}

	// Rebuild router with original startup credentials to verify persisted
	// credentials override boot-time config.
	router = api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "secret-pass",
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("frontend"))
		}),
	)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"persisted-pass"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected persisted password to be accepted, got %d: %s", loginResp.Code, loginResp.Body.String())
	}
}

func TestRouterUpgradesLegacyPersistedAdminPasswordHash(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "thism.db")

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	s.Close()

	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := rawDB.Exec(`
	INSERT INTO admin_auth (id, username, password, updated_at)
	VALUES (1, 'admin', 'legacy-plain-pass', ?)
	`, time.Now().Unix()); err != nil {
		t.Fatalf("insert legacy admin auth: %v", err)
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	s, err = store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New reopen: %v", err)
	}
	defer s.Close()

	h := hub.New(s)
	go h.Run()

	_ = api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{
			AdminToken: "admin-token",
			Username:   "admin",
			Password:   "boot-pass",
		},
		nil,
	)

	username, password, found, err := s.GetAdminAuth()
	if err != nil {
		t.Fatalf("GetAdminAuth: %v", err)
	}
	if !found {
		t.Fatal("expected persisted admin auth to remain present")
	}
	if username != "admin" {
		t.Fatalf("expected username to remain admin, got %q", username)
	}
	if password == "legacy-plain-pass" {
		t.Fatal("expected legacy plaintext password to be upgraded to a hash")
	}
}

func TestRegisterNode(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "admin-token", nil)

	body := strings.NewReader(`{"name":"web-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/register", body)
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == "" {
		t.Error("expected non-empty token in response")
	}
	if resp["id"] == "" {
		t.Error("expected non-empty id in response")
	}
}

func TestNodeManagementActions(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "admin-token", nil)

	err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "old-name",
		Token:     "node-token-1",
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}

	renameBody := bytes.NewBufferString(`{"name":"new-name"}`)
	renameReq := httptest.NewRequest(http.MethodPatch, "/api/nodes/node-1", renameBody)
	renameReq.Header.Set("Authorization", "Bearer admin-token")
	renameReq.Header.Set("Content-Type", "application/json")
	renameResp := httptest.NewRecorder()
	router.ServeHTTP(renameResp, renameReq)
	if renameResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for rename, got %d: %s", renameResp.Code, renameResp.Body.String())
	}

	var renamed models.Node
	if err := json.Unmarshal(renameResp.Body.Bytes(), &renamed); err != nil {
		t.Fatalf("decode renamed node: %v", err)
	}
	if renamed.Name != "new-name" {
		t.Fatalf("expected renamed name to be new-name, got %s", renamed.Name)
	}

	commandReq := httptest.NewRequest(http.MethodGet, "/api/nodes/node-1/install-command", nil)
	commandReq.Host = "example.com:12026"
	commandReq.Header.Set("Authorization", "Bearer admin-token")
	commandResp := httptest.NewRecorder()
	router.ServeHTTP(commandResp, commandReq)
	if commandResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for install command, got %d: %s", commandResp.Code, commandResp.Body.String())
	}

	var commandPayload map[string]string
	if err := json.Unmarshal(commandResp.Body.Bytes(), &commandPayload); err != nil {
		t.Fatalf("decode install command payload: %v", err)
	}
	command := commandPayload["command"]
	if !strings.Contains(command, "/install.sh?") || !strings.Contains(command, "token=node-token-1") || !strings.Contains(command, "name=new-name") {
		t.Fatalf("expected install command with token and renamed name, got: %s", command)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/nodes/node-1", nil)
	deleteReq.Header.Set("Authorization", "Bearer admin-token")
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for delete, got %d: %s", deleteResp.Code, deleteResp.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	getReq.Header.Set("Authorization", "Bearer admin-token")
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for list after delete, got %d: %s", getResp.Code, getResp.Body.String())
	}

	var body map[string][]models.Node
	if err := json.Unmarshal(getResp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list payload: %v", err)
	}
	if len(body["nodes"]) != 0 {
		t.Fatalf("expected empty node list after delete, got %d", len(body["nodes"]))
	}
}

func TestAgentMetricsUpdateNodeMetadata(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token-1",
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	baseURL.Scheme = "ws"
	baseURL.Path = "/ws/agent"
	query := baseURL.Query()
	query.Set("token", "agent-token-1")
	baseURL.RawQuery = query.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(baseURL.String(), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	payload := models.MetricsPayload{
		Type: "metrics",
		TS:   time.Now().Unix(),
		CPU:  10.5,
		IP:   "10.0.0.5",
		OS:   "linux",
		Arch: "arm64",
		Mem: models.MemStats{
			Used:  1024,
			Total: 2048,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	node, err := s.GetNodeByID("node-1")
	if err != nil {
		t.Fatalf("load node: %v", err)
	}
	if node == nil {
		t.Fatal("expected node to exist")
	}
	if node.OS != "linux" || node.Arch != "arm64" {
		t.Fatalf("expected metadata update to set os/arch, got %s/%s", node.OS, node.Arch)
	}
	if node.IP != "10.0.0.5" {
		t.Fatalf("expected fallback node ip from payload, got %s", node.IP)
	}
}

func TestAgentMetricsPreferPublicIPFromForwardedHeader(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	err := s.UpsertNode(&models.Node{
		ID:        "node-2",
		Name:      "agent-node-public",
		Token:     "agent-token-2",
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	baseURL.Scheme = "ws"
	baseURL.Path = "/ws/agent"
	query := baseURL.Query()
	query.Set("token", "agent-token-2")
	baseURL.RawQuery = query.Encode()

	headers := http.Header{}
	headers.Set("X-Forwarded-For", "198.51.100.27")
	conn, _, err := websocket.DefaultDialer.Dial(baseURL.String(), headers)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	payload := models.MetricsPayload{
		Type: "metrics",
		TS:   time.Now().Unix(),
		CPU:  8.1,
		IP:   "10.0.0.9",
		OS:   "linux",
		Arch: "amd64",
		Mem: models.MemStats{
			Used:  1024,
			Total: 2048,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	node, err := s.GetNodeByID("node-2")
	if err != nil {
		t.Fatalf("load node: %v", err)
	}
	if node == nil {
		t.Fatal("expected node to exist")
	}
	if node.IP != "198.51.100.27" {
		t.Fatalf("expected forwarded public ip to win, got %s", node.IP)
	}
}

func TestGetMetricsRetentionDefaultsToSevenDays(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "test-admin-token", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/metrics-retention", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		RetentionDays int   `json:"retention_days"`
		Options       []int `json:"options"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.RetentionDays != 7 {
		t.Fatalf("expected default retention 7 days, got %d", body.RetentionDays)
	}
	if len(body.Options) != 2 || body.Options[0] != 7 || body.Options[1] != 30 {
		t.Fatalf("unexpected retention options: %#v", body.Options)
	}
}

func TestUpdateMetricsRetentionPersistsAndPrunesOldMetrics(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "test-admin-token", nil)

	if err := s.UpsertNode(&models.Node{ID: "node-1", Token: "token-1", Name: "node-1"}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	oldTS := time.Now().AddDate(0, 0, -40).Unix()
	recentTS := time.Now().Unix()
	if err := s.InsertMetrics("node-1", &models.MetricsPayload{TS: oldTS, CPU: 10}); err != nil {
		t.Fatalf("InsertMetrics old: %v", err)
	}
	if err := s.InsertMetrics("node-1", &models.MetricsPayload{TS: recentTS, CPU: 20}); err != nil {
		t.Fatalf("InsertMetrics recent: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-retention", bytes.NewBufferString(`{"retention_days":30}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	days, err := s.GetMetricsRetentionDays()
	if err != nil {
		t.Fatalf("GetMetricsRetentionDays: %v", err)
	}
	if days != 30 {
		t.Fatalf("expected retention 30 days after update, got %d", days)
	}

	rows, err := s.QueryMetrics("node-1", oldTS-1, recentTS+1)
	if err != nil {
		t.Fatalf("QueryMetrics: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 metric row after prune, got %d", len(rows))
	}
	if rows[0].TS != recentTS {
		t.Fatalf("expected recent metric to remain, got ts %d", rows[0].TS)
	}
}

func TestUpdateMetricsRetentionRejectsInvalidValue(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()
	router := api.NewRouter(s, h, "test-admin-token", nil)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/metrics-retention", bytes.NewBufferString(`{"retention_days":14}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
