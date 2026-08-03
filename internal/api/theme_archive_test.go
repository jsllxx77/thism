package api_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/store"
)

func themeArchive(t *testing.T, manifest string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create("thism-theme.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestThemeArchiveImportEndpoint(t *testing.T) {
	storeInstance, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer storeInstance.Close()
	hubInstance := hub.New(storeInstance)
	go hubInstance.Run()

	router := api.NewRouter(storeInstance, hubInstance, "test-admin-token", nil)
	archive := themeArchive(t, `{"type":"thism-theme","version":1,"id":"ops-theme","name":"Ops Theme"}`)
	payload, err := json.Marshal(map[string]string{
		"name": "ops-theme.thism-theme.zip",
		"data": base64.StdEncoding.EncodeToString(archive),
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/settings/theme/import", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Theme struct {
			ID string `json:"id"`
		} `json:"theme"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if body.Theme.ID != "ops-theme" {
		t.Fatalf("unexpected theme response: %#v", body)
	}
}
