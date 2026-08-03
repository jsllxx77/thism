package themes

import (
	"archive/zip"
	"bytes"
	"testing"
)

func buildArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, contents := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestExtractThemePackage(t *testing.T) {
	archive := buildArchive(t, map[string]string{
		"thism-theme.json": `{"type":"thism-theme","version":1,"id":"classic-plus"}`,
		"README.md":        "theme package",
	})

	manifest, err := ExtractThemePackage("classic-plus.thism-theme.zip", archive)
	if err != nil {
		t.Fatalf("ExtractThemePackage: %v", err)
	}
	if string(manifest) != `{"type":"thism-theme","version":1,"id":"classic-plus"}` {
		t.Fatalf("unexpected manifest: %s", manifest)
	}
}

func TestExtractThemePackageRejectsInvalidArchives(t *testing.T) {
	tests := []struct {
		name    string
		archive []byte
	}{
		{name: "wrong filename", archive: buildArchive(t, map[string]string{"thism-theme.json": `{"type":"thism-theme","version":1}`})},
		{name: "missing manifest", archive: buildArchive(t, map[string]string{"README.md": "missing"})},
		{name: "invalid manifest", archive: buildArchive(t, map[string]string{"thism-theme.json": `{"type":"other","version":1}`})},
		{name: "unsafe path", archive: buildArchive(t, map[string]string{"../thism-theme.json": `{"type":"thism-theme","version":1}`})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filename := "theme.zip"
			if test.name == "wrong filename" {
				filename = "theme.json"
			}
			if _, err := ExtractThemePackage(filename, test.archive); err == nil {
				t.Fatal("expected archive validation error")
			}
		})
	}
}
