package alerting

import (
	"fmt"
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

type blockingSender struct {
	started chan struct{}
	release chan struct{}
	events  chan models.AlertEvent
}

func (s *blockingSender) Send(_ models.NotificationSettings, event models.AlertEvent) error {
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-s.release
	s.events <- event
	return nil
}

type dropAwareSender struct {
	started chan struct{}
	release chan struct{}
	events  chan models.AlertEvent
}

func (s *dropAwareSender) Send(_ models.NotificationSettings, event models.AlertEvent) error {
	if event.Metric == models.ResourceMetricDispatcherQueue {
		s.events <- event
		return nil
	}
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-s.release
	s.events <- event
	return nil
}

func TestDispatcherEnqueueMetricsDoesNotBlockOnSlowSender(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()

	if err := st.UpsertNotificationSettings(testNotificationSettings()); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}

	sender := &blockingSender{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		events:  make(chan models.AlertEvent, 4),
	}
	dispatcher := NewDispatcher(st, sender)
	defer dispatcher.Close()

	node := &models.Node{ID: "node-1", Name: "alpha"}
	metrics := &models.MetricsPayload{
		TS:   1000,
		CPU:  95,
		Mem:  models.MemStats{Used: 10, Total: 100},
		Disk: []models.DiskStats{{Used: 10, Total: 100}},
	}

	returned := make(chan struct{})
	go func() {
		dispatcher.EnqueueMetrics(node, metrics)
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected enqueue to return before sender completes")
	}

	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("expected dispatcher worker to start processing queued metric")
	}

	close(sender.release)

	select {
	case event := <-sender.events:
		if event.NodeID != "node-1" || event.Metric != models.ResourceMetricCPU {
			t.Fatalf("unexpected alert event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected queued alert to reach sender after release")
	}
}

func TestDispatcherDropsWhenQueueIsFullAndTracksStats(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()

	if err := st.UpsertNotificationSettings(testNotificationSettings()); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}

	sender := &blockingSender{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		events:  make(chan models.AlertEvent, 4),
	}
	dispatcher := NewDispatcherWithCapacity(st, sender, 1)

	node := &models.Node{ID: "node-1", Name: "alpha"}
	metrics := &models.MetricsPayload{
		TS:   1000,
		CPU:  95,
		Mem:  models.MemStats{Used: 10, Total: 100},
		Disk: []models.DiskStats{{Used: 10, Total: 100}},
	}

	if ok := dispatcher.EnqueueMetrics(node, metrics); !ok {
		t.Fatal("expected first enqueue to succeed")
	}
	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("expected worker to begin first send")
	}

	if ok := dispatcher.EnqueueHeartbeat(node, false, 1001); !ok {
		t.Fatal("expected second enqueue to fill queue successfully")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, true, 1002); ok {
		t.Fatal("expected enqueue to fail when bounded queue is full")
	}

	stats := dispatcher.Stats()
	if stats.Capacity != 1 {
		t.Fatalf("expected capacity 1, got %d", stats.Capacity)
	}
	if stats.QueueDepth != 1 {
		t.Fatalf("expected queued depth 1 while worker blocked, got %d", stats.QueueDepth)
	}
	if stats.Enqueued != 2 {
		t.Fatalf("expected 2 accepted enqueues, got %d", stats.Enqueued)
	}
	if stats.Dropped != 1 {
		t.Fatalf("expected 1 dropped enqueue, got %d", stats.Dropped)
	}
	if stats.HighWatermark != 1 {
		t.Fatalf("expected high watermark 1, got %d", stats.HighWatermark)
	}

	close(sender.release)
	dispatcher.Close()
}

func TestDispatcherRateLimitsDroppedQueueLogs(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()

	if err := st.UpsertNotificationSettings(testNotificationSettings()); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}

	sender := &blockingSender{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		events:  make(chan models.AlertEvent, 4),
	}
	dispatcher := NewDispatcherWithCapacity(st, sender, 1)

	currentTime := time.Unix(1000, 0)
	dispatcher.now = func() time.Time { return currentTime }
	dispatcher.dropLogInterval = time.Minute
	var logs []string
	dispatcher.logf = func(format string, args ...any) {
		logs = append(logs, formatMessage(format, args...))
	}

	node := &models.Node{ID: "node-1", Name: "alpha"}
	metrics := &models.MetricsPayload{
		TS:   1000,
		CPU:  95,
		Mem:  models.MemStats{Used: 10, Total: 100},
		Disk: []models.DiskStats{{Used: 10, Total: 100}},
	}

	if ok := dispatcher.EnqueueMetrics(node, metrics); !ok {
		t.Fatal("expected first enqueue to succeed")
	}
	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("expected worker to begin first send")
	}

	if ok := dispatcher.EnqueueHeartbeat(node, false, 1001); !ok {
		t.Fatal("expected second enqueue to fill queue")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, true, 1002); ok {
		t.Fatal("expected third enqueue to drop")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, true, 1003); ok {
		t.Fatal("expected fourth enqueue to drop")
	}

	if len(logs) != 1 {
		t.Fatalf("expected a single rate-limited log entry, got %d: %#v", len(logs), logs)
	}
	if logs[0] != "alert dispatcher: dropping queued jobs due to full queue (dropped_since_last_log=1 total_dropped=1 queue_depth=1 capacity=1)" {
		t.Fatalf("unexpected first drop log: %q", logs[0])
	}

	currentTime = currentTime.Add(time.Minute)
	if ok := dispatcher.EnqueueHeartbeat(node, true, 1004); ok {
		t.Fatal("expected enqueue after interval to still drop while queue is full")
	}
	if len(logs) != 2 {
		t.Fatalf("expected a second log entry after log interval, got %d: %#v", len(logs), logs)
	}
	if logs[1] != "alert dispatcher: dropping queued jobs due to full queue (dropped_since_last_log=2 total_dropped=3 queue_depth=1 capacity=1)" {
		t.Fatalf("unexpected second drop log: %q", logs[1])
	}

	close(sender.release)
	dispatcher.Close()
}

func formatMessage(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func TestDispatcherRuntimeStatsSnapshotTracksAggregateDeltas(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()

	if err := st.UpsertNotificationSettings(testNotificationSettings()); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}

	baseline := DispatcherRuntimeStatsSnapshot()

	sender := &blockingSender{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		events:  make(chan models.AlertEvent, 4),
	}
	dispatcher := NewDispatcherWithCapacity(st, sender, 1)

	node := &models.Node{ID: "node-1", Name: "alpha"}
	metrics := &models.MetricsPayload{
		TS:   1000,
		CPU:  95,
		Mem:  models.MemStats{Used: 10, Total: 100},
		Disk: []models.DiskStats{{Used: 10, Total: 100}},
	}

	if ok := dispatcher.EnqueueMetrics(node, metrics); !ok {
		t.Fatal("expected first enqueue to succeed")
	}
	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("expected worker to begin first send")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, false, 1001); !ok {
		t.Fatal("expected second enqueue to succeed")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, true, 1002); ok {
		t.Fatal("expected third enqueue to be dropped")
	}

	stats := DispatcherRuntimeStatsSnapshot()
	if stats.ActiveDispatchers < baseline.ActiveDispatchers+1 {
		t.Fatalf("expected active dispatchers to increase, baseline=%+v current=%+v", baseline, stats)
	}
	if stats.TotalCapacity < baseline.TotalCapacity+1 {
		t.Fatalf("expected total capacity to increase, baseline=%+v current=%+v", baseline, stats)
	}
	if stats.QueueDepth < baseline.QueueDepth+1 {
		t.Fatalf("expected aggregate queue depth to increase, baseline=%+v current=%+v", baseline, stats)
	}
	if stats.Enqueued < baseline.Enqueued+2 {
		t.Fatalf("expected aggregate enqueued count to increase by 2, baseline=%+v current=%+v", baseline, stats)
	}
	if stats.Dropped < baseline.Dropped+1 {
		t.Fatalf("expected aggregate dropped count to increase by 1, baseline=%+v current=%+v", baseline, stats)
	}

	close(sender.release)
	dispatcher.Close()
}

func TestDispatcherRefreshesQueueCapacityFromNotificationSettings(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()

	settings := testNotificationSettings()
	settings.DispatcherQueueCapacity = 1
	if err := st.UpsertNotificationSettings(settings); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}

	sender := &blockingSender{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		events:  make(chan models.AlertEvent, 4),
	}
	dispatcher := NewDispatcherWithCapacity(st, sender, 1)

	node := &models.Node{ID: "node-1", Name: "alpha"}
	metrics := &models.MetricsPayload{
		TS:   1000,
		CPU:  95,
		Mem:  models.MemStats{Used: 10, Total: 100},
		Disk: []models.DiskStats{{Used: 10, Total: 100}},
	}

	if ok := dispatcher.EnqueueMetrics(node, metrics); !ok {
		t.Fatal("expected first enqueue to succeed")
	}
	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("expected worker to begin first send")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, false, 1001); !ok {
		t.Fatal("expected second enqueue to fill queue successfully")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, true, 1002); ok {
		t.Fatal("expected third enqueue to be dropped before capacity update")
	}

	settings.DispatcherQueueCapacity = 2
	if err := st.UpsertNotificationSettings(settings); err != nil {
		t.Fatalf("UpsertNotificationSettings update: %v", err)
	}
	if ok := dispatcher.EnqueueHeartbeat(node, true, 1003); !ok {
		t.Fatal("expected enqueue to succeed after runtime capacity update")
	}

	stats := dispatcher.Stats()
	if stats.Capacity != 2 {
		t.Fatalf("expected dispatcher capacity 2 after settings refresh, got %d", stats.Capacity)
	}
	if stats.QueueDepth != 2 {
		t.Fatalf("expected queue depth 2 after capacity refresh, got %d", stats.QueueDepth)
	}

	close(sender.release)
	dispatcher.Close()
}

func TestDispatcherSendsDropAlertWhenConfigured(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()

	settings := testNotificationSettings()
	settings.DispatcherQueueCapacity = 1
	settings.NotifyDispatcherDrops = true
	if err := st.UpsertNotificationSettings(settings); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}

	sender := &dropAwareSender{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		events:  make(chan models.AlertEvent, 4),
	}
	dispatcher := NewDispatcherWithCapacity(st, sender, 1)

	node := &models.Node{ID: "node-1", Name: "alpha"}
	metrics := &models.MetricsPayload{
		TS:   1000,
		CPU:  95,
		Mem:  models.MemStats{Used: 10, Total: 100},
		Disk: []models.DiskStats{{Used: 10, Total: 100}},
	}

	if ok := dispatcher.EnqueueMetrics(node, metrics); !ok {
		t.Fatal("expected first enqueue to succeed")
	}
	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("expected worker to begin first send")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, false, 1001); !ok {
		t.Fatal("expected second enqueue to fill queue successfully")
	}
	if ok := dispatcher.EnqueueHeartbeat(node, true, 1002); ok {
		t.Fatal("expected third enqueue to be dropped")
	}

	select {
	case event := <-sender.events:
		if event.Metric != models.ResourceMetricDispatcherQueue {
			t.Fatalf("expected dispatcher queue alert, got %#v", event)
		}
		if event.Severity != models.AlertSeverityWarning {
			t.Fatalf("expected warning severity for dispatcher drop alert, got %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected dispatcher drop alert to be emitted")
	}

	close(sender.release)
	dispatcher.Close()
}
