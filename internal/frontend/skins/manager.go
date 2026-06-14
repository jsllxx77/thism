package skins

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	MaxArchiveBytes          = 32 << 20
	maxExtractedBytes        = 96 << 20
	maxArchiveFiles          = 2048
	maxManifestBytes         = 128 << 10
	temporaryDirectoryPrefix = ".install-"
)

type Manager struct {
	root string
	mu   sync.RWMutex
}

func NewManager(root string) (*Manager, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "frontend-skins"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create frontend skin directory: %w", err)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve frontend skin directory: %w", err)
	}
	return &Manager{root: absRoot}, nil
}

func (m *Manager) Root() string {
	if m == nil {
		return ""
	}
	return m.root
}

func (m *Manager) List() ([]Skin, error) {
	skins := []Skin{BuiltInSkin()}
	if m == nil {
		return skins, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return skins, nil
		}
		return nil, err
	}

	custom := make([]Skin, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		manifest, err := m.loadManifest(entry.Name())
		if err != nil {
			continue
		}
		custom = append(custom, SkinFromManifest(manifest))
	}
	sort.Slice(custom, func(i, j int) bool {
		return strings.ToLower(custom[i].Name) < strings.ToLower(custom[j].Name)
	})
	return append(skins, custom...), nil
}

func (m *Manager) Has(id string) bool {
	id = strings.TrimSpace(id)
	if id == BuiltInSkinID {
		return true
	}
	if m == nil || !ValidSkinID(id) {
		return false
	}
	_, err := m.loadManifest(id)
	return err == nil
}

func (m *Manager) InstallArchive(filename string, data []byte) (Skin, error) {
	if m == nil {
		return Skin{}, errors.New("frontend skin manager is unavailable")
	}
	if len(data) == 0 {
		return Skin{}, errors.New("frontend skin archive is empty")
	}
	if len(data) > MaxArchiveBytes {
		return Skin{}, fmt.Errorf("frontend skin archive exceeds %d bytes", MaxArchiveBytes)
	}
	if filename != "" && !isArchiveFilename(filename) {
		return Skin{}, errors.New("frontend skin archive must be a .zip file")
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Skin{}, fmt.Errorf("open frontend skin archive: %w", err)
	}
	if len(reader.File) > maxArchiveFiles {
		return Skin{}, fmt.Errorf("frontend skin archive contains more than %d files", maxArchiveFiles)
	}

	manifest, err := manifestFromArchive(reader)
	if err != nil {
		return Skin{}, err
	}
	presentFiles, totalSize, err := inspectArchive(reader)
	if err != nil {
		return Skin{}, err
	}
	if totalSize > maxExtractedBytes {
		return Skin{}, fmt.Errorf("frontend skin archive expands beyond %d bytes", maxExtractedBytes)
	}
	if !presentFiles[manifest.Entry] {
		return Skin{}, fmt.Errorf("frontend skin entry %q is missing from archive", manifest.Entry)
	}
	for _, asset := range manifest.Assets {
		if !presentFiles[asset] {
			return Skin{}, fmt.Errorf("frontend skin asset %q is missing from archive", asset)
		}
	}
	if manifest.Preview != "" && !presentFiles[manifest.Preview] {
		return Skin{}, fmt.Errorf("frontend skin preview %q is missing from archive", manifest.Preview)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.root, 0o755); err != nil {
		return Skin{}, err
	}
	tempDir, err := os.MkdirTemp(m.root, temporaryDirectoryPrefix+manifest.ID+"-")
	if err != nil {
		return Skin{}, err
	}
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.RemoveAll(tempDir)
		}
	}()

	if err := extractArchive(reader, tempDir); err != nil {
		return Skin{}, err
	}
	targetDir := filepath.Join(m.root, manifest.ID)
	if err := os.RemoveAll(targetDir); err != nil {
		return Skin{}, err
	}
	if err := os.Rename(tempDir, targetDir); err != nil {
		return Skin{}, err
	}
	cleanupTemp = false

	return SkinFromManifest(manifest), nil
}

func (m *Manager) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == BuiltInSkinID {
		return errors.New("classic frontend skin cannot be deleted")
	}
	if !ValidSkinID(id) {
		return errors.New("invalid frontend skin id")
	}
	if m == nil {
		return errors.New("frontend skin manager is unavailable")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return os.RemoveAll(filepath.Join(m.root, id))
}

func (m *Manager) Handler(id string) (http.Handler, bool) {
	id = strings.TrimSpace(id)
	if id == "" || id == BuiltInSkinID || m == nil || !ValidSkinID(id) {
		return nil, false
	}

	m.mu.RLock()
	manifest, err := m.loadManifest(id)
	m.mu.RUnlock()
	if err != nil {
		return nil, false
	}
	return &spaHandler{fs: http.Dir(filepath.Join(m.root, id)), entry: "/" + manifest.Entry}, true
}

func (m *Manager) loadManifest(id string) (Manifest, error) {
	if !ValidSkinID(id) {
		return Manifest{}, errors.New("invalid frontend skin id")
	}
	raw, err := os.ReadFile(filepath.Join(m.root, id, ManifestFilename))
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := ParseManifest(raw)
	if err != nil {
		return Manifest{}, err
	}
	if manifest.ID != id {
		return Manifest{}, errors.New("frontend skin manifest id does not match directory")
	}
	return manifest, nil
}

type spaHandler struct {
	fs    http.FileSystem
	entry string
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, err := h.fs.Open(r.URL.Path)
	if err != nil {
		r2 := *r
		r2.URL.Path = h.entry
		http.FileServer(h.fs).ServeHTTP(w, &r2)
		return
	}
	_ = f.Close()
	http.FileServer(h.fs).ServeHTTP(w, r)
}

func isArchiveFilename(filename string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(filename)), ".zip")
}

func manifestFromArchive(reader *zip.Reader) (Manifest, error) {
	for _, file := range reader.File {
		normalized, ignored, err := archivePath(file.Name)
		if err != nil {
			return Manifest{}, err
		}
		if ignored || normalized != ManifestFilename {
			continue
		}
		if file.UncompressedSize64 > maxManifestBytes {
			return Manifest{}, errors.New("frontend skin manifest is too large")
		}
		raw, err := readZipFile(file, maxManifestBytes)
		if err != nil {
			return Manifest{}, err
		}
		return ParseManifest(raw)
	}
	return Manifest{}, fmt.Errorf("%s is missing from frontend skin archive", ManifestFilename)
}

func inspectArchive(reader *zip.Reader) (map[string]bool, uint64, error) {
	present := map[string]bool{}
	var total uint64
	for _, file := range reader.File {
		normalized, ignored, err := archivePath(file.Name)
		if err != nil {
			return nil, 0, err
		}
		if ignored {
			continue
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, 0, fmt.Errorf("frontend skin archive contains symlink %q", file.Name)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		total += file.UncompressedSize64
		if total > maxExtractedBytes {
			return nil, 0, fmt.Errorf("frontend skin archive expands beyond %d bytes", maxExtractedBytes)
		}
		present[normalized] = true
	}
	return present, total, nil
}

func extractArchive(reader *zip.Reader, destination string) error {
	for _, file := range reader.File {
		normalized, ignored, err := archivePath(file.Name)
		if err != nil {
			return err
		}
		if ignored {
			continue
		}
		target := filepath.Join(destination, filepath.FromSlash(normalized))
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		raw, err := readZipFile(file, maxExtractedBytes)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func archivePath(value string) (string, bool, error) {
	if strings.HasPrefix(value, "__MACOSX/") || strings.HasSuffix(value, "/.DS_Store") {
		return "", true, nil
	}
	normalized, err := SafeRelativePath(value)
	if err != nil {
		return "", false, fmt.Errorf("unsafe frontend skin archive path %q: %w", value, err)
	}
	return normalized, false, nil
}

func readZipFile(file *zip.File, limit uint64) ([]byte, error) {
	if file.UncompressedSize64 > limit {
		return nil, fmt.Errorf("frontend skin archive file %q is too large", file.Name)
	}
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
		return nil, fmt.Errorf("frontend skin archive file %q is too large", file.Name)
	}
	return buffer.Bytes(), nil
}
