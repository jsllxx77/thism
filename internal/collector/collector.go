package collector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/thism-dev/thism/internal/models"
)

const DefaultReportInterval = 5 * time.Second
const DefaultAutoUpdateInterval = 30 * time.Minute

const (
	ipv4DefaultRoutePath = "/proc/net/route"
	ipv6DefaultRoutePath = "/proc/net/ipv6_route"
)

type dialMode string

const (
	dialModeAuto dialMode = "auto"
	dialModeIPv4 dialMode = "ipv4"
)

type websocketConn interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (int, []byte, error)
	Close() error
	RemoteAddr() net.Addr
}

type websocketDialFunc func(mode dialMode, targetURL string) (websocketConn, error)

var (
	cpuInfoFunc        = cpu.Info
	cpuCountsFunc      = cpu.Counts
	virtualMemoryFunc  = mem.VirtualMemory
	hostInfoFunc       = host.Info
	diskPartitionsFunc = disk.Partitions
	diskUsageFunc      = disk.Usage
	ioCountersFunc     = psnet.IOCounters
	netInterfacesFunc  = net.Interfaces
	readFileFunc       = os.ReadFile
	httpClient         = &http.Client{Timeout: 2 * time.Minute}
)

// Collector gathers system metrics and pushes them to the ThisM server via WebSocket.
type Collector struct {
	serverURL          string
	token              string
	name               string
	nodeIP             string
	agentVersion       string
	reportInterval     time.Duration
	preferIPv4Fallback bool
	dialWebsocket      websocketDialFunc
	hardwareProfile    *models.NodeHardware
	selfUpdateFunc     func(models.AgentCommandPayload, func(models.UpdateJobTargetStatus, string, string) error) error
	updateMu           sync.Mutex
	updateInProgress   bool
	autoUpdateInterval time.Duration
}

// New creates a new Collector with the default report interval.
func New(serverURL, token, name, nodeIP string) *Collector {
	return NewWithInterval(serverURL, token, name, nodeIP, DefaultReportInterval)
}

// NewWithInterval creates a new Collector with the provided report interval.
func NewWithInterval(serverURL, token, name, nodeIP string, reportInterval time.Duration) *Collector {
	if reportInterval <= 0 {
		reportInterval = DefaultReportInterval
	}

	c := &Collector{
		serverURL:          serverURL,
		token:              token,
		name:               name,
		nodeIP:             strings.TrimSpace(nodeIP),
		agentVersion:       "dev",
		reportInterval:     reportInterval,
		dialWebsocket:      defaultWebsocketDial,
		preferIPv4Fallback: false,
		autoUpdateInterval: DefaultAutoUpdateInterval,
	}
	c.selfUpdateFunc = c.runSelfUpdate
	c.SetAgentVersion(c.agentVersion)
	return c
}

// ReportInterval returns the effective metrics push interval.
func (c *Collector) ReportInterval() time.Duration {
	return c.reportInterval
}

func (c *Collector) SetAgentVersion(version string) {
	trimmed := strings.TrimSpace(version)
	if trimmed != "" && trimmed != "dev" {
		c.agentVersion = trimmed
		return
	}
	if persisted := readPersistedAgentVersion(); persisted != "" {
		c.agentVersion = persisted
		return
	}
	if trimmed != "" {
		c.agentVersion = trimmed
	}
}

type agentReleaseManifest struct {
	TargetVersion        string `json:"target_version"`
	DownloadURL          string `json:"download_url"`
	SHA256               string `json:"sha256"`
	CheckIntervalSeconds int    `json:"check_interval_seconds"`
}

func (c *Collector) currentExecutableSHA256() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(exePath)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return strings.ToLower(hex.EncodeToString(digest[:])), nil
}

func (c *Collector) fetchAgentReleaseManifest() (agentReleaseManifest, error) {
	base, err := url.Parse(c.serverURL)
	if err != nil {
		return agentReleaseManifest{}, err
	}
	switch base.Scheme {
	case "ws":
		base.Scheme = "http"
	case "wss":
		base.Scheme = "https"
	}
	base.Path = "/api/agent-release"
	query := base.Query()
	query.Set("os", runtime.GOOS)
	query.Set("arch", runtime.GOARCH)
	base.RawQuery = query.Encode()
	resp, err := httpClient.Get(base.String())
	if err != nil {
		return agentReleaseManifest{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agentReleaseManifest{}, fmt.Errorf("release manifest returned %d", resp.StatusCode)
	}
	var manifest agentReleaseManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return agentReleaseManifest{}, err
	}
	return manifest, nil
}

func (c *Collector) maybeApplyRelease(manifest agentReleaseManifest, currentChecksum string) error {
	if strings.TrimSpace(manifest.SHA256) == "" || strings.TrimSpace(manifest.DownloadURL) == "" {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(manifest.SHA256), strings.TrimSpace(currentChecksum)) {
		return nil
	}
	cmd := models.AgentCommandPayload{
		JobID:         "auto-update",
		Kind:          models.AgentCommandKindSelfUpdate,
		TargetVersion: manifest.TargetVersion,
		DownloadURL:   manifest.DownloadURL,
		SHA256:        manifest.SHA256,
	}
	return c.selfUpdateFunc(cmd, func(models.UpdateJobTargetStatus, string, string) error { return nil })
}

func (c *Collector) checkForAutoUpdate() error {
	manifest, err := c.fetchAgentReleaseManifest()
	if err != nil {
		return err
	}
	currentChecksum, err := c.currentExecutableSHA256()
	if err != nil {
		return err
	}
	if manifest.CheckIntervalSeconds > 0 {
		c.autoUpdateInterval = time.Duration(manifest.CheckIntervalSeconds) * time.Second
	}
	return c.maybeApplyRelease(manifest, currentChecksum)
}

func isLoopbackInterfaceName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return normalized == "lo" || strings.HasPrefix(normalized, "lo")
}

func parseIPv4DefaultRouteInterfaceNames(raw []byte) map[string]struct{} {
	names := map[string]struct{}{}
	for index, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		if index == 0 && strings.EqualFold(fields[0], "Iface") {
			continue
		}
		interfaceName := strings.TrimSpace(fields[0])
		if interfaceName == "" || isLoopbackInterfaceName(interfaceName) {
			continue
		}
		if fields[1] != "00000000" || fields[7] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(fields[3], 16, 64)
		if err != nil || flags&0x1 == 0 {
			continue
		}
		names[interfaceName] = struct{}{}
	}
	return names
}

func parseIPv6DefaultRouteInterfaceNames(raw []byte) map[string]struct{} {
	names := map[string]struct{}{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		interfaceName := strings.TrimSpace(fields[9])
		if interfaceName == "" || isLoopbackInterfaceName(interfaceName) {
			continue
		}
		if fields[0] != "00000000000000000000000000000000" || fields[1] != "00" {
			continue
		}
		flags, err := strconv.ParseUint(fields[8], 16, 64)
		if err != nil || flags&0x1 == 0 {
			continue
		}
		names[interfaceName] = struct{}{}
	}
	return names
}

func defaultRouteInterfaceNames() map[string]struct{} {
	if runtime.GOOS != "linux" {
		return nil
	}

	names := map[string]struct{}{}
	for _, routeFile := range []struct {
		path  string
		parse func([]byte) map[string]struct{}
	}{
		{path: ipv4DefaultRoutePath, parse: parseIPv4DefaultRouteInterfaceNames},
		{path: ipv6DefaultRoutePath, parse: parseIPv6DefaultRouteInterfaceNames},
	} {
		raw, err := readFileFunc(routeFile.path)
		if err != nil || len(raw) == 0 {
			continue
		}
		for interfaceName := range routeFile.parse(raw) {
			names[interfaceName] = struct{}{}
		}
	}

	return names
}

func nonLoopbackInterfaceNames() map[string]struct{} {
	names := map[string]struct{}{}
	interfaces, err := netInterfacesFunc()
	if err != nil {
		return names
	}
	for _, iface := range interfaces {
		interfaceName := strings.TrimSpace(iface.Name)
		if interfaceName == "" || iface.Flags&net.FlagLoopback != 0 || isLoopbackInterfaceName(interfaceName) {
			continue
		}
		names[interfaceName] = struct{}{}
	}
	return names
}

func collectNetworkStats() models.NetStats {
	selectedInterfaces := defaultRouteInterfaceNames()
	if len(selectedInterfaces) == 0 && runtime.GOOS != "linux" {
		selectedInterfaces = nonLoopbackInterfaceNames()
	}
	if len(selectedInterfaces) == 0 {
		return models.NetStats{}
	}

	ioCounters, err := ioCountersFunc(true)
	if err != nil || len(ioCounters) == 0 {
		return models.NetStats{}
	}

	var rxBytes uint64
	var txBytes uint64
	for _, counter := range ioCounters {
		name := strings.TrimSpace(counter.Name)
		if name == "" || name == "all" {
			continue
		}
		if _, ok := selectedInterfaces[name]; !ok {
			continue
		}
		rxBytes += counter.BytesRecv
		txBytes += counter.BytesSent
	}

	return models.NetStats{RxBytes: rxBytes, TxBytes: txBytes}
}

// Collect gathers a single snapshot of system metrics.
func (c *Collector) Collect() (*models.MetricsPayload, error) {
	ip := c.nodeIP
	if net.ParseIP(ip) == nil {
		ip = detectLocalIP()
	}

	payload := &models.MetricsPayload{
		Type:         "metrics",
		TS:           time.Now().Unix(),
		IP:           ip,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		AgentVersion: c.agentVersion,
		Services:     []models.Service{},
	}
	if payload.Hardware = c.hardware(); payload.Hardware != nil && payload.Hardware.IsEmpty() {
		payload.Hardware = nil
	}
	if hostInfo, err := hostInfoFunc(); err == nil && hostInfo != nil {
		payload.UptimeSeconds = hostInfo.Uptime
	}

	// CPU — blocks for 1 second for an accurate reading.
	cpuPercents, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercents) > 0 {
		payload.CPU = cpuPercents[0]
	}

	// Memory.
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		payload.Mem = models.MemStats{
			Used:  vmStat.Used,
			Total: vmStat.Total,
		}
	}

	// Disk — iterate partitions, skip any that fail Usage().
	partitions, err := disk.Partitions(false)
	if err == nil {
		for _, p := range partitions {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}
			payload.Disk = append(payload.Disk, models.DiskStats{
				Mount: p.Mountpoint,
				Used:  usage.Used,
				Total: usage.Total,
			})
		}
	}

	// Network — aggregate across non-loopback interfaces only.
	payload.Net = collectNetworkStats()

	// Processes — collect up to 30 processes.
	procs, err := process.Processes()
	processSamples := make([]models.Process, 0, 64)
	if err == nil {
		for _, p := range procs {
			name, err := p.Name()
			if err != nil {
				continue
			}
			memInfo, err := p.MemoryInfo()
			if err != nil {
				continue
			}
			var rss uint64
			if memInfo != nil {
				rss = memInfo.RSS
			}
			// Kernel/system helper threads are typically memory-less and add mostly
			// noisy 0.0 CPU rows in the UI; skip them to keep the snapshot useful.
			if rss == 0 {
				continue
			}
			cpuPct, err := p.CPUPercent()
			if err != nil {
				continue
			}
			processSamples = append(processSamples, models.Process{
				PID:        p.Pid,
				Name:       name,
				CPUPercent: cpuPct,
				MemRSS:     rss,
			})
		}
	}
	payload.Processes = selectTopProcesses(processSamples, 30)
	if payload.Processes == nil {
		payload.Processes = []models.Process{}
	}

	// Docker containers — graceful degradation when Docker is unavailable.
	containers, dockerAvailable, _ := collectDockerContainers()
	payload.DockerAvailable = &dockerAvailable
	if dockerAvailable {
		payload.Containers = containers
	}

	return payload, nil
}

func (c *Collector) hardware() *models.NodeHardware {
	if c.hardwareProfile == nil {
		c.hardwareProfile = collectHardwareProfile()
	}
	return c.hardwareProfile
}

func collectHardwareProfile() *models.NodeHardware {
	profile := &models.NodeHardware{}

	if info, err := cpuInfoFunc(); err == nil {
		for _, cpuInfo := range info {
			if strings.TrimSpace(cpuInfo.ModelName) != "" {
				profile.CPUModel = strings.TrimSpace(cpuInfo.ModelName)
				break
			}
		}
	}

	if physicalCores, err := cpuCountsFunc(false); err == nil && physicalCores > 0 {
		profile.CPUCores = physicalCores
	}
	if logicalThreads, err := cpuCountsFunc(true); err == nil && logicalThreads > 0 {
		profile.CPUThreads = logicalThreads
	}

	if vmStat, err := virtualMemoryFunc(); err == nil && vmStat != nil {
		profile.MemoryTotal = vmStat.Total
	}

	if hostInfo, err := hostInfoFunc(); err == nil && hostInfo != nil {
		profile.VirtualizationSystem = strings.TrimSpace(hostInfo.VirtualizationSystem)
		profile.VirtualizationRole = strings.TrimSpace(hostInfo.VirtualizationRole)
	}

	if partitions, err := diskPartitionsFunc(false); err == nil {
		for _, partition := range partitions {
			usage, err := diskUsageFunc(partition.Mountpoint)
			if err != nil || usage == nil {
				continue
			}
			profile.DiskTotal += usage.Total
		}
	}

	if profile.IsEmpty() {
		return nil
	}

	return profile
}

// Run starts a blocking loop that connects to the server with exponential backoff.
func (c *Collector) Run() {
	backoff := time.Second
	for {
		err := c.connect()
		if err != nil {
			log.Printf("collector: disconnected: %v — retrying in %s", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
		} else {
			// Successful clean exit — reset backoff.
			backoff = time.Second
		}
	}
}

// connect dials the server, then sends metrics on the configured interval until an error occurs.
func (c *Collector) connect() error {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return err
	}
	u.Path = "/ws/agent"
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()
	targetURL := u.String()

	conn, mode, err := c.dialAgent(targetURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Reset backoff on successful connection.
	if mode == dialModeIPv4 {
		log.Printf("collector: connected with IPv4 fallback to %s", targetURL)
	} else {
		log.Printf("collector: connected to %s", targetURL)
	}

	ticker := time.NewTicker(c.reportInterval)
	defer ticker.Stop()
	autoUpdateTicker := time.NewTicker(c.autoUpdateInterval)
	defer autoUpdateTicker.Stop()
	autoUpdateNow := make(chan struct{}, 1)
	autoUpdateNow <- struct{}{}

	var writeMu sync.Mutex
	readErrCh := make(chan error, 1)
	go func() {
		readErrCh <- c.readAgentCommands(conn, &writeMu)
	}()

	for {
		select {
		case err := <-readErrCh:
			if err != nil && !errors.Is(err, net.ErrClosed) {
				return err
			}
			return nil
		case <-autoUpdateNow:
			if err := c.checkForAutoUpdate(); err != nil {
				log.Printf("collector: auto update check failed: %v", err)
			}
		case <-autoUpdateTicker.C:
			if err := c.checkForAutoUpdate(); err != nil {
				log.Printf("collector: auto update check failed: %v", err)
			}
		case <-ticker.C:
			metrics, err := c.Collect()
			if err != nil {
				return err
			}
			data, err := json.Marshal(metrics)
			if err != nil {
				return err
			}
			writeMu.Lock()
			err = conn.WriteMessage(websocket.TextMessage, data)
			writeMu.Unlock()
			if err != nil {
				c.noteConnectionError(mode, conn.RemoteAddr(), err)
				return err
			}
		}
	}
}

func (c *Collector) readAgentCommands(conn websocketConn, writeMu *sync.Mutex) error {
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var envelope struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			continue
		}
		if envelope.Type != "agent_command" {
			continue
		}
		var payload models.AgentCommandPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			continue
		}
		c.dispatchAgentCommand(payload, conn, writeMu)
	}
}

func (c *Collector) dispatchAgentCommand(cmd models.AgentCommandPayload, conn websocketConn, writeMu *sync.Mutex) {
	if cmd.Kind != models.AgentCommandKindSelfUpdate {
		_ = c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{JobID: cmd.JobID, Status: models.UpdateJobTargetStatusFailed, Message: "unsupported agent command"})
		return
	}
	c.updateMu.Lock()
	if c.updateInProgress {
		c.updateMu.Unlock()
		_ = c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{JobID: cmd.JobID, Status: models.UpdateJobTargetStatusFailed, Message: "another self update is already in progress"})
		return
	}
	c.updateInProgress = true
	c.updateMu.Unlock()

	go func() {
		defer func() {
			c.updateMu.Lock()
			c.updateInProgress = false
			c.updateMu.Unlock()
		}()
		report := func(status models.UpdateJobTargetStatus, message, version string) error {
			return c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{JobID: cmd.JobID, Status: status, Message: message, ReportedVersion: version})
		}
		if err := report(models.UpdateJobTargetStatusAccepted, "accepted", c.agentVersion); err != nil {
			return
		}
		if err := c.selfUpdateFunc(cmd, report); err != nil {
			_ = report(models.UpdateJobTargetStatusFailed, err.Error(), c.agentVersion)
		}
	}()
}

func (c *Collector) sendAgentCommandStatus(conn websocketConn, writeMu *sync.Mutex, payload models.AgentCommandStatusPayload) error {
	raw, err := json.Marshal(models.WSMessage{Type: "agent_command_status", Payload: payload})
	if err != nil {
		return err
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, raw)
}

func persistedAgentVersionPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	resolvedPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = resolvedPath
	}
	return filepath.Join(filepath.Dir(exePath), ".thism-agent.version")
}

func readPersistedAgentVersion() string {
	versionPath := persistedAgentVersionPath()
	if versionPath == "" {
		return ""
	}
	raw, err := os.ReadFile(versionPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func (c *Collector) runSelfUpdate(cmd models.AgentCommandPayload, report func(models.UpdateJobTargetStatus, string, string) error) error {
	if err := validateSelfUpdateSource(c.serverURL, cmd.DownloadURL); err != nil {
		return err
	}
	if err := report(models.UpdateJobTargetStatusDownloading, "downloading replacement binary", c.agentVersion); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, cmd.DownloadURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	binaryData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := report(models.UpdateJobTargetStatusVerifying, "verifying binary checksum", c.agentVersion); err != nil {
		return err
	}
	digest := sha256.Sum256(binaryData)
	actualChecksum := strings.ToLower(hex.EncodeToString(digest[:]))
	if strings.ToLower(strings.TrimSpace(cmd.SHA256)) != actualChecksum {
		return fmt.Errorf("sha256 mismatch")
	}
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return err
	}
	fileInfo, err := os.Stat(exePath)
	if err != nil {
		return err
	}
	tmpPath := filepath.Join(filepath.Dir(exePath), ".thism-agent-update.tmp")
	if err := os.WriteFile(tmpPath, binaryData, fileInfo.Mode()); err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	if err := os.Rename(tmpPath, exePath); err != nil {
		return err
	}
	versionPath := persistedAgentVersionPath()
	if versionPath != "" {
		if err := os.WriteFile(versionPath, []byte(strings.TrimSpace(cmd.TargetVersion)+"\n"), 0644); err != nil {
			return err
		}
	}
	if err := report(models.UpdateJobTargetStatusRestarting, "restarting agent", cmd.TargetVersion); err != nil {
		return err
	}
	return syscall.Exec(exePath, os.Args, os.Environ())
}

func validateSelfUpdateSource(serverURL, downloadURL string) error {
	target, err := url.Parse(downloadURL)
	if err != nil {
		return err
	}
	if target.Scheme == "https" {
		return nil
	}
	if target.Scheme != "http" {
		return fmt.Errorf("unsupported update url scheme")
	}
	base, err := url.Parse(serverURL)
	if err != nil {
		return err
	}
	if !strings.EqualFold(target.Host, base.Host) {
		return fmt.Errorf("http update source must match server host")
	}
	return nil
}

func (c *Collector) dialModes() []dialMode {
	if c.preferIPv4Fallback {
		return []dialMode{dialModeIPv4, dialModeAuto}
	}
	return []dialMode{dialModeAuto}
}

func (c *Collector) dialAgent(targetURL string) (websocketConn, dialMode, error) {
	var lastErr error

	for index, mode := range c.dialModes() {
		conn, err := c.dialWebsocket(mode, targetURL)
		if err == nil {
			return conn, mode, nil
		}
		lastErr = err

		if mode == dialModeIPv4 && index < len(c.dialModes())-1 {
			log.Printf("collector: IPv4 fallback dial failed: %v — falling back to automatic network selection", err)
		}
	}

	if lastErr == nil {
		lastErr = errors.New("collector: no websocket dial modes available")
	}

	return nil, "", lastErr
}

func (c *Collector) noteConnectionError(mode dialMode, remoteAddr net.Addr, err error) {
	if c.preferIPv4Fallback || mode != dialModeAuto || !shouldEnableIPv4Fallback(remoteAddr, err) {
		return
	}

	c.preferIPv4Fallback = true
	if remoteAddr != nil {
		log.Printf("collector: enabling IPv4 fallback after IPv6 connection error to %s: %v", remoteAddr.String(), err)
		return
	}
	log.Printf("collector: enabling IPv4 fallback after IPv6 connection error: %v", err)
}

func shouldEnableIPv4Fallback(remoteAddr net.Addr, err error) bool {
	return isIPv6Addr(remoteAddr) && isResetLikeError(err)
}

func isIPv6Addr(addr net.Addr) bool {
	ip := ipFromAddr(addr)
	return ip != nil && ip.To4() == nil && ip.To16() != nil
}

func ipFromAddr(addr net.Addr) net.IP {
	if addr == nil {
		return nil
	}

	switch value := addr.(type) {
	case *net.TCPAddr:
		return value.IP
	case *net.UDPAddr:
		return value.IP
	case *net.IPAddr:
		return value.IP
	default:
		host, _, err := net.SplitHostPort(addr.String())
		if err == nil {
			return net.ParseIP(host)
		}
		return net.ParseIP(addr.String())
	}
}

func isResetLikeError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "connection reset by peer") || strings.Contains(message, "broken pipe")
}

func defaultWebsocketDial(mode dialMode, targetURL string) (websocketConn, error) {
	dialer := *websocket.DefaultDialer
	if mode == dialModeIPv4 {
		netDialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return netDialer.DialContext(ctx, "tcp4", addr)
		}
	}

	conn, _, err := dialer.Dial(targetURL, nil)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func selectTopProcesses(processes []models.Process, limit int) []models.Process {
	if limit <= 0 || len(processes) == 0 {
		return []models.Process{}
	}

	sorted := make([]models.Process, 0, len(processes))
	for _, process := range processes {
		if process.MemRSS == 0 {
			continue
		}
		sorted = append(sorted, process)
	}
	if len(sorted) == 0 {
		return []models.Process{}
	}
	sort.Slice(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		if left.CPUPercent != right.CPUPercent {
			return left.CPUPercent > right.CPUPercent
		}
		if left.MemRSS != right.MemRSS {
			return left.MemRSS > right.MemRSS
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.PID < right.PID
	})

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func detectLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP == nil {
			continue
		}
		if ipNet.IP.IsLoopback() {
			continue
		}
		if ipv4 := ipNet.IP.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
			continue
		}
		return ipNet.IP.String()
	}

	return ""
}
