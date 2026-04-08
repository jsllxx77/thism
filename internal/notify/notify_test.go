package notify

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestTelegramSenderIncludesTopicID(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		text := string(body)
		if !strings.Contains(req.URL.String(), "/sendMessage") {
			t.Fatalf("expected sendMessage endpoint, got %s", req.URL.String())
		}
		if !strings.Contains(text, "\"message_thread_id\":22") {
			t.Fatalf("expected topic id in payload, got %s", text)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	sender := NewTelegramSender(client)
	err := sender.Send(models.NotificationSettings{TelegramBotToken: "token", TelegramTargets: []models.TelegramTarget{{ChatID: "-1001", TopicID: 22}}}, models.AlertEvent{NodeName: "alpha", Metric: models.ResourceMetricCPU, Severity: models.AlertSeverityCritical, Value: 95, Threshold: 90, ObservedAt: 1710000000})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestFormatTelegramMessageForNodeStatus(t *testing.T) {
	message := formatTelegramMessage(models.AlertEvent{NodeName: "alpha", Metric: models.ResourceMetricNodeStatus, Severity: models.AlertSeverityInfo, Value: 0, ObservedAt: 1710000000})
	if !strings.Contains(message, "Node offline") {
		t.Fatalf("expected offline node status message, got %s", message)
	}
	message = formatTelegramMessage(models.AlertEvent{NodeName: "alpha", Metric: models.ResourceMetricNodeStatus, Severity: models.AlertSeverityInfo, Value: 1, ObservedAt: 1710000000})
	if !strings.Contains(message, "Node online") {
		t.Fatalf("expected online node status message, got %s", message)
	}
}

func TestResolveNotificationLocationPrefersCustomTimezone(t *testing.T) {
	systemLocation := time.FixedZone("SYSTEM", 9*60*60)
	settings := models.NotificationSettings{
		TimeZoneMode: models.NotificationTimeZoneModeCustom,
		TimeZone:     "America/New_York",
	}

	location := resolveNotificationLocation(settings, systemLocation)
	if got := location.String(); got != "America/New_York" {
		t.Fatalf("expected custom timezone, got %s", got)
	}
}

func TestFormatTelegramMessageInLocationUsesProvidedTimezone(t *testing.T) {
	location := time.FixedZone("SYSTEM", 9*60*60)
	observedAt := time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC).Unix()

	message := formatTelegramMessageInLocation(models.AlertEvent{
		NodeName:   "alpha",
		Metric:     models.ResourceMetricCPU,
		Severity:   models.AlertSeverityCritical,
		Value:      95,
		Threshold:  90,
		ObservedAt: observedAt,
	}, location)

	expectedTimestamp := escapeTelegramMarkdown(time.Unix(observedAt, 0).In(location).Format(time.RFC3339))
	if !strings.Contains(message, expectedTimestamp) {
		t.Fatalf("expected timestamp %s in message, got %s", expectedTimestamp, message)
	}
}
