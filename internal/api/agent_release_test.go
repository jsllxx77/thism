package api_test

import (
  "encoding/json"
  "net/http"
  "net/http/httptest"
  "testing"

  "github.com/thism-dev/thism/internal/api"
  "github.com/thism-dev/thism/internal/hub"
  "github.com/thism-dev/thism/internal/store"
)

func TestAgentReleaseManifest(t *testing.T) {
  s, _ := store.New(":memory:")
  defer s.Close()
  h := hub.New(s)
  go h.Run()

  router := api.NewRouter(s, h, "admin-token", nil)
  req := httptest.NewRequest(http.MethodGet, "/api/agent-release?os=linux&arch=amd64", nil)
  w := httptest.NewRecorder()
  router.ServeHTTP(w, req)

  if w.Code != http.StatusOK {
    t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
  }

  var body map[string]any
  if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
    t.Fatalf("decode response: %v", err)
  }
  for _, key := range []string{"target_version", "download_url", "sha256", "check_interval_seconds"} {
    if _, ok := body[key]; !ok {
      t.Fatalf("expected manifest field %q, got %#v", key, body)
    }
  }
}
