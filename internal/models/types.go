package models

// Node represents a registered monitored server.
type Node struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Token         string               `json:"-"` // never serialized to API responses
	IP            string               `json:"ip"`
	OS            string               `json:"os"`
	Arch          string               `json:"arch"`
	CountryCode   string               `json:"country_code,omitempty"`
	AgentVersion  string               `json:"agent_version,omitempty"`
	IPFamilies    []string             `json:"ip_families,omitempty"`
	CreatedAt     int64                `json:"created_at"`
	LastSeen      int64                `json:"last_seen"`
	Online        bool                 `json:"online"`
	Tags          []string             `json:"tags"`
	Hardware      *NodeHardware        `json:"hardware,omitempty"`
	DiskHealth    []DiskHealthStats    `json:"disk_health,omitempty"`
	LatestMetrics *NodeMetricsSnapshot `json:"latest_metrics,omitempty"`
}

type NodeHardware struct {
	CPUModel             string `json:"cpu_model,omitempty"`
	CPUCores             int    `json:"cpu_cores,omitempty"`
	CPUThreads           int    `json:"cpu_threads,omitempty"`
	MemoryTotal          uint64 `json:"memory_total,omitempty"`
	DiskTotal            uint64 `json:"disk_total,omitempty"`
	VirtualizationSystem string `json:"virtualization_system,omitempty"`
	VirtualizationRole   string `json:"virtualization_role,omitempty"`
}

func (h *NodeHardware) IsEmpty() bool {
	if h == nil {
		return true
	}

	return h.CPUModel == "" &&
		h.CPUCores == 0 &&
		h.CPUThreads == 0 &&
		h.MemoryTotal == 0 &&
		h.DiskTotal == 0 &&
		h.VirtualizationSystem == "" &&
		h.VirtualizationRole == ""
}

type NodeMetricsSnapshot struct {
	TS                 int64   `json:"ts"`
	CPU                float64 `json:"cpu"`
	Load1              float64 `json:"load1"`
	Load5              float64 `json:"load5"`
	Load15             float64 `json:"load15"`
	CPUIOWaitPercent   float64 `json:"cpu_iowait_percent"`
	CPUStealPercent    float64 `json:"cpu_steal_percent"`
	PressureCPUSome    float64 `json:"pressure_cpu_some"`
	PressureMemorySome float64 `json:"pressure_memory_some"`
	PressureMemoryFull float64 `json:"pressure_memory_full"`
	PressureIOSome     float64 `json:"pressure_io_some"`
	PressureIOFull     float64 `json:"pressure_io_full"`
	MemUsed            uint64  `json:"mem_used"`
	MemTotal           uint64  `json:"mem_total"`
	SwapUsed           uint64  `json:"swap_used"`
	SwapTotal          uint64  `json:"swap_total"`
	SwapIn             uint64  `json:"swap_in"`
	SwapOut            uint64  `json:"swap_out"`
	OOMKills           uint64  `json:"oom_kills"`
	DiskUsed           uint64  `json:"disk_used"`
	DiskTotal          uint64  `json:"disk_total"`
	DiskReadBytes      uint64  `json:"disk_read_bytes"`
	DiskWriteBytes     uint64  `json:"disk_write_bytes"`
	NetRx              uint64  `json:"net_rx"`
	NetTx              uint64  `json:"net_tx"`
	UptimeSeconds      uint64  `json:"uptime_seconds,omitempty"`
}

type AvailabilityReportRange struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type AvailabilityReportFilter struct {
	Tag string `json:"tag,omitempty"`
}

type AvailabilityReportOverview struct {
	TotalNodes                  int      `json:"total_nodes"`
	AverageAvailabilityPercent  float64  `json:"average_availability_percent"`
	NodesBelow99                int      `json:"nodes_below_99"`
	TotalOfflineDurationSeconds int64    `json:"total_offline_duration_seconds"`
	HighestLatencyP95Ms         *float64 `json:"highest_latency_p95_ms,omitempty"`
}

type NodeAvailabilityReport struct {
	NodeID                 string   `json:"node_id"`
	Name                   string   `json:"name"`
	Tags                   []string `json:"tags"`
	LastSeen               int64    `json:"last_seen"`
	AvailabilityPercent    float64  `json:"availability_percent"`
	ExpectedSamples        int      `json:"expected_samples"`
	ObservedSamples        int      `json:"observed_samples"`
	OfflineDurationSeconds int64    `json:"offline_duration_seconds"`
	OutageCount            int      `json:"outage_count"`
	LastOutageStart        *int64   `json:"last_outage_start,omitempty"`
	LastOutageEnd          *int64   `json:"last_outage_end,omitempty"`
	LatencyP50Ms           *float64 `json:"latency_p50_ms,omitempty"`
	LatencyP95Ms           *float64 `json:"latency_p95_ms,omitempty"`
}

type AvailabilityReport struct {
	Range         AvailabilityReportRange    `json:"range"`
	Filter        AvailabilityReportFilter   `json:"filter"`
	AvailableTags []string                   `json:"available_tags"`
	Overview      AvailabilityReportOverview `json:"overview"`
	Nodes         []NodeAvailabilityReport   `json:"nodes"`
}

// MetricsPayload is the JSON message sent by agents over WebSocket.
type MetricsPayload struct {
	Type            string            `json:"type"`
	TS              int64             `json:"ts"`
	CPU             float64           `json:"cpu"`
	IP              string            `json:"ip,omitempty"`
	IPFamilies      []string          `json:"ip_families,omitempty"`
	OS              string            `json:"os,omitempty"`
	Arch            string            `json:"arch,omitempty"`
	AgentVersion    string            `json:"agent_version,omitempty"`
	UptimeSeconds   uint64            `json:"uptime_seconds,omitempty"`
	Load            LoadStats         `json:"load"`
	CPUStats        CPUStats          `json:"cpu_stats"`
	Pressure        PressureStats     `json:"pressure"`
	Swap            SwapStats         `json:"swap"`
	OOMKills        uint64            `json:"oom_kills,omitempty"`
	Hardware        *NodeHardware     `json:"hardware,omitempty"`
	Mem             MemStats          `json:"mem"`
	Disk            []DiskStats       `json:"disk"`
	DiskHealth      []DiskHealthStats `json:"disk_health"`
	DiskIO          DiskIOStats       `json:"disk_io"`
	Net             NetStats          `json:"net"`
	Processes       []Process         `json:"processes"`
	Services        []Service         `json:"services"`
	DockerAvailable *bool             `json:"docker_available,omitempty"`
	Containers      []DockerContainer `json:"containers,omitempty"`
}

type LoadStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type CPUStats struct {
	IOWaitPercent float64 `json:"iowait_percent"`
	StealPercent  float64 `json:"steal_percent"`
}

type PressureStats struct {
	CPUSomeAvg10    float64 `json:"cpu_some_avg10"`
	MemorySomeAvg10 float64 `json:"memory_some_avg10"`
	MemoryFullAvg10 float64 `json:"memory_full_avg10"`
	IOSomeAvg10     float64 `json:"io_some_avg10"`
	IOFullAvg10     float64 `json:"io_full_avg10"`
}

type SwapStats struct {
	Used  uint64 `json:"used"`
	Total uint64 `json:"total"`
	In    uint64 `json:"in"`
	Out   uint64 `json:"out"`
}

type MemStats struct {
	Used  uint64 `json:"used"`
	Total uint64 `json:"total"`
}

type DiskStats struct {
	Mount string `json:"mount"`
	Used  uint64 `json:"used"`
	Total uint64 `json:"total"`
}

const (
	DiskHealthTypeNVMe    = "nvme"
	DiskHealthTypeATA     = "ata"
	DiskHealthTypeVirtual = "virtual"
	DiskHealthTypeUnknown = "unknown"

	DiskHealthStatusOK          = "ok"
	DiskHealthStatusWarning     = "warning"
	DiskHealthStatusCritical    = "critical"
	DiskHealthStatusUnsupported = "unsupported"
	DiskHealthStatusUnknown     = "unknown"
)

type DiskHealthStats struct {
	Name                  string   `json:"name"`
	Path                  string   `json:"path,omitempty"`
	Type                  string   `json:"type"`
	Model                 string   `json:"model,omitempty"`
	Serial                string   `json:"serial,omitempty"`
	Firmware              string   `json:"firmware,omitempty"`
	SizeBytes             uint64   `json:"size_bytes,omitempty"`
	Status                string   `json:"status"`
	TemperatureC          *float64 `json:"temperature_c,omitempty"`
	LifeUsedPercent       *float64 `json:"life_used_percent,omitempty"`
	AvailableSparePercent *float64 `json:"available_spare_percent,omitempty"`
	PowerOnHours          uint64   `json:"power_on_hours,omitempty"`
	UnsafeShutdowns       uint64   `json:"unsafe_shutdowns,omitempty"`
	MediaErrors           uint64   `json:"media_errors,omitempty"`
	ReallocatedSectors    uint64   `json:"reallocated_sectors,omitempty"`
	PendingSectors        uint64   `json:"pending_sectors,omitempty"`
	OfflineUncorrectable  uint64   `json:"offline_uncorrectable,omitempty"`
	InterfaceCRCErrors    uint64   `json:"interface_crc_errors,omitempty"`
	CriticalWarning       uint8    `json:"critical_warning,omitempty"`
	Message               string   `json:"message,omitempty"`
}

type DiskIOStats struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
}

type NetStats struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type Process struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu"`
	MemRSS     uint64  `json:"mem"`
}

type Service struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type LatencyMonitorType string

const (
	LatencyMonitorTypeICMP LatencyMonitorType = "icmp"
	LatencyMonitorTypeTCP  LatencyMonitorType = "tcp"
	LatencyMonitorTypeHTTP LatencyMonitorType = "http"
)

type LatencyMonitor struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Type               LatencyMonitorType `json:"type"`
	Target             string             `json:"target"`
	IntervalSeconds    int                `json:"interval_seconds"`
	AutoAssignNewNodes bool               `json:"auto_assign_new_nodes"`
	AssignedNodeCount  int                `json:"assigned_node_count,omitempty"`
	AssignedNodeIDs    []string           `json:"assigned_node_ids,omitempty"`
	CreatedAt          int64              `json:"created_at"`
	UpdatedAt          int64              `json:"updated_at"`
}

type LatencyMonitorResult struct {
	MonitorID    string   `json:"monitor_id"`
	NodeID       string   `json:"node_id"`
	TS           int64    `json:"ts"`
	LatencyMs    *float64 `json:"latency_ms"`
	LossPercent  *float64 `json:"loss_percent,omitempty"`
	JitterMs     *float64 `json:"jitter_ms,omitempty"`
	Success      bool     `json:"success"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

type LatencyMonitorConfigPayload struct {
	Monitors []LatencyMonitor `json:"monitors"`
}

// DockerContainer represents a running or stopped Docker container.
type DockerContainer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Status string `json:"status"`
}

type AgentCommandKind string

const (
	AgentCommandKindSelfUpdate AgentCommandKind = "self_update"
)

type UpdateJobStatus string

const (
	UpdateJobStatusPending       UpdateJobStatus = "pending"
	UpdateJobStatusRunning       UpdateJobStatus = "running"
	UpdateJobStatusCompleted     UpdateJobStatus = "completed"
	UpdateJobStatusPartialFailed UpdateJobStatus = "partial_failed"
	UpdateJobStatusFailed        UpdateJobStatus = "failed"
)

type UpdateJobTargetStatus string

const (
	UpdateJobTargetStatusPending        UpdateJobTargetStatus = "pending"
	UpdateJobTargetStatusDispatched     UpdateJobTargetStatus = "dispatched"
	UpdateJobTargetStatusAccepted       UpdateJobTargetStatus = "accepted"
	UpdateJobTargetStatusDownloading    UpdateJobTargetStatus = "downloading"
	UpdateJobTargetStatusVerifying      UpdateJobTargetStatus = "verifying"
	UpdateJobTargetStatusRestarting     UpdateJobTargetStatus = "restarting"
	UpdateJobTargetStatusSucceeded      UpdateJobTargetStatus = "succeeded"
	UpdateJobTargetStatusFailed         UpdateJobTargetStatus = "failed"
	UpdateJobTargetStatusTimeout        UpdateJobTargetStatus = "timeout"
	UpdateJobTargetStatusOfflineSkipped UpdateJobTargetStatus = "offline_skipped"
)

type AgentCommand struct {
	JobID         string           `json:"job_id"`
	Kind          AgentCommandKind `json:"kind"`
	TargetVersion string           `json:"target_version"`
	DownloadURL   string           `json:"download_url"`
	SHA256        string           `json:"sha256"`
	Signature     string           `json:"signature,omitempty"`
}

type AgentCommandStatus struct {
	JobID           string                `json:"job_id"`
	Status          UpdateJobTargetStatus `json:"status"`
	Message         string                `json:"message,omitempty"`
	ReportedVersion string                `json:"reported_version,omitempty"`
}

type UpdateJob struct {
	ID            string           `json:"id"`
	Kind          AgentCommandKind `json:"kind"`
	TargetVersion string           `json:"target_version"`
	DownloadURL   string           `json:"download_url"`
	SHA256        string           `json:"sha256"`
	Signature     string           `json:"signature,omitempty"`
	CreatedAt     int64            `json:"created_at"`
	UpdatedAt     int64            `json:"updated_at"`
	CreatedBy     string           `json:"created_by"`
	Status        UpdateJobStatus  `json:"status"`
}

type UpdateJobTarget struct {
	JobID           string                `json:"job_id"`
	NodeID          string                `json:"node_id"`
	Status          UpdateJobTargetStatus `json:"status"`
	Message         string                `json:"message,omitempty"`
	UpdatedAt       int64                 `json:"updated_at"`
	ReportedVersion string                `json:"reported_version,omitempty"`
}

type AgentCommandPayload struct {
	JobID         string           `json:"job_id"`
	Kind          AgentCommandKind `json:"kind"`
	TargetVersion string           `json:"target_version"`
	DownloadURL   string           `json:"download_url"`
	SHA256        string           `json:"sha256"`
	Signature     string           `json:"signature,omitempty"`
}

type AgentCommandStatusPayload struct {
	JobID           string                `json:"job_id"`
	Status          UpdateJobTargetStatus `json:"status"`
	Message         string                `json:"message,omitempty"`
	ReportedVersion string                `json:"reported_version,omitempty"`
}

// WSMessage wraps any WebSocket message with a type discriminator.
type WSMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}
