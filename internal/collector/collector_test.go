package collector_test

import (
	"testing"

	"github.com/thism-dev/thism/internal/collector"
)

func TestCollectMetrics(t *testing.T) {
	c := collector.New("ws://localhost:9999", "token", "test", "")
	m, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if m.CPU < 0 || m.CPU > 100 {
		t.Errorf("CPU out of range: %v", m.CPU)
	}
	if m.Mem.Total == 0 {
		t.Error("mem total should not be zero")
	}
	if len(m.Disk) == 0 {
		t.Error("expected at least one disk partition")
	}
}

func TestCollectMetricsWithExplicitIP(t *testing.T) {
	c := collector.New("ws://localhost:9999", "token", "test", "203.0.113.10")
	m, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if m.IP != "203.0.113.10" {
		t.Fatalf("expected explicit ip override, got %s", m.IP)
	}
}
