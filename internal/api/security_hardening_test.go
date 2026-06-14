package api_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func TestRouterAddsSecureResponseHeaders(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	headers := resp.Result().Header
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options nosniff, got %q", headers.Get("X-Content-Type-Options"))
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY, got %q", headers.Get("X-Frame-Options"))
	}
	if !strings.Contains(headers.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("expected CSP frame-ancestors directive, got %q", headers.Get("Content-Security-Policy"))
	}
	if !strings.Contains(headers.Get("Content-Security-Policy"), "https://api.github.com") {
		t.Fatalf("expected CSP to allow GitHub API theme imports, got %q", headers.Get("Content-Security-Policy"))
	}
	if !strings.Contains(headers.Get("Content-Security-Policy"), "https://raw.githubusercontent.com") {
		t.Fatalf("expected CSP to allow raw GitHub theme imports, got %q", headers.Get("Content-Security-Policy"))
	}
	if headers.Get("Referrer-Policy") == "" {
		t.Fatal("expected Referrer-Policy header")
	}
}

func TestCookieAuthenticatedStateChangeRequiresCSRFToken(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{AdminToken: "admin-token", Username: "admin", Password: "secret-pass"},
		nil,
	)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret-pass"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var adminCookie, csrfCookie *http.Cookie
	for _, cookie := range loginResp.Result().Cookies() {
		switch cookie.Name {
		case "thism_admin":
			adminCookie = cookie
		case "thism_csrf":
			csrfCookie = cookie
		}
	}
	if adminCookie == nil || csrfCookie == nil {
		t.Fatalf("expected admin and csrf cookies, got admin=%v csrf=%v", adminCookie, csrfCookie)
	}

	rejected := httptest.NewRecorder()
	rejectedReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rejectedReq.AddCookie(adminCookie)
	rejectedReq.AddCookie(csrfCookie)
	router.ServeHTTP(rejected, rejectedReq)
	if rejected.Code != http.StatusForbidden {
		t.Fatalf("expected missing csrf token to be rejected with 403, got %d: %s", rejected.Code, rejected.Body.String())
	}

	accepted := httptest.NewRecorder()
	acceptedReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	acceptedReq.AddCookie(adminCookie)
	acceptedReq.AddCookie(csrfCookie)
	acceptedReq.Header.Set("X-CSRF-Token", csrfCookie.Value)
	router.ServeHTTP(accepted, acceptedReq)
	if accepted.Code != http.StatusOK {
		t.Fatalf("expected csrf-matched logout to succeed, got %d: %s", accepted.Code, accepted.Body.String())
	}
}

func TestOversizedJSONBodyIsRejected(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)
	body := strings.NewReader(`{"name":"` + strings.Repeat("x", 2<<20) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/register", body)
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized body, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPasswordLoginLocksOutRepeatedFailures(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{AdminToken: "admin-token", Username: "admin", Password: "secret-pass"},
		nil,
	)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected failed login %d to return 401, got %d", i+1, resp.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected lockout to reject valid credentials after repeated failures, got %d", resp.Code)
	}
}

func TestAgentWebsocketPrefersBearerTokenOverQueryToken(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	if err := s.UpsertNode(&models.Node{ID: "query-node", Name: "query", Token: "query-token", CreatedAt: 1}); err != nil {
		t.Fatalf("seed query node: %v", err)
	}
	if err := s.UpsertNode(&models.Node{ID: "bearer-node", Name: "bearer", Token: "bearer-token", CreatedAt: 1}); err != nil {
		t.Fatalf("seed bearer node: %v", err)
	}

	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	wsURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	wsURL.Scheme = "ws"
	wsURL.Path = "/ws/agent"
	query := wsURL.Query()
	query.Set("token", "query-token")
	wsURL.RawQuery = query.Encode()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer bearer-token")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(models.MetricsPayload{Type: "metrics", TS: 1, CPU: 9, OS: "linux", Arch: "amd64"}); err != nil {
		t.Fatalf("write metrics: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		node, err := s.GetNodeByID("bearer-node")
		if err != nil {
			t.Fatalf("load bearer node: %v", err)
		}
		if node != nil && node.OS == "linux" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected bearer-token node to receive websocket metrics")
}
