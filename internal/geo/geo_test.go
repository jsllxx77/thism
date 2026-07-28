package geo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thism-dev/thism/internal/models"
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

func TestNormalizeCountryCode(t *testing.T) {
	cases := map[string]string{
		"hk":   "HK",
		" US ": "US",
		"-":    "",
		"--":   "",
		"ZZ":   "",
		"USA":  "",
		"This parameter is unavailable for selected data file. Please upgrade the data file.": "",
		"": "",
	}
	for in, want := range cases {
		if got := normalizeCountryCode(in); got != want {
			t.Fatalf("normalizeCountryCode(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestDetectDBFormat(t *testing.T) {
	cases := map[string]dbFormat{
		"/opt/1panel/geo/GeoIP.mmdb":                    dbFormatMaxMind,
		"/opt/1panel/geo/IP2LOCATION-LITE-DB1.IPV6.BIN": dbFormatIP2Location,
		"/tmp/IP2LOCATION-LITE-DB1.BIN":                 dbFormatIP2Location,
		"/tmp/something.unknown":                        dbFormatUnknown,
	}
	for path, want := range cases {
		if got := detectDBFormat(path); got != want {
			t.Fatalf("detectDBFormat(%q)=%v, want %v", path, got, want)
		}
	}
}

func TestDefaultCandidatePathsPreferGenericDirAndKeepLegacyFallback(t *testing.T) {
	paths := DefaultCandidatePaths()
	want := []string{DefaultDBPath, DefaultIP2LocationPath, LegacyMaxMindDBPath, LegacyIP2LocationPath}
	if len(paths) < len(want) {
		t.Fatalf("DefaultCandidatePaths() returned %d paths, want at least %d", len(paths), len(want))
	}
	for index, path := range want {
		if paths[index] != path {
			t.Fatalf("DefaultCandidatePaths()[%d] = %q, want %q", index, paths[index], path)
		}
	}
}

func TestIP2LocationResolverLookup(t *testing.T) {
	candidates := []string{
		DefaultDBPath,
		"/tmp/ip2location-dl/IP2LOCATION-LITE-DB1.IPV6.BIN",
		filepath.Join("testdata", "IP2LOCATION-LITE-DB1.IPV6.BIN"),
	}
	var resolver *Resolver
	var opened string
	for _, path := range candidates {
		r, err := NewResolver(path)
		if err != nil {
			continue
		}
		resolver = r
		opened = path
		break
	}
	if resolver == nil {
		t.Skip("no IP2Location BIN available for integration lookup test")
	}
	defer resolver.Close()

	if got := resolver.ResolveCountryCode("8.8.8.8"); got != "US" {
		t.Fatalf("ResolveCountryCode(8.8.8.8) via %s = %q, want US", opened, got)
	}
	if got := resolver.ResolveCountryCode("2001:4860:4860::8888"); got != "US" {
		t.Fatalf("ResolveCountryCode(IPv6) via %s = %q, want US", opened, got)
	}
	if got := resolver.ResolveCountryCode("10.0.0.1"); got != "" {
		t.Fatalf("private IP should stay empty, got %q", got)
	}
}

func TestOpenBestResolverLoadsBothSourcesWhenPresent(t *testing.T) {
	ip2l := DefaultIP2LocationPath
	mmdb := DefaultDBPath
	if _, err := NewResolver(ip2l); err != nil {
		t.Skipf("IP2Location DB unavailable: %v", err)
	}
	if _, err := NewResolver(mmdb); err != nil {
		t.Skipf("MaxMind DB unavailable: %v", err)
	}

	resolver, err := NewResolverWithFallback(mmdb, ip2l)
	if err != nil {
		t.Fatalf("NewResolverWithFallback: %v", err)
	}
	defer resolver.Close()

	sources := resolver.Sources()
	if len(sources) < 2 {
		t.Fatalf("expected both geo sources loaded, got %#v", sources)
	}
	if got := resolver.ResolveCountryCode("8.8.8.8"); got != "US" {
		t.Fatalf("dual-source lookup = %q, want US", got)
	}
}

func TestManagerDefaultsToMaxMind(t *testing.T) {
	manager := NewManager(DefaultDir)
	view := manager.View()
	if view.Provider != "maxmind" {
		t.Fatalf("default provider = %q, want maxmind", view.Provider)
	}
	_ = manager.ApplySettings(models.GeoIPSettings{Provider: models.GeoIPProviderMaxMind})
	if got := manager.ResolveCountryCode("8.8.8.8"); got != "" && got != "US" {
		t.Fatalf("unexpected country code %q", got)
	}
}

func TestManagedDatabasePathFallsBackToLegacyOnlyWhenManagedFileIsMissing(t *testing.T) {
	managedDir := t.TempDir()
	legacyDir := t.TempDir()
	managedPath := filepath.Join(managedDir, DefaultMaxMindName)
	legacyPath := filepath.Join(legacyDir, DefaultMaxMindName)

	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o600); err != nil {
		t.Fatalf("write legacy database placeholder: %v", err)
	}
	if got := managedDatabasePath(managedDir, legacyDir, models.GeoIPProviderMaxMind, true); got != legacyPath {
		t.Fatalf("managedDatabasePath() = %q, want legacy path %q", got, legacyPath)
	}

	if err := os.WriteFile(managedPath, []byte("managed"), 0o600); err != nil {
		t.Fatalf("write managed database placeholder: %v", err)
	}
	if got := managedDatabasePath(managedDir, legacyDir, models.GeoIPProviderMaxMind, true); got != managedPath {
		t.Fatalf("managedDatabasePath() = %q, want managed path %q", got, managedPath)
	}

	if err := os.Remove(managedPath); err != nil {
		t.Fatalf("remove managed database placeholder: %v", err)
	}
	if got := managedDatabasePath(managedDir, legacyDir, models.GeoIPProviderMaxMind, false); got != managedPath {
		t.Fatalf("managedDatabasePath() without fallback = %q, want managed path %q", got, managedPath)
	}
}
