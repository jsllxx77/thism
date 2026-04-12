package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

type TelegramSender struct {
	Client *http.Client
}

func NewTelegramSender(client *http.Client) *TelegramSender {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &TelegramSender{Client: client}
}

func (s *TelegramSender) Send(settings models.NotificationSettings, event models.AlertEvent) error {
	if strings.TrimSpace(settings.TelegramBotToken) == "" {
		return fmt.Errorf("telegram bot token is required")
	}
	if len(settings.TelegramTargets) == 0 {
		return fmt.Errorf("at least one telegram target is required")
	}

	message := formatTelegramMessageInLocation(event, resolveNotificationLocation(settings, time.Now().Location()))
	for _, rawTarget := range settings.TelegramTargets {
		target := rawTarget.Normalized()
		if target.ChatID == "" {
			continue
		}
		if err := s.sendTelegramMessage(settings.TelegramBotToken, target, message); err != nil {
			return err
		}
	}
	return nil
}

func formatTelegramMessage(event models.AlertEvent) string {
	return formatTelegramMessageInLocation(event, time.UTC)
}

func formatTelegramMessageInLocation(event models.AlertEvent, location *time.Location) string {
	location = normalizeSystemLocation(location)
	metricLabel := map[models.ResourceMetric]string{
		models.ResourceMetricCPU:             "CPU",
		models.ResourceMetricMemory:          "Memory",
		models.ResourceMetricDisk:            "Disk",
		models.ResourceMetricNodeStatus:      "Node status",
		models.ResourceMetricDispatcherQueue: "Dispatcher queue",
	}[event.Metric]
	severityLabel := map[models.AlertSeverity]string{
		models.AlertSeverityWarning:  "Warning",
		models.AlertSeverityCritical: "Critical",
		models.AlertSeverityResolved: "Resolved",
		models.AlertSeverityInfo:     "Info",
	}[event.Severity]
	if metricLabel == "" {
		metricLabel = string(event.Metric)
	}
	if severityLabel == "" {
		severityLabel = string(event.Severity)
	}
	timestampBlock := formatTelegramTimestampBlock(time.Unix(event.ObservedAt, 0).In(location))
	if event.Metric == models.ResourceMetricNodeStatus {
		status := "offline"
		emoji := "🔴"
		if event.Value > 0 {
			status = "online"
			emoji = "🟢"
		}
		return fmt.Sprintf(
			"%s *Node %s*\nNode: *%s*\n%s",
			emoji,
			escapeTelegramMarkdown(status),
			escapeTelegramMarkdown(event.NodeName),
			timestampBlock,
		)
	}
	if event.Metric == models.ResourceMetricDispatcherQueue {
		details := strings.TrimSpace(event.Details)
		if details == "" {
			details = fmt.Sprintf("Dropped jobs: %.0f\nQueue capacity: %.0f", event.Value, event.Threshold)
		}
		return fmt.Sprintf(
			"🚨 *%s dispatcher alert*\nComponent: *%s*\n%s\n%s",
			escapeTelegramMarkdown(severityLabel),
			escapeTelegramMarkdown(event.NodeName),
			escapeTelegramMarkdown(details),
			timestampBlock,
		)
	}
	usage := escapeTelegramMarkdown(fmt.Sprintf("%.1f%%", event.Value))
	threshold := escapeTelegramMarkdown(fmt.Sprintf("%.1f%%", event.Threshold))
	if event.Severity == models.AlertSeverityResolved {
		return fmt.Sprintf(
			"✅ *%s resource alert*\nNode: *%s*\nMetric: *%s*\nUsage: *%s*\n%s",
			escapeTelegramMarkdown(severityLabel),
			escapeTelegramMarkdown(event.NodeName),
			escapeTelegramMarkdown(metricLabel),
			usage,
			timestampBlock,
		)
	}
	return fmt.Sprintf(
		"🚨 *%s resource alert*\nNode: *%s*\nMetric: *%s*\nUsage: *%s*\nThreshold: *%s*\n%s",
		escapeTelegramMarkdown(severityLabel),
		escapeTelegramMarkdown(event.NodeName),
		escapeTelegramMarkdown(metricLabel),
		usage,
		threshold,
		timestampBlock,
	)
}

func formatTelegramTimestampBlock(observedAt time.Time) string {
	dateLabel := escapeTelegramMarkdown(observedAt.Format("2006-01-02"))
	clockLabel := escapeTelegramMarkdown(observedAt.Format("15:04:05"))
	zoneLabel := escapeTelegramMarkdown(formatUTCOffset(observedAt))
	return fmt.Sprintf("Time:\n• Date: *%s*\n• Clock: *%s*\n• Zone: *%s*", dateLabel, clockLabel, zoneLabel)
}

func formatUTCOffset(observedAt time.Time) string {
	_, offsetSeconds := observedAt.Zone()
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("UTC%s%02d:%02d", sign, hours, minutes)
}

func escapeTelegramMarkdown(value string) string {
	replacer := strings.NewReplacer("_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!")
	return replacer.Replace(value)
}

func (s *TelegramSender) sendTelegramMessage(botToken string, target models.TelegramTarget, text string) error {
	payload := map[string]any{
		"chat_id":                  target.ChatID,
		"text":                     text,
		"parse_mode":               "MarkdownV2",
		"disable_web_page_preview": true,
	}
	if target.TopicID > 0 {
		payload["message_thread_id"] = target.TopicID
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := s.Client.Post("https://api.telegram.org/bot"+botToken+"/sendMessage", "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return fmt.Errorf("telegram send failed: %s", resp.Status)
		}
		return fmt.Errorf("telegram send failed: %s: %s", resp.Status, msg)
	}
	return nil
}
