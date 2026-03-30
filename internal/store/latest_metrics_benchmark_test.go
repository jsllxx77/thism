package store_test

import (
	"fmt"
	"testing"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func BenchmarkStoreListNodesWithLatestMetrics(b *testing.B) {
	s, err := store.New(":memory:")
	if err != nil {
		b.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	const nodeCount = 8
	const metricsPerNode = 4000

	nodeIDs := make([]string, 0, nodeCount)

	b.StopTimer()
	for nodeIndex := 0; nodeIndex < nodeCount; nodeIndex++ {
		nodeID := fmt.Sprintf("node-%02d", nodeIndex)
		nodeIDs = append(nodeIDs, nodeID)
		if err := s.UpsertNode(&models.Node{
			ID:        nodeID,
			Name:      fmt.Sprintf("node-%02d", nodeIndex),
			Token:     fmt.Sprintf("token-%02d", nodeIndex),
			CreatedAt: 1700000000 + int64(nodeIndex),
		}); err != nil {
			b.Fatalf("UpsertNode %s: %v", nodeID, err)
		}

		for sampleIndex := 0; sampleIndex < metricsPerNode; sampleIndex++ {
			if err := s.InsertMetrics(nodeID, &models.MetricsPayload{
				TS:            1700000000 + int64(sampleIndex),
				CPU:           float64((nodeIndex + sampleIndex) % 100),
				UptimeSeconds: uint64(sampleIndex),
				Mem: models.MemStats{
					Used:  uint64(1024 + sampleIndex),
					Total: 8192,
				},
				Net: models.NetStats{
					RxBytes: uint64(sampleIndex * 128),
					TxBytes: uint64(sampleIndex * 256),
				},
			}); err != nil {
				b.Fatalf("InsertMetrics %s/%d: %v", nodeID, sampleIndex, err)
			}
		}
	}
	b.StartTimer()

	b.ReportAllocs()
	for iteration := 0; iteration < b.N; iteration++ {
		nodes, err := s.ListNodesWithLatestMetrics()
		if err != nil {
			b.Fatalf("ListNodesWithLatestMetrics: %v", err)
		}
		if len(nodes) != len(nodeIDs) {
			b.Fatalf("expected %d nodes, got %d", len(nodeIDs), len(nodes))
		}
	}
}
