package api_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/store"
	sharedversion "github.com/thism-dev/thism/internal/version"
)

func TestAgentReleaseManifest(t *testing.T) {
	originalVersion := sharedversion.Version
	sharedversion.Version = "v1.2.3"
	t.Cleanup(func() {
		sharedversion.Version = originalVersion
	})

	fixture := []byte("test-agent-release-binary")
	tempDir := t.TempDir()
	distDir := filepath.Join(tempDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("create dist dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "thism-agent-linux-amd64"), fixture, 0o755); err != nil {
		t.Fatalf("write agent fixture: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

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

	checksum := sha256.Sum256(fixture)
	expectedSHA := hex.EncodeToString(checksum[:])
	expectedVersion := "v1.2.3"

	if body["sha256"] != expectedSHA {
		t.Fatalf("expected sha256 %q, got %#v", expectedSHA, body["sha256"])
	}
	if body["target_version"] != expectedVersion {
		t.Fatalf("expected target_version %q, got %#v", expectedVersion, body["target_version"])
	}
	if body["download_url"] != "http://example.com/dl/thism-agent-linux-amd64" {
		t.Fatalf("expected download_url for amd64 binary, got %#v", body["download_url"])
	}
	if body["check_interval_seconds"] != float64(1800) {
		t.Fatalf("expected 1800 second interval, got %#v", body["check_interval_seconds"])
	}
}
