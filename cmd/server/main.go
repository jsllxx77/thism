package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/thism-dev/thism/frontend"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/store"
)

func main() {
	port := flag.String("port", "12026", "HTTP port")
	dbPath := flag.String("db", "./thism.db", "SQLite database path")
	adminToken := flag.String("token", "", "Admin token for API auth")
	adminUser := flag.String("admin-user", "", "Admin username for login page authentication")
	adminPass := flag.String("admin-pass", "", "Admin password for login page authentication")
	flag.Parse()

	if *adminToken == "" {
		log.Fatal("--token is required")
	}
	if (*adminUser == "") != (*adminPass == "") {
		log.Fatal("--admin-user and --admin-pass must be provided together")
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

	router := api.NewRouterWithAuth(s, h, api.AuthConfig{
		AdminToken: *adminToken,
		Username:   *adminUser,
		Password:   *adminPass,
	}, frontend.Handler())

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
