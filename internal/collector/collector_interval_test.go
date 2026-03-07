package collector_test

import (
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/collector"
)

func TestCollectorUsesDefaultReportInterval(t *testing.T) {
	c := collector.New("ws://localhost:9999", "token", "test", "")

	if got := c.ReportInterval(); got != collector.DefaultReportInterval {
		t.Fatalf("expected default interval %s, got %s", collector.DefaultReportInterval, got)
	}
}

func TestCollectorUsesCustomReportInterval(t *testing.T) {
	want := 12 * time.Second
	c := collector.NewWithInterval("ws://localhost:9999", "token", "test", "", want)

	if got := c.ReportInterval(); got != want {
		t.Fatalf("expected interval %s, got %s", want, got)
	}
}
