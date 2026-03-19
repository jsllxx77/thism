package notify

import (
	"io"
	"net/http"
	"strings"
	"testing"

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
