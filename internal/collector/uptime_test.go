package collector

import (
	"testing"

	"github.com/shirou/gopsutil/v3/host"
)

func TestCollectIncludesUptimeSeconds(t *testing.T) {
	originalHostInfoFunc := hostInfoFunc
	defer func() {
		hostInfoFunc = originalHostInfoFunc
	}()

	hostInfoFunc = func() (*host.InfoStat, error) {
		return &host.InfoStat{Uptime: 3723}, nil
	}

	c := New("ws://localhost:9999", "token", "test", "")
	metrics, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if metrics.UptimeSeconds != 3723 {
		t.Fatalf("expected uptime 3723, got %d", metrics.UptimeSeconds)
	}
}
