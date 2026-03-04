package models

// Node represents a registered monitored server.
type Node struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	IP        string `json:"ip"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CreatedAt int64  `json:"created_at"`
	LastSeen  int64  `json:"last_seen"`
	Online    bool   `json:"online"`
}

// MetricsPayload is the JSON message sent by agents over WebSocket.
type MetricsPayload struct {
	Type      string      `json:"type"`
	TS        int64       `json:"ts"`
	CPU       float64     `json:"cpu"`
	Mem       MemStats    `json:"mem"`
	Disk      []DiskStats `json:"disk"`
	Net       NetStats    `json:"net"`
	Processes []Process   `json:"processes"`
	Services  []Service   `json:"services"`
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

// WSMessage wraps any WebSocket message with a type discriminator.
type WSMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}
