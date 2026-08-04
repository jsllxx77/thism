package collector

import (
	"path/filepath"
	"testing"
	"time"
)

// newTestCollector builds a collector whose durable agent state lives in an
// isolated per-test directory so tests never read leftover records or
// watermarks from other tests.
func newTestCollector(t *testing.T, serverURL, token, name, nodeIP string, reportInterval time.Duration) *Collector {
	t.Helper()
	collector := NewWithInterval(serverURL, token, name, nodeIP, reportInterval)
	collector.statePath = filepath.Join(t.TempDir(), "agent-state.json")
	collector.loadAgentState()
	return collector
}
