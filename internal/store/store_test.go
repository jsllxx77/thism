package store_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func TestStoreNodeCRUD(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	node := &models.Node{
		ID:        "node-1",
		Name:      "test-server",
		Token:     "secret-token",
		IP:        "1.2.3.4",
		OS:        "linux",
		Arch:      "amd64",
		CreatedAt: time.Now().Unix(),
		LastSeen:  time.Now().Unix(),
	}

	if err := s.UpsertNode(node); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	got, err := s.GetNodeByToken("secret-token")
	if err != nil {
		t.Fatalf("GetNodeByToken: %v", err)
	}
	if got.Name != "test-server" {
		t.Errorf("expected name test-server, got %s", got.Name)
	}

	nodes, err := s.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
}

func TestStoreMetrics(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	s.UpsertNode(&models.Node{ID: "n1", Token: "t1", Name: "n1"})

	m := &models.MetricsPayload{
		TS:            time.Now().Unix(),
		CPU:           55.0,
		UptimeSeconds: 3723,
		Mem:           models.MemStats{Used: 1024, Total: 4096},
	}
	if err := s.InsertMetrics("n1", m); err != nil {
		t.Fatalf("InsertMetrics: %v", err)
	}

	rows, err := s.QueryMetrics("n1", time.Now().Add(-time.Minute).Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("QueryMetrics: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
	if rows[0].UptimeSeconds != 3723 {
		t.Fatalf("expected uptime 3723, got %d", rows[0].UptimeSeconds)
	}
}

func TestStoreNodeHardwareMetadata(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.UpsertNode(&models.Node{
		ID:    "n1",
		Token: "t1",
		Name:  "node-1",
		Hardware: &models.NodeHardware{
			CPUModel:             "AMD EPYC 7B13",
			CPUCores:             8,
			CPUThreads:           16,
			MemoryTotal:          34359738368,
			DiskTotal:            322122547200,
			VirtualizationSystem: "kvm",
			VirtualizationRole:   "guest",
		},
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	node, err := s.GetNodeByID("n1")
	if err != nil {
		t.Fatalf("GetNodeByID: %v", err)
	}
	if node == nil || node.Hardware == nil {
		t.Fatal("expected node hardware to be persisted")
	}
	if node.Hardware.CPUModel != "AMD EPYC 7B13" {
		t.Fatalf("expected cpu model to round-trip, got %q", node.Hardware.CPUModel)
	}
	if node.Hardware.DiskTotal != 322122547200 {
		t.Fatalf("expected disk total to round-trip, got %d", node.Hardware.DiskTotal)
	}
}

func TestStoreAdminAuthCRUD(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	_, _, found, err := s.GetAdminAuth()
	if err != nil {
		t.Fatalf("GetAdminAuth initial: %v", err)
	}
	if found {
		t.Fatal("expected no persisted admin auth initially")
	}

	if err := s.UpsertAdminAuth("admin", "secret-pass"); err != nil {
		t.Fatalf("UpsertAdminAuth: %v", err)
	}

	username, password, found, err := s.GetAdminAuth()
	if err != nil {
		t.Fatalf("GetAdminAuth after insert: %v", err)
	}
	if !found {
		t.Fatal("expected persisted admin auth after insert")
	}
	if username != "admin" {
		t.Fatalf("unexpected username after insert: %q", username)
	}
	if password == "" {
		t.Fatal("expected persisted password hash after insert")
	}
	if password == "secret-pass" {
		t.Fatalf("unexpected admin auth values: username=%q password=%q", username, password)
	}

	if err := s.UpsertAdminAuth("admin", "new-pass"); err != nil {
		t.Fatalf("UpsertAdminAuth update: %v", err)
	}
	_, password, found, err = s.GetAdminAuth()
	if err != nil {
		t.Fatalf("GetAdminAuth after update: %v", err)
	}
	if !found {
		t.Fatal("expected persisted admin auth after update")
	}
	if password == "" {
		t.Fatal("expected updated persisted password hash")
	}
	if password == "new-pass" {
		t.Fatalf("expected updated password to be hashed, got %q", password)
	}
}

func TestDeleteNodeRemovesAggregatedMetrics(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.UpsertNode(&models.Node{ID: "n1", Token: "t1", Name: "n1"}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	ts := time.Now().Add(-2 * time.Minute).Unix()
	bucketTS := (ts / 60) * 60
	if err := s.InsertMetrics("n1", &models.MetricsPayload{
		TS:  ts,
		CPU: 55,
		Mem: models.MemStats{Used: 1024, Total: 4096},
	}); err != nil {
		t.Fatalf("InsertMetrics: %v", err)
	}
	if err := s.RollupMetrics1m(ts, ts+59); err != nil {
		t.Fatalf("RollupMetrics1m: %v", err)
	}

	rows, err := s.QueryMetrics1m("n1", bucketTS, bucketTS)
	if err != nil {
		t.Fatalf("QueryMetrics1m before delete: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one aggregated row before delete, got %d", len(rows))
	}

	if err := s.DeleteNode("n1"); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}

	rows, err = s.QueryMetrics1m("n1", bucketTS, bucketTS)
	if err != nil {
		t.Fatalf("QueryMetrics1m after delete: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected aggregated metrics to be removed with node delete, got %d rows", len(rows))
	}
}

func TestPruneOldMetricsRemovesAggregatedMetrics(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.UpsertNode(&models.Node{ID: "n1", Token: "t1", Name: "n1"}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	oldTS := time.Now().AddDate(0, 0, -10).Unix()
	bucketTS := (oldTS / 60) * 60
	if err := s.InsertMetrics("n1", &models.MetricsPayload{
		TS:  oldTS,
		CPU: 40,
		Mem: models.MemStats{Used: 512, Total: 4096},
	}); err != nil {
		t.Fatalf("InsertMetrics old: %v", err)
	}
	if err := s.RollupMetrics1m(oldTS, oldTS+59); err != nil {
		t.Fatalf("RollupMetrics1m: %v", err)
	}

	rows, err := s.QueryMetrics1m("n1", bucketTS, bucketTS)
	if err != nil {
		t.Fatalf("QueryMetrics1m before prune: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one aggregated row before prune, got %d", len(rows))
	}

	if err := s.PruneOldMetrics(7); err != nil {
		t.Fatalf("PruneOldMetrics: %v", err)
	}

	rows, err = s.QueryMetrics1m("n1", bucketTS, bucketTS)
	if err != nil {
		t.Fatalf("QueryMetrics1m after prune: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected aggregated metrics older than retention to be pruned, got %d rows", len(rows))
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestStoreUpdateJobLifecycle(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	job := &models.UpdateJob{
		ID:            "job-1",
		Kind:          "self_update",
		TargetVersion: "1.2.3",
		DownloadURL:   "https://example.com/agent",
		SHA256:        "abc123",
		CreatedAt:     time.Now().Unix(),
		CreatedBy:     "admin",
		Status:        models.UpdateJobStatusPending,
	}
	if err := s.CreateUpdateJob(job); err != nil {
		t.Fatalf("CreateUpdateJob: %v", err)
	}
	if err := s.CreateUpdateJobTargets(job.ID, []string{"node-1", "node-2"}); err != nil {
		t.Fatalf("CreateUpdateJobTargets: %v", err)
	}

	storedJob, err := s.GetUpdateJob(job.ID)
	if err != nil {
		t.Fatalf("GetUpdateJob: %v", err)
	}
	if storedJob == nil || storedJob.Status != models.UpdateJobStatusPending {
		t.Fatalf("expected pending job, got %#v", storedJob)
	}

	if err := s.UpdateUpdateJobTargetStatus(job.ID, "node-1", models.UpdateJobTargetStatusAccepted, "accepted", ""); err != nil {
		t.Fatalf("UpdateUpdateJobTargetStatus accepted: %v", err)
	}
	storedJob, err = s.GetUpdateJob(job.ID)
	if err != nil {
		t.Fatalf("GetUpdateJob running: %v", err)
	}
	if storedJob.Status != models.UpdateJobStatusRunning {
		t.Fatalf("expected running job after active target, got %q", storedJob.Status)
	}

	if err := s.UpdateUpdateJobTargetStatus(job.ID, "node-1", models.UpdateJobTargetStatusSucceeded, "", "1.2.3"); err != nil {
		t.Fatalf("UpdateUpdateJobTargetStatus success: %v", err)
	}
	if err := s.UpdateUpdateJobTargetStatus(job.ID, "node-2", models.UpdateJobTargetStatusFailed, "checksum mismatch", ""); err != nil {
		t.Fatalf("UpdateUpdateJobTargetStatus failed: %v", err)
	}
	storedJob, err = s.GetUpdateJob(job.ID)
	if err != nil {
		t.Fatalf("GetUpdateJob partial_failed: %v", err)
	}
	if storedJob.Status != models.UpdateJobStatusPartialFailed {
		t.Fatalf("expected partial_failed, got %q", storedJob.Status)
	}
	targets, err := s.ListUpdateJobTargets(job.ID)
	if err != nil {
		t.Fatalf("ListUpdateJobTargets: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0].ReportedVersion == "" && targets[1].ReportedVersion == "" {
		t.Fatalf("expected one target to store reported version")
	}
}

func TestStoreMetricsRetentionDefaultsToSevenDays(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	days, err := s.GetMetricsRetentionDays()
	if err != nil {
		t.Fatalf("GetMetricsRetentionDays: %v", err)
	}
	if days != 7 {
		t.Fatalf("expected default retention 7 days, got %d", days)
	}
}

func TestStoreMetricsRetentionRoundTrip(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.SetMetricsRetentionDays(30); err != nil {
		t.Fatalf("SetMetricsRetentionDays: %v", err)
	}

	days, err := s.GetMetricsRetentionDays()
	if err != nil {
		t.Fatalf("GetMetricsRetentionDays: %v", err)
	}
	if days != 30 {
		t.Fatalf("expected persisted retention 30 days, got %d", days)
	}

	if err := s.SetMetricsRetentionDays(14); err == nil {
		t.Fatal("expected unsupported retention value to fail")
	}
}

func TestStoreNotificationSettingsRoundTripAndCooldown(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	settings := models.NotificationSettings{
		Enabled:                 true,
		Channel:                 string(models.NotificationChannelTelegram),
		TelegramBotToken:        "secret-token",
		TelegramTargets:         []models.TelegramTarget{{Name: "Ops", ChatID: "-100123", TopicID: 42}},
		CPUWarningPercent:       80,
		CPUCriticalPercent:      90,
		MemWarningPercent:       81,
		MemCriticalPercent:      91,
		DiskWarningPercent:      82,
		DiskCriticalPercent:     92,
		CooldownMinutes:         15,
		DispatcherQueueCapacity: 512,
		NotifyDispatcherDrops:   true,
	}
	if err := s.UpsertNotificationSettings(settings); err != nil {
		t.Fatalf("UpsertNotificationSettings: %v", err)
	}
	stored, err := s.GetNotificationSettings()
	if err != nil {
		t.Fatalf("GetNotificationSettings: %v", err)
	}
	if stored.TelegramBotToken != "secret-token" || len(stored.TelegramTargets) != 1 || stored.TelegramTargets[0].TopicID != 42 {
		t.Fatalf("unexpected stored notification settings: %#v", stored)
	}
	if stored.DispatcherQueueCapacity != 512 || !stored.NotifyDispatcherDrops {
		t.Fatalf("expected dispatcher settings to round-trip, got %#v", stored)
	}
	view, err := s.NotificationSettingsView(false)
	if err != nil {
		t.Fatalf("NotificationSettingsView: %v", err)
	}
	if !view.TelegramBotTokenSet || view.TelegramBotToken != "" {
		t.Fatalf("expected masked token in notification settings view, got %#v", view)
	}
	if view.DispatcherQueueCapacity != 512 || !view.NotifyDispatcherDrops {
		t.Fatalf("expected dispatcher settings in masked view, got %#v", view)
	}
	allowed, err := s.ShouldSendAlert("node-1", "cpu", "critical", 30*time.Minute, 1000)
	if err != nil || !allowed {
		t.Fatalf("expected first alert to be allowed, got allowed=%v err=%v", allowed, err)
	}
	if err := s.RecordAlertDelivery("node-1", "cpu", "critical", 95, 90, 1000); err != nil {
		t.Fatalf("RecordAlertDelivery: %v", err)
	}
	allowed, err = s.ShouldSendAlert("node-1", "cpu", "critical", 30*time.Minute, 1200)
	if err != nil {
		t.Fatalf("ShouldSendAlert cooldown: %v", err)
	}
	if allowed {
		t.Fatal("expected cooldown window to suppress duplicate alert")
	}
	allowed, err = s.ShouldSendAlert("node-1", "cpu", "critical", 30*time.Minute, 3000)
	if err != nil || !allowed {
		t.Fatalf("expected alert after cooldown to be allowed, got allowed=%v err=%v", allowed, err)
	}

	count, err := s.IncrementRecoveryStreak("node-1", "cpu", 3100)
	if err != nil || count != 1 {
		t.Fatalf("expected first recovery streak increment, count=%d err=%v", count, err)
	}
	count, err = s.IncrementRecoveryStreak("node-1", "cpu", 3200)
	if err != nil || count != 2 {
		t.Fatalf("expected second recovery streak increment, count=%d err=%v", count, err)
	}
	if err := s.ResetRecoveryState("node-1", "cpu"); err != nil {
		t.Fatalf("ResetRecoveryState: %v", err)
	}
	count, err = s.IncrementRecoveryStreak("node-1", "cpu", 3300)
	if err != nil || count != 1 {
		t.Fatalf("expected recovery streak reset to 1, count=%d err=%v", count, err)
	}

	allowed, err = s.ShouldSendRecovery("node-1", "cpu", 30*time.Minute, 3300)
	if err != nil || !allowed {
		t.Fatalf("expected first recovery notice to be allowed, got allowed=%v err=%v", allowed, err)
	}
	if err := s.RecordRecoveryDelivery("node-1", "cpu", 3300); err != nil {
		t.Fatalf("RecordRecoveryDelivery: %v", err)
	}
	allowed, err = s.ShouldSendRecovery("node-1", "cpu", 30*time.Minute, 3400)
	if err != nil {
		t.Fatalf("ShouldSendRecovery cooldown: %v", err)
	}
	if allowed {
		t.Fatal("expected recovery cooldown window to suppress duplicate recovery")
	}
}

func TestStoreNotificationSettingsDefaultDispatcherValues(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	settings, err := s.GetNotificationSettings()
	if err != nil {
		t.Fatalf("GetNotificationSettings: %v", err)
	}
	if settings.DispatcherQueueCapacity != models.DefaultDispatcherQueueCapacity {
		t.Fatalf("expected default dispatcher queue capacity %d, got %d", models.DefaultDispatcherQueueCapacity, settings.DispatcherQueueCapacity)
	}
	if settings.NotifyDispatcherDrops {
		t.Fatalf("expected dispatcher drop alerts disabled by default, got %#v", settings)
	}
}

func TestStoreDashboardSettingsDefaultsAndRoundTrip(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	defaults, err := s.GetDashboardSettings()
	if err != nil {
		t.Fatalf("GetDashboardSettings defaults: %v", err)
	}
	if !defaults.ShowDashboardCardIP {
		t.Fatal("expected dashboard card IP visibility to default to true")
	}

	if err := s.UpsertDashboardSettings(models.DashboardSettings{ShowDashboardCardIP: false}); err != nil {
		t.Fatalf("UpsertDashboardSettings: %v", err)
	}

	stored, err := s.GetDashboardSettings()
	if err != nil {
		t.Fatalf("GetDashboardSettings stored: %v", err)
	}
	if stored.ShowDashboardCardIP {
		t.Fatal("expected stored dashboard card IP visibility to round-trip as false")
	}
}

func TestStoreListNodesWithLatestMetrics(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.UpsertNode(&models.Node{ID: "node-b", Name: "beta", Token: "token-b", CreatedAt: 1700000001}); err != nil {
		t.Fatalf("UpsertNode beta: %v", err)
	}
	if err := s.UpsertNode(&models.Node{ID: "node-a", Name: "alpha", Token: "token-a", CreatedAt: 1700000000}); err != nil {
		t.Fatalf("UpsertNode alpha: %v", err)
	}

	if err := s.InsertMetrics("node-a", &models.MetricsPayload{
		TS:            1700000010,
		CPU:           11.5,
		UptimeSeconds: 123,
		Mem:           models.MemStats{Used: 512, Total: 1024},
		Net:           models.NetStats{RxBytes: 100, TxBytes: 200},
	}); err != nil {
		t.Fatalf("InsertMetrics alpha old: %v", err)
	}
	if err := s.InsertMetrics("node-a", &models.MetricsPayload{
		TS:            1700000020,
		CPU:           23.5,
		UptimeSeconds: 456,
		Mem:           models.MemStats{Used: 768, Total: 1024},
		Net:           models.NetStats{RxBytes: 300, TxBytes: 400},
	}); err != nil {
		t.Fatalf("InsertMetrics alpha latest: %v", err)
	}
	if err := s.InsertMetrics("node-b", &models.MetricsPayload{
		TS:            1700000030,
		CPU:           44.0,
		UptimeSeconds: 789,
		Mem:           models.MemStats{Used: 2048, Total: 4096},
		Net:           models.NetStats{RxBytes: 500, TxBytes: 600},
	}); err != nil {
		t.Fatalf("InsertMetrics beta latest: %v", err)
	}

	nodes, err := s.ListNodesWithLatestMetrics()
	if err != nil {
		t.Fatalf("ListNodesWithLatestMetrics: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Name != "alpha" || nodes[1].Name != "beta" {
		t.Fatalf("expected nodes ordered by name, got %q then %q", nodes[0].Name, nodes[1].Name)
	}
	if nodes[0].LatestMetrics == nil {
		t.Fatal("expected alpha latest metrics to be populated")
	}
	if nodes[0].LatestMetrics.CPU != 23.5 || nodes[0].LatestMetrics.UptimeSeconds != 456 {
		t.Fatalf("unexpected alpha latest metrics: %#v", nodes[0].LatestMetrics)
	}
	if nodes[1].LatestMetrics == nil {
		t.Fatal("expected beta latest metrics to be populated")
	}
	if nodes[1].LatestMetrics.CPU != 44.0 || nodes[1].LatestMetrics.NetTx != 600 {
		t.Fatalf("unexpected beta latest metrics: %#v", nodes[1].LatestMetrics)
	}
}

func TestStoreApplyAgentSnapshot(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token",
		CreatedAt: 1700000000,
	}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	dockerAvailable := true
	payload := &models.MetricsPayload{
		TS:            1700000100,
		CPU:           31.25,
		OS:            "linux",
		Arch:          "amd64",
		AgentVersion:  "v1.2.3",
		UptimeSeconds: 900,
		Hardware: &models.NodeHardware{
			CPUModel:    "EPYC",
			MemoryTotal: 17179869184,
		},
		Mem: models.MemStats{Used: 2048, Total: 8192},
		Disk: []models.DiskStats{
			{Mount: "/", Used: 4096, Total: 16384},
		},
		Net: models.NetStats{RxBytes: 555, TxBytes: 777},
		Processes: []models.Process{
			{PID: 10, Name: "nginx", CPUPercent: 1.2, MemRSS: 1234},
		},
		Services: []models.Service{
			{Name: "nginx", Status: "running"},
			{Name: "sshd", Status: "running"},
		},
		DockerAvailable: &dockerAvailable,
		Containers: []models.DockerContainer{
			{ID: "abcdef123456", Name: "web", Image: "nginx:latest", State: "running", Status: "Up 1 hour"},
		},
	}

	if err := s.ApplyAgentSnapshot("node-1", payload, "198.51.100.10", 1700000101); err != nil {
		t.Fatalf("ApplyAgentSnapshot: %v", err)
	}

	rows, err := s.QueryMetrics("node-1", 1700000000, 1700000200)
	if err != nil {
		t.Fatalf("QueryMetrics: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 metrics row, got %d", len(rows))
	}
	if rows[0].CPU != 31.25 || rows[0].DiskTotal != 16384 || rows[0].NetTx != 777 {
		t.Fatalf("unexpected metrics row: %#v", rows[0])
	}

	node, err := s.GetNodeByID("node-1")
	if err != nil {
		t.Fatalf("GetNodeByID: %v", err)
	}
	if node == nil {
		t.Fatal("expected node to exist")
	}
	if node.IP != "198.51.100.10" || node.AgentVersion != "v1.2.3" || node.LastSeen != 1700000101 {
		t.Fatalf("unexpected node metadata after snapshot: %#v", node)
	}
	if node.Hardware == nil || node.Hardware.CPUModel != "EPYC" {
		t.Fatalf("expected hardware metadata to persist, got %#v", node.Hardware)
	}

	processesJSON, err := s.GetProcesses("node-1")
	if err != nil {
		t.Fatalf("GetProcesses: %v", err)
	}
	var processes []models.Process
	if err := json.Unmarshal([]byte(processesJSON), &processes); err != nil {
		t.Fatalf("unmarshal processes: %v", err)
	}
	if len(processes) != 1 || processes[0].Name != "nginx" {
		t.Fatalf("unexpected processes snapshot: %#v", processes)
	}

	services, err := s.GetServiceChecks("node-1")
	if err != nil {
		t.Fatalf("GetServiceChecks: %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("expected 2 service checks, got %d", len(services))
	}

	dockerReady, containersJSON, err := s.GetDockerContainers("node-1")
	if err != nil {
		t.Fatalf("GetDockerContainers: %v", err)
	}
	if !dockerReady {
		t.Fatal("expected docker availability to persist")
	}
	var containers []models.DockerContainer
	if err := json.Unmarshal([]byte(containersJSON), &containers); err != nil {
		t.Fatalf("unmarshal containers: %v", err)
	}
	if len(containers) != 1 || containers[0].Name != "web" {
		t.Fatalf("unexpected docker snapshot: %#v", containers)
	}
}

func TestStoreLatencyMonitorsRoundTrip(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	for _, node := range []*models.Node{
		{ID: "node-1", Token: "token-1", Name: "alpha"},
		{ID: "node-2", Token: "token-2", Name: "beta"},
	} {
		if err := s.UpsertNode(node); err != nil {
			t.Fatalf("UpsertNode %s: %v", node.ID, err)
		}
	}

	monitor := &models.LatencyMonitor{
		ID:                 "monitor-1",
		Name:               "Guangdong Telecom IPv4",
		Type:               models.LatencyMonitorTypeTCP,
		Target:             "gd-ct-v4.ip.zstaticcdn.com:80",
		IntervalSeconds:    60,
		AutoAssignNewNodes: true,
		CreatedAt:          1700000000,
		UpdatedAt:          1700000000,
	}

	if err := s.CreateLatencyMonitor(monitor, []string{"node-1", "node-2"}); err != nil {
		t.Fatalf("CreateLatencyMonitor: %v", err)
	}

	monitors, err := s.ListLatencyMonitors()
	if err != nil {
		t.Fatalf("ListLatencyMonitors: %v", err)
	}
	if len(monitors) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(monitors))
	}
	if monitors[0].Name != monitor.Name || monitors[0].AssignedNodeCount != 2 {
		t.Fatalf("unexpected listed monitor: %#v", monitors[0])
	}

	nodeMonitors, err := s.ListLatencyMonitorsByNodeID("node-2")
	if err != nil {
		t.Fatalf("ListLatencyMonitorsByNodeID: %v", err)
	}
	if len(nodeMonitors) != 1 || nodeMonitors[0].ID != "monitor-1" {
		t.Fatalf("expected node-2 to see monitor-1, got %#v", nodeMonitors)
	}
}

func TestNewNodeAutoAssignedLatencyMonitors(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.UpsertNode(&models.Node{ID: "node-1", Token: "token-1", Name: "alpha"}); err != nil {
		t.Fatalf("UpsertNode node-1: %v", err)
	}

	if err := s.CreateLatencyMonitor(&models.LatencyMonitor{
		ID:                 "monitor-1",
		Name:               "Default monitor",
		Type:               models.LatencyMonitorTypeICMP,
		Target:             "1.1.1.1",
		IntervalSeconds:    30,
		AutoAssignNewNodes: true,
		CreatedAt:          1700000000,
		UpdatedAt:          1700000000,
	}, []string{"node-1"}); err != nil {
		t.Fatalf("CreateLatencyMonitor: %v", err)
	}

	if err := s.UpsertNode(&models.Node{ID: "node-2", Token: "token-2", Name: "beta"}); err != nil {
		t.Fatalf("UpsertNode node-2: %v", err)
	}
	if err := s.AssignAutoLatencyMonitorsToNode("node-2"); err != nil {
		t.Fatalf("AssignAutoLatencyMonitorsToNode: %v", err)
	}

	nodeMonitors, err := s.ListLatencyMonitorsByNodeID("node-2")
	if err != nil {
		t.Fatalf("ListLatencyMonitorsByNodeID: %v", err)
	}
	if len(nodeMonitors) != 1 || nodeMonitors[0].ID != "monitor-1" {
		t.Fatalf("expected node-2 to be auto-assigned, got %#v", nodeMonitors)
	}
}

func TestLatencyResultsQueryByNode(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.UpsertNode(&models.Node{ID: "node-1", Token: "token-1", Name: "alpha"}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	if err := s.CreateLatencyMonitor(&models.LatencyMonitor{
		ID:                 "monitor-1",
		Name:               "TCP 80",
		Type:               models.LatencyMonitorTypeTCP,
		Target:             "example.com:80",
		IntervalSeconds:    60,
		AutoAssignNewNodes: true,
		CreatedAt:          1700000000,
		UpdatedAt:          1700000000,
	}, []string{"node-1"}); err != nil {
		t.Fatalf("CreateLatencyMonitor: %v", err)
	}

	latency := 23.75
	loss := 20.0
	jitter := 4.25
	if err := s.InsertLatencyResult(&models.LatencyMonitorResult{
		MonitorID:    "monitor-1",
		NodeID:       "node-1",
		TS:           1700000100,
		LatencyMs:    &latency,
		LossPercent:  &loss,
		JitterMs:     &jitter,
		Success:      true,
		ErrorMessage: "",
	}); err != nil {
		t.Fatalf("InsertLatencyResult success: %v", err)
	}
	if err := s.InsertLatencyResult(&models.LatencyMonitorResult{
		MonitorID:    "monitor-1",
		NodeID:       "node-1",
		TS:           1700000160,
		LatencyMs:    nil,
		Success:      false,
		ErrorMessage: "dial timeout",
	}); err != nil {
		t.Fatalf("InsertLatencyResult failure: %v", err)
	}

	results, err := s.QueryLatencyResultsByNodeID("node-1", 1700000000, 1700000200)
	if err != nil {
		t.Fatalf("QueryLatencyResultsByNodeID: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].LatencyMs == nil || *results[0].LatencyMs != latency || !results[0].Success {
		t.Fatalf("unexpected first result: %#v", results[0])
	}
	if results[0].LossPercent == nil || *results[0].LossPercent != loss {
		t.Fatalf("unexpected loss percent on first result: %#v", results[0])
	}
	if results[0].JitterMs == nil || *results[0].JitterMs != jitter {
		t.Fatalf("unexpected jitter on first result: %#v", results[0])
	}
	if results[1].LatencyMs != nil || results[1].Success || results[1].ErrorMessage != "dial timeout" {
		t.Fatalf("unexpected second result: %#v", results[1])
	}
	if results[1].LossPercent != nil || results[1].JitterMs != nil {
		t.Fatalf("expected failed result summaries to remain empty, got %#v", results[1])
	}
}

func TestDeleteLatencyMonitorCascades(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if err := s.UpsertNode(&models.Node{ID: "node-1", Token: "token-1", Name: "alpha"}); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	if err := s.CreateLatencyMonitor(&models.LatencyMonitor{
		ID:                 "monitor-1",
		Name:               "Healthcheck",
		Type:               models.LatencyMonitorTypeHTTP,
		Target:             "https://example.com/healthz",
		IntervalSeconds:    60,
		AutoAssignNewNodes: true,
		CreatedAt:          1700000000,
		UpdatedAt:          1700000000,
	}, []string{"node-1"}); err != nil {
		t.Fatalf("CreateLatencyMonitor: %v", err)
	}

	latency := 11.5
	if err := s.InsertLatencyResult(&models.LatencyMonitorResult{
		MonitorID: "monitor-1",
		NodeID:    "node-1",
		TS:        1700000100,
		LatencyMs: &latency,
		Success:   true,
	}); err != nil {
		t.Fatalf("InsertLatencyResult: %v", err)
	}

	if err := s.DeleteLatencyMonitor("monitor-1"); err != nil {
		t.Fatalf("DeleteLatencyMonitor: %v", err)
	}

	monitors, err := s.ListLatencyMonitors()
	if err != nil {
		t.Fatalf("ListLatencyMonitors: %v", err)
	}
	if len(monitors) != 0 {
		t.Fatalf("expected monitor to be deleted, got %#v", monitors)
	}

	nodeMonitors, err := s.ListLatencyMonitorsByNodeID("node-1")
	if err != nil {
		t.Fatalf("ListLatencyMonitorsByNodeID: %v", err)
	}
	if len(nodeMonitors) != 0 {
		t.Fatalf("expected node bindings to be deleted, got %#v", nodeMonitors)
	}

	results, err := s.QueryLatencyResultsByNodeID("node-1", 1700000000, 1700000200)
	if err != nil {
		t.Fatalf("QueryLatencyResultsByNodeID: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected results to be deleted, got %#v", results)
	}
}
