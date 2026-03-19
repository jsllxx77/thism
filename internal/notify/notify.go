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

	message := formatTelegramMessage(event)
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
	metricLabel := map[models.ResourceMetric]string{
		models.ResourceMetricCPU:        "CPU",
		models.ResourceMetricMemory:     "Memory",
		models.ResourceMetricDisk:       "Disk",
		models.ResourceMetricNodeStatus: "Node status",
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
	timestamp := escapeTelegramMarkdown(time.Unix(event.ObservedAt, 0).UTC().Format(time.RFC3339))
	if event.Metric == models.ResourceMetricNodeStatus {
		status := "offline"
		emoji := "🔴"
		if event.Value > 0 {
			status = "online"
			emoji = "🟢"
		}
		return fmt.Sprintf(
			"%s *Node %s*\nNode: *%s*\nTime: `%s`",
			emoji,
			escapeTelegramMarkdown(status),
			escapeTelegramMarkdown(event.NodeName),
			timestamp,
		)
	}
	usage := escapeTelegramMarkdown(fmt.Sprintf("%.1f%%", event.Value))
	threshold := escapeTelegramMarkdown(fmt.Sprintf("%.1f%%", event.Threshold))
	if event.Severity == models.AlertSeverityResolved {
		return fmt.Sprintf(
			"✅ *%s resource alert*\nNode: *%s*\nMetric: *%s*\nUsage: *%s*\nTime: `%s`",
			escapeTelegramMarkdown(severityLabel),
			escapeTelegramMarkdown(event.NodeName),
			escapeTelegramMarkdown(metricLabel),
			usage,
			timestamp,
		)
	}
	return fmt.Sprintf(
		"🚨 *%s resource alert*\nNode: *%s*\nMetric: *%s*\nUsage: *%s*\nThreshold: *%s*\nTime: `%s`",
		escapeTelegramMarkdown(severityLabel),
		escapeTelegramMarkdown(event.NodeName),
		escapeTelegramMarkdown(metricLabel),
		usage,
		threshold,
		timestamp,
	)
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
