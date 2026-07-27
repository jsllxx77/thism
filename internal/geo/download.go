package geo

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ip2locationDownloadURL = "https://www.ip2location.com/download?token=%s&file=%s"
	maxmindDownloadURL     = "https://download.maxmind.com/app/geoip_download?edition_id=%s&license_key=%s&suffix=tar.gz"
	defaultIP2LocationCode = "DB1LITEBINIPV6"
	defaultMaxMindEdition  = "GeoLite2-City"
	downloadHTTPTimeout    = 5 * time.Minute
)

// DownloadIP2LocationDB fetches the LITE country BIN into destPath.
func DownloadIP2LocationDB(ctx context.Context, token, destPath string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("ip2location token is required")
	}
	url := fmt.Sprintf(ip2locationDownloadURL, token, defaultIP2LocationCode)
	body, err := downloadBytes(ctx, url)
	if err != nil {
		return err
	}
	if looksLikeHTML(body) {
		return fmt.Errorf("ip2location download returned an error page; check token")
	}
	bin, err := extractNamedFileFromZip(body, ".bin")
	if err != nil {
		return fmt.Errorf("extract ip2location bin: %w", err)
	}
	return writeFileAtomic(destPath, bin, 0o644)
}

// DownloadMaxMindDB fetches GeoLite2-City MMDB into destPath.
func DownloadMaxMindDB(ctx context.Context, licenseKey, destPath string) error {
	licenseKey = strings.TrimSpace(licenseKey)
	if licenseKey == "" {
		return fmt.Errorf("maxmind license key is required")
	}
	url := fmt.Sprintf(maxmindDownloadURL, defaultMaxMindEdition, licenseKey)
	body, err := downloadBytes(ctx, url)
	if err != nil {
		return err
	}
	if looksLikeHTML(body) || looksLikePlainError(body) {
		return fmt.Errorf("maxmind download failed; check license key (%s)", firstLine(body))
	}
	mmdb, err := extractNamedFileFromTarGz(body, ".mmdb")
	if err != nil {
		return fmt.Errorf("extract maxmind mmdb: %w", err)
	}
	return writeFileAtomic(destPath, mmdb, 0o644)
}

func downloadBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: downloadHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20)) // 256 MiB cap
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download status %d: %s", resp.StatusCode, firstLine(body))
	}
	return body, nil
}

func extractNamedFileFromZip(archive []byte, ext string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}
	ext = strings.ToLower(ext)
	var preferred *zip.File
	var fallback *zip.File
	for _, file := range reader.File {
		name := strings.ToLower(file.Name)
		if file.FileInfo().IsDir() || !strings.HasSuffix(name, ext) {
			continue
		}
		if strings.Contains(name, "ipv6") {
			preferred = file
			break
		}
		if fallback == nil {
			fallback = file
		}
	}
	target := preferred
	if target == nil {
		target = fallback
	}
	if target == nil {
		return nil, fmt.Errorf("no %s file found in zip", ext)
	}
	rc, err := target.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 256<<20))
}

func extractNamedFileFromTarGz(archive []byte, ext string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	ext = strings.ToLower(ext)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(header.Name), ext) {
			continue
		}
		return io.ReadAll(io.LimitReader(tr, 256<<20))
	}
	return nil, fmt.Errorf("no %s file found in tar.gz", ext)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func looksLikeHTML(body []byte) bool {
	sample := strings.ToLower(string(body[:min(len(body), 256)]))
	return strings.Contains(sample, "<html") || strings.Contains(sample, "<!doctype")
}

func looksLikePlainError(body []byte) bool {
	if len(body) == 0 {
		return true
	}
	// Valid gzip starts with 1f 8b.
	if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
		return false
	}
	sample := strings.ToLower(string(body[:min(len(body), 200)]))
	return strings.Contains(sample, "invalid") || strings.Contains(sample, "error") || strings.Contains(sample, "unauthorized")
}

func firstLine(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "empty response"
	}
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	if len(text) > 160 {
		text = text[:160] + "..."
	}
	return text
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
