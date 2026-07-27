package models

const (
	GeoIPProviderMaxMind     = "maxmind"
	GeoIPProviderIP2Location = "ip2location"
)

// GeoIPSettings is the persisted configuration for offline GeoIP resolution.
type GeoIPSettings struct {
	Provider           string `json:"provider"`
	IP2LocationToken   string `json:"ip2location_token,omitempty"`
	MaxMindLicenseKey  string `json:"maxmind_license_key,omitempty"`
	// Dir overrides the on-disk geo database directory when non-empty.
	Dir string `json:"dir,omitempty"`
}

// GeoIPSettingsUpdate is the admin API write payload.
// Empty secret fields keep the previously stored values.
type GeoIPSettingsUpdate struct {
	Provider              string `json:"provider"`
	IP2LocationToken      string `json:"ip2location_token,omitempty"`
	MaxMindLicenseKey     string `json:"maxmind_license_key,omitempty"`
	ClearIP2LocationToken bool   `json:"clear_ip2location_token,omitempty"`
	ClearMaxMindLicenseKey bool  `json:"clear_maxmind_license_key,omitempty"`
}

// GeoIPSettingsView is the admin/viewer-safe status payload.
type GeoIPSettingsView struct {
	Provider                string `json:"provider"`
	IP2LocationTokenSet     bool   `json:"ip2location_token_set"`
	MaxMindLicenseKeySet    bool   `json:"maxmind_license_key_set"`
	Enabled                 bool   `json:"enabled"`
	DatabasePath            string `json:"database_path,omitempty"`
	DatabaseExists          bool   `json:"database_exists"`
	DatabaseSizeBytes       int64  `json:"database_size_bytes,omitempty"`
	DatabaseModifiedAt      int64  `json:"database_modified_at,omitempty"`
	DatabaseVersion         string `json:"database_version,omitempty"`
	LastError               string `json:"last_error,omitempty"`
	SupportedProviders      []string `json:"supported_providers"`
}
