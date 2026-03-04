package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/store"
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
