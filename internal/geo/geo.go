package geo

import (
	"errors"
	"log"
	"net"
	"net/netip"
	"os"
	"strings"

	maxminddb "github.com/oschwald/maxminddb-golang/v2"
)

const DefaultDBPath = "/opt/1panel/geo/GeoIP.mmdb"

type CountryResolver interface {
	ResolveCountryCode(ip string) string
}

type Resolver struct {
	db *maxminddb.Reader
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

func NewResolver(dbPath string) (*Resolver, error) {
	path := strings.TrimSpace(dbPath)
	if path == "" {
		path = DefaultDBPath
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	db, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	return &Resolver{db: db}, nil
}

func MustNewResolver(dbPath string) *Resolver {
	resolver, err := NewResolver(dbPath)
	if err != nil {
		log.Fatalf("geoip: failed to open database: %v", err)
	}
	return resolver
}

func (r *Resolver) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *Resolver) ResolveCountryCode(ip string) string {
	if r == nil || r.db == nil {
		return ""
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil || isPrivateOrLocal(parsed) {
		return ""
	}
	addr, err := netip.ParseAddr(parsed.String())
	if err != nil {
		return ""
	}
	var result countryLookup
	if err := r.db.Lookup(addr).Decode(&result); err != nil {
		return ""
	}
	code := strings.ToUpper(strings.TrimSpace(result.Country.ISOCode))
	if len(code) != 2 {
		code = strings.ToUpper(strings.TrimSpace(result.RegisteredCountry.ISOCode))
	}
	if len(code) != 2 {
		code = strings.ToUpper(strings.TrimSpace(result.ISO))
	}
	if len(code) != 2 {
		return ""
	}
	return code
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
