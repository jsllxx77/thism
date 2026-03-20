package models

import "strings"

type NotificationChannel string

type AlertSeverity string

type ResourceMetric string

const (
	NotificationChannelTelegram NotificationChannel = "telegram"
)

const (
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityResolved AlertSeverity = "resolved"
	AlertSeverityInfo     AlertSeverity = "info"
)

const (
	ResourceMetricCPU        ResourceMetric = "cpu"
	ResourceMetricMemory     ResourceMetric = "memory"
	ResourceMetricDisk       ResourceMetric = "disk"
	ResourceMetricNodeStatus ResourceMetric = "node_status"
)

type TelegramTarget struct {
	Name    string `json:"name,omitempty"`
	ChatID  string `json:"chat_id"`
	TopicID int64  `json:"topic_id,omitempty"`
}

func (t TelegramTarget) Normalized() TelegramTarget {
	return TelegramTarget{
		Name:    strings.TrimSpace(t.Name),
		ChatID:  strings.TrimSpace(t.ChatID),
		TopicID: t.TopicID,
	}
}

type NotificationSettings struct {
	Enabled                             bool             `json:"enabled"`
	Channel                             string           `json:"channel"`
	TelegramBotToken                    string           `json:"telegram_bot_token,omitempty"`
	TelegramTargets                     []TelegramTarget `json:"telegram_targets,omitempty"`
	EnabledNodeIDs                      []string         `json:"enabled_node_ids,omitempty"`
	CPUWarningPercent                   float64          `json:"cpu_warning_percent"`
	CPUCriticalPercent                  float64          `json:"cpu_critical_percent"`
	MemWarningPercent                   float64          `json:"mem_warning_percent"`
	MemCriticalPercent                  float64          `json:"mem_critical_percent"`
	DiskWarningPercent                  float64          `json:"disk_warning_percent"`
	DiskCriticalPercent                 float64          `json:"disk_critical_percent"`
	CooldownMinutes                     int              `json:"cooldown_minutes"`
	RecoverySuccessiveSamples           int              `json:"recovery_successive_samples"`
	RecoveryNotificationCooldownMinutes int              `json:"recovery_notification_cooldown_minutes"`
	NotifyNodeOffline                   bool             `json:"notify_node_offline"`
	NotifyNodeOnline                    bool             `json:"notify_node_online"`
	NodeOfflineGraceMinutes             int              `json:"node_offline_grace_minutes"`
}

type NotificationSettingsView struct {
	Enabled                             bool             `json:"enabled"`
	Channel                             string           `json:"channel"`
	TelegramBotTokenSet                 bool             `json:"telegram_bot_token_set"`
	TelegramBotToken                    string           `json:"telegram_bot_token,omitempty"`
	TelegramTargets                     []TelegramTarget `json:"telegram_targets,omitempty"`
	EnabledNodeIDs                      []string         `json:"enabled_node_ids,omitempty"`
	CPUWarningPercent                   float64          `json:"cpu_warning_percent"`
	CPUCriticalPercent                  float64          `json:"cpu_critical_percent"`
	MemWarningPercent                   float64          `json:"mem_warning_percent"`
	MemCriticalPercent                  float64          `json:"mem_critical_percent"`
	DiskWarningPercent                  float64          `json:"disk_warning_percent"`
	DiskCriticalPercent                 float64          `json:"disk_critical_percent"`
	CooldownMinutes                     int              `json:"cooldown_minutes"`
	RecoverySuccessiveSamples           int              `json:"recovery_successive_samples"`
	RecoveryNotificationCooldownMinutes int              `json:"recovery_notification_cooldown_minutes"`
	NotifyNodeOffline                   bool             `json:"notify_node_offline"`
	NotifyNodeOnline                    bool             `json:"notify_node_online"`
	NodeOfflineGraceMinutes             int              `json:"node_offline_grace_minutes"`
}

type AlertEvent struct {
	NodeID     string         `json:"node_id"`
	NodeName   string         `json:"node_name"`
	Metric     ResourceMetric `json:"metric"`
	Severity   AlertSeverity  `json:"severity"`
	Value      float64        `json:"value"`
	Threshold  float64        `json:"threshold"`
	ObservedAt int64          `json:"observed_at"`
}
