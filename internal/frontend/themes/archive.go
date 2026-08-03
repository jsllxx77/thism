package themes

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

const (
	MaxArchiveBytes   = 32 << 20
	maxExtractedBytes = 96 << 20
	maxArchiveFiles   = 2048
	maxThemeBytes     = 128 << 10
)

// ExtractThemePackage returns the validated thism-theme.json manifest from a
// theme archive. The manifest is parsed again by the frontend for full token
// and appearance validation.
func ExtractThemePackage(filename string, data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > MaxArchiveBytes {
		return nil, errors.New("theme archive size is invalid")
	}
	if filename != "" && !strings.HasSuffix(strings.ToLower(strings.TrimSpace(filename)), ".zip") {
		return nil, errors.New("theme archive must be a .zip file")
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open theme archive: %w", err)
	}
	if len(reader.File) > maxArchiveFiles {
		return nil, fmt.Errorf("theme archive contains more than %d files", maxArchiveFiles)
	}

	var totalSize uint64
	for _, file := range reader.File {
		normalized, ignored, err := archivePath(file.Name)
		if err != nil {
			return nil, err
		}
		if ignored {
			continue
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("theme archive contains symlink %q", file.Name)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		totalSize += file.UncompressedSize64
		if totalSize > maxExtractedBytes {
			return nil, fmt.Errorf("theme archive expands beyond %d bytes", maxExtractedBytes)
		}
		if normalized != "thism-theme.json" {
			continue
		}
		if file.UncompressedSize64 > maxThemeBytes {
			return nil, errors.New("theme manifest is too large")
		}
		manifest, err := readZipFile(file, maxThemeBytes)
		if err != nil {
			return nil, err
		}
		if err := validateManifest(manifest); err != nil {
			return nil, err
		}
		return manifest, nil
	}

	return nil, errors.New("theme archive is missing thism-theme.json")
}

func archivePath(value string) (string, bool, error) {
	if strings.HasPrefix(value, "__MACOSX/") || strings.HasSuffix(value, "/.DS_Store") {
		return "", true, nil
	}
	if strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return "", false, fmt.Errorf("unsafe theme archive path %q", value)
	}
	normalized := path.Clean(value)
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", false, fmt.Errorf("unsafe theme archive path %q", value)
	}
	return normalized, false, nil
}

func readZipFile(file *zip.File, limit uint64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buffer bytes.Buffer
	if _, err := io.CopyN(&buffer, reader, int64(limit)+1); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if uint64(buffer.Len()) > limit {
		return nil, fmt.Errorf("theme archive file %q is too large", file.Name)
	}
	return buffer.Bytes(), nil
}

func validateManifest(raw []byte) error {
	var manifest struct {
		Type    string `json:"type"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("invalid theme manifest: %w", err)
	}
	if manifest.Type != "thism-theme" || manifest.Version != 1 {
		return errors.New("theme manifest must be a version 1 thism-theme package")
	}
	return nil
}
