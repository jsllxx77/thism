package store_test

import (
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
