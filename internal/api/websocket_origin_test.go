package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func TestDashboardWebsocketRejectsCrossOriginGuest(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{AdminToken: "admin-token", Username: "admin", Password: "secret-pass"},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	)
	server := newIPv4TestServer(router)
	defer server.Close()

	guestReq := httptest.NewRequest(http.MethodPost, "/api/auth/guest", nil)
	guestResp := httptest.NewRecorder()
	router.ServeHTTP(guestResp, guestReq)
	if guestResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for guest login, got %d", guestResp.Code)
	}

	var guestCookie *http.Cookie
	for _, cookie := range guestResp.Result().Cookies() {
		if cookie.Name == "thism_guest" {
			guestCookie = cookie
			break
		}
	}
	if guestCookie == nil {
		t.Fatal("expected guest cookie")
	}

	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/ws/dashboard"
	headers := http.Header{}
	headers.Set("Origin", "http://evil.example")
	headers.Add("Cookie", guestCookie.String())

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err == nil {
		t.Fatal("expected cross-origin guest websocket handshake to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-origin guest websocket, got %#v (err=%v)", resp, err)
	}
}

func TestDashboardWebsocketAllowsSameOriginAdmin(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	if err := s.UpsertNode(&models.Node{ID: "node-1", Name: "edge-1", Token: "node-token-1", CreatedAt: time.Now().Unix()}); err != nil {
		t.Fatalf("seed node: %v", err)
	}

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{AdminToken: "admin-token", Username: "admin", Password: "secret-pass"},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	)
	server := newIPv4TestServer(router)
	defer server.Close()

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret-pass"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin login, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var adminCookie *http.Cookie
	for _, cookie := range loginResp.Result().Cookies() {
		if cookie.Name == "thism_admin" {
			adminCookie = cookie
			break
		}
	}
	if adminCookie == nil {
		t.Fatal("expected admin cookie")
	}

	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/ws/dashboard"
	headers := http.Header{}
	headers.Set("Origin", server.URL)
	headers.Add("Cookie", adminCookie.String())

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("expected same-origin admin websocket to succeed, got status=%d err=%v", status, err)
	}
	defer conn.Close()
}
