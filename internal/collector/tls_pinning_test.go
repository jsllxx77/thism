package collector

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func spkiSHA256Base64(cert *x509.Certificate) string {
	digest := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(digest[:])
}

func TestPinnedTLSConfigAcceptsMatchingSPKIPin(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	originalPin := ServerTLSSPKISHA256Base64
	ServerTLSSPKISHA256Base64 = spkiSHA256Base64(server.Certificate())
	t.Cleanup(func() {
		ServerTLSSPKISHA256Base64 = originalPin
	})

	transport := server.Client().Transport.(*http.Transport).Clone()
	pinnedConfig, err := tlsConfigWithServerPin(transport.TLSClientConfig)
	if err != nil {
		t.Fatalf("build pinned tls config: %v", err)
	}
	transport.TLSClientConfig = pinnedConfig
	client := &http.Client{Transport: transport}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected matching SPKI pin to connect: %v", err)
	}
	resp.Body.Close()
}

func TestPinnedTLSConfigRejectsMismatchedSPKIPin(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	originalPin := ServerTLSSPKISHA256Base64
	ServerTLSSPKISHA256Base64 = base64.StdEncoding.EncodeToString(make([]byte, sha256.Size))
	t.Cleanup(func() {
		ServerTLSSPKISHA256Base64 = originalPin
	})

	transport := server.Client().Transport.(*http.Transport).Clone()
	pinnedConfig, err := tlsConfigWithServerPin(transport.TLSClientConfig)
	if err != nil {
		t.Fatalf("build pinned tls config: %v", err)
	}
	transport.TLSClientConfig = pinnedConfig
	client := &http.Client{Transport: transport}

	_, err = client.Get(server.URL)
	if err == nil {
		t.Fatal("expected mismatched SPKI pin to reject the connection")
	}
	if !strings.Contains(err.Error(), "tls spki pin mismatch") {
		t.Fatalf("expected pin mismatch error, got %v", err)
	}
}

func TestSelfUpdateHTTPClientRejectsMismatchedSPKIPin(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	originalPin := ServerTLSSPKISHA256Base64
	ServerTLSSPKISHA256Base64 = base64.StdEncoding.EncodeToString(make([]byte, sha256.Size))
	t.Cleanup(func() {
		ServerTLSSPKISHA256Base64 = originalPin
	})

	client := newSelfUpdateHTTPClient()
	client.Transport = trustTestServerTransport(t, client.Transport, server)

	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("expected self-update HTTP client to reject mismatched SPKI pin")
	}
	if !strings.Contains(err.Error(), "tls spki pin mismatch") {
		t.Fatalf("expected pin mismatch error, got %v", err)
	}
}

func TestPinnedTLSConfigPreservesExistingVerifyConnection(t *testing.T) {
	originalPin := ServerTLSSPKISHA256Base64
	ServerTLSSPKISHA256Base64 = base64.StdEncoding.EncodeToString(make([]byte, sha256.Size))
	t.Cleanup(func() {
		ServerTLSSPKISHA256Base64 = originalPin
	})

	expectedErr := errors.New("existing verifier rejected")
	base := &tls.Config{
		VerifyConnection: func(tls.ConnectionState) error {
			return expectedErr
		},
	}

	pinnedConfig, err := tlsConfigWithServerPin(base)
	if err != nil {
		t.Fatalf("build pinned tls config: %v", err)
	}

	err = pinnedConfig.VerifyConnection(tls.ConnectionState{})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected existing verifier error, got %v", err)
	}
}

func trustTestServerTransport(t *testing.T, base http.RoundTripper, server *httptest.Server) http.RoundTripper {
	t.Helper()

	transport, ok := base.(*http.Transport)
	if !ok {
		pinningTransport, ok := base.(pinningRoundTripper)
		if !ok {
			t.Fatalf("unexpected transport type %T", base)
		}
		transport, ok = pinningTransport.base.(*http.Transport)
		if !ok {
			t.Fatalf("unexpected wrapped transport type %T", pinningTransport.base)
		}
		transport = transport.Clone()
		pinningTransport.base = transport
		base = pinningTransport
	} else {
		transport = transport.Clone()
		base = transport
	}
	tlsConfig, err := tlsConfigWithServerPin(server.Client().Transport.(*http.Transport).TLSClientConfig)
	if err != nil {
		t.Fatalf("build test tls config: %v", err)
	}
	transport.TLSClientConfig = tlsConfig
	return base
}

func TestDefaultWebsocketDialRejectsMismatchedSPKIPin(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			conn.Close()
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	serverURL.Scheme = "wss"

	originalPin := ServerTLSSPKISHA256Base64
	originalTLSConfig := websocket.DefaultDialer.TLSClientConfig
	ServerTLSSPKISHA256Base64 = base64.StdEncoding.EncodeToString(make([]byte, sha256.Size))
	websocket.DefaultDialer.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig
	t.Cleanup(func() {
		ServerTLSSPKISHA256Base64 = originalPin
		websocket.DefaultDialer.TLSClientConfig = originalTLSConfig
	})

	_, err = defaultWebsocketDial(dialModeAuto, serverURL.String(), nil)
	if err == nil {
		t.Fatal("expected mismatched SPKI pin to reject websocket dial")
	}
	if !strings.Contains(err.Error(), "tls spki pin mismatch") {
		t.Fatalf("expected pin mismatch error, got %v", err)
	}
}

func TestDefaultWebsocketDialIgnoresPinForPlainWebSocket(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			conn.Close()
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	serverURL.Scheme = "ws"

	originalPin := ServerTLSSPKISHA256Base64
	ServerTLSSPKISHA256Base64 = "not-base64"
	t.Cleanup(func() {
		ServerTLSSPKISHA256Base64 = originalPin
	})

	conn, err := defaultWebsocketDial(dialModeAuto, serverURL.String(), nil)
	if err != nil {
		t.Fatalf("expected plain websocket dial to ignore TLS pin: %v", err)
	}
	conn.Close()
}
