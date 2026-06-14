package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/thism-dev/thism/frontend"
	"github.com/thism-dev/thism/internal/api"
	frontendSkins "github.com/thism-dev/thism/internal/frontend/skins"
	"github.com/thism-dev/thism/internal/geo"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/store"
)

func openCountryResolver(dbPath string) geo.CountryResolver {
	countryResolver, err := geo.NewResolver(dbPath)
	if err != nil {
		log.Printf("geoip: disabled country resolver: %v", err)
		return nil
	}
	return countryResolver
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

func defaultFrontendSkinsDir(dbPath string) string {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" || dbPath == ":memory:" {
		return "./frontend-skins"
	}
	return filepath.Join(filepath.Dir(dbPath), "frontend-skins")
}

func main() {
	port := flag.String("port", envOr("THISM_PORT", "12026"), "HTTP port")
	dbPath := flag.String("db", envOr("THISM_DB", "./thism.db"), "SQLite database path")
	adminToken := flag.String("token", os.Getenv("THISM_TOKEN"), "Admin token for API auth (env: THISM_TOKEN)")
	adminUser := flag.String("admin-user", os.Getenv("THISM_ADMIN_USER"), "Admin username for login page authentication (env: THISM_ADMIN_USER)")
	adminPass := flag.String("admin-pass", os.Getenv("THISM_ADMIN_PASS"), "Admin password for login page authentication (env: THISM_ADMIN_PASS)")
	geoIPDBPath := flag.String("geoip-db", envOr("THISM_GEOIP_DB", geo.DefaultDBPath), "Path to local GeoIP mmdb database")
	frontendSkinsDir := flag.String("frontend-skins-dir", os.Getenv("THISM_FRONTEND_SKINS_DIR"), "Directory for installed frontend skin packages (env: THISM_FRONTEND_SKINS_DIR)")
	flag.Parse()

	if *adminToken == "" {
		log.Fatal("admin token is required: pass --token or set THISM_TOKEN")
	}
	if (*adminUser == "") != (*adminPass == "") {
		log.Fatal("admin-user and admin-pass must be provided together (via flags or THISM_ADMIN_USER / THISM_ADMIN_PASS)")
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

	skinDir := strings.TrimSpace(*frontendSkinsDir)
	if skinDir == "" {
		skinDir = defaultFrontendSkinsDir(*dbPath)
	}
	skinManager, err := frontendSkins.NewManager(skinDir)
	if err != nil {
		log.Fatalf("failed to open frontend skin manager: %v", err)
	}

	countryResolver := openCountryResolver(*geoIPDBPath)
	if closer, ok := countryResolver.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	frontendHandler := frontend.HandlerWithSkins(frontend.Handler(), skinManager, s)
	router := api.NewRouterWithAuthGeoAndFrontendSkins(s, h, api.AuthConfig{
		AdminToken: *adminToken,
		Username:   *adminUser,
		Password:   *adminPass,
	}, frontendHandler, countryResolver, skinManager)

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
