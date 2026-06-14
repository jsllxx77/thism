package skins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const githubAPIBase = "https://api.github.com"

var skinArchiveCandidatePaths = []string{
	"thism-frontend-skin.zip",
	"frontend-skin.zip",
	"skins/thism-frontend-skin.zip",
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (m *Manager) InstallFromGitHub(ctx context.Context, input string) (Skin, error) {
	return m.InstallFromGitHubWithClient(ctx, input, &http.Client{Timeout: 30 * time.Second})
}

func (m *Manager) InstallFromGitHubWithClient(ctx context.Context, input string, client HTTPClient) (Skin, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	archiveURL, filename, err := resolveGitHubSkinArchive(ctx, strings.TrimSpace(input), client)
	if err != nil {
		return Skin{}, err
	}
	data, err := downloadBytes(ctx, client, archiveURL, MaxArchiveBytes)
	if err != nil {
		return Skin{}, err
	}
	return m.InstallArchive(filename, data)
}

type parsedGitHubURL struct {
	rawURL string
	owner  string
	repo   string
}

func parseGitHubURL(input string) (parsedGitHubURL, error) {
	if input == "" {
		return parsedGitHubURL{}, errors.New("enter a GitHub repository URL")
	}
	parsed, err := url.Parse(input)
	if err != nil || parsed.Scheme != "https" {
		return parsedGitHubURL{}, errors.New("enter a GitHub repository URL")
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if parsed.Host == "raw.githubusercontent.com" {
		if len(parts) < 4 || !isSkinArchiveAsset(parts[len(parts)-1]) {
			return parsedGitHubURL{}, errors.New("enter a thisM frontend skin archive URL")
		}
		return parsedGitHubURL{rawURL: parsed.String()}, nil
	}
	if parsed.Host != "github.com" || len(parts) < 2 {
		return parsedGitHubURL{}, errors.New("enter a GitHub repository URL")
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return parsedGitHubURL{}, errors.New("enter a GitHub repository URL")
	}
	if len(parts) >= 5 && parts[2] == "releases" && parts[3] == "download" && isSkinArchiveAsset(parts[len(parts)-1]) {
		return parsedGitHubURL{rawURL: parsed.String()}, nil
	}
	if len(parts) >= 5 && parts[2] == "raw" && isSkinArchiveAsset(parts[len(parts)-1]) {
		return parsedGitHubURL{rawURL: parsed.String()}, nil
	}
	return parsedGitHubURL{owner: owner, repo: repo}, nil
}

func resolveGitHubSkinArchive(ctx context.Context, input string, client HTTPClient) (string, string, error) {
	parsed, err := parseGitHubURL(input)
	if err != nil {
		return "", "", err
	}
	if parsed.rawURL != "" {
		return parsed.rawURL, filenameFromURL(parsed.rawURL), nil
	}

	if archiveURL, filename, err := latestReleaseSkinArchive(ctx, client, parsed.owner, parsed.repo); err == nil && archiveURL != "" {
		return archiveURL, filename, nil
	}
	if archiveURL, filename, err := repositoryContentSkinArchive(ctx, client, parsed.owner, parsed.repo); err == nil && archiveURL != "" {
		return archiveURL, filename, nil
	}
	return "", "", errors.New("no thisM frontend skin package found in that GitHub repository")
}

func latestReleaseSkinArchive(ctx context.Context, client HTTPClient, owner, repo string) (string, string, error) {
	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			URL                string `json:"url"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := getJSON(ctx, client, fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPIBase, owner, repo), &release); err != nil {
		return "", "", err
	}

	for _, asset := range release.Assets {
		if !isSkinArchiveAsset(asset.Name) {
			continue
		}
		if asset.URL != "" {
			return asset.URL, asset.Name, nil
		}
		if asset.BrowserDownloadURL != "" {
			return asset.BrowserDownloadURL, asset.Name, nil
		}
	}
	return "", "", nil
}

func repositoryContentSkinArchive(ctx context.Context, client HTTPClient, owner, repo string) (string, string, error) {
	for _, candidate := range skinArchiveCandidatePaths {
		var content struct {
			Name        string `json:"name"`
			DownloadURL string `json:"download_url"`
		}
		err := getJSON(ctx, client, fmt.Sprintf("%s/repos/%s/%s/contents/%s", githubAPIBase, owner, repo, candidate), &content)
		if err != nil || content.DownloadURL == "" {
			continue
		}
		return content.DownloadURL, content.Name, nil
	}
	return "", "", nil
}

func getJSON(ctx context.Context, client HTTPClient, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func downloadBytes(ctx context.Context, client HTTPClient, endpoint string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if strings.Contains(endpoint, "api.github.com/repos/") && strings.Contains(endpoint, "/releases/assets/") {
		req.Header.Set("Accept", "application/octet-stream")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download frontend skin archive: %s", resp.Status)
	}
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("frontend skin archive exceeds %d bytes", limit)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("frontend skin archive exceeds %d bytes", limit)
	}
	return data, nil
}

func isSkinArchiveAsset(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return lower == "thism-frontend-skin.zip" || strings.HasSuffix(lower, ".thism-frontend-skin.zip")
}

func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "thism-frontend-skin.zip"
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return "thism-frontend-skin.zip"
	}
	return parts[len(parts)-1]
}
