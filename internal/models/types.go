package models

// Node represents a registered monitored server.
type Node struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Token         string               `json:"-"` // never serialized to API responses
	IP            string               `json:"ip"`
	OS            string               `json:"os"`
	Arch          string               `json:"arch"`
	AgentVersion  string               `json:"agent_version,omitempty"`
	CreatedAt     int64                `json:"created_at"`
	LastSeen      int64                `json:"last_seen"`
	Online        bool                 `json:"online"`
	Hardware      *NodeHardware        `json:"hardware,omitempty"`
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
	TS        int64   `json:"ts"`
	CPU       float64 `json:"cpu"`
	MemUsed   uint64  `json:"mem_used"`
	MemTotal  uint64  `json:"mem_total"`
	DiskUsed  uint64  `json:"disk_used"`
	DiskTotal uint64  `json:"disk_total"`
	NetRx     uint64  `json:"net_rx"`
	NetTx     uint64  `json:"net_tx"`
}

// MetricsPayload is the JSON message sent by agents over WebSocket.
type MetricsPayload struct {
	Type         string        `json:"type"`
	TS           int64         `json:"ts"`
	CPU          float64       `json:"cpu"`
	IP           string        `json:"ip,omitempty"`
	OS           string        `json:"os,omitempty"`
	Arch         string        `json:"arch,omitempty"`
	AgentVersion string        `json:"agent_version,omitempty"`
	Hardware     *NodeHardware `json:"hardware,omitempty"`
	Mem          MemStats      `json:"mem"`
	Disk         []DiskStats   `json:"disk"`
	Net          NetStats      `json:"net"`
	Processes    []Process     `json:"processes"`
	Services     []Service     `json:"services"`
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
