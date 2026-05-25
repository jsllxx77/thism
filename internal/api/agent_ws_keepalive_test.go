package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func newAgentKeepaliveTestServer(handler http.Handler) *httptest.Server {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	server := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler},
	}
	server.Start()
	return server
}

func dialAgentKeepaliveTestConn(t *testing.T, serverURL, token string) *websocket.Conn {
	t.Helper()

	agentURL, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	query := agentURL.Query()
	query.Set("token", token)
	agentURL.RawQuery = query.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	return conn
}

func waitForAgentKeepaliveOnline(t *testing.T, h *hub.Hub, nodeID string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h.IsOnline(nodeID) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("node %s did not become online in time", nodeID)
}

func TestAgentMetricsHeartbeatsKeepConnectionOnlinePastReadDeadline(t *testing.T) {
	oldPongWait := websocketPongWait
	websocketPongWait = 200 * time.Millisecond
	t.Cleanup(func() {
		websocketPongWait = oldPongWait
	})

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	h := hub.New(s)
	go h.Run()

	router := NewRouter(s, h, "admin-token", nil)
	server := newAgentKeepaliveTestServer(router)
	defer server.Close()

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token-1",
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatalf("seed node: %v", err)
	}

	conn := dialAgentKeepaliveTestConn(t, server.URL, "agent-token-1")
	defer conn.Close()
	waitForAgentKeepaliveOnline(t, h, "node-1")

	deadline := time.Now().Add(550 * time.Millisecond)
	for time.Now().Before(deadline) {
		payload := models.MetricsPayload{
			Type: "metrics",
			TS:   time.Now().Unix(),
			CPU:  10,
			Mem:  models.MemStats{Used: 1, Total: 2},
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal metrics payload: %v", err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
			t.Fatalf("write metrics heartbeat: %v", err)
		}
		time.Sleep(75 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)
	if !h.IsOnline("node-1") {
		t.Fatal("expected metrics heartbeats to keep the agent websocket online past the read deadline")
	}
}

func TestAgentWebSocketSendsServerPingKeepalive(t *testing.T) {
	oldPingPeriod := websocketPingPeriod
	websocketPingPeriod = 25 * time.Millisecond
	t.Cleanup(func() {
		websocketPingPeriod = oldPingPeriod
	})

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	h := hub.New(s)
	go h.Run()

	router := NewRouter(s, h, "admin-token", nil)
	server := newAgentKeepaliveTestServer(router)
	defer server.Close()

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token-1",
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatalf("seed node: %v", err)
	}

	conn := dialAgentKeepaliveTestConn(t, server.URL, "agent-token-1")
	defer conn.Close()
	waitForAgentKeepaliveOnline(t, h, "node-1")

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	for {
		var msg struct {
			Type string `json:"type"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read agent websocket message: %v", err)
		}
		if msg.Type == "server_ping" {
			return
		}
	}
}
