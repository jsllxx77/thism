package alerting

import (
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

type senderStub struct{ events []models.AlertEvent }

func (s *senderStub) Send(_ models.NotificationSettings, event models.AlertEvent) error {
	s.events = append(s.events, event)
	return nil
}

func testNotificationSettings() models.NotificationSettings {
	return models.NotificationSettings{
		Enabled:                true,
		Channel:                string(models.NotificationChannelTelegram),
		TelegramBotToken:       "token",
		TelegramTargets:        []models.TelegramTarget{{ChatID: "-1001", TopicID: 22}},
		CPUWarningPercent:      80,
		CPUCriticalPercent:     90,
		MemWarningPercent:      80,
		MemCriticalPercent:     90,
		DiskWarningPercent:     80,
		DiskCriticalPercent:    90,
		CooldownMinutes:        30,
		NotifyNodeOffline:      true,
		NotifyNodeOnline:       true,
		NodeOfflineGraceMinutes: 2,
	}
}

func TestEvaluatorSendsCriticalAndRespectsCooldown(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if err := s.UpsertNotificationSettings(testNotificationSettings()); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}
	stub := &senderStub{}
	evaluator := &Evaluator{Store: s, Sender: stub}
	node := &models.Node{ID: "node-1", Name: "alpha"}
	metrics := &models.MetricsPayload{TS: time.Now().Unix(), CPU: 95, Mem: models.MemStats{Used: 91, Total: 100}, Disk: []models.DiskStats{{Used: 85, Total: 100}}}

	if err := evaluator.Process(node, metrics); err != nil {
		t.Fatalf("Process first: %v", err)
	}
	if len(stub.events) != 3 {
		t.Fatalf("expected 3 events on first pass, got %d", len(stub.events))
	}
	if err := evaluator.Process(node, metrics); err != nil {
		t.Fatalf("Process second: %v", err)
	}
	if len(stub.events) != 3 {
		t.Fatalf("expected cooldown to suppress duplicate alerts, got %d", len(stub.events))
	}
}

func TestEvaluatorSendsResolvedWhenMetricRecovers(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if err := s.UpsertNotificationSettings(testNotificationSettings()); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}
	stub := &senderStub{}
	evaluator := &Evaluator{Store: s, Sender: stub}
	node := &models.Node{ID: "node-1", Name: "alpha"}

	if err := evaluator.Process(node, &models.MetricsPayload{TS: 1000, CPU: 95, Mem: models.MemStats{Used: 50, Total: 100}, Disk: []models.DiskStats{{Used: 50, Total: 100}}}); err != nil {
		t.Fatalf("Process alert: %v", err)
	}
	if len(stub.events) != 1 || stub.events[0].Severity != models.AlertSeverityCritical {
		t.Fatalf("expected critical cpu alert, got %#v", stub.events)
	}

	if err := evaluator.Process(node, &models.MetricsPayload{TS: 1100, CPU: 20, Mem: models.MemStats{Used: 50, Total: 100}, Disk: []models.DiskStats{{Used: 50, Total: 100}}}); err != nil {
		t.Fatalf("Process recovery: %v", err)
	}
	if len(stub.events) != 2 {
		t.Fatalf("expected resolved event after recovery, got %d", len(stub.events))
	}
	if stub.events[1].Severity != models.AlertSeverityResolved || stub.events[1].Metric != models.ResourceMetricCPU {
		t.Fatalf("expected cpu resolved event, got %#v", stub.events[1])
	}

	active, err := s.HasActiveAlertDelivery("node-1", "cpu")
	if err != nil {
		t.Fatalf("HasActiveAlertDelivery: %v", err)
	}
	if active {
		t.Fatal("expected active cpu alert to be cleared after recovery")
	}
}

func TestEvaluatorCooldownAppliesAcrossSeverityChanges(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if err := s.UpsertNotificationSettings(testNotificationSettings()); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}
	stub := &senderStub{}
	evaluator := &Evaluator{Store: s, Sender: stub}
	node := &models.Node{ID: "node-1", Name: "alpha"}

	if err := evaluator.Process(node, &models.MetricsPayload{TS: 1000, CPU: 95, Mem: models.MemStats{Used: 10, Total: 100}, Disk: []models.DiskStats{{Used: 10, Total: 100}}}); err != nil {
		t.Fatalf("Process critical: %v", err)
	}
	if err := evaluator.Process(node, &models.MetricsPayload{TS: 1100, CPU: 85, Mem: models.MemStats{Used: 10, Total: 100}, Disk: []models.DiskStats{{Used: 10, Total: 100}}}); err != nil {
		t.Fatalf("Process warning: %v", err)
	}
	if len(stub.events) != 1 {
		t.Fatalf("expected warning to be suppressed during cooldown, got %#v", stub.events)
	}
}

func TestEvaluatorSendsOfflineAndOnlineNotifications(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()
	if err := s.UpsertNotificationSettings(testNotificationSettings()); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}
	stub := &senderStub{}
	evaluator := &Evaluator{Store: s, Sender: stub}
	node := &models.Node{ID: "node-1", Name: "alpha"}

	if err := evaluator.ProcessHeartbeat(node, true, 1000); err != nil {
		t.Fatalf("ProcessHeartbeat initial: %v", err)
	}
	if err := evaluator.ProcessHeartbeat(node, false, 1100); err != nil {
		t.Fatalf("ProcessHeartbeat offline: %v", err)
	}
	if err := evaluator.ProcessHeartbeat(node, true, 1200); err != nil {
		t.Fatalf("ProcessHeartbeat online: %v", err)
	}
	if len(stub.events) != 2 {
		t.Fatalf("expected offline and online notifications, got %#v", stub.events)
	}
	if stub.events[0].Metric != models.ResourceMetricNodeStatus || stub.events[0].Value != 0 {
		t.Fatalf("expected offline node status event, got %#v", stub.events[0])
	}
	if stub.events[1].Metric != models.ResourceMetricNodeStatus || stub.events[1].Value != 1 {
		t.Fatalf("expected online node status event, got %#v", stub.events[1])
	}
}
