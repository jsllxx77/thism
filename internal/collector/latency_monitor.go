package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/models"
)

type latencyProbeFunc func(target string) (float64, error)

type latencyMonitorState struct {
	monitor models.LatencyMonitor
	lastRun time.Time
}

const latencyProbeSamplesPerCycle = 10

var pingLatencyPattern = regexp.MustCompile(`time[=<]([0-9.]+)\s*ms`)

func (c *Collector) applyLatencyMonitorConfig(monitors []models.LatencyMonitor) {
	c.latencyMu.Lock()
	defer c.latencyMu.Unlock()

	next := make(map[string]*latencyMonitorState, len(monitors))
	for _, monitor := range monitors {
		if strings.TrimSpace(monitor.ID) == "" || monitor.IntervalSeconds <= 0 {
			continue
		}
		state := &latencyMonitorState{monitor: monitor}
		if existing := c.latencyMonitors[monitor.ID]; existing != nil {
			state.lastRun = existing.lastRun
		}
		next[monitor.ID] = state
	}
	c.latencyMonitors = next
}

func (c *Collector) latencyMonitorSnapshot() []models.LatencyMonitor {
	c.latencyMu.Lock()
	defer c.latencyMu.Unlock()

	monitors := make([]models.LatencyMonitor, 0, len(c.latencyMonitors))
	for _, state := range c.latencyMonitors {
		monitors = append(monitors, state.monitor)
	}
	sort.Slice(monitors, func(i, j int) bool {
		if monitors[i].Name != monitors[j].Name {
			return monitors[i].Name < monitors[j].Name
		}
		return monitors[i].ID < monitors[j].ID
	})
	return monitors
}

func (c *Collector) runDueLatencyMonitors(conn websocketConn, writeMu *sync.Mutex, currentTime time.Time) error {
	c.latencyMu.Lock()
	due := make([]models.LatencyMonitor, 0, len(c.latencyMonitors))
	for _, state := range c.latencyMonitors {
		interval := time.Duration(state.monitor.IntervalSeconds) * time.Second
		if interval <= 0 {
			continue
		}
		if !state.lastRun.IsZero() && currentTime.Sub(state.lastRun) < interval {
			continue
		}
		state.lastRun = currentTime
		due = append(due, state.monitor)
	}
	c.latencyMu.Unlock()

	sort.Slice(due, func(i, j int) bool {
		return due[i].ID < due[j].ID
	})

	for _, monitor := range due {
		if err := c.executeLatencyMonitor(monitor, conn, writeMu, currentTime); err != nil {
			return err
		}
	}

	return nil
}

func (c *Collector) executeLatencyMonitor(monitor models.LatencyMonitor, conn websocketConn, writeMu *sync.Mutex, currentTime time.Time) error {
	result := models.LatencyMonitorResult{
		MonitorID: monitor.ID,
		TS:        currentTime.Unix(),
	}

	latency, lossPercent, jitterMs, err := c.probeLatencyMonitorSummary(monitor)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
	} else {
		result.Success = true
		result.LatencyMs = &latency
	}
	result.LossPercent = &lossPercent
	result.JitterMs = jitterMs

	if err := c.sendLatencyResult(conn, writeMu, result); err != nil {
		log.Printf("collector: send latency result for monitor %s failed: %v", monitor.ID, err)
		return err
	}

	return nil
}

func (c *Collector) probeLatencyMonitor(monitor models.LatencyMonitor) (float64, error) {
	switch monitor.Type {
	case models.LatencyMonitorTypeICMP:
		return c.icmpLatencyProbe(monitor.Target)
	case models.LatencyMonitorTypeTCP:
		return c.tcpLatencyProbe(monitor.Target)
	case models.LatencyMonitorTypeHTTP:
		return c.httpLatencyProbe(monitor.Target)
	default:
		return 0, fmt.Errorf("unsupported latency monitor type")
	}
}

func (c *Collector) probeLatencyMonitorSummary(monitor models.LatencyMonitor) (float64, float64, *float64, error) {
	latencies := make([]float64, 0, latencyProbeSamplesPerCycle)
	failures := 0
	attempts := 0
	var lastErr error

	// Bound the total time spent probing within a single cycle. Probes run
	// serially and a dead target can stall on each probe's own timeout (HTTP
	// up to 10s), so a full sample run could otherwise exceed the monitor's
	// interval. We spend at most 3/4 of the interval, leaving headroom for the
	// cycle to finish before the next tick, and always take at least one
	// sample regardless of budget.
	start := c.currentTime()
	var deadline time.Time
	if monitor.IntervalSeconds > 0 {
		budget := time.Duration(monitor.IntervalSeconds) * time.Second * 3 / 4
		deadline = start.Add(budget)
	}

	for attempt := 0; attempt < latencyProbeSamplesPerCycle; attempt++ {
		if attempt > 0 && !deadline.IsZero() && !c.currentTime().Before(deadline) {
			break
		}
		attempts++
		latency, err := c.probeLatencyMonitor(monitor)
		if err != nil {
			failures++
			lastErr = err
			continue
		}
		latencies = append(latencies, latency)
	}

	// Loss is measured against attempts actually made, not the configured
	// sample count, so a budget-truncated cycle never reports unmeasured
	// samples as lost.
	lossPercent := 0.0
	if attempts > 0 {
		lossPercent = (float64(failures) / float64(attempts)) * 100
	}
	if len(latencies) == 0 {
		if lastErr == nil {
			lastErr = fmt.Errorf("probe failed")
		}
		return 0, lossPercent, nil, lastErr
	}

	average := averageLatency(latencies)
	jitter := jitterLatency(latencies)
	return average, lossPercent, jitter, nil
}

func averageLatency(latencies []float64) float64 {
	if len(latencies) == 0 {
		return 0
	}
	total := 0.0
	for _, latency := range latencies {
		total += latency
	}
	return total / float64(len(latencies))
}

// jitterLatency returns packet delay variation following the RFC 3550 spirit:
// the mean of the absolute differences between consecutive latency samples.
// This reflects how much latency swings probe-to-probe, which is what network
// "jitter" conventionally means (as opposed to the standard deviation around
// the mean). Requires at least two samples to have a consecutive pair.
func jitterLatency(latencies []float64) *float64 {
	if len(latencies) < 2 {
		return nil
	}
	totalDelta := 0.0
	for i := 1; i < len(latencies); i++ {
		totalDelta += math.Abs(latencies[i] - latencies[i-1])
	}
	value := totalDelta / float64(len(latencies)-1)
	return &value
}

func (c *Collector) sendLatencyResult(conn websocketConn, writeMu *sync.Mutex, payload models.LatencyMonitorResult) error {
	if conn == nil {
		return fmt.Errorf("websocket connection is nil")
	}
	raw, err := json.Marshal(models.WSMessage{Type: "latency_result", Payload: payload})
	if err != nil {
		return err
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, raw)
}

func defaultTCPLatencyProbe(target string) (float64, error) {
	startedAt := time.Now()
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return float64(time.Since(startedAt).Microseconds()) / 1000, nil
}

func defaultHTTPLatencyProbe(target string) (float64, error) {
	startedAt := time.Now()
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return float64(time.Since(startedAt).Microseconds()) / 1000, nil
}

func defaultICMPLatencyProbe(target string) (float64, error) {
	args := []string{"-c", "1", target}
	switch runtime.GOOS {
	case "windows":
		args = []string{"-n", "1", "-w", "2000", target}
	case "darwin":
		args = []string{"-c", "1", "-W", "2000", target}
	default:
		args = []string{"-c", "1", "-W", "2", target}
	}

	cmd := exec.Command("ping", args...)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return 0, errors.New(strings.TrimSpace(string(raw)))
	}
	return parsePingLatencyMs(string(raw))
}

func parsePingLatencyMs(output string) (float64, error) {
	matches := pingLatencyPattern.FindStringSubmatch(output)
	if len(matches) != 2 {
		return 0, fmt.Errorf("unable to parse ping latency")
	}
	if strings.Contains(matches[1], "<") {
		return 0.5, nil
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}
