package api

import (
	"net/http/httptest"
	"testing"
)

func TestResolveNodeIPPrefersPayloadIPv4OverTransportIPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws/agent", nil)
	req.RemoteAddr = "[2606:4700:3031::6815:5969]:443"
	req.Header.Set("X-Forwarded-For", "2606:4700:3031::6815:5969")

	got := resolveNodeIP(req, "10.0.0.5")
	if got != "10.0.0.5" {
		t.Fatalf("expected payload IPv4 to win over transport IPv6, got %q", got)
	}
}

func TestResolveNodeIPFallsBackToIPv6WhenNoIPv4Exists(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws/agent", nil)
	req.RemoteAddr = "[2606:4700:3031::6815:5969]:443"

	got := resolveNodeIP(req, "")
	if got != "2606:4700:3031::6815:5969" {
		t.Fatalf("expected IPv6 fallback when no IPv4 exists, got %q", got)
	}
}
