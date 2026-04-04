package collector

import (
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/models"
)

type latencyTestConn struct {
	reads      [][]byte
	readErr    error
	readIndex  int
	writes     [][]byte
	writeErr   error
	remoteAddr net.Addr
}

func (c *latencyTestConn) WriteMessage(_ int, data []byte) error {
	if c.writeErr != nil {
		return c.writeErr
	}
	copied := make([]byte, len(data))
	copy(copied, data)
	c.writes = append(c.writes, copied)
	return nil
}

func (c *latencyTestConn) ReadMessage() (int, []byte, error) {
	if c.readIndex >= len(c.reads) {
		if c.readErr != nil {
			return websocket.TextMessage, nil, c.readErr
		}
		return websocket.TextMessage, nil, errors.New("eof")
	}
	raw := c.reads[c.readIndex]
	c.readIndex++
	return websocket.TextMessage, raw, nil
}

func (c *latencyTestConn) Close() error         { return nil }
func (c *latencyTestConn) RemoteAddr() net.Addr { return c.remoteAddr }

func decodeLatencyResults(t *testing.T, writes [][]byte) []models.LatencyMonitorResult {
	t.Helper()
	results := make([]models.LatencyMonitorResult, 0, len(writes))
	for _, raw := range writes {
		var msg struct {
			Type    string                      `json:"type"`
			Payload models.LatencyMonitorResult `json:"payload"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal latency message: %v", err)
		}
		if msg.Type == "latency_result" {
			results = append(results, msg.Payload)
		}
	}
	return results
}

func TestCollectorReplacesLatencyMonitorConfig(t *testing.T) {
	collector := NewWithInterval("ws://localhost:12026", "token", "node", "", DefaultReportInterval)

	raw, err := json.Marshal(models.WSMessage{
		Type: "latency_monitor_config",
		Payload: models.LatencyMonitorConfigPayload{
			Monitors: []models.LatencyMonitor{
				{ID: "monitor-1", Name: "TCP 80", Type: models.LatencyMonitorTypeTCP, Target: "example.com:80", IntervalSeconds: 60},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal config payload: %v", err)
	}

	conn := &latencyTestConn{reads: [][]byte{raw}, readErr: errors.New("done")}
	var writeMu sync.Mutex

	_ = collector.readAgentCommands(conn, &writeMu)

	monitors := collector.latencyMonitorSnapshot()
	if len(monitors) != 1 || monitors[0].ID != "monitor-1" {
		t.Fatalf("expected config to be replaced, got %#v", monitors)
	}
}

func TestCollectorLatencyProbeScheduling(t *testing.T) {
	collector := NewWithInterval("ws://localhost:12026", "token", "node", "", DefaultReportInterval)
	collector.applyLatencyMonitorConfig([]models.LatencyMonitor{
		{ID: "monitor-1", Name: "TCP 80", Type: models.LatencyMonitorTypeTCP, Target: "example.com:80", IntervalSeconds: 60},
	})

	callCount := 0
	collector.tcpLatencyProbe = func(_ string) (float64, error) {
		callCount++
		return 12.5, nil
	}

	conn := &latencyTestConn{}
	var writeMu sync.Mutex

	collector.runDueLatencyMonitors(conn, &writeMu, time.Unix(100, 0))
	collector.runDueLatencyMonitors(conn, &writeMu, time.Unix(159, 0))
	collector.runDueLatencyMonitors(conn, &writeMu, time.Unix(160, 0))

	if callCount != 10 {
		t.Fatalf("expected 10 probe attempts across 2 cycles, got %d", callCount)
	}

	results := decodeLatencyResults(t, conn.writes)
	if len(results) != 2 {
		t.Fatalf("expected 2 latency result messages, got %#v", results)
	}
}

func TestCollectorLatencyProbeResultPayloads(t *testing.T) {
	collector := NewWithInterval("ws://localhost:12026", "token", "node", "", DefaultReportInterval)
	collector.applyLatencyMonitorConfig([]models.LatencyMonitor{
		{ID: "icmp-1", Name: "ICMP", Type: models.LatencyMonitorTypeICMP, Target: "1.1.1.1", IntervalSeconds: 60},
		{ID: "tcp-1", Name: "TCP", Type: models.LatencyMonitorTypeTCP, Target: "example.com:80", IntervalSeconds: 60},
		{ID: "http-1", Name: "HTTP", Type: models.LatencyMonitorTypeHTTP, Target: "https://example.com/healthz", IntervalSeconds: 60},
	})

	collector.icmpLatencyProbe = func(target string) (float64, error) {
		if target != "1.1.1.1" {
			t.Fatalf("unexpected icmp target %q", target)
		}
		return 8.5, nil
	}
	tcpAttempts := 0
	collector.tcpLatencyProbe = func(target string) (float64, error) {
		if target != "example.com:80" {
			t.Fatalf("unexpected tcp target %q", target)
		}
		tcpAttempts++
		if tcpAttempts <= 2 {
			return 0, errors.New("dial timeout")
		}
		return 15, nil
	}
	httpAttemptValues := []float64{20, 24, 22, 26, 28}
	collector.httpLatencyProbe = func(target string) (float64, error) {
		if target != "https://example.com/healthz" {
			t.Fatalf("unexpected http target %q", target)
		}
		value := httpAttemptValues[0]
		httpAttemptValues = httpAttemptValues[1:]
		return value, nil
	}

	conn := &latencyTestConn{}
	var writeMu sync.Mutex

	collector.runDueLatencyMonitors(conn, &writeMu, time.Unix(100, 0))

	results := decodeLatencyResults(t, conn.writes)
	if len(results) != 3 {
		t.Fatalf("expected 3 latency result messages, got %#v", results)
	}

	byID := map[string]models.LatencyMonitorResult{}
	for _, result := range results {
		byID[result.MonitorID] = result
	}

	if byID["icmp-1"].LatencyMs == nil || *byID["icmp-1"].LatencyMs != 8.5 || !byID["icmp-1"].Success {
		t.Fatalf("unexpected icmp result: %#v", byID["icmp-1"])
	}
	if byID["icmp-1"].LossPercent == nil || *byID["icmp-1"].LossPercent != 0 {
		t.Fatalf("unexpected icmp loss: %#v", byID["icmp-1"])
	}
	if byID["icmp-1"].JitterMs == nil || *byID["icmp-1"].JitterMs != 0 {
		t.Fatalf("unexpected icmp jitter: %#v", byID["icmp-1"])
	}

	if byID["tcp-1"].LatencyMs == nil || !byID["tcp-1"].Success || byID["tcp-1"].ErrorMessage != "" {
		t.Fatalf("unexpected tcp result: %#v", byID["tcp-1"])
	}
	if byID["tcp-1"].LossPercent == nil || *byID["tcp-1"].LossPercent != 40 {
		t.Fatalf("unexpected tcp loss: %#v", byID["tcp-1"])
	}
	if byID["tcp-1"].JitterMs == nil {
		t.Fatalf("expected tcp jitter to be computed: %#v", byID["tcp-1"])
	}

	if byID["http-1"].LatencyMs == nil || *byID["http-1"].LatencyMs != 24 || !byID["http-1"].Success {
		t.Fatalf("unexpected http result: %#v", byID["http-1"])
	}
	if byID["http-1"].LossPercent == nil || *byID["http-1"].LossPercent != 0 {
		t.Fatalf("unexpected http loss: %#v", byID["http-1"])
	}
	if byID["http-1"].JitterMs == nil {
		t.Fatalf("expected http jitter to be computed: %#v", byID["http-1"])
	}
}

func TestCollectorLatencyProbeAggregationSkipsJitterWithSingleSuccess(t *testing.T) {
	collector := NewWithInterval("ws://localhost:12026", "token", "node", "", DefaultReportInterval)
	collector.applyLatencyMonitorConfig([]models.LatencyMonitor{
		{ID: "tcp-1", Name: "TCP", Type: models.LatencyMonitorTypeTCP, Target: "example.com:80", IntervalSeconds: 60},
	})

	attempt := 0
	collector.tcpLatencyProbe = func(_ string) (float64, error) {
		attempt++
		if attempt == 5 {
			return 18, nil
		}
		return 0, errors.New("dial timeout")
	}

	conn := &latencyTestConn{}
	var writeMu sync.Mutex

	collector.runDueLatencyMonitors(conn, &writeMu, time.Unix(100, 0))

	results := decodeLatencyResults(t, conn.writes)
	if len(results) != 1 {
		t.Fatalf("expected 1 latency result message, got %#v", results)
	}
	if results[0].LatencyMs == nil || *results[0].LatencyMs != 18 {
		t.Fatalf("unexpected latency: %#v", results[0])
	}
	if results[0].LossPercent == nil || *results[0].LossPercent != 80 {
		t.Fatalf("unexpected loss percent: %#v", results[0])
	}
	if results[0].JitterMs != nil {
		t.Fatalf("expected jitter to remain empty with a single success, got %#v", results[0])
	}
}

func TestCollectorLatencyMonitorWriteErrorIsReturned(t *testing.T) {
	collector := NewWithInterval("ws://localhost:12026", "token", "node", "", DefaultReportInterval)
	collector.applyLatencyMonitorConfig([]models.LatencyMonitor{
		{ID: "tcp-1", Name: "TCP", Type: models.LatencyMonitorTypeTCP, Target: "example.com:80", IntervalSeconds: 60},
	})

	collector.tcpLatencyProbe = func(_ string) (float64, error) {
		return 12.5, nil
	}

	wantErr := errors.New("broken pipe")
	conn := &latencyTestConn{writeErr: wantErr}
	var writeMu sync.Mutex

	err := collector.runDueLatencyMonitors(conn, &writeMu, time.Unix(100, 0))
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected write error %v, got %v", wantErr, err)
	}
}
