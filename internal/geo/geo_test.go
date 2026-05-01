package geo

import (
	"testing"
)

func TestResolverSkipsPrivateAndInvalidIPs(t *testing.T) {
	resolver := &Resolver{}
	for _, candidate := range []string{"", "bad-ip", "10.0.0.1", "192.168.1.9", "127.0.0.1", "::1", "fc00::1"} {
		if got := resolver.ResolveCountryCode(candidate); got != "" {
			t.Fatalf("expected empty country code for %q, got %q", candidate, got)
		}
	}
}

func TestValidateResolver(t *testing.T) {
	if err := ValidateResolver(nil); err == nil {
		t.Fatal("expected nil resolver to be rejected")
	}
}
