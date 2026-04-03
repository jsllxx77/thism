package api_test

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func TestAgentMetricsBroadcastIncludesLastSeen(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := newIPv4TestServer(router)
	defer server.Close()

	err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token-1",
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	bootstrapURL := *baseURL
	bootstrapQuery := bootstrapURL.Query()
	bootstrapQuery.Set("token", "admin-token")
	bootstrapURL.RawQuery = bootstrapQuery.Encode()

	client := server.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	bootstrapResp, err := client.Get(bootstrapURL.String())
	if err != nil {
		t.Fatalf("bootstrap admin session: %v", err)
	}
	defer bootstrapResp.Body.Close()

	var sessionCookie *http.Cookie
	for _, cookie := range bootstrapResp.Cookies() {
		if cookie.Name == "thism_admin" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("expected bootstrap request to return a non-empty admin session cookie")
	}

	dashboardURL := *baseURL
	dashboardURL.Scheme = "ws"
	dashboardURL.Path = "/ws/dashboard"

	dashboardHeader := http.Header{}
	dashboardHeader.Add("Cookie", sessionCookie.String())

	dashboardConn, _, err := websocket.DefaultDialer.Dial(dashboardURL.String(), dashboardHeader)
	if err != nil {
		t.Fatalf("dial dashboard websocket: %v", err)
	}
	defer dashboardConn.Close()

	agentURL := *baseURL
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	agentQuery := agentURL.Query()
	agentQuery.Set("token", "agent-token-1")
	agentURL.RawQuery = agentQuery.Encode()

	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer agentConn.Close()

	payload := models.MetricsPayload{
		Type: "metrics",
		TS:   time.Now().Unix(),
		CPU:  12.5,
		IP:   "10.0.0.5",
		OS:   "linux",
		Arch: "amd64",
		Mem:  models.MemStats{Used: 1024, Total: 2048},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := agentConn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write agent metrics: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	_ = dashboardConn.SetReadDeadline(deadline)

	var lastSeen int64
	for {
		var msg struct {
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
		}
		if err := dashboardConn.ReadJSON(&msg); err != nil {
			t.Fatalf("read dashboard message: %v", err)
		}
		if msg.Type != "metrics" {
			continue
		}

		rawLastSeen, ok := msg.Payload["last_seen"].(float64)
		if !ok {
			t.Fatalf("expected metrics payload to include numeric last_seen, got %#v", msg.Payload["last_seen"])
		}
		lastSeen = int64(rawLastSeen)
		if lastSeen <= 0 {
			t.Fatalf("expected positive last_seen, got %d", lastSeen)
		}
		if msg.Payload["node_id"] != "node-1" {
			t.Fatalf("expected node_id node-1, got %#v", msg.Payload["node_id"])
		}
		break
	}

	node, err := s.GetNodeByID("node-1")
	if err != nil {
		t.Fatalf("load node: %v", err)
	}
	if node == nil {
		t.Fatal("expected node to exist")
	}
	if node.LastSeen != lastSeen {
		t.Fatalf("expected stored last_seen %d to match broadcast %d", node.LastSeen, lastSeen)
	}
}

func TestAgentMetricsDoesNotBroadcastWhenPersistenceFails(t *testing.T) {
	s, _ := store.New(":memory:")
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := newIPv4TestServer(router)
	defer server.Close()

	err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token-1",
		CreatedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	dashboardURL := *baseURL
	dashboardURL.Scheme = "ws"
	dashboardURL.Path = "/ws/dashboard"
	dashboardHeader := http.Header{}
	dashboardHeader.Set("Authorization", "Bearer admin-token")

	dashboardConn, _, err := websocket.DefaultDialer.Dial(dashboardURL.String(), dashboardHeader)
	if err != nil {
		t.Fatalf("dial dashboard websocket: %v", err)
	}
	defer dashboardConn.Close()

	var initial struct {
		Type string `json:"type"`
	}
	_ = dashboardConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := dashboardConn.ReadJSON(&initial); err != nil {
		t.Fatalf("read initial dashboard message: %v", err)
	}
	if initial.Type != "node_status" {
		t.Fatalf("expected initial node_status message, got %q", initial.Type)
	}

	agentURL := *baseURL
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	agentQuery := agentURL.Query()
	agentQuery.Set("token", "agent-token-1")
	agentURL.RawQuery = agentQuery.Encode()

	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer agentConn.Close()

	var onlineMsg struct {
		Type string `json:"type"`
	}
	_ = dashboardConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := dashboardConn.ReadJSON(&onlineMsg); err != nil {
		t.Fatalf("read online node status: %v", err)
	}
	if onlineMsg.Type != "node_status" {
		t.Fatalf("expected node_status after agent connection, got %q", onlineMsg.Type)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("close store to force persistence failure: %v", err)
	}

	payload := models.MetricsPayload{
		Type: "metrics",
		TS:   time.Now().Unix(),
		CPU:  12.5,
		Mem:  models.MemStats{Used: 1024, Total: 2048},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := agentConn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write agent metrics: %v", err)
	}

	_ = dashboardConn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var msg map[string]any
	err = dashboardConn.ReadJSON(&msg)
	if err == nil {
		t.Fatalf("expected no dashboard broadcast when persistence fails, got %#v", msg)
	}
	netErr, ok := err.(net.Error)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected read timeout when no broadcast is sent, got %v", err)
	}
}

func TestDashboardWebSocketBroadcastsLatencyResults(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := newIPv4TestServer(router)
	defer server.Close()

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token-1",
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		t.Fatalf("seed node: %v", err)
	}
	if err := s.CreateLatencyMonitor(&models.LatencyMonitor{
		ID:                 "monitor-1",
		Name:               "TCP 80",
		Type:               models.LatencyMonitorTypeTCP,
		Target:             "example.com:80",
		IntervalSeconds:    60,
		AutoAssignNewNodes: true,
		CreatedAt:          1700000000,
		UpdatedAt:          1700000000,
	}, []string{"node-1"}); err != nil {
		t.Fatalf("CreateLatencyMonitor: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	dashboardURL := *baseURL
	dashboardURL.Scheme = "ws"
	dashboardURL.Path = "/ws/dashboard"
	dashboardHeader := http.Header{}
	dashboardHeader.Set("Authorization", "Bearer admin-token")

	dashboardConn, _, err := websocket.DefaultDialer.Dial(dashboardURL.String(), dashboardHeader)
	if err != nil {
		t.Fatalf("dial dashboard websocket: %v", err)
	}
	defer dashboardConn.Close()

	agentURL := *baseURL
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	agentQuery := agentURL.Query()
	agentQuery.Set("token", "agent-token-1")
	agentURL.RawQuery = agentQuery.Encode()

	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer agentConn.Close()

	latency := 24.5
	loss := 40.0
	jitter := 8.25
	payload := models.WSMessage{
		Type: "latency_result",
		Payload: models.LatencyMonitorResult{
			MonitorID:   "monitor-1",
			NodeID:      "node-1",
			TS:          time.Now().Unix(),
			LatencyMs:   &latency,
			LossPercent: &loss,
			JitterMs:    &jitter,
			Success:     true,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal latency payload: %v", err)
	}
	if err := agentConn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write latency payload: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	_ = dashboardConn.SetReadDeadline(deadline)

	for {
		var msg struct {
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
		}
		if err := dashboardConn.ReadJSON(&msg); err != nil {
			t.Fatalf("read dashboard message: %v", err)
		}
		if msg.Type != "latency_result" {
			continue
		}
		if msg.Payload["node_id"] != "node-1" {
			t.Fatalf("expected node_id node-1, got %#v", msg.Payload["node_id"])
		}
		data, ok := msg.Payload["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data payload, got %#v", msg.Payload["data"])
		}
		if data["monitor_id"] != "monitor-1" {
			t.Fatalf("expected monitor_id monitor-1, got %#v", data["monitor_id"])
		}
		if data["success"] != true {
			t.Fatalf("expected success true, got %#v", data["success"])
		}
		if data["latency_ms"] != latency {
			t.Fatalf("expected latency %v, got %#v", latency, data["latency_ms"])
		}
		if data["loss_percent"] != loss {
			t.Fatalf("expected loss %v, got %#v", loss, data["loss_percent"])
		}
		if data["jitter_ms"] != jitter {
			t.Fatalf("expected jitter %v, got %#v", jitter, data["jitter_ms"])
		}
		break
	}
}
