package api

import (
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func TestDashboardInitialMessagesIncludeLatestSnapshot(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	h := hub.New(s)
	go h.Run()

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token",
		CreatedAt: time.Now().Unix(),
		LastSeen:  1733011260,
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	if err := s.InsertMetrics("node-1", &models.MetricsPayload{
		TS:  1733011200,
		CPU: 37.5,
		Mem: models.MemStats{Used: 2048, Total: 4096},
		Net: models.NetStats{RxBytes: 1234, TxBytes: 5678},
	}); err != nil {
		t.Fatalf("InsertMetrics: %v", err)
	}

	messages, err := dashboardInitialMessages(s, h)
	if err != nil {
		t.Fatalf("dashboardInitialMessages: %v", err)
	}

	var foundStatus bool
	var foundMetrics bool

	for _, msg := range messages {
		switch msg.Type {
		case "node_status":
			payload, ok := msg.Payload.(map[string]any)
			if !ok || payload["node_id"] != "node-1" {
				continue
			}
			foundStatus = true
			if payload["online"] != false {
				t.Fatalf("expected offline snapshot status, got %#v", payload["online"])
			}
		case "metrics":
			payload, ok := msg.Payload.(map[string]any)
			if !ok || payload["node_id"] != "node-1" {
				continue
			}
			foundMetrics = true

			if payload["last_seen"] != int64(1733011260) {
				t.Fatalf("expected last_seen 1733011260, got %#v", payload["last_seen"])
			}

			data, ok := payload["data"].(map[string]any)
			if !ok {
				t.Fatalf("expected metrics data map, got %#v", payload["data"])
			}
			if data["cpu"] != 37.5 {
				t.Fatalf("expected cpu 37.5, got %#v", data["cpu"])
			}

			mem, ok := data["mem"].(map[string]any)
			if !ok {
				t.Fatalf("expected mem map, got %#v", data["mem"])
			}
			if mem["used"] != uint64(2048) || mem["total"] != uint64(4096) {
				t.Fatalf("unexpected mem snapshot: %#v", mem)
			}
		}
	}

	if !foundStatus {
		t.Fatal("expected initial node_status snapshot message")
	}
	if !foundMetrics {
		t.Fatal("expected initial metrics snapshot message")
	}
}
