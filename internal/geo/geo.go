package geo

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/ip2location/ip2location-go/v9"
	maxminddb "github.com/oschwald/maxminddb-golang/v2"
)

// Default directory used by local 1Panel-style deployments.
const DefaultDir = "/opt/1panel/geo"

// DefaultDBPath points at the default MaxMind GeoLite2 City database.
const DefaultDBPath = DefaultDir + "/GeoIP.mmdb"

// DefaultIP2LocationPath is used when the active provider is IP2Location.
const DefaultIP2LocationPath = DefaultDir + "/IP2LOCATION-LITE-DB1.IPV6.BIN"

// LegacyMaxMindDBPath keeps the historical MaxMind path name used by local installs.
const LegacyMaxMindDBPath = DefaultDBPath

// DefaultIP2LocationName and DefaultMaxMindName are the canonical filenames
// written by deploy/fetch-geoip-dbs.sh and the settings-page updater.
const (
	DefaultIP2LocationName = "IP2LOCATION-LITE-DB1.IPV6.BIN"
	DefaultMaxMindName     = "GeoIP.mmdb"
)

type CountryResolver interface {
	ResolveCountryCode(ip string) string
}

type Resolver struct {
	mmdb          *maxminddb.Reader
	ip2l          *ip2location.DB
	preferIP2L    bool
	primaryPath   string
	fallbackPath  string
	loadedSources []string
}

type countryLookup struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	RegisteredCountry struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"registered_country"`
	ISO string `maxminddb:"iso"`
}

// NewResolver opens a single database path.
func NewResolver(dbPath string) (*Resolver, error) {
	return NewResolverWithFallback(dbPath, "")
}

// NewResolverWithFallback opens a primary database and an optional fallback.
// When both IP2Location and MaxMind files are available, lookups try the
// primary source first and fall back to the other if the country code is empty.
func NewResolverWithFallback(primaryPath, fallbackPath string) (*Resolver, error) {
	primaryPath = strings.TrimSpace(primaryPath)
	fallbackPath = strings.TrimSpace(fallbackPath)
	if primaryPath == "" && fallbackPath != "" {
		primaryPath, fallbackPath = fallbackPath, ""
	}
	if primaryPath == "" {
		primaryPath = DefaultDBPath
	}

	candidates := uniqueNonEmpty(primaryPath, fallbackPath)
	resolver := &Resolver{
		preferIP2L:   false,
		primaryPath:  primaryPath,
		fallbackPath: fallbackPath,
	}

	var openErrs []string
	for _, path := range candidates {
		if err := resolver.openPath(path); err != nil {
			openErrs = append(openErrs, fmt.Sprintf("%s: %v", path, err))
		}
	}
	if resolver.ip2l == nil && resolver.mmdb == nil {
		if len(openErrs) == 0 {
			return nil, fmt.Errorf("no geoip database found at %q", primaryPath)
		}
		return nil, errors.New(strings.Join(openErrs, "; "))
	}

	switch detectDBFormat(primaryPath) {
	case dbFormatMaxMind:
		resolver.preferIP2L = false
	case dbFormatIP2Location:
		resolver.preferIP2L = true
	default:
		// If only one backend loaded, prefer whatever is available.
		resolver.preferIP2L = resolver.ip2l != nil
	}
	return resolver, nil
}

// OpenBestResolver tries explicit paths first, then common deploy locations.
// It loads every readable IP2Location/MaxMind database so both sources can
// coexist on a fresh host after running deploy/fetch-geoip-dbs.sh.
func OpenBestResolver(explicitPaths ...string) (*Resolver, error) {
	paths := make([]string, 0, len(explicitPaths)+len(DefaultCandidatePaths()))
	for _, path := range explicitPaths {
		path = strings.TrimSpace(path)
		if path != "" {
			paths = append(paths, path)
		}
	}
	paths = append(paths, DefaultCandidatePaths()...)
	paths = uniqueNonEmpty(paths...)

	var primary, fallback string
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if primary == "" {
			primary = path
			continue
		}
		// Prefer pairing different formats when possible.
		if detectDBFormat(primary) != detectDBFormat(path) {
			fallback = path
			break
		}
		if fallback == "" {
			fallback = path
		}
	}
	if primary == "" {
		return nil, fmt.Errorf("no geoip database found in candidates: %s", strings.Join(paths, ", "))
	}
	return NewResolverWithFallback(primary, fallback)
}

// DefaultCandidatePaths returns common on-disk locations for both providers.
// MaxMind is preferred first so fresh installs default to GeoLite2.
func DefaultCandidatePaths() []string {
	return []string{
		DefaultDBPath,
		LegacyMaxMindDBPath,
		filepath.Join(DefaultDir, DefaultMaxMindName),
		filepath.Join(DefaultDir, DefaultIP2LocationName),
		DefaultIP2LocationPath,
		filepath.Join("geo", DefaultMaxMindName),
		filepath.Join("geo", DefaultIP2LocationName),
		filepath.Join("/var/lib/thism/geo", DefaultMaxMindName),
		filepath.Join("/var/lib/thism/geo", DefaultIP2LocationName),
		DefaultMaxMindName,
		DefaultIP2LocationName,
	}
}

func MustNewResolver(dbPath string) *Resolver {
	resolver, err := NewResolver(dbPath)
	if err != nil {
		log.Fatalf("geoip: failed to open database: %v", err)
	}
	return resolver
}

func (r *Resolver) Close() error {
	if r == nil {
		return nil
	}
	var firstErr error
	if r.mmdb != nil {
		if err := r.mmdb.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		r.mmdb = nil
	}
	if r.ip2l != nil {
		r.ip2l.Close()
		r.ip2l = nil
	}
	return firstErr
}

// Sources returns the database paths successfully loaded into this resolver.
func (r *Resolver) Sources() []string {
	if r == nil {
		return nil
	}
	out := make([]string, len(r.loadedSources))
	copy(out, r.loadedSources)
	return out
}

func (r *Resolver) ResolveCountryCode(ip string) string {
	if r == nil {
		return ""
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil || isPrivateOrLocal(parsed) {
		return ""
	}

	tryIP2L := func() string {
		if r.ip2l == nil {
			return ""
		}
		return r.resolveIP2Location(parsed.String())
	}
	tryMaxMind := func() string {
		if r.mmdb == nil {
			return ""
		}
		return r.resolveMaxMind(parsed)
	}

	if r.preferIP2L {
		if code := tryIP2L(); code != "" {
			return code
		}
		return tryMaxMind()
	}
	if code := tryMaxMind(); code != "" {
		return code
	}
	return tryIP2L()
}

func (r *Resolver) openPath(path string) error {
	if path == "" {
		return errors.New("empty geoip path")
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	switch detectDBFormat(path) {
	case dbFormatIP2Location:
		if r.ip2l != nil {
			return nil
		}
		db, err := ip2location.OpenDB(path)
		if err != nil {
			return err
		}
		r.ip2l = db
		r.loadedSources = append(r.loadedSources, path)
		return nil
	case dbFormatMaxMind:
		if r.mmdb != nil {
			return nil
		}
		db, err := maxminddb.Open(path)
		if err != nil {
			return err
		}
		r.mmdb = db
		r.loadedSources = append(r.loadedSources, path)
		return nil
	default:
		// Prefer MaxMind first for unknown extensions, then IP2Location.
		if r.mmdb == nil {
			if db, err := maxminddb.Open(path); err == nil {
				r.mmdb = db
				r.loadedSources = append(r.loadedSources, path)
				return nil
			}
		}
		if r.ip2l == nil {
			db, err := ip2location.OpenDB(path)
			if err != nil {
				return err
			}
			r.ip2l = db
			r.loadedSources = append(r.loadedSources, path)
			return nil
		}
		return nil
	}
}

func (r *Resolver) resolveIP2Location(ip string) string {
	rec, err := r.ip2l.Get_country_short(ip)
	if err != nil {
		return ""
	}
	return normalizeCountryCode(rec.Country_short)
}

func (r *Resolver) resolveMaxMind(parsed net.IP) string {
	addr, err := netip.ParseAddr(parsed.String())
	if err != nil {
		return ""
	}
	var result countryLookup
	if err := r.mmdb.Lookup(addr).Decode(&result); err != nil {
		return ""
	}
	code := normalizeCountryCode(result.Country.ISOCode)
	if code == "" {
		code = normalizeCountryCode(result.RegisteredCountry.ISOCode)
	}
	if code == "" {
		code = normalizeCountryCode(result.ISO)
	}
	return code
}

func normalizeCountryCode(raw string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) != 2 {
		return ""
	}
	// IP2Location uses "-" for unknown / reserved rows.
	if code == "--" || code == "ZZ" {
		return ""
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return code
}

type dbFormat int

const (
	dbFormatUnknown dbFormat = iota
	dbFormatMaxMind
	dbFormatIP2Location
)

func detectDBFormat(path string) dbFormat {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mmdb":
		return dbFormatMaxMind
	case ".bin":
		return dbFormatIP2Location
	default:
		base := strings.ToLower(filepath.Base(path))
		switch {
		case strings.Contains(base, "ip2location"):
			return dbFormatIP2Location
		case strings.Contains(base, "geoip") || strings.Contains(base, "geolite"):
			return dbFormatMaxMind
		default:
			return dbFormatUnknown
		}
	}
}

func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isPrivateOrLocal(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		if ipv4[0] == 169 && ipv4[1] == 254 {
			return true
		}
		return false
	}
	if isUniqueLocalIPv6(ip) {
		return true
	}
	return false
}

func isUniqueLocalIPv6(ip net.IP) bool {
	if ip == nil || ip.To4() != nil {
		return false
	}
	if len(ip) < net.IPv6len {
		return false
	}
	return ip[0]&0xfe == 0xfc
}

func ValidateResolver(r CountryResolver) error {
	if r == nil {
		return errors.New("country resolver is nil")
	}
	return nil
}
