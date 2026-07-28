package geo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

// Manager owns the active offline GeoIP resolver and can hot-reload after
// settings or database updates from the admin UI.
type Manager struct {
	mu         sync.RWMutex
	dir        string
	settings   models.GeoIPSettings
	resolver   *Resolver
	lastError  string
	dbVersion  string
	dbPath     string
	dbExists   bool
	dbSize     int64
	dbModified int64
}

func NewManager(dir string) *Manager {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = DefaultDir
	}
	return &Manager{
		dir: dir,
		settings: models.GeoIPSettings{
			Provider: models.GeoIPProviderMaxMind,
		},
	}
}

func (m *Manager) ResolveCountryCode(ip string) string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	resolver := m.resolver
	m.mu.RUnlock()
	if resolver == nil {
		return ""
	}
	return resolver.ResolveCountryCode(ip)
}

func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeLocked()
}

func (m *Manager) ApplySettings(settings models.GeoIPSettings) error {
	if m == nil {
		return fmt.Errorf("geo manager is nil")
	}
	settings = normalizeGeoIPSettings(settings, m.dir)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings = settings
	if settings.Dir != "" {
		m.dir = settings.Dir
	}
	return m.reloadLocked()
}

func (m *Manager) Settings() models.GeoIPSettings {
	if m == nil {
		return models.GeoIPSettings{Provider: models.GeoIPProviderMaxMind}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings
}

func (m *Manager) View() models.GeoIPSettingsView {
	if m == nil {
		return models.GeoIPSettingsView{
			Provider:           models.GeoIPProviderMaxMind,
			SupportedProviders: []string{models.GeoIPProviderMaxMind, models.GeoIPProviderIP2Location},
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return models.GeoIPSettingsView{
		Provider:             m.settings.Provider,
		IP2LocationTokenSet:  strings.TrimSpace(m.settings.IP2LocationToken) != "",
		MaxMindLicenseKeySet: strings.TrimSpace(m.settings.MaxMindLicenseKey) != "",
		Enabled:              m.resolver != nil,
		DatabasePath:         m.dbPath,
		DatabaseExists:       m.dbExists,
		DatabaseSizeBytes:    m.dbSize,
		DatabaseModifiedAt:   m.dbModified,
		DatabaseVersion:      m.dbVersion,
		LastError:            m.lastError,
		SupportedProviders:   []string{models.GeoIPProviderMaxMind, models.GeoIPProviderIP2Location},
	}
}

func (m *Manager) UpdateSettings(update models.GeoIPSettingsUpdate) (models.GeoIPSettingsView, error) {
	if m == nil {
		return models.GeoIPSettingsView{}, fmt.Errorf("geo manager is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.settings
	provider := strings.ToLower(strings.TrimSpace(update.Provider))
	if provider != "" {
		next.Provider = provider
	}
	if update.ClearIP2LocationToken {
		next.IP2LocationToken = ""
	} else if strings.TrimSpace(update.IP2LocationToken) != "" {
		next.IP2LocationToken = strings.TrimSpace(update.IP2LocationToken)
	}
	if update.ClearMaxMindLicenseKey {
		next.MaxMindLicenseKey = ""
	} else if strings.TrimSpace(update.MaxMindLicenseKey) != "" {
		next.MaxMindLicenseKey = strings.TrimSpace(update.MaxMindLicenseKey)
	}
	next = normalizeGeoIPSettings(next, m.dir)
	m.settings = next
	if err := m.reloadLocked(); err != nil {
		// Keep settings even if the current DB file is missing; UI can trigger update.
		m.lastError = err.Error()
	}
	return m.viewLocked(), nil
}

func (m *Manager) RefreshDatabase(ctx context.Context) (models.GeoIPSettingsView, error) {
	if m == nil {
		return models.GeoIPSettingsView{}, fmt.Errorf("geo manager is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	m.mu.RLock()
	settings := m.settings
	dir := m.dir
	m.mu.RUnlock()

	var (
		dest string
		err  error
	)
	switch settings.Provider {
	case models.GeoIPProviderIP2Location:
		dest = filepath.Join(dir, DefaultIP2LocationName)
		err = DownloadIP2LocationDB(ctx, settings.IP2LocationToken, dest)
	default:
		dest = filepath.Join(dir, DefaultMaxMindName)
		err = DownloadMaxMindDB(ctx, settings.MaxMindLicenseKey, dest)
	}
	if err != nil {
		m.mu.Lock()
		m.lastError = err.Error()
		view := m.viewLocked()
		m.mu.Unlock()
		return view, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.reloadLocked(); err != nil {
		m.lastError = err.Error()
		return m.viewLocked(), err
	}
	m.lastError = ""
	return m.viewLocked(), nil
}

func (m *Manager) reloadLocked() error {
	_ = m.closeLocked()
	path := m.pathForProviderLocked(m.settings.Provider)
	m.dbPath = path
	m.inspectPathLocked(path)

	resolver, err := NewResolver(path)
	if err != nil {
		m.resolver = nil
		m.dbVersion = ""
		m.lastError = err.Error()
		return err
	}
	m.resolver = resolver
	m.dbVersion = detectDatabaseVersion(resolver, path)
	m.lastError = ""
	m.inspectPathLocked(path)
	return nil
}

func (m *Manager) closeLocked() error {
	if m.resolver == nil {
		return nil
	}
	err := m.resolver.Close()
	m.resolver = nil
	return err
}

func (m *Manager) pathForProviderLocked(provider string) string {
	allowLegacyFallback := filepath.Clean(m.dir) == filepath.Clean(DefaultDir)
	return managedDatabasePath(m.dir, LegacyDir, provider, allowLegacyFallback)
}

func managedDatabasePath(dir, legacyDir, provider string, allowLegacyFallback bool) string {
	name := DefaultMaxMindName
	if provider == models.GeoIPProviderIP2Location {
		name = DefaultIP2LocationName
	}

	managedPath := filepath.Join(dir, name)
	if _, err := os.Stat(managedPath); err == nil || !allowLegacyFallback {
		return managedPath
	}

	legacyPath := filepath.Join(legacyDir, name)
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath
	}
	return managedPath
}

func (m *Manager) inspectPathLocked(path string) {
	info, err := os.Stat(path)
	if err != nil {
		m.dbExists = false
		m.dbSize = 0
		m.dbModified = 0
		return
	}
	m.dbExists = true
	m.dbSize = info.Size()
	m.dbModified = info.ModTime().Unix()
}

func (m *Manager) viewLocked() models.GeoIPSettingsView {
	return models.GeoIPSettingsView{
		Provider:             m.settings.Provider,
		IP2LocationTokenSet:  strings.TrimSpace(m.settings.IP2LocationToken) != "",
		MaxMindLicenseKeySet: strings.TrimSpace(m.settings.MaxMindLicenseKey) != "",
		Enabled:              m.resolver != nil,
		DatabasePath:         m.dbPath,
		DatabaseExists:       m.dbExists,
		DatabaseSizeBytes:    m.dbSize,
		DatabaseModifiedAt:   m.dbModified,
		DatabaseVersion:      m.dbVersion,
		LastError:            m.lastError,
		SupportedProviders:   []string{models.GeoIPProviderMaxMind, models.GeoIPProviderIP2Location},
	}
}

func normalizeGeoIPSettings(settings models.GeoIPSettings, defaultDir string) models.GeoIPSettings {
	settings.Provider = strings.ToLower(strings.TrimSpace(settings.Provider))
	switch settings.Provider {
	case models.GeoIPProviderIP2Location:
		// ok
	default:
		settings.Provider = models.GeoIPProviderMaxMind
	}
	settings.IP2LocationToken = strings.TrimSpace(settings.IP2LocationToken)
	settings.MaxMindLicenseKey = strings.TrimSpace(settings.MaxMindLicenseKey)
	settings.Dir = strings.TrimSpace(settings.Dir)
	if settings.Dir == "" {
		settings.Dir = strings.TrimSpace(defaultDir)
	}
	if settings.Dir == "" {
		settings.Dir = DefaultDir
	}
	return settings
}

func detectDatabaseVersion(resolver *Resolver, path string) string {
	if resolver == nil {
		return ""
	}
	if resolver.ip2l != nil {
		return resolver.ip2l.DatabaseVersion()
	}
	if info, err := os.Stat(path); err == nil {
		return info.ModTime().UTC().Format(time.RFC3339)
	}
	return ""
}
