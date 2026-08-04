package collector

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
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
	"unsafe"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/security/release"
)

const DefaultReportInterval = 5 * time.Second
const DefaultAutoUpdateInterval = 30 * time.Minute
const DefaultHeavySnapshotInterval = time.Minute
const networkTopologyCacheTTL = 30 * time.Second
const processSnapshotLimit = 10

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

type websocketDialFunc func(mode dialMode, targetURL string, headers http.Header) (websocketConn, error)

var (
	// ServerTLSSPKISHA256Base64 pins the agent to a server certificate public key
	// when injected at build time with -ldflags.
	ServerTLSSPKISHA256Base64 string

	cpuInfoFunc                 = cpu.Info
	cpuPercentFunc              = cpu.Percent
	cpuTimesFunc                = cpu.Times
	cpuCountsFunc               = cpu.Counts
	loadAvgFunc                 = load.Avg
	virtualMemoryFunc           = mem.VirtualMemory
	swapMemoryFunc              = mem.SwapMemory
	hostInfoFunc                = host.Info
	diskPartitionsFunc          = disk.Partitions
	diskUsageFunc               = disk.Usage
	diskIOCountersFunc          = disk.IOCounters
	ioCountersFunc              = psnet.IOCounters
	netInterfacesFunc           = net.Interfaces
	interfaceAddrsFunc          = net.InterfaceAddrs
	readFileFunc                = os.ReadFile
	nowFunc                     = time.Now
	collectProcessSamplesFunc   = collectProcessSamples
	collectDockerContainersFunc = collectDockerContainers
	readNVMeHealthLogFunc       = readNVMeHealthLog
	readATAHealthFunc           = readATAHealth
	httpClient                  = newSelfUpdateHTTPClient()
)

var (
	diskHealthSysBlockPath   = "/sys/block"
	diskHealthDevRootPath    = "/dev"
	diskHealthProcMountsPath = "/proc/mounts"
)

func newSelfUpdateHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	tlsConfig, err := tlsConfigWithServerPin(transport.TLSClientConfig)
	var roundTripper http.RoundTripper = transport
	if err != nil {
		roundTripper = pinningRoundTripper{base: transport, tlsPinErr: err}
	} else if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}
	return &http.Client{
		Timeout:   2 * time.Minute,
		Transport: roundTripper,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

type pinningRoundTripper struct {
	base      http.RoundTripper
	tlsPinErr error
}

func (rt pinningRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req != nil && req.URL != nil && strings.EqualFold(req.URL.Scheme, "https") && rt.tlsPinErr != nil {
		return nil, rt.tlsPinErr
	}
	return rt.base.RoundTrip(req)
}

func tlsConfigWithServerPin(base *tls.Config) (*tls.Config, error) {
	pin, err := parseServerTLSSPKIPin()
	if err != nil {
		return nil, err
	}
	if pin == nil {
		if base == nil {
			return nil, nil
		}
		return base.Clone(), nil
	}

	var config *tls.Config
	if base != nil {
		config = base.Clone()
	} else {
		config = &tls.Config{}
	}
	existingVerifyConnection := config.VerifyConnection
	config.VerifyConnection = func(state tls.ConnectionState) error {
		if existingVerifyConnection != nil {
			if err := existingVerifyConnection(state); err != nil {
				return err
			}
		}
		if matchesPinnedSPKI(state.PeerCertificates, pin) {
			return nil
		}
		return errors.New("tls spki pin mismatch")
	}
	return config, nil
}

func parseServerTLSSPKIPin() ([]byte, error) {
	raw := strings.TrimSpace(ServerTLSSPKISHA256Base64)
	if raw == "" {
		return nil, nil
	}
	pin, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid tls spki pin: %w", err)
	}
	if len(pin) != sha256.Size {
		return nil, fmt.Errorf("invalid tls spki pin: expected %d-byte sha256 digest", sha256.Size)
	}
	return pin, nil
}

func matchesPinnedSPKI(certs []*x509.Certificate, pin []byte) bool {
	if len(certs) == 0 {
		return false
	}
	digest := sha256.Sum256(certs[0].RawSubjectPublicKeyInfo)
	return subtle.ConstantTimeCompare(digest[:], pin) == 1
}

// Collector gathers system metrics and pushes them to the ThisM server via WebSocket.
type Collector struct {
	serverURL             string
	token                 string
	name                  string
	nodeIP                string
	agentVersion          string
	reportInterval        time.Duration
	preferIPv4Fallback    bool
	dialWebsocket         websocketDialFunc
	hardwareProfile       *models.NodeHardware
	heavySnapshotInterval time.Duration
	lastHeavySnapshotAt   time.Time
	heavySnapshotSent     bool
	now                   func() time.Time
	selfUpdateFunc        func(models.AgentCommandPayload, func(models.UpdateJobTargetStatus, string, string) error) error
	statePath             string
	state                 *agentState
	stateMu               sync.Mutex
	taskCancel            context.CancelFunc
	taskCtx               context.Context
	taskMu                sync.Mutex
	taskWG                sync.WaitGroup
	autoUpdateInterval    time.Duration
	cpuSampleMu           sync.Mutex
	cpuSampleReady        bool
	lastCPUTotal          float64
	lastCPUIdle           float64
	lastCPUIOWait         float64
	lastCPUSteal          float64
	networkCacheMu        sync.Mutex
	cachedLocalIP         string
	cachedIPFamilies      []string
	cachedLocalIPAt       time.Time
	cachedInterfaces      map[string]struct{}
	cachedInterfacesAt    time.Time
	latencyMu             sync.Mutex
	latencyMonitors       map[string]*latencyMonitorState
	icmpLatencyProbe      latencyProbeFunc
	tcpLatencyProbe       latencyProbeFunc
	httpLatencyProbe      latencyProbeFunc
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
		serverURL:             serverURL,
		token:                 token,
		name:                  name,
		nodeIP:                strings.TrimSpace(nodeIP),
		agentVersion:          "dev",
		reportInterval:        reportInterval,
		dialWebsocket:         defaultWebsocketDial,
		preferIPv4Fallback:    false,
		heavySnapshotInterval: DefaultHeavySnapshotInterval,
		now:                   nowFunc,
		autoUpdateInterval:    DefaultAutoUpdateInterval,
		latencyMonitors:       make(map[string]*latencyMonitorState),
		icmpLatencyProbe:      defaultICMPLatencyProbe,
		tcpLatencyProbe:       defaultTCPLatencyProbe,
		httpLatencyProbe:      defaultHTTPLatencyProbe,
	}
	c.selfUpdateFunc = c.runSelfUpdate
	c.SetAgentVersion(c.agentVersion)
	c.primeCPUSample()
	c.loadAgentState()
	return c
}

// ReportInterval returns the effective metrics push interval.
func (c *Collector) ReportInterval() time.Duration {
	return c.reportInterval
}

func (c *Collector) primeCPUSample() {
	sample, ok := readCPUSample()
	if !ok {
		return
	}

	c.cpuSampleMu.Lock()
	c.lastCPUTotal = sample.total
	c.lastCPUIdle = sample.idle
	c.lastCPUIOWait = sample.iowait
	c.lastCPUSteal = sample.steal
	c.cpuSampleReady = true
	c.cpuSampleMu.Unlock()
}

type cpuSample struct {
	total  float64
	idle   float64
	iowait float64
	steal  float64
}

func readCPUSample() (cpuSample, bool) {
	times, err := cpuTimesFunc(false)
	if err != nil || len(times) == 0 {
		return cpuSample{}, false
	}
	return cpuSampleFromTimes(times[0]), true
}

func cpuSampleFromTimes(times cpu.TimesStat) cpuSample {
	idle := times.Idle + times.Iowait
	total := times.User + times.System + times.Idle + times.Nice + times.Iowait + times.Irq + times.Softirq + times.Steal + times.Guest + times.GuestNice
	return cpuSample{total: total, idle: idle, iowait: times.Iowait, steal: times.Steal}
}

func clampCPUPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

type cpuDeltaStats struct {
	UsagePercent  float64
	IOWaitPercent float64
	StealPercent  float64
}

func (c *Collector) sampleCPUStats() (cpuDeltaStats, bool) {
	sample, ok := readCPUSample()
	if !ok {
		return cpuDeltaStats{}, false
	}

	c.cpuSampleMu.Lock()
	defer c.cpuSampleMu.Unlock()

	if !c.cpuSampleReady {
		c.lastCPUTotal = sample.total
		c.lastCPUIdle = sample.idle
		c.lastCPUIOWait = sample.iowait
		c.lastCPUSteal = sample.steal
		c.cpuSampleReady = true
		return cpuDeltaStats{}, false
	}

	deltaTotal := sample.total - c.lastCPUTotal
	deltaIdle := sample.idle - c.lastCPUIdle
	deltaIOWait := sample.iowait - c.lastCPUIOWait
	deltaSteal := sample.steal - c.lastCPUSteal
	c.lastCPUTotal = sample.total
	c.lastCPUIdle = sample.idle
	c.lastCPUIOWait = sample.iowait
	c.lastCPUSteal = sample.steal
	if deltaTotal <= 0 || deltaIdle < 0 {
		return cpuDeltaStats{}, false
	}

	usage := ((deltaTotal - deltaIdle) / deltaTotal) * 100
	stats := cpuDeltaStats{UsagePercent: clampCPUPercent(usage)}
	if deltaIOWait > 0 {
		stats.IOWaitPercent = clampCPUPercent((deltaIOWait / deltaTotal) * 100)
	}
	if deltaSteal > 0 {
		stats.StealPercent = clampCPUPercent((deltaSteal / deltaTotal) * 100)
	}
	return stats, true
}

func (c *Collector) sampleCPUPercent() (float64, bool) {
	stats, ok := c.sampleCPUStats()
	return stats.UsagePercent, ok
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
	Signature            string `json:"signature,omitempty"`
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
		Signature:     manifest.Signature,
	}
	// Auto-update goes through the same durable executor and the same global
	// serialization as remote commands: they are mutually exclusive.
	ok, replay, rejectCode, rejectMsg := c.beginUpdateTask(cmd)
	if !ok {
		if replay != nil {
			return fmt.Errorf("auto update already processed: %s", replay.Status)
		}
		if rejectCode != "" {
			return fmt.Errorf("auto update rejected: %s %s", rejectCode, rejectMsg)
		}
		return fmt.Errorf("auto update rejected: %s", rejectMsg)
	}
	return c.runUpdateTask(cmd, func(models.UpdateJobTargetStatus, string, string) error { return nil })
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

func isTunnelInterfaceName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for _, prefix := range []string{"tun", "tap", "wg", "tailscale", "zt", "ppp", "ipsec", "utun"} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
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
		if interfaceName == "" || iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || isLoopbackInterfaceName(interfaceName) {
			continue
		}
		names[interfaceName] = struct{}{}
	}
	return names
}

func supplementalLinuxInterfaceNames() map[string]struct{} {
	names := map[string]struct{}{}
	if runtime.GOOS != "linux" {
		return names
	}

	interfaces, err := netInterfacesFunc()
	if err != nil {
		return names
	}

	for _, iface := range interfaces {
		interfaceName := strings.TrimSpace(iface.Name)
		if interfaceName == "" || iface.Flags&net.FlagLoopback != 0 || isLoopbackInterfaceName(interfaceName) {
			continue
		}
		if iface.Flags&net.FlagPointToPoint != 0 || isTunnelInterfaceName(interfaceName) {
			names[interfaceName] = struct{}{}
		}
	}

	return names
}

func cloneInterfaceNames(names map[string]struct{}) map[string]struct{} {
	if len(names) == 0 {
		return nil
	}
	clone := make(map[string]struct{}, len(names))
	for name := range names {
		clone[name] = struct{}{}
	}
	return clone
}

func collectNetworkStatsForInterfaces(selectedInterfaces map[string]struct{}) models.NetStats {
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

func collectNetworkStats() models.NetStats {
	return collectNetworkStatsForInterfaces(nonLoopbackInterfaceNames())
}

func collectDiskIOStats() models.DiskIOStats {
	counters, err := diskIOCountersFunc()
	if err != nil || len(counters) == 0 {
		return models.DiskIOStats{}
	}

	var readBytes uint64
	var writeBytes uint64
	for _, counter := range counters {
		readBytes += counter.ReadBytes
		writeBytes += counter.WriteBytes
	}

	return models.DiskIOStats{ReadBytes: readBytes, WriteBytes: writeBytes}
}

type nvmePassthruCmd struct {
	Opcode      uint8
	Flags       uint8
	Rsvd1       uint16
	Nsid        uint32
	Cdw2        uint32
	Cdw3        uint32
	Metadata    uint64
	Addr        uint64
	MetadataLen uint32
	DataLen     uint32
	Cdw10       uint32
	Cdw11       uint32
	Cdw12       uint32
	Cdw13       uint32
	Cdw14       uint32
	Cdw15       uint32
	TimeoutMS   uint32
	Result      uint32
}

type nvmeHealthLog struct {
	CriticalWarning uint8
	TemperatureC    *float64
	LifeUsedPercent *float64
	AvailableSpare  *float64
	PowerOnHours    uint64
	UnsafeShutdowns uint64
	MediaErrors     uint64
}

type ataHealthLog struct {
	TemperatureC          *float64
	LifeUsedPercent       *float64
	PowerOnHours          uint64
	ReallocatedSectors    uint64
	PendingSectors        uint64
	OfflineUncorrectable  uint64
	InterfaceCRCErrors    uint64
	ReportedUncorrectable uint64
}

const (
	nvmeIoctlAdminCmd = uintptr(0xC0484E41)
	nvmeAdminGetLog   = uint8(0x02)
	nvmeSmartLogID    = uint32(0x02)
	nvmeGlobalNSID    = uint32(0xffffffff)
)

const (
	sgIoctl              = uintptr(0x2285)
	sgDxferFromDev       = int32(-3)
	sgInterfaceID        = int32('S')
	ataPassThrough16     = byte(0x85)
	ataProtocolPIODataIn = byte(4)
	ataSmartCommand      = byte(0xB0)
	ataSmartReadData     = byte(0xD0)
	ataSmartLBAHigh      = byte(0xC2)
	ataSmartLBAMid       = byte(0x4F)
)

var errATASmartUnsupported = errors.New("ATA SMART passthrough is not supported by this block device")

type sgIOHdr struct {
	InterfaceID    int32
	DxferDirection int32
	CmdLen         uint8
	MxSbLen        uint8
	IovecCount     uint16
	DxferLen       uint32
	Dxferp         uintptr
	Cmdp           uintptr
	Sbp            uintptr
	Timeout        uint32
	Flags          uint32
	PackID         int32
	UsrPtr         uintptr
	Status         uint8
	MaskedStatus   uint8
	MsgStatus      uint8
	SbLenWr        uint8
	HostStatus     uint16
	DriverStatus   uint16
	Resid          int32
	Duration       uint32
	Info           uint32
}

func collectDiskHealth() []models.DiskHealthStats {
	if runtime.GOOS != "linux" {
		return nil
	}

	entries, err := os.ReadDir(diskHealthSysBlockPath)
	if err != nil {
		return nil
	}

	disks := make([]models.DiskHealthStats, 0, len(entries))
	for _, entry := range entries {
		if !isDiskHealthBlockEntry(entry) {
			continue
		}
		name := entry.Name()
		if shouldSkipDiskHealthDevice(name) {
			continue
		}

		stats := collectDiskHealthForDevice(name)
		if stats.Name == "" {
			continue
		}
		disks = append(disks, stats)
	}
	sort.Slice(disks, func(i, j int) bool {
		return disks[i].Name < disks[j].Name
	})
	return disks
}

func collectDiskHealthForDevice(name string) models.DiskHealthStats {
	deviceRoot := filepath.Join(diskHealthSysBlockPath, name)
	stats := models.DiskHealthStats{
		Name:          name,
		Path:          filepath.Join(diskHealthDevRootPath, name),
		Type:          classifyDiskHealthType(name, deviceRoot),
		Status:        models.DiskHealthStatusUnknown,
		SupportStatus: models.DiskHealthSupportUnknown,
		Model:         readSysfsString(filepath.Join(deviceRoot, "device/model")),
		Serial:        readSysfsString(filepath.Join(deviceRoot, "device/serial")),
	}
	stats.ReadOnly = readBlockDeviceReadOnly(filepath.Join(deviceRoot, "ro"))
	stats.Mounts = readDiskHealthMounts(name)
	for _, mount := range stats.Mounts {
		if mount.ReadOnly {
			stats.ReadOnly = true
			break
		}
	}
	if firmware := readSysfsString(filepath.Join(deviceRoot, "device/firmware_rev")); firmware != "" {
		stats.Firmware = firmware
	} else if firmware := readSysfsString(filepath.Join(deviceRoot, "device/rev")); firmware != "" {
		stats.Firmware = firmware
	}
	stats.SizeBytes = readBlockDeviceSizeBytes(filepath.Join(deviceRoot, "size"))

	switch stats.Type {
	case models.DiskHealthTypeNVMe:
		controllerPath, ok := nvmeControllerPathForBlockDevice(name)
		if !ok {
			stats.Status = models.DiskHealthStatusUnknown
			stats.SupportStatus = models.DiskHealthSupportDegraded
			stats.Message = "NVMe controller path not found"
			return stats
		}
		log, err := readNVMeHealthLogFunc(controllerPath)
		if err != nil {
			stats.Status = models.DiskHealthStatusUnknown
			stats.SupportStatus = models.DiskHealthSupportDegraded
			stats.Message = "NVMe health log unavailable: " + err.Error()
			return stats
		}
		return diskHealthFromNVMeLog(stats, log)
	case models.DiskHealthTypeATA:
		log, err := readATAHealthFunc(stats.Path)
		if err != nil {
			stats.Status = models.DiskHealthStatusUnknown
			stats.SupportStatus = models.DiskHealthSupportDegraded
			stats.Message = "ATA SMART data unavailable: " + err.Error()
			if errors.Is(err, errATASmartUnsupported) {
				stats.Message = "ATA SMART passthrough unavailable: " + err.Error()
			}
			return stats
		}
		return diskHealthFromATAHealthLog(stats, log)
	case models.DiskHealthTypeVirtual:
		stats.Status = models.DiskHealthStatusUnsupported
		stats.SupportStatus = models.DiskHealthSupportUnsupported
		stats.Message = "cloud or virtual block device does not expose physical disk health"
		return stats
	default:
		stats.Status = models.DiskHealthStatusUnsupported
		stats.SupportStatus = models.DiskHealthSupportUnsupported
		stats.Message = "block device health is not supported"
		return stats
	}
}

func isDiskHealthBlockEntry(entry os.DirEntry) bool {
	if entry == nil {
		return false
	}
	if entry.IsDir() {
		return true
	}
	info, err := os.Stat(filepath.Join(diskHealthSysBlockPath, entry.Name()))
	return err == nil && info.IsDir()
}

func shouldSkipDiskHealthDevice(name string) bool {
	for _, prefix := range []string{"loop", "ram", "zram", "fd", "sr"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func classifyDiskHealthType(name, deviceRoot string) string {
	resolvedRoot, err := filepath.EvalSymlinks(deviceRoot)
	if err == nil && (strings.Contains(resolvedRoot, "/devices/virtual/block/") || strings.Contains(resolvedRoot, "/virtio")) {
		return models.DiskHealthTypeVirtual
	}
	if strings.HasPrefix(name, "vd") || strings.HasPrefix(name, "xvd") || strings.HasPrefix(name, "dm-") {
		return models.DiskHealthTypeVirtual
	}
	identity := strings.Join([]string{
		readSysfsString(filepath.Join(deviceRoot, "device/model")),
		readSysfsString(filepath.Join(deviceRoot, "device/vendor")),
	}, " ")
	if diskHealthIdentityLooksVirtual(identity) {
		return models.DiskHealthTypeVirtual
	}
	if strings.HasPrefix(name, "nvme") {
		return models.DiskHealthTypeNVMe
	}
	rotational := strings.TrimSpace(string(readFileOrNil(filepath.Join(deviceRoot, "queue/rotational"))))
	if strings.HasPrefix(name, "sd") || strings.HasPrefix(name, "hd") || rotational == "1" {
		return models.DiskHealthTypeATA
	}
	return models.DiskHealthTypeUnknown
}

func diskHealthIdentityLooksVirtual(identity string) bool {
	normalized := strings.ToLower(strings.TrimSpace(identity))
	if normalized == "" {
		return false
	}
	for _, marker := range []string{
		"amazon elastic block store",
		"amazon ec2 nvme instance storage",
		"google persistentdisk",
		"persistent disk",
		"microsoft virtual",
		"virtual disk",
		"qemu",
		"vmware",
		"vbox",
		"xen",
		"virtio",
		"alibaba cloud",
		"tencent cloud",
		"cloud block",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func readSysfsString(path string) string {
	return strings.TrimSpace(string(readFileOrNil(path)))
}

func readFileOrNil(path string) []byte {
	data, err := readFileFunc(path)
	if err != nil {
		return nil
	}
	return data
}

func readBlockDeviceSizeBytes(path string) uint64 {
	raw := strings.TrimSpace(string(readFileOrNil(path)))
	if raw == "" {
		return 0
	}
	sectors, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return sectors * 512
}

func readBlockDeviceReadOnly(path string) bool {
	return strings.TrimSpace(string(readFileOrNil(path))) == "1"
}

func readDiskHealthMounts(deviceName string) []models.DiskHealthMount {
	raw := readFileOrNil(diskHealthProcMountsPath)
	if len(raw) == 0 {
		return nil
	}

	lines := strings.Split(string(raw), "\n")
	mounts := make([]models.DiskHealthMount, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if !mountSourceMatchesBlockDevice(fields[0], deviceName) {
			continue
		}
		mounts = append(mounts, models.DiskHealthMount{
			Mount:    decodeProcMountField(fields[1]),
			FSType:   fields[2],
			ReadOnly: mountOptionsReadOnly(fields[3]),
		})
	}
	sort.Slice(mounts, func(i, j int) bool {
		return mounts[i].Mount < mounts[j].Mount
	})
	return mounts
}

func mountSourceMatchesBlockDevice(source, deviceName string) bool {
	base := filepath.Base(decodeProcMountField(source))
	if base == deviceName {
		return true
	}
	if !strings.HasPrefix(base, deviceName) {
		return false
	}
	return isPartitionSuffix(strings.TrimPrefix(base, deviceName))
}

func isPartitionSuffix(suffix string) bool {
	if suffix == "" {
		return false
	}
	if suffix[0] == 'p' {
		suffix = suffix[1:]
	}
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func decodeProcMountField(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\`,
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, `\`,
	)
	return replacer.Replace(value)
}

func mountOptionsReadOnly(options string) bool {
	for _, option := range strings.Split(options, ",") {
		if option == "ro" {
			return true
		}
	}
	return false
}

func nvmeControllerPathForBlockDevice(name string) (string, bool) {
	if !strings.HasPrefix(name, "nvme") {
		return "", false
	}
	namespaceIndex := strings.LastIndex(name, "n")
	if namespaceIndex <= len("nvme") {
		return "", false
	}
	controllerName := name[:namespaceIndex]
	return filepath.Join(diskHealthDevRootPath, controllerName), true
}

func readNVMeHealthLog(controllerPath string) (nvmeHealthLog, error) {
	file, err := os.OpenFile(controllerPath, os.O_RDONLY, 0)
	if err != nil {
		return nvmeHealthLog{}, err
	}
	defer file.Close()

	data := make([]byte, 512)
	cmd := nvmePassthruCmd{
		Opcode:    nvmeAdminGetLog,
		Nsid:      nvmeGlobalNSID,
		Addr:      uint64(uintptr(unsafe.Pointer(&data[0]))),
		DataLen:   uint32(len(data)),
		Cdw10:     ((uint32(len(data)/4) - 1) << 16) | nvmeSmartLogID,
		TimeoutMS: 1000,
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), nvmeIoctlAdminCmd, uintptr(unsafe.Pointer(&cmd)))
	if errno != 0 {
		return nvmeHealthLog{}, errno
	}
	return parseNVMeHealthLog(data), nil
}

func readATAHealth(devicePath string) (ataHealthLog, error) {
	file, err := os.OpenFile(devicePath, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return ataHealthLog{}, err
	}
	defer file.Close()

	data, err := readATASmartData(file.Fd())
	if err != nil {
		return ataHealthLog{}, err
	}
	return parseATAHealthLog(data), nil
}

func readATASmartData(fd uintptr) ([]byte, error) {
	data := make([]byte, 512)
	sense := make([]byte, 32)
	cdb := [16]byte{
		0:  ataPassThrough16,
		1:  ataProtocolPIODataIn << 1,
		2:  0x0e, // T_DIR=device-to-host, BYT_BLOK=blocks, T_LENGTH=sector count.
		4:  ataSmartReadData,
		6:  1,
		10: ataSmartLBAMid,
		12: ataSmartLBAHigh,
		14: ataSmartCommand,
	}
	hdr := sgIOHdr{
		InterfaceID:    sgInterfaceID,
		DxferDirection: sgDxferFromDev,
		CmdLen:         uint8(len(cdb)),
		MxSbLen:        uint8(len(sense)),
		DxferLen:       uint32(len(data)),
		Dxferp:         uintptr(unsafe.Pointer(&data[0])),
		Cmdp:           uintptr(unsafe.Pointer(&cdb[0])),
		Sbp:            uintptr(unsafe.Pointer(&sense[0])),
		Timeout:        5000,
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, sgIoctl, uintptr(unsafe.Pointer(&hdr)))
	if errno != 0 {
		if errno == syscall.ENOTTY || errno == syscall.EINVAL || errno == syscall.ENODEV {
			return nil, fmt.Errorf("%w: %v", errATASmartUnsupported, errno)
		}
		return nil, errno
	}
	if hdr.Status != 0 || hdr.HostStatus != 0 || hdr.DriverStatus != 0 {
		if ataSenseReportsUnsupported(sense[:min(int(hdr.SbLenWr), len(sense))]) {
			return nil, errATASmartUnsupported
		}
		return nil, fmt.Errorf("SG_IO status=%#x host=%#x driver=%#x", hdr.Status, hdr.HostStatus, hdr.DriverStatus)
	}
	return data, nil
}

func ataSenseReportsUnsupported(sense []byte) bool {
	if len(sense) < 4 {
		return false
	}
	responseCode := sense[0] & 0x7f
	switch responseCode {
	case 0x70, 0x71:
		if len(sense) < 14 {
			return false
		}
		senseKey := sense[2] & 0x0f
		return senseKey == 0x05
	case 0x72, 0x73:
		senseKey := sense[1] & 0x0f
		return senseKey == 0x05
	default:
		return false
	}
}

func parseNVMeHealthLog(data []byte) nvmeHealthLog {
	if len(data) < 176 {
		return nvmeHealthLog{}
	}

	var temperature *float64
	kelvin := binary.LittleEndian.Uint16(data[1:3])
	if kelvin > 0 {
		celsius := float64(kelvin) - 273.15
		temperature = &celsius
	}
	lifeUsed := float64(data[5])
	availableSpare := float64(data[3])
	return nvmeHealthLog{
		CriticalWarning: data[0],
		TemperatureC:    temperature,
		LifeUsedPercent: &lifeUsed,
		AvailableSpare:  &availableSpare,
		PowerOnHours:    binary.LittleEndian.Uint64(data[128:136]),
		UnsafeShutdowns: binary.LittleEndian.Uint64(data[144:152]),
		MediaErrors:     binary.LittleEndian.Uint64(data[160:168]),
	}
}

func parseATAHealthLog(data []byte) ataHealthLog {
	if len(data) < 362 {
		return ataHealthLog{}
	}

	var log ataHealthLog
	for offset := 2; offset+12 <= 362; offset += 12 {
		id := data[offset]
		if id == 0 {
			continue
		}
		current := data[offset+3]
		raw := ataAttributeRawValue(data[offset+5 : offset+11])
		switch id {
		case 5:
			log.ReallocatedSectors = raw
		case 9:
			log.PowerOnHours = raw
		case 187:
			log.ReportedUncorrectable = raw
		case 190, 194:
			if temperature := ataTemperatureFromRaw(raw); temperature != nil {
				log.TemperatureC = temperature
			}
		case 197:
			log.PendingSectors = raw
		case 198:
			log.OfflineUncorrectable = raw
		case 199:
			log.InterfaceCRCErrors = raw
		case 177, 202, 231, 233:
			if lifeUsed, ok := ataLifeUsedFromAttribute(current, raw); ok {
				log.LifeUsedPercent = &lifeUsed
			}
		}
	}
	return log
}

func ataAttributeRawValue(raw []byte) uint64 {
	var value uint64
	for index := 0; index < len(raw) && index < 6; index++ {
		value |= uint64(raw[index]) << (8 * index)
	}
	return value
}

func ataTemperatureFromRaw(raw uint64) *float64 {
	if raw == 0 {
		return nil
	}
	temperature := raw & 0xff
	if temperature == 0 || temperature > 150 {
		temperature = raw
	}
	if temperature == 0 || temperature > 150 {
		return nil
	}
	result := float64(temperature)
	return &result
}

func ataLifeUsedFromAttribute(current byte, raw uint64) (float64, bool) {
	if raw > 0 && raw <= 100 {
		return float64(raw), true
	}
	if current > 0 && current <= 100 {
		return 100 - float64(current), true
	}
	return 0, false
}

func diskHealthFromATAHealthLog(stats models.DiskHealthStats, log ataHealthLog) models.DiskHealthStats {
	stats.Status = models.DiskHealthStatusOK
	stats.SupportStatus = models.DiskHealthSupportSupported
	stats.TemperatureC = log.TemperatureC
	stats.LifeUsedPercent = log.LifeUsedPercent
	stats.PowerOnHours = log.PowerOnHours
	stats.ReallocatedSectors = log.ReallocatedSectors
	stats.PendingSectors = log.PendingSectors
	stats.OfflineUncorrectable = log.OfflineUncorrectable
	stats.InterfaceCRCErrors = log.InterfaceCRCErrors
	stats.MediaErrors = log.ReallocatedSectors + log.PendingSectors + log.OfflineUncorrectable + log.ReportedUncorrectable

	if log.PendingSectors > 0 || log.OfflineUncorrectable > 0 {
		stats.Status = models.DiskHealthStatusCritical
		stats.Message = "ATA SMART reports unstable or uncorrectable sectors"
		return stats
	}
	if log.TemperatureC != nil && *log.TemperatureC >= 80 {
		stats.Status = models.DiskHealthStatusCritical
		stats.Message = "ATA temperature is critical"
		return stats
	}
	if log.LifeUsedPercent != nil && *log.LifeUsedPercent >= 100 {
		stats.Status = models.DiskHealthStatusCritical
		stats.Message = "ATA estimated life is exhausted"
		return stats
	}
	if log.ReallocatedSectors > 0 || log.ReportedUncorrectable > 0 {
		stats.Status = models.DiskHealthStatusWarning
		stats.Message = "ATA SMART reports media errors"
		return stats
	}
	if log.TemperatureC != nil && *log.TemperatureC >= 70 {
		stats.Status = models.DiskHealthStatusWarning
		stats.Message = "ATA temperature is elevated"
		return stats
	}
	if log.LifeUsedPercent != nil && *log.LifeUsedPercent >= 90 {
		stats.Status = models.DiskHealthStatusWarning
		stats.Message = "ATA estimated life is near exhaustion"
		return stats
	}
	if log.InterfaceCRCErrors > 0 {
		stats.Status = models.DiskHealthStatusWarning
		stats.Message = "ATA interface CRC errors were reported"
		return stats
	}
	return stats
}

func diskHealthFromNVMeLog(stats models.DiskHealthStats, log nvmeHealthLog) models.DiskHealthStats {
	stats.Status = models.DiskHealthStatusOK
	stats.SupportStatus = models.DiskHealthSupportSupported
	stats.CriticalWarning = log.CriticalWarning
	stats.TemperatureC = log.TemperatureC
	stats.LifeUsedPercent = log.LifeUsedPercent
	stats.AvailableSparePercent = log.AvailableSpare
	stats.PowerOnHours = log.PowerOnHours
	stats.UnsafeShutdowns = log.UnsafeShutdowns
	stats.MediaErrors = log.MediaErrors

	if log.CriticalWarning != 0 {
		stats.Status = models.DiskHealthStatusCritical
		stats.Message = "NVMe critical warning is set"
		return stats
	}
	if log.TemperatureC != nil && *log.TemperatureC >= 80 {
		stats.Status = models.DiskHealthStatusCritical
		stats.Message = "NVMe temperature is critical"
		return stats
	}
	if log.LifeUsedPercent != nil && *log.LifeUsedPercent >= 100 {
		stats.Status = models.DiskHealthStatusCritical
		stats.Message = "NVMe estimated life is exhausted"
		return stats
	}
	if log.MediaErrors > 0 {
		stats.Status = models.DiskHealthStatusWarning
		stats.Message = "NVMe media errors were reported"
		return stats
	}
	if log.TemperatureC != nil && *log.TemperatureC >= 70 {
		stats.Status = models.DiskHealthStatusWarning
		stats.Message = "NVMe temperature is elevated"
		return stats
	}
	if log.LifeUsedPercent != nil && *log.LifeUsedPercent >= 90 {
		stats.Status = models.DiskHealthStatusWarning
		stats.Message = "NVMe estimated life is near exhaustion"
		return stats
	}
	if log.AvailableSpare != nil && *log.AvailableSpare <= 10 {
		stats.Status = models.DiskHealthStatusWarning
		stats.Message = "NVMe available spare is low"
		return stats
	}
	return stats
}

func (c *Collector) cachedNonLoopbackInterfaceNames(currentTime time.Time) map[string]struct{} {
	c.networkCacheMu.Lock()
	if !c.cachedInterfacesAt.IsZero() && currentTime.Sub(c.cachedInterfacesAt) < networkTopologyCacheTTL && len(c.cachedInterfaces) > 0 {
		cached := cloneInterfaceNames(c.cachedInterfaces)
		c.networkCacheMu.Unlock()
		return cached
	}
	c.networkCacheMu.Unlock()

	selectedInterfaces := nonLoopbackInterfaceNames()
	if len(selectedInterfaces) == 0 {
		return nil
	}

	cloned := cloneInterfaceNames(selectedInterfaces)
	c.networkCacheMu.Lock()
	c.cachedInterfaces = cloned
	c.cachedInterfacesAt = currentTime
	c.networkCacheMu.Unlock()
	return cloneInterfaceNames(cloned)
}

func (c *Collector) cachedDetectedNetworkMetadata(currentTime time.Time) (string, []string) {
	c.networkCacheMu.Lock()
	if !c.cachedLocalIPAt.IsZero() && currentTime.Sub(c.cachedLocalIPAt) < networkTopologyCacheTTL && (c.cachedLocalIP != "" || len(c.cachedIPFamilies) > 0) {
		cachedIP := c.cachedLocalIP
		cachedFamilies := cloneStrings(c.cachedIPFamilies)
		c.networkCacheMu.Unlock()
		return cachedIP, cachedFamilies
	}
	c.networkCacheMu.Unlock()

	ip, families := detectLocalNetworkMetadata()
	if ip == "" && len(families) == 0 {
		return "", nil
	}

	c.networkCacheMu.Lock()
	c.cachedLocalIP = ip
	c.cachedIPFamilies = cloneStrings(families)
	c.cachedLocalIPAt = currentTime
	c.networkCacheMu.Unlock()
	return ip, cloneStrings(families)
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func detectedIPFamilies() []string {
	_, families := detectLocalNetworkMetadata()
	return families
}

func detectLocalNetworkMetadata() (string, []string) {
	addrs, err := interfaceAddrsFunc()
	if err != nil {
		return "", nil
	}
	return detectLocalIPFromAddrs(addrs), detectedIPFamiliesFromAddrs(addrs)
}

func detectedIPFamiliesFromAddrs(addrs []net.Addr) []string {
	hasIPv4 := false
	hasIPv6 := false
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP == nil {
			continue
		}
		if ipNet.IP.To4() != nil {
			if isUsableIPv4ForExternalProbes(ipNet.IP) {
				hasIPv4 = true
			}
			continue
		}
		if isUsableIPv6ForExternalProbes(ipNet.IP) {
			hasIPv6 = true
		}
	}
	families := make([]string, 0, 2)
	if hasIPv4 {
		families = append(families, "ipv4")
	}
	if hasIPv6 {
		families = append(families, "ipv6")
	}
	return families
}

func isUsableIPv4ForExternalProbes(ip net.IP) bool {
	if ip == nil || ip.To4() == nil {
		return false
	}
	return !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}

func isUsableIPv6ForExternalProbes(ip net.IP) bool {
	if ip == nil || ip.To4() != nil {
		return false
	}
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !isDocumentationIPv6(ip) && !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}

func isDocumentationIPv6(ip net.IP) bool {
	parsed := ip.To16()
	if parsed == nil || ip.To4() != nil {
		return false
	}
	return parsed[0] == 0x20 && parsed[1] == 0x01 && parsed[2] == 0x0d && parsed[3] == 0xb8
}

func (c *Collector) collectNetworkStats(currentTime time.Time) models.NetStats {
	return collectNetworkStatsForInterfaces(c.cachedNonLoopbackInterfaceNames(currentTime))
}

type pressureFileStats struct {
	SomeAvg10 float64
	FullAvg10 float64
}

func parsePressureFile(data []byte) (pressureFileStats, error) {
	var stats pressureFileStats
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		var target *float64
		switch fields[0] {
		case "some":
			target = &stats.SomeAvg10
		case "full":
			target = &stats.FullAvg10
		default:
			continue
		}
		for _, field := range fields[1:] {
			valueText, ok := strings.CutPrefix(field, "avg10=")
			if !ok {
				continue
			}
			value, err := strconv.ParseFloat(valueText, 64)
			if err != nil {
				return pressureFileStats{}, err
			}
			*target = value
			break
		}
	}
	return stats, nil
}

func readPressureFile(path string) pressureFileStats {
	data, err := readFileFunc(path)
	if err != nil {
		return pressureFileStats{}
	}
	stats, err := parsePressureFile(data)
	if err != nil {
		return pressureFileStats{}
	}
	return stats
}

func collectPressureStats() models.PressureStats {
	cpuPressure := readPressureFile("/proc/pressure/cpu")
	memoryPressure := readPressureFile("/proc/pressure/memory")
	ioPressure := readPressureFile("/proc/pressure/io")
	return models.PressureStats{
		CPUSomeAvg10:    cpuPressure.SomeAvg10,
		MemorySomeAvg10: memoryPressure.SomeAvg10,
		MemoryFullAvg10: memoryPressure.FullAvg10,
		IOSomeAvg10:     ioPressure.SomeAvg10,
		IOFullAvg10:     ioPressure.FullAvg10,
	}
}

func collectSwapStats() models.SwapStats {
	swap, err := swapMemoryFunc()
	if err != nil || swap == nil {
		return models.SwapStats{}
	}
	return models.SwapStats{
		Used:  swap.Used,
		Total: swap.Total,
		In:    swap.Sin,
		Out:   swap.Sout,
	}
}

func collectOOMKills() uint64 {
	if count, ok := readOOMKillCountFromVMStat("/proc/vmstat"); ok {
		return count
	}
	return countOOMKillEvents([]string{"/var/log/kern.log", "/var/log/messages", "/var/log/syslog"})
}

func readOOMKillCountFromVMStat(path string) (uint64, bool) {
	data, err := readFileFunc(path)
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != "oom_kill" {
			continue
		}
		count, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, false
		}
		return count, true
	}
	return 0, false
}

func countOOMKillEvents(paths []string) uint64 {
	var count uint64
	for _, path := range paths {
		data, err := readFileFunc(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.ToLower(string(data)), "\n") {
			if strings.Contains(line, "out of memory: killed process") || strings.Contains(line, "oom-kill:") {
				count++
			}
		}
	}
	return count
}

// Collect gathers a single snapshot of system metrics.
func (c *Collector) Collect() (*models.MetricsPayload, error) {
	currentTime := c.currentTime()
	detectedIP, ipFamilies := c.cachedDetectedNetworkMetadata(currentTime)
	ip := c.nodeIP
	if net.ParseIP(ip) == nil {
		ip = detectedIP
	}

	payload := &models.MetricsPayload{
		Type:         "metrics",
		TS:           currentTime.Unix(),
		IP:           ip,
		IPFamilies:   ipFamilies,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		AgentVersion: c.agentVersion,
		Services:     []models.Service{},
	}
	if hostInfo, err := hostInfoFunc(); err == nil && hostInfo != nil {
		payload.UptimeSeconds = hostInfo.Uptime
	}
	if loadAvg, err := loadAvgFunc(); err == nil && loadAvg != nil {
		payload.Load = models.LoadStats{
			Load1:  loadAvg.Load1,
			Load5:  loadAvg.Load5,
			Load15: loadAvg.Load15,
		}
	}
	payload.Pressure = collectPressureStats()
	payload.Swap = collectSwapStats()
	payload.OOMKills = collectOOMKills()

	// CPU — compute usage from non-blocking cumulative CPU time deltas.
	if cpuStats, ok := c.sampleCPUStats(); ok {
		payload.CPU = cpuStats.UsagePercent
		payload.CPUStats = models.CPUStats{
			IOWaitPercent: cpuStats.IOWaitPercent,
			StealPercent:  cpuStats.StealPercent,
		}
	}

	// Memory.
	vmStat, err := virtualMemoryFunc()
	if err == nil {
		payload.Mem = models.MemStats{
			Used:  vmStat.Used,
			Total: vmStat.Total,
		}
	}

	// Disk — iterate partitions, skip any that fail Usage().
	// Device is included so server-side aggregation can de-dupe bind mounts of
	// the same underlying block device (they report identical capacity).
	partitions, err := diskPartitionsFunc(false)
	if err == nil {
		for _, p := range partitions {
			usage, err := diskUsageFunc(p.Mountpoint)
			if err != nil {
				continue
			}
			payload.Disk = append(payload.Disk, models.DiskStats{
				Mount:  p.Mountpoint,
				Device: p.Device,
				Used:   usage.Used,
				Total:  usage.Total,
			})
		}
	}
	payload.DiskIO = collectDiskIOStats()

	// Network — aggregate across active non-loopback interfaces with cached topology selection.
	payload.Net = c.collectNetworkStats(currentTime)

	if c.shouldCollectHeavySnapshot(currentTime) {
		if !c.heavySnapshotSent {
			if payload.Hardware = c.hardware(); payload.Hardware != nil && payload.Hardware.IsEmpty() {
				payload.Hardware = nil
			}
		}
		payload.DiskHealth = collectDiskHealth()
		processSamples, err := collectProcessSamplesFunc()
		if err == nil {
			payload.Processes = selectTopProcesses(processSamples, processSnapshotLimit)
			if payload.Processes == nil {
				payload.Processes = []models.Process{}
			}
		}

		// Docker containers — graceful degradation when Docker is unavailable.
		containers, dockerAvailable, _ := collectDockerContainersFunc()
		payload.DockerAvailable = &dockerAvailable
		if dockerAvailable {
			payload.Containers = containers
		}
		c.lastHeavySnapshotAt = currentTime
		c.heavySnapshotSent = true
	}

	return payload, nil
}

func (c *Collector) currentTime() time.Time {
	if c != nil && c.now != nil {
		return c.now()
	}
	return nowFunc()
}

func (c *Collector) shouldCollectHeavySnapshot(currentTime time.Time) bool {
	if c == nil {
		return true
	}
	if !c.heavySnapshotSent {
		return true
	}
	interval := c.heavySnapshotInterval
	if interval <= 0 {
		interval = DefaultHeavySnapshotInterval
	}
	return currentTime.Sub(c.lastHeavySnapshotAt) >= interval
}

func collectProcessSamples() ([]models.Process, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	processSamples := make([]models.Process, 0, 64)
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
	return processSamples, nil
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
		// Count each block device once. Bind mounts re-export the same
		// filesystem total and would otherwise inflate DiskTotal.
		seenDevices := make(map[string]struct{}, len(partitions))
		for _, partition := range partitions {
			if partition.Device != "" {
				if _, seen := seenDevices[partition.Device]; seen {
					continue
				}
				seenDevices[partition.Device] = struct{}{}
			}
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
	c.heavySnapshotSent = false
	c.lastHeavySnapshotAt = time.Time{}

	var writeMu sync.Mutex

	// Send the stream/watermark handshake before anything else. The server
	// withholds control commands until it has reconciled this state.
	if err := c.sendHandshake(conn, &writeMu); err != nil {
		log.Printf("collector: handshake send failed: %v", err)
	}

	ticker := time.NewTicker(c.reportInterval)
	defer ticker.Stop()
	latencyTicker := time.NewTicker(time.Second)
	defer latencyTicker.Stop()
	autoUpdateTicker := time.NewTicker(c.autoUpdateInterval)
	defer autoUpdateTicker.Stop()
	autoUpdateNow := make(chan struct{}, 1)
	autoUpdateNow <- struct{}{}
	autoUpdateDone := make(chan struct{}, 1)
	autoUpdateRunning := false
	latencyErrCh := make(chan error, 1)
	latencyRunDone := make(chan struct{}, 1)
	latencyRunning := false
	startAutoUpdateCheck := func() {
		if autoUpdateRunning {
			return
		}
		autoUpdateRunning = true
		go func() {
			if err := c.checkForAutoUpdate(); err != nil {
				log.Printf("collector: auto update check failed: %v", err)
			}
			select {
			case autoUpdateDone <- struct{}{}:
			default:
			}
		}()
	}

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
		case err := <-latencyErrCh:
			c.noteConnectionError(mode, conn.RemoteAddr(), err)
			return err
		case <-autoUpdateDone:
			autoUpdateRunning = false
		case <-latencyRunDone:
			latencyRunning = false
		case <-autoUpdateNow:
			startAutoUpdateCheck()
		case <-autoUpdateTicker.C:
			startAutoUpdateCheck()
		case <-latencyTicker.C:
			if latencyRunning {
				continue
			}
			latencyRunning = true
			go func(currentTime time.Time) {
				if err := c.runDueLatencyMonitors(conn, &writeMu, currentTime); err != nil {
					select {
					case latencyErrCh <- err:
					default:
					}
				}
				select {
				case latencyRunDone <- struct{}{}:
				default:
				}
			}(c.currentTime())
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

func (c *Collector) sendHandshake(conn websocketConn, writeMu *sync.Mutex) error {
	raw, err := json.Marshal(models.WSMessage{Type: models.WSMessageTypeAgentHandshake, Payload: c.handshakePayload()})
	if err != nil {
		return err
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, raw)
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
		switch envelope.Type {
		case models.WSMessageTypeAgentCommand:
			var payload models.AgentCommandPayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				continue
			}
			c.dispatchAgentCommand(payload, conn, writeMu)
		case models.WSMessageTypeAgentHandshakeAck:
			var payload models.AgentHandshakeAckPayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				continue
			}
			c.applyHandshakeAck(payload)
			if payload.ControlState == models.ControlStateNone {
				log.Printf("collector: handshake complete (stream %s, watermark %d)", payload.StreamID, payload.RetiredWatermark)
			}
		case models.WSMessageTypeAgentCommandCancel:
			var payload models.AgentCommandCancelPayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				continue
			}
			c.handleCancelRequest(payload, conn, writeMu)
		case "latency_monitor_config":
			var payload models.LatencyMonitorConfigPayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				continue
			}
			c.applyLatencyMonitorConfig(payload.Monitors)
		}
	}
}

// handleCancelRequest applies a persistent cancel intent. Before the rename
// commit point the task is cancelled, the temp file removed and the sequence
// retired with a cancelled tombstone. After the commit point cancellation is
// refused: the committed binary must be Exec'd or reconciled by SHA256.
func (c *Collector) handleCancelRequest(payload models.AgentCommandCancelPayload, conn websocketConn, writeMu *sync.Mutex) {
	c.stateMu.Lock()
	inProgress := c.state.InProgress
	c.stateMu.Unlock()
	if inProgress == nil || (payload.Seq > 0 && inProgress.Seq != payload.Seq) || (payload.JobID != "" && inProgress.JobID != payload.JobID) {
		_ = c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{
			JobID: payload.JobID, Seq: payload.Seq, StreamID: c.stateStreamID(), RetiredWatermark: c.stateWatermark(), RejectCode: models.CommandRejectAlreadyProcessed, Message: "no matching in-progress task",
		})
		return
	}
	if inProgress.CommitPoint {
		_ = c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{
			JobID: payload.JobID, Seq: payload.Seq, StreamID: c.stateStreamID(), RetiredWatermark: c.stateWatermark(), Status: models.UpdateJobTargetStatusRestarting,
			Message: "cancel refused: rename already committed; update continues",
		})
		return
	}
	c.stateMu.Lock()
	if c.state.InProgress != nil && c.state.InProgress.Key == inProgress.Key {
		c.state.InProgress.CancelIntent = true
		_ = c.saveAgentState()
	}
	c.stateMu.Unlock()
	if c.cancelTask(payload.Seq, payload.JobID) {
		// The task reports the durable cancelled status when it observes the
		// cancellation; acknowledge promptly on the current socket as well.
		_ = c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{
			JobID: payload.JobID, Seq: payload.Seq, StreamID: c.stateStreamID(), RetiredWatermark: c.stateWatermark(), Status: models.UpdateJobTargetStatusCancelled,
			Message: "cancel intent applied",
		})
	}
}

func (c *Collector) dispatchAgentCommand(cmd models.AgentCommandPayload, conn websocketConn, writeMu *sync.Mutex) {
	if cmd.Kind != models.AgentCommandKindSelfUpdate {
		_ = c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{JobID: cmd.JobID, Status: models.UpdateJobTargetStatusFailed, Message: "unsupported agent command"})
		return
	}
	ok, replay, rejectCode, rejectMsg := c.beginUpdateTask(cmd)
	if !ok {
		if replay != nil {
			// Exact replay of a persisted state (dedup, in-progress duplicate,
			// retired sequence). No download, rename or exec side effect.
			_ = c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{
				JobID: replay.JobID, Status: replay.Status, Message: replay.Message,
				ReportedVersion: replay.ReportedVersion, StreamID: c.stateStreamID(), Seq: replay.Seq,
				RetiredWatermark: c.stateWatermark(),
			})
			return
		}
		payload := models.AgentCommandStatusPayload{
			JobID: cmd.JobID, StreamID: c.stateStreamID(), Seq: cmd.Seq,
			RetiredWatermark: c.stateWatermark(), Message: rejectMsg,
		}
		if rejectCode != "" {
			payload.RejectCode = rejectCode
		} else {
			payload.Status = models.UpdateJobTargetStatusFailed
		}
		_ = c.sendAgentCommandStatus(conn, writeMu, payload)
		return
	}

	c.taskWG.Add(1)
	go func() {
		defer c.taskWG.Done()
		report := func(status models.UpdateJobTargetStatus, message, version string) error {
			return c.sendAgentCommandStatus(conn, writeMu, models.AgentCommandStatusPayload{
				JobID: cmd.JobID, Status: status, Message: message,
				ReportedVersion: version, StreamID: c.stateStreamID(), Seq: cmd.Seq,
			})
		}
		if err := report(models.UpdateJobTargetStatusAccepted, "accepted", c.agentVersion); err != nil {
			// This socket is gone; revert the durable admission so the command
			// is redelivered after reconnect instead of being stuck.
			c.abortUpdateTask(cmd)
			return
		}
		_ = c.runUpdateTask(cmd, report)
	}()
}

func (c *Collector) stateStreamID() string {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.state.StreamID
}

// WaitForTasks blocks until all in-flight update tasks have finalized their
// durable state. Intended for tests and orderly agent shutdown.
func (c *Collector) WaitForTasks() {
	c.taskWG.Wait()
}

func (c *Collector) stateWatermark() int64 {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.state.RetiredWatermark
}

func (c *Collector) currentTaskContext() context.Context {
	c.taskMu.Lock()
	defer c.taskMu.Unlock()
	if c.taskCtx != nil {
		return c.taskCtx
	}
	return context.Background()
}

// abortUpdateTask reverts the durable accepted/in-progress admission when the
// task cannot even report its acceptance on the current socket.
func (c *Collector) abortUpdateTask(cmd models.AgentCommandPayload) {
	key := commandDedupKey(cmd)
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.state.InProgress == nil || c.state.InProgress.Key != key {
		return
	}
	c.state.InProgress = nil
	kept := c.state.Records[:0]
	for _, record := range c.state.Records {
		if record.Key != key {
			kept = append(kept, record)
		}
	}
	c.state.Records = kept
	_ = c.saveAgentState()
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

	// The task context carries the persistent cancel intent; the download
	// stage lease bounds the network read.
	taskCtx := c.currentTaskContext()

	if err := report(models.UpdateJobTargetStatusDownloading, "downloading replacement binary", c.agentVersion); err != nil {
		return err
	}
	c.updateTaskProgress(models.UpdateJobTargetStatusDownloading, "downloading replacement binary")

	downloadCtx, cancelDownload := context.WithTimeout(taskCtx, leaseDownload)
	defer cancelDownload()
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, cmd.DownloadURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return errSelfUpdateCancelled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("download stage lease expired: %w", err)
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	binaryData, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return errSelfUpdateCancelled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("download stage lease expired: %w", err)
		}
		return err
	}

	if err := report(models.UpdateJobTargetStatusVerifying, "verifying binary checksum", c.agentVersion); err != nil {
		return err
	}
	c.updateTaskProgress(models.UpdateJobTargetStatusVerifying, "verifying binary checksum")
	verifyCtx, cancelVerify := context.WithTimeout(taskCtx, leaseVerify)
	defer cancelVerify()
	if err := verifyCtx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return errSelfUpdateCancelled
		}
		return fmt.Errorf("verify stage lease expired: %w", err)
	}

	digest := sha256.Sum256(binaryData)
	actualChecksum := strings.ToLower(hex.EncodeToString(digest[:]))
	if strings.ToLower(strings.TrimSpace(cmd.SHA256)) != actualChecksum {
		return fmt.Errorf("sha256 mismatch")
	}
	if err := release.VerifyBinary(binaryData, cmd.Signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	if err := verifyCtx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return errSelfUpdateCancelled
		}
		return fmt.Errorf("verify stage lease expired: %w", err)
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
	if err := writeSyncedFile(tmpPath, binaryData, fileInfo.Mode().Perm()); err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if err := taskCtx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return errSelfUpdateCancelled
		}
		return err
	}

	// Commit point: os.Rename is the irreversible substitution. From here on
	// cancellation is refused and every failure is a conservative post-commit
	// recovery (SHA256 reconciliation), never a clean "did not execute".
	if err := c.markCommitPoint(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		return c.postCommitFailure(cmd, report, fmt.Errorf("rename failed after commit point: %w", err))
	}

	restartCtx, cancelRestart := context.WithTimeout(context.Background(), leaseRestart)
	defer cancelRestart()
	if err := restartCtx.Err(); err != nil {
		return c.postCommitFailure(cmd, report, fmt.Errorf("restart stage lease expired: %w", err))
	}

	if err := report(models.UpdateJobTargetStatusRestarting, "restarting agent", cmd.TargetVersion); err != nil {
		log.Printf("collector: failed to report self-update restart status: %v", err)
	}
	c.updateTaskProgress(models.UpdateJobTargetStatusRestarting, "restarting agent")

	versionPath := persistedAgentVersionPath()
	if versionPath != "" {
		if err := writeSyncedFile(versionPath, []byte(strings.TrimSpace(cmd.TargetVersion)+"\n"), 0644); err != nil {
			return c.postCommitFailure(cmd, report, fmt.Errorf("version file write failed after commit point: %w", err))
		}
	}
	if err := restartCtx.Err(); err != nil {
		return c.postCommitFailure(cmd, report, fmt.Errorf("restart stage lease expired: %w", err))
	}

	if err := syscall.Exec(exePath, os.Args, os.Environ()); err != nil {
		return c.postCommitFailure(cmd, report, fmt.Errorf("exec failed after commit point: %w", err))
	}
	return nil
}

// postCommitFailure records an unknown execution state and blocks the node's
// control plane: the binary was replaced but the outcome cannot be proven.
// Recovery happens through SHA256 evidence (startup recovery or operator
// disposition).
func (c *Collector) postCommitFailure(cmd models.AgentCommandPayload, report func(models.UpdateJobTargetStatus, string, string) error, err error) error {
	c.finishUpdateTask(models.UpdateJobTargetStatusUnknown, "post-commit failure: "+err.Error())
	_ = report(models.UpdateJobTargetStatusUnknown, "post-commit failure: "+err.Error(), cmd.TargetVersion)
	return err
}

func writeSyncedFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
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
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+c.token)

	for index, mode := range c.dialModes() {
		conn, err := c.dialWebsocket(mode, targetURL, headers)
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

func defaultWebsocketDial(mode dialMode, targetURL string, headers http.Header) (websocketConn, error) {
	dialer := *websocket.DefaultDialer
	parsedTarget, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(parsedTarget.Scheme, "wss") {
		tlsConfig, err := tlsConfigWithServerPin(dialer.TLSClientConfig)
		if err != nil {
			return nil, err
		}
		dialer.TLSClientConfig = tlsConfig
	}
	if mode == dialModeIPv4 {
		netDialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return netDialer.DialContext(ctx, "tcp4", addr)
		}
	}

	conn, _, err := dialer.Dial(targetURL, headers)
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
	ip, _ := detectLocalNetworkMetadata()
	return ip
}

func detectLocalIPFromAddrs(addrs []net.Addr) string {
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
