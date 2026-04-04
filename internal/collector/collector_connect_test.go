package collector

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/thism-dev/thism/internal/models"
)

type connectTestRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f connectTestRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type blockingConnectTestConn struct {
	closeCh     chan struct{}
	metricsSeen chan struct{}
	remoteAddr  net.Addr
}

func (c *blockingConnectTestConn) WriteMessage(_ int, data []byte) error {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if msgType, _ := payload["type"].(string); msgType == "metrics" {
		select {
		case c.metricsSeen <- struct{}{}:
		default:
		}
		return errors.New("stop after metrics")
	}
	return nil
}

func (c *blockingConnectTestConn) ReadMessage() (int, []byte, error) {
	<-c.closeCh
	return websocket.TextMessage, nil, io.EOF
}

func (c *blockingConnectTestConn) Close() error {
	select {
	case <-c.closeCh:
	default:
		close(c.closeCh)
	}
	return nil
}

func (c *blockingConnectTestConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func stubFastCollectorDependencies(t *testing.T, transport http.RoundTripper) {
	t.Helper()

	originalHTTPClient := httpClient
	originalCPUPercentFunc := cpuPercentFunc
	originalVirtualMemoryFunc := virtualMemoryFunc
	originalHostInfoFunc := hostInfoFunc
	originalDiskPartitionsFunc := diskPartitionsFunc
	originalIOCountersFunc := ioCountersFunc
	originalReadFileFunc := readFileFunc
	originalCollectProcessSamplesFunc := collectProcessSamplesFunc
	originalCollectDockerContainersFunc := collectDockerContainersFunc
	t.Cleanup(func() {
		httpClient = originalHTTPClient
		cpuPercentFunc = originalCPUPercentFunc
		virtualMemoryFunc = originalVirtualMemoryFunc
		hostInfoFunc = originalHostInfoFunc
		diskPartitionsFunc = originalDiskPartitionsFunc
		ioCountersFunc = originalIOCountersFunc
		readFileFunc = originalReadFileFunc
		collectProcessSamplesFunc = originalCollectProcessSamplesFunc
		collectDockerContainersFunc = originalCollectDockerContainersFunc
	})

	httpClient = &http.Client{
		Transport: transport,
	}
	cpuPercentFunc = func(time.Duration, bool) ([]float64, error) {
		return []float64{25}, nil
	}
	virtualMemoryFunc = func() (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Used: 1024, Total: 4096}, nil
	}
	hostInfoFunc = func() (*host.InfoStat, error) {
		return &host.InfoStat{Uptime: 3600}, nil
	}
	diskPartitionsFunc = func(bool) ([]disk.PartitionStat, error) {
		return nil, nil
	}
	ioCountersFunc = func(bool) ([]psnet.IOCountersStat, error) {
		return nil, nil
	}
	readFileFunc = func(string) ([]byte, error) {
		return nil, errors.New("no route file")
	}
	collectProcessSamplesFunc = func() ([]models.Process, error) {
		return nil, nil
	}
	collectDockerContainersFunc = func() ([]models.DockerContainer, bool, error) {
		return nil, false, nil
	}
}

func TestCollectorConnectSendsMetricsWhileLatencyProbeIsInFlight(t *testing.T) {
	stubFastCollectorDependencies(t, connectTestRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := io.NopCloser(bytes.NewBufferString(`{"target_version":"test","download_url":"","sha256":"","check_interval_seconds":1800}`))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
			Header:     make(http.Header),
		}, nil
	}))

	conn := &blockingConnectTestConn{
		closeCh:     make(chan struct{}),
		metricsSeen: make(chan struct{}, 1),
		remoteAddr:  &net.TCPAddr{IP: net.ParseIP("104.21.89.105"), Port: 443},
	}

	c := NewWithInterval("wss://example.com", "token", "node", "", 1200*time.Millisecond)
	c.dialWebsocket = func(mode dialMode, targetURL string, headers http.Header) (websocketConn, error) {
		return conn, nil
	}
	c.hardwareProfile = &models.NodeHardware{CPUModel: "Test CPU"}
	c.applyLatencyMonitorConfig([]models.LatencyMonitor{
		{ID: "monitor-1", Name: "TCP 80", Type: models.LatencyMonitorTypeTCP, Target: "example.com:80", IntervalSeconds: 1},
	})

	probeStarted := make(chan struct{})
	probeRelease := make(chan struct{})
	firstProbe := true
	c.tcpLatencyProbe = func(_ string) (float64, error) {
		if firstProbe {
			firstProbe = false
			close(probeStarted)
			<-probeRelease
		}
		return 12.5, nil
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.connect()
	}()

	select {
	case <-probeStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("expected latency probe to start")
	}

	select {
	case <-conn.metricsSeen:
	case <-time.After(750 * time.Millisecond):
		close(probeRelease)
		t.Fatal("expected metrics to be sent while latency probe was still in flight")
	}

	close(probeRelease)

	select {
	case err := <-errCh:
		if err == nil || err.Error() != "stop after metrics" {
			t.Fatalf("expected stop-after-metrics error, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected connect to stop after metrics write")
	}
}

func TestCollectorConnectSendsMetricsWhileAutoUpdateCheckIsInFlight(t *testing.T) {
	updateStarted := make(chan struct{})
	updateRelease := make(chan struct{})
	stubFastCollectorDependencies(t, connectTestRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		close(updateStarted)
		<-updateRelease
		body := io.NopCloser(bytes.NewBufferString(`{"target_version":"test","download_url":"","sha256":"","check_interval_seconds":1800}`))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
			Header:     make(http.Header),
		}, nil
	}))

	conn := &blockingConnectTestConn{
		closeCh:     make(chan struct{}),
		metricsSeen: make(chan struct{}, 1),
		remoteAddr:  &net.TCPAddr{IP: net.ParseIP("104.21.89.105"), Port: 443},
	}

	c := NewWithInterval("wss://example.com", "token", "node", "", 200*time.Millisecond)
	c.dialWebsocket = func(mode dialMode, targetURL string, headers http.Header) (websocketConn, error) {
		return conn, nil
	}
	c.hardwareProfile = &models.NodeHardware{CPUModel: "Test CPU"}

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.connect()
	}()

	select {
	case <-updateStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("expected auto update check to start")
	}

	select {
	case <-conn.metricsSeen:
	case <-time.After(750 * time.Millisecond):
		close(updateRelease)
		t.Fatal("expected metrics to be sent while auto update check was still in flight")
	}

	close(updateRelease)

	select {
	case err := <-errCh:
		if err == nil || err.Error() != "stop after metrics" {
			t.Fatalf("expected stop-after-metrics error, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected connect to stop after metrics write")
	}
}
