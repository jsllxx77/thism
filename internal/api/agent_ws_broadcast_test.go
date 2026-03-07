package api_test

import (
	"encoding/json"
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

	router := api.NewRouter(s, h, "admin-token", nil)
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
	dashboardQuery := dashboardURL.Query()
	dashboardQuery.Set("token", "admin-token")
	dashboardURL.RawQuery = dashboardQuery.Encode()

	dashboardConn, _, err := websocket.DefaultDialer.Dial(dashboardURL.String(), nil)
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
