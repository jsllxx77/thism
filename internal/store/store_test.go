package store_test

import (
	"os"
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func TestStoreNodeCRUD(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	node := &models.Node{
		ID:        "node-1",
		Name:      "test-server",
		Token:     "secret-token",
		IP:        "1.2.3.4",
		OS:        "linux",
		Arch:      "amd64",
		CreatedAt: time.Now().Unix(),
		LastSeen:  time.Now().Unix(),
	}

	if err := s.UpsertNode(node); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	got, err := s.GetNodeByToken("secret-token")
	if err != nil {
		t.Fatalf("GetNodeByToken: %v", err)
	}
	if got.Name != "test-server" {
		t.Errorf("expected name test-server, got %s", got.Name)
	}

	nodes, err := s.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
}

func TestStoreMetrics(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	s.UpsertNode(&models.Node{ID: "n1", Token: "t1", Name: "n1"})

	m := &models.MetricsPayload{
		TS:  time.Now().Unix(),
		CPU: 55.0,
		Mem: models.MemStats{Used: 1024, Total: 4096},
	}
	if err := s.InsertMetrics("n1", m); err != nil {
		t.Fatalf("InsertMetrics: %v", err)
	}

	rows, err := s.QueryMetrics("n1", time.Now().Add(-time.Minute).Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("QueryMetrics: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
