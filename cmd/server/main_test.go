package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenCountryResolverReturnsNilWhenGeoIPDatabaseIsUnavailable(t *testing.T) {
	resolver := openCountryResolver(filepath.Join(t.TempDir(), "missing.mmdb"))
	if resolver != nil {
		t.Fatal("expected missing GeoIP database to disable country resolver without failing startup")
	}
}

func TestOpenCountryResolverWithFallbackUsesSecondaryWhenPrimaryMissing(t *testing.T) {
	fallbackCandidates := []string{
		geoDefaultIP2LocationPathForTest(),
		geoDefaultMaxMindPathForTest(),
	}
	var fallback string
	for _, candidate := range fallbackCandidates {
		if _, err := os.Stat(candidate); err == nil {
			fallback = candidate
			break
		}
	}
	if fallback == "" {
		t.Skip("no local geo database available for fallback test")
	}

	missingPrimary := filepath.Join(t.TempDir(), "missing-primary.mmdb")
	resolver := openCountryResolverWithFallback(missingPrimary, fallback)
	if resolver == nil {
		t.Fatal("expected fallback geo database to keep country resolver enabled")
	}
	if closer, ok := resolver.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

func geoDefaultIP2LocationPathForTest() string {
	return "/opt/1panel/geo/IP2LOCATION-LITE-DB1.IPV6.BIN"
}

func geoDefaultMaxMindPathForTest() string {
	return "/opt/1panel/geo/GeoIP.mmdb"
}

func TestNewHTTPServerConfiguresTimeouts(t *testing.T) {
	server := newHTTPServer(":12026", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	if server.Addr != ":12026" {
		t.Fatalf("expected server addr to be preserved, got %q", server.Addr)
	}
	if server.ReadHeaderTimeout <= 0 {
		t.Fatal("expected ReadHeaderTimeout to be configured")
	}
	if server.ReadTimeout <= 0 {
		t.Fatal("expected ReadTimeout to be configured")
	}
	if server.WriteTimeout <= 0 {
		t.Fatal("expected WriteTimeout to be configured")
	}
	if server.IdleTimeout <= 0 {
		t.Fatal("expected IdleTimeout to be configured")
	}
}

func TestServeHTTPServerStopsOnContextCancel(t *testing.T) {
	server := newHTTPServer("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveHTTPServer(ctx, server, listener)
	}()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, reqErr := client.Get("http://" + listener.Addr().String())
		if reqErr == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became ready: %v", reqErr)
		}
		time.Sleep(25 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected graceful shutdown without error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for serveHTTPServer to stop after context cancellation")
	}
}

func TestEnvOrReturnsEnvWhenSet(t *testing.T) {
	t.Setenv("THISM_TEST_KEY", "from-env")
	if got := envOr("THISM_TEST_KEY", "fallback"); got != "from-env" {
		t.Fatalf("expected env value to win over fallback, got %q", got)
	}
}

func TestEnvOrReturnsFallbackWhenUnset(t *testing.T) {
	if got := envOr("THISM_TEST_KEY_UNSET", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback when env unset, got %q", got)
	}
}

func TestEnvOrReturnsEmptyEnvValue(t *testing.T) {
	// Distinguish "set to empty" from "unset" — operators explicitly setting
	// THISM_FOO= should not silently fall back to a non-empty default.
	t.Setenv("THISM_TEST_KEY_EMPTY", "")
	if got := envOr("THISM_TEST_KEY_EMPTY", "fallback"); got != "" {
		t.Fatalf("expected explicit empty env value to be respected, got %q", got)
	}
}

func TestEnvOrFileReturnsEnvWhenSet(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretPath, []byte("from-file\n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	t.Setenv("THISM_TEST_SECRET", "from-env")
	t.Setenv("THISM_TEST_SECRET_FILE", secretPath)

	got, err := envOrFile("THISM_TEST_SECRET", "THISM_TEST_SECRET_FILE", "fallback")
	if err != nil {
		t.Fatalf("envOrFile returned error: %v", err)
	}
	if got != "from-env" {
		t.Fatalf("expected env value to win over file, got %q", got)
	}
}

func TestEnvOrFileReturnsFileValueWhenEnvUnset(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretPath, []byte("from-file\n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	t.Setenv("THISM_TEST_SECRET_FILE", secretPath)

	got, err := envOrFile("THISM_TEST_SECRET", "THISM_TEST_SECRET_FILE", "fallback")
	if err != nil {
		t.Fatalf("envOrFile returned error: %v", err)
	}
	if got != "from-file" {
		t.Fatalf("expected file value without trailing newline, got %q", got)
	}
}

func TestEnvOrFileReturnsFallbackWhenEnvAndFileUnset(t *testing.T) {
	got, err := envOrFile("THISM_TEST_SECRET_UNSET", "THISM_TEST_SECRET_FILE_UNSET", "fallback")
	if err != nil {
		t.Fatalf("envOrFile returned error: %v", err)
	}
	if got != "fallback" {
		t.Fatalf("expected fallback when env and file env are unset, got %q", got)
	}
}

func TestEnvOrFileReturnsEmptyEnvValue(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretPath, []byte("from-file\n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	t.Setenv("THISM_TEST_SECRET_EMPTY", "")
	t.Setenv("THISM_TEST_SECRET_EMPTY_FILE", secretPath)

	got, err := envOrFile("THISM_TEST_SECRET_EMPTY", "THISM_TEST_SECRET_EMPTY_FILE", "fallback")
	if err != nil {
		t.Fatalf("envOrFile returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected explicit empty env value to be respected, got %q", got)
	}
}

func TestEnvOrFileReturnsErrorWhenFileCannotBeRead(t *testing.T) {
	t.Setenv("THISM_TEST_SECRET_FILE", filepath.Join(t.TempDir(), "missing"))

	if _, err := envOrFile("THISM_TEST_SECRET", "THISM_TEST_SECRET_FILE", "fallback"); err == nil {
		t.Fatal("expected unreadable secret file to return an error")
	}
}
