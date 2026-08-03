package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thism-dev/thism/frontend"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/geo"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/store"
)

func openCountryResolver(dbPath string) geo.CountryResolver {
	return openCountryResolverWithFallback(dbPath, "")
}

func openCountryResolverWithFallback(primaryPath, fallbackPath string) geo.CountryResolver {
	primaryPath = strings.TrimSpace(primaryPath)
	fallbackPath = strings.TrimSpace(fallbackPath)

	var (
		countryResolver geo.CountryResolver
		err             error
	)
	switch {
	case primaryPath != "" || fallbackPath != "":
		countryResolver, err = geo.NewResolverWithFallback(primaryPath, fallbackPath)
	default:
		countryResolver, err = geo.OpenBestResolver()
	}
	if err != nil {
		log.Printf("geoip: disabled country resolver: %v", err)
		return nil
	}
	if resolver, ok := countryResolver.(*geo.Resolver); ok {
		if sources := resolver.Sources(); len(sources) > 0 {
			log.Printf("geoip: loaded database sources: %s", strings.Join(sources, ", "))
		}
	}
	return countryResolver
}

func openGeoIPManager(s *store.Store, dir string) *geo.Manager {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = geo.DefaultDir
	}
	manager := geo.NewManager(dir)
	if s != nil {
		if settings, err := s.GetGeoIPSettings(); err != nil {
			log.Printf("geoip: failed to load settings: %v", err)
		} else {
			settings.Dir = dir
			if err := manager.ApplySettings(settings); err != nil {
				log.Printf("geoip: disabled country resolver: %v", err)
			}
		}
	} else if err := manager.ApplySettings(manager.Settings()); err != nil {
		log.Printf("geoip: disabled country resolver: %v", err)
	}
	if view := manager.View(); view.Enabled {
		log.Printf("geoip: provider=%s path=%s version=%s", view.Provider, view.DatabasePath, view.DatabaseVersion)
	}
	return manager
}

// envOr returns the value of the named environment variable when set,
// otherwise the provided fallback. Used so flag defaults can read
// sensitive values from the environment, keeping them off the command
// line (and out of /proc/<pid>/cmdline).
func envOr(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return fallback
}

// envOrFile returns envName when set, otherwise it reads a value from
// fileEnvName when that environment variable points to a Docker/Kubernetes
// secret file. It trims only trailing newlines so generated secret files work
// without changing intentional spaces in the secret.
func envOrFile(envName, fileEnvName, fallback string) (string, error) {
	if v, ok := os.LookupEnv(envName); ok {
		return v, nil
	}
	path, ok := os.LookupEnv(fileEnvName)
	if !ok {
		return fallback, nil
	}
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s (%s): %w", fileEnvName, path, err)
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}

func mustEnvOrFile(envName, fileEnvName, fallback string) string {
	v, err := envOrFile(envName, fileEnvName, fallback)
	if err != nil {
		log.Fatalf("failed to load %s: %v", envName, err)
	}
	return v
}

func main() {
	port := flag.String("port", envOr("THISM_PORT", "12026"), "HTTP port")
	dbPath := flag.String("db", envOr("THISM_DB", "./thism.db"), "SQLite database path")
	adminToken := flag.String("token", mustEnvOrFile("THISM_TOKEN", "THISM_TOKEN_FILE", ""), "Admin token for API auth (env: THISM_TOKEN or THISM_TOKEN_FILE)")
	adminUser := flag.String("admin-user", mustEnvOrFile("THISM_ADMIN_USER", "THISM_ADMIN_USER_FILE", ""), "Admin username for login page authentication (env: THISM_ADMIN_USER or THISM_ADMIN_USER_FILE)")
	adminPass := flag.String("admin-pass", mustEnvOrFile("THISM_ADMIN_PASS", "THISM_ADMIN_PASS_FILE", ""), "Admin password for login page authentication (env: THISM_ADMIN_PASS or THISM_ADMIN_PASS_FILE)")
	geoIPDir := flag.String("geoip-dir", envOr("THISM_GEOIP_DIR", geo.DefaultDir), "Directory for offline GeoIP databases")
	geoIPDBPath := flag.String("geoip-db", os.Getenv("THISM_GEOIP_DB"), "Legacy single GeoIP database path override. Empty = use settings-managed provider")
	geoIPDBFallback := flag.String("geoip-db-fallback", os.Getenv("THISM_GEOIP_DB_FALLBACK"), "Optional legacy fallback GeoIP database path")
	flag.Parse()

	if *adminToken == "" {
		log.Fatal("admin token is required: pass --token or set THISM_TOKEN / THISM_TOKEN_FILE")
	}
	if (*adminUser == "") != (*adminPass == "") {
		log.Fatal("admin-user and admin-pass must be provided together (via flags, THISM_ADMIN_USER / THISM_ADMIN_PASS, or THISM_ADMIN_USER_FILE / THISM_ADMIN_PASS_FILE)")
	}

	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	startMetricsRetentionPruner(s)
	startMetricsRolluper(s)

	h := hub.New(s)
	go h.Run()

	var countryResolver geo.CountryResolver
	var geoManager *geo.Manager
	if strings.TrimSpace(*geoIPDBPath) != "" || strings.TrimSpace(*geoIPDBFallback) != "" {
		countryResolver = openCountryResolverWithFallback(*geoIPDBPath, *geoIPDBFallback)
		if closer, ok := countryResolver.(interface{ Close() error }); ok {
			defer closer.Close()
		}
	} else {
		geoManager = openGeoIPManager(s, *geoIPDir)
		defer geoManager.Close()
		countryResolver = geoManager
	}

	frontendHandler := frontend.Handler()
	router := api.NewRouterWithAuthGeoManager(s, h, api.AuthConfig{
		AdminToken: *adminToken,
		Username:   *adminUser,
		Password:   *adminPass,
	}, frontendHandler, countryResolver, geoManager)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := newHTTPServer(":"+*port, router)
	log.Printf("ThisM server listening on :%s", *port)
	if err := serveHTTPServer(ctx, server, nil); err != nil {
		log.Fatal(err)
	}
}

const metricsRetentionPruneInterval = time.Hour
const metricsRollupInterval = time.Minute
const (
	serverReadHeaderTimeout = 5 * time.Second
	serverReadTimeout       = 15 * time.Second
	serverWriteTimeout      = 30 * time.Second
	serverIdleTimeout       = 60 * time.Second
	serverShutdownTimeout   = 10 * time.Second
)

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}
}

func serveHTTPServer(ctx context.Context, server *http.Server, listener net.Listener) error {
	if server == nil {
		return errors.New("http server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var err error
	if listener == nil {
		listener, err = net.Listen("tcp", server.Addr)
		if err != nil {
			return err
		}
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- server.Serve(listener)
	}()

	select {
	case err := <-serveErrCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancel()

		shutdownErr := server.Shutdown(shutdownCtx)
		serveErr := <-serveErrCh
		if shutdownErr != nil {
			return shutdownErr
		}
		if errors.Is(serveErr, http.ErrServerClosed) {
			return nil
		}
		return serveErr
	}
}

func startMetricsRetentionPruner(s *store.Store) {
	go func() {
		pruneMetrics(s)
		ticker := time.NewTicker(metricsRetentionPruneInterval)
		defer ticker.Stop()
		for range ticker.C {
			pruneMetrics(s)
		}
	}()
}

func pruneMetrics(s *store.Store) {
	days, err := s.GetMetricsRetentionDays()
	if err != nil {
		log.Printf("metrics retention: failed to load retention days: %v", err)
		return
	}
	if err := s.PruneOldMetrics(days); err != nil {
		log.Printf("metrics retention: failed to prune old metrics: %v", err)
		return
	}
	if err := s.ReclaimSpace(); err != nil {
		log.Printf("metrics retention: failed to reclaim freed space: %v", err)
	}
}

func startMetricsRolluper(s *store.Store) {
	go func() {
		ticker := time.NewTicker(metricsRollupInterval)
		defer ticker.Stop()

		// Give the server a moment to finish booting.
		time.Sleep(2 * time.Second)
		rollupMetrics(s)
		for range ticker.C {
			rollupMetrics(s)
		}
	}()
}

func rollupMetrics(s *store.Store) {
	now := time.Now().Unix()
	// Roll up the last 15 minutes to cover delayed arrivals.
	from := now - int64((15 * time.Minute).Seconds())
	to := now
	if err := s.RollupMetrics1m(from, to); err != nil {
		log.Printf("metrics rollup: failed: %v", err)
	}
	if err := s.RollupLatencyResults1m(from, to); err != nil {
		log.Printf("latency rollup: failed: %v", err)
	}
}
