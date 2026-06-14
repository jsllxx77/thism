package skins

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeManifestRejectsUnsafePaths(t *testing.T) {
	base := Manifest{
		Type:       ManifestType,
		Version:    ManifestVersion,
		ID:         "shadcn-dashboard",
		Name:       "Shadcn Dashboard",
		Entry:      "index.html",
		APIVersion: APIVersion,
	}

	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{
			name: "entry traversal",
			mutate: func(m *Manifest) {
				m.Entry = "../index.html"
			},
		},
		{
			name: "external preview",
			mutate: func(m *Manifest) {
				m.Preview = "https://example.com/preview.png"
			},
		},
		{
			name: "asset traversal",
			mutate: func(m *Manifest) {
				m.Assets = []string{"assets/app.js", "../../secret"}
			},
		},
		{
			name: "reserved classic id",
			mutate: func(m *Manifest) {
				m.ID = BuiltInSkinID
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manifest := base
			tc.mutate(&manifest)
			if _, err := NormalizeManifest(manifest); err == nil {
				t.Fatalf("expected manifest validation to fail")
			}
		})
	}
}

func TestInstallArchiveInstallsAndListsSkin(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	archive := buildSkinArchive(t, map[string]string{
		ManifestFilename: `{
			"type": "thism-frontend-skin",
			"version": 1,
			"id": "shadcn-dashboard",
			"name": "Shadcn Dashboard",
			"description": "A shadcn blocks style dashboard.",
			"entry": "index.html",
			"apiVersion": "thism.v1",
			"assets": ["index.html", "assets/app.js"],
			"preview": "preview.png"
		}`,
		"index.html":    "<div id=\"root\"></div><script src=\"/assets/app.js\"></script>",
		"assets/app.js": "console.log('skin')",
		"preview.png":   "png",
	})

	installed, err := manager.InstallArchive("shadcn-dashboard.thism-frontend-skin.zip", archive)
	if err != nil {
		t.Fatalf("InstallArchive: %v", err)
	}
	if installed.ID != "shadcn-dashboard" || installed.Source != "custom" {
		t.Fatalf("unexpected installed skin: %#v", installed)
	}
	if _, err := os.Stat(filepath.Join(manager.Root(), "shadcn-dashboard", "assets", "app.js")); err != nil {
		t.Fatalf("expected extracted asset: %v", err)
	}

	skins, err := manager.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(skins) != 2 {
		t.Fatalf("expected built-in plus installed skin, got %#v", skins)
	}
	if skins[0].ID != BuiltInSkinID || skins[1].ID != "shadcn-dashboard" {
		t.Fatalf("unexpected skin order: %#v", skins)
	}
}

func TestInstallArchiveRejectsTraversalFile(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	archive := buildSkinArchive(t, map[string]string{
		ManifestFilename: validManifestJSON(t, "safe-skin"),
		"index.html":     "ok",
		"../outside":     "nope",
	})
	if _, err := manager.InstallArchive("safe-skin.thism-frontend-skin.zip", archive); err == nil {
		t.Fatalf("expected traversal archive to be rejected")
	}
}

func validManifestJSON(t *testing.T, id string) string {
	t.Helper()
	raw, err := json.Marshal(Manifest{
		Type:       ManifestType,
		Version:    ManifestVersion,
		ID:         id,
		Name:       "Safe Skin",
		Entry:      "index.html",
		APIVersion: APIVersion,
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	return string(raw)
}

func buildSkinArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		fileWriter, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip file %s: %v", name, err)
		}
		if _, err := fileWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write zip file %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}
