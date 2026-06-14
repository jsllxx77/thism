package skins

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const (
	BuiltInSkinID    = "classic"
	ManifestFilename = "thism-frontend-skin.json"
	ManifestType     = "thism-frontend-skin"
	ManifestVersion  = 1
	APIVersion       = "thism.v1"
)

var skinIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

type Manifest struct {
	Type        string   `json:"type"`
	Version     int      `json:"version"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Entry       string   `json:"entry"`
	APIVersion  string   `json:"apiVersion"`
	Assets      []string `json:"assets,omitempty"`
	Preview     string   `json:"preview,omitempty"`
}

type Skin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
	Entry       string `json:"entry"`
	APIVersion  string `json:"api_version"`
	Preview     string `json:"preview,omitempty"`
}

func BuiltInSkin() Skin {
	return Skin{
		ID:          BuiltInSkinID,
		Name:        "Classic",
		Description: "Bundled thisM React frontend.",
		Source:      "built-in",
		Entry:       "index.html",
		APIVersion:  APIVersion,
	}
}

func ParseManifest(raw []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("invalid frontend skin manifest: %w", err)
	}
	return NormalizeManifest(manifest)
}

func NormalizeManifest(manifest Manifest) (Manifest, error) {
	manifest.Type = strings.TrimSpace(manifest.Type)
	manifest.ID = strings.TrimSpace(strings.ToLower(manifest.ID))
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Description = strings.TrimSpace(manifest.Description)
	manifest.Entry = strings.TrimSpace(manifest.Entry)
	manifest.APIVersion = strings.TrimSpace(manifest.APIVersion)
	manifest.Preview = strings.TrimSpace(manifest.Preview)

	if manifest.Type != ManifestType {
		return Manifest{}, errors.New("frontend skin manifest type must be thism-frontend-skin")
	}
	if manifest.Version != ManifestVersion {
		return Manifest{}, fmt.Errorf("unsupported frontend skin manifest version %d", manifest.Version)
	}
	if !ValidSkinID(manifest.ID) {
		return Manifest{}, errors.New("frontend skin id must use lowercase letters, numbers, and hyphens")
	}
	if manifest.ID == BuiltInSkinID {
		return Manifest{}, errors.New("classic is reserved for the built-in frontend skin")
	}
	if manifest.Name == "" || len(manifest.Name) > 80 {
		return Manifest{}, errors.New("frontend skin name must be 1-80 characters")
	}
	if len(manifest.Description) > 240 {
		return Manifest{}, errors.New("frontend skin description must be 240 characters or less")
	}
	if manifest.APIVersion != APIVersion {
		return Manifest{}, fmt.Errorf("unsupported frontend skin apiVersion %q", manifest.APIVersion)
	}

	entry, err := SafeRelativePath(manifest.Entry)
	if err != nil {
		return Manifest{}, fmt.Errorf("invalid frontend skin entry: %w", err)
	}
	if !strings.HasSuffix(strings.ToLower(entry), ".html") {
		return Manifest{}, errors.New("frontend skin entry must be an HTML file")
	}
	manifest.Entry = entry

	assets := make([]string, 0, len(manifest.Assets))
	seenAssets := map[string]struct{}{}
	for _, asset := range manifest.Assets {
		normalized, err := SafeRelativePath(asset)
		if err != nil {
			return Manifest{}, fmt.Errorf("invalid frontend skin asset %q: %w", asset, err)
		}
		if _, seen := seenAssets[normalized]; seen {
			continue
		}
		seenAssets[normalized] = struct{}{}
		assets = append(assets, normalized)
	}
	manifest.Assets = assets

	if manifest.Preview != "" {
		preview, err := SafeRelativePath(manifest.Preview)
		if err != nil {
			return Manifest{}, fmt.Errorf("invalid frontend skin preview: %w", err)
		}
		manifest.Preview = preview
	}

	return manifest, nil
}

func ValidSkinID(id string) bool {
	return skinIDPattern.MatchString(strings.TrimSpace(id))
}

func SafeRelativePath(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("path is required")
	}
	if strings.Contains(trimmed, "\x00") || strings.Contains(trimmed, "\\") {
		return "", errors.New("path contains unsafe characters")
	}
	if strings.HasPrefix(trimmed, "/") {
		return "", errors.New("absolute paths are not allowed")
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" {
		return "", errors.New("external URLs are not allowed")
	}
	if strings.Contains(trimmed, "?") || strings.Contains(trimmed, "#") {
		return "", errors.New("query strings and fragments are not allowed")
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", errors.New("path traversal is not allowed")
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", errors.New("path traversal is not allowed")
		}
	}
	return cleaned, nil
}

func SkinFromManifest(manifest Manifest) Skin {
	return Skin{
		ID:          manifest.ID,
		Name:        manifest.Name,
		Description: manifest.Description,
		Source:      "custom",
		Entry:       manifest.Entry,
		APIVersion:  manifest.APIVersion,
		Preview:     manifest.Preview,
	}
}
