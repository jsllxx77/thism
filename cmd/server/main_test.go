package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerConfiguresTimeouts(t *testing.T) {
	server := newHTTPServer(":12026", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	if server.Addr != ":12026" {
		t.Fatalf("expected server addr to be preserved, got %q", server.Addr)
	}
	if server.ReadHeaderTimeout <= 0 {
		t.Fatal("expected ReadHeaderTimeout to be configured")
	}
	if server.ReadTimeout <= 0 {
		t.Fatal("expected ReadTimeout to be configured")
	}
	if server.WriteTimeout <= 0 {
		t.Fatal("expected WriteTimeout to be configured")
	}
	if server.IdleTimeout <= 0 {
		t.Fatal("expected IdleTimeout to be configured")
	}
}

func TestServeHTTPServerStopsOnContextCancel(t *testing.T) {
	server := newHTTPServer("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveHTTPServer(ctx, server, listener)
	}()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, reqErr := client.Get("http://" + listener.Addr().String())
		if reqErr == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became ready: %v", reqErr)
		}
		time.Sleep(25 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected graceful shutdown without error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for serveHTTPServer to stop after context cancellation")
	}
}
