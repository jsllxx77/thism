package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func TestGuestSessionCannotAccessAdminEndpoints(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	if err := s.UpsertNode(&models.Node{ID: "node-1", Name: "edge-1", Token: "node-token-1", CreatedAt: time.Now().Unix()}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	router := api.NewRouterWithAuth(
		s,
		h,
		api.AuthConfig{AdminToken: "admin-token", Username: "admin", Password: "secret-pass"},
		nil,
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

	protected := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/nodes/register"},
		{method: http.MethodGet, path: "/api/nodes/node-1/install-command"},
		{method: http.MethodGet, path: "/api/nodes/node-1/metrics"},
		{method: http.MethodGet, path: "/api/nodes/node-1/services"},
		{method: http.MethodGet, path: "/api/nodes/node-1/docker"},
	}

	for _, item := range protected {
		req := httptest.NewRequest(item.method, item.path, nil)
		req.AddCookie(guestCookie)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for guest access to %s %s, got %d: %s", item.method, item.path, resp.Code, resp.Body.String())
		}
	}
}
