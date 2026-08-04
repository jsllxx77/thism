package store

import (
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

func newStoreForTest(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateUpdateJobWithTargetsAllocatesContiguousSequences(t *testing.T) {
	s := newStoreForTest(t)

	job := &models.UpdateJob{ID: "job-seq-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	targets, err := s.CreateUpdateJobWithTargets(job, []string{"node-1", "node-2"})
	if err != nil {
		t.Fatalf("CreateUpdateJobWithTargets: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0].Seq != 1 || targets[1].Seq != 1 {
		t.Fatalf("expected both fresh nodes to get seq 1, got %d and %d", targets[0].Seq, targets[1].Seq)
	}
	if targets[0].StreamID == "" || targets[0].StreamID == targets[1].StreamID {
		t.Fatalf("expected a distinct stream per node, got %q and %q", targets[0].StreamID, targets[1].StreamID)
	}
	if targets[0].DeliveryState != models.DeliveryStatePending {
		t.Fatalf("expected pending delivery, got %q", targets[0].DeliveryState)
	}
	if targets[0].TargetSHA256 != "abc" {
		t.Fatalf("expected target sha256 to be stored, got %q", targets[0].TargetSHA256)
	}

	// Second job: each node gets the next contiguous sequence (no gaps).
	job2 := &models.UpdateJob{ID: "job-seq-2", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.4", DownloadURL: "https://example.com/agent", SHA256: "def", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	targets2, err := s.CreateUpdateJobWithTargets(job2, []string{"node-1"})
	if err != nil {
		t.Fatalf("CreateUpdateJobWithTargets: %v", err)
	}
	if targets2[0].Seq != 2 {
		t.Fatalf("expected node-1 seq 2, got %d", targets2[0].Seq)
	}
	if targets2[0].StreamID != targets[0].StreamID {
		t.Fatalf("expected node-1 to reuse its stream, got %q", targets2[0].StreamID)
	}

	stream, err := s.GetCommandStream("node-1")
	if err != nil || stream == nil {
		t.Fatalf("GetCommandStream: %v %v", stream, err)
	}
	if stream.NextSeq != 3 {
		t.Fatalf("expected next_seq 3, got %d", stream.NextSeq)
	}
}

func TestRollbackReconcileAdvancesNextSeqAndAudits(t *testing.T) {
	s := newStoreForTest(t)

	job := &models.UpdateJob{ID: "job-rollback-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	_, err := s.CreateUpdateJobWithTargets(job, []string{"node-1"})
	if err != nil {
		t.Fatalf("CreateUpdateJobWithTargets: %v", err)
	}

	// Simulate a database rollback: the agent retired sequences 1..5 but the
	// restored server only knows seq 1 (next_seq=1 after restore).
	stream, err := s.ReconcileCommandStream("node-1", "", 0)
	if err != nil || stream == nil {
		t.Fatalf("initial reconcile: %v", err)
	}
	reconciled, err := s.ReconcileCommandStream("node-1", stream.StreamID, 5)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reconciled.NextSeq != 6 {
		t.Fatalf("expected next_seq advanced to 6 from agent watermark 5, got %d", reconciled.NextSeq)
	}
	if reconciled.RetiredWatermark != 5 {
		t.Fatalf("expected retired watermark 5, got %d", reconciled.RetiredWatermark)
	}

	audit, err := s.ListControlAudit("node-1", 10)
	if err != nil {
		t.Fatalf("ListControlAudit: %v", err)
	}
	foundRollback := false
	for _, event := range audit {
		if event.Event == "rollback-reconciled" {
			foundRollback = true
		}
	}
	if !foundRollback {
		t.Fatalf("expected rollback-reconciled audit event, got %#v", audit)
	}

	// Sequences must never be reused: next allocation is 6.
	job2 := &models.UpdateJob{ID: "job-rollback-2", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.4", DownloadURL: "https://example.com/agent", SHA256: "def", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	targets, err := s.CreateUpdateJobWithTargets(job2, []string{"node-1"})
	if err != nil {
		t.Fatalf("CreateUpdateJobWithTargets: %v", err)
	}
	if targets[0].Seq != 6 {
		t.Fatalf("expected next allocated seq 6, got %d", targets[0].Seq)
	}
}

func TestStreamMismatchQuarantinesNode(t *testing.T) {
	s := newStoreForTest(t)
	job := &models.UpdateJob{ID: "job-mm-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	_, err := s.CreateUpdateJobWithTargets(job, []string{"node-1"})
	if err != nil {
		t.Fatalf("CreateUpdateJobWithTargets: %v", err)
	}

	stream, _ := s.GetCommandStream("node-1")
	reconciled, err := s.ReconcileCommandStream("node-1", "stream-different", 3)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reconciled.ControlState != models.ControlStateQuarantined {
		t.Fatalf("expected quarantine on stream mismatch, got %q", reconciled.ControlState)
	}
	// A matching stream cannot un-quarantine automatically.
	reconciled2, err := s.ReconcileCommandStream("node-1", stream.StreamID, 3)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reconciled2.ControlState != models.ControlStateQuarantined {
		t.Fatalf("expected quarantine to persist, got %q", reconciled2.ControlState)
	}
}

func TestExecutionTransitionsAreMonotonicAndSticky(t *testing.T) {
	s := newStoreForTest(t)
	job := &models.UpdateJob{ID: "job-cas-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if _, err := s.CreateUpdateJobWithTargets(job, []string{"node-1"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	apply := func(status models.UpdateJobTargetStatus) bool {
		t.Helper()
		applied, err := s.ApplyExecutionTransition(job.ID, "node-1", status, "msg", "")
		if err != nil {
			t.Fatalf("ApplyExecutionTransition(%s): %v", status, err)
		}
		return applied
	}

	if !apply(models.UpdateJobTargetStatusAccepted) {
		t.Fatal("pending -> accepted must apply")
	}
	if apply(models.UpdateJobTargetStatusDispatched) {
		t.Fatal("late dispatched after accepted must be rejected")
	}
	if !apply(models.UpdateJobTargetStatusDownloading) {
		t.Fatal("accepted -> downloading must apply")
	}
	if !apply(models.UpdateJobTargetStatusSucceeded) {
		t.Fatal("downloading -> succeeded must apply")
	}
	if apply(models.UpdateJobTargetStatusFailed) {
		t.Fatal("succeeded is sticky; failed must be rejected")
	}
	// Same-value idempotent.
	if !apply(models.UpdateJobTargetStatusSucceeded) {
		t.Fatal("same-value succeeded must be idempotent")
	}

	target, err := s.GetUpdateJobTarget(job.ID, "node-1")
	if err != nil || target == nil {
		t.Fatalf("GetUpdateJobTarget: %v", err)
	}
	if target.Status != models.UpdateJobTargetStatusSucceeded {
		t.Fatalf("expected sticky succeeded, got %q", target.Status)
	}
}

func TestSendFailedDoesNotOverwriteAccepted(t *testing.T) {
	s := newStoreForTest(t)
	job := &models.UpdateJob{ID: "job-sf-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if _, err := s.CreateUpdateJobWithTargets(job, []string{"node-1"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.ApplyExecutionTransition(job.ID, "node-1", models.UpdateJobTargetStatusAccepted, "accepted", "1.2.2"); err != nil {
		t.Fatalf("accepted: %v", err)
	}
	// Delivery and execution are independent dimensions.
	if err := s.RecordDelivery(job.ID, "node-1", models.DeliveryStateSendFailed, "socket error", 7); err != nil {
		t.Fatalf("RecordDelivery: %v", err)
	}
	target, err := s.GetUpdateJobTarget(job.ID, "node-1")
	if err != nil || target == nil {
		t.Fatalf("GetUpdateJobTarget: %v", err)
	}
	if target.Status != models.UpdateJobTargetStatusAccepted {
		t.Fatalf("send_failed must not overwrite accepted, got %q", target.Status)
	}
	if target.DeliveryState != models.DeliveryStateSendFailed {
		t.Fatalf("expected send_failed delivery state, got %q", target.DeliveryState)
	}
	if target.DeliveryError != "socket error" {
		t.Fatalf("expected delivery error recorded, got %q", target.DeliveryError)
	}
}

func TestUnknownBlocksNodeUntilResolved(t *testing.T) {
	s := newStoreForTest(t)
	job := &models.UpdateJob{ID: "job-unknown-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if _, err := s.CreateUpdateJobWithTargets(job, []string{"node-1"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.ApplyExecutionTransition(job.ID, "node-1", models.UpdateJobTargetStatusUnknown, "post-commit failure", ""); err != nil {
		t.Fatalf("unknown: %v", err)
	}

	stream, err := s.GetCommandStream("node-1")
	if err != nil || stream == nil {
		t.Fatalf("GetCommandStream: %v", err)
	}
	if stream.ControlState != models.ControlStateBlocked {
		t.Fatalf("expected control-blocked after unknown, got %q", stream.ControlState)
	}

	// Illegal resolution (back to running) is refused.
	if _, err := s.ResolveUnknownTarget(job.ID, "node-1", models.UpdateJobTargetStatusAccepted, "", ""); err == nil {
		t.Fatal("expected non-terminal resolution to fail")
	}
	// Resolution without evidence or reason is refused.
	if _, err := s.ResolveUnknownTarget(job.ID, "node-1", models.UpdateJobTargetStatusSucceeded, "", ""); err == nil {
		t.Fatal("expected resolution without evidence to fail")
	}

	applied, err := s.ResolveUnknownTarget(job.ID, "node-1", models.UpdateJobTargetStatusSucceeded, "sha256:abc123", "verified on host")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !applied {
		t.Fatal("expected resolution to apply")
	}
	stream, _ = s.GetCommandStream("node-1")
	if stream.ControlState != models.ControlStateNone {
		t.Fatalf("expected control state cleared after disposition, got %q", stream.ControlState)
	}
	audit, _ := s.ListControlAudit("node-1", 20)
	found := false
	for _, event := range audit {
		if event.Event == "unknown-disposition" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unknown-disposition audit event, got %#v", audit)
	}
}

func TestCancelTombstoneClosesSequence(t *testing.T) {
	s := newStoreForTest(t)
	job := &models.UpdateJob{ID: "job-cancel-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	targets, err := s.CreateUpdateJobWithTargets(job, []string{"node-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateCancelTombstone(job.ID, "node-1", targets[0].Seq); err != nil {
		t.Fatalf("CreateCancelTombstone: %v", err)
	}
	watermark, err := s.RetiredWatermarkForNode("node-1")
	if err != nil {
		t.Fatalf("RetiredWatermarkForNode: %v", err)
	}
	if watermark != 1 {
		t.Fatalf("expected watermark 1 from tombstone, got %d", watermark)
	}
	target, _ := s.GetUpdateJobTarget(job.ID, "node-1")
	if target.CancelIntentAt == 0 {
		t.Fatal("expected cancel_intent_at to be recorded")
	}
}

func TestPendingDeliveriesExcludesTerminalAndRetired(t *testing.T) {
	s := newStoreForTest(t)
	job := &models.UpdateJob{ID: "job-pd-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	targets, err := s.CreateUpdateJobWithTargets(job, []string{"node-1", "node-2"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// node-1 target is terminal (retired), node-2 still pending.
	if _, err := s.ApplyExecutionTransition(job.ID, "node-1", models.UpdateJobTargetStatusSucceeded, "done", "1.2.3"); err != nil {
		t.Fatalf("succeeded: %v", err)
	}
	pending, err := s.PendingDeliveriesForNode("node-2", 0, 10)
	if err != nil {
		t.Fatalf("PendingDeliveriesForNode: %v", err)
	}
	if len(pending) != 1 || pending[0].NodeID != "node-2" || pending[0].Seq != targets[1].Seq {
		t.Fatalf("expected only node-2 pending delivery, got %#v", pending)
	}
	// A node whose watermark covers the seq gets nothing.
	pending, err = s.PendingDeliveriesForNode("node-1", 1, 10)
	if err != nil {
		t.Fatalf("PendingDeliveriesForNode: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending deliveries past watermark, got %#v", pending)
	}
}

func TestPruneUpdateDetailKeepsInProgressAndUnknown(t *testing.T) {
	s := newStoreForTest(t)
	old := time.Now().Add(-400 * 24 * time.Hour).Unix()
	job := &models.UpdateJob{ID: "job-prune-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: old, CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if _, err := s.CreateUpdateJobWithTargets(job, []string{"node-1", "node-2", "node-3"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Make them old.
	if _, err := s.db.Exec(`UPDATE update_job_targets SET updated_at = ?`, old); err != nil {
		t.Fatalf("age rows: %v", err)
	}
	// node-1 in-progress (protected), node-2 unknown (protected), node-3 terminal.
	if _, err := s.ApplyExecutionTransition(job.ID, "node-1", models.UpdateJobTargetStatusAccepted, "active", ""); err != nil {
		t.Fatalf("accepted: %v", err)
	}
	if _, err := s.ApplyExecutionTransition(job.ID, "node-2", models.UpdateJobTargetStatusUnknown, "unknown", ""); err != nil {
		t.Fatalf("unknown: %v", err)
	}
	if _, err := s.ApplyExecutionTransition(job.ID, "node-3", models.UpdateJobTargetStatusSucceeded, "done", "1.2.3"); err != nil {
		t.Fatalf("succeeded: %v", err)
	}
	// Re-age all rows: the transitions refreshed updated_at to now.
	if _, err := s.db.Exec(`UPDATE update_job_targets SET updated_at = ?`, old); err != nil {
		t.Fatalf("re-age rows: %v", err)
	}
	pruned, err := s.PruneUpdateDetail()
	if err != nil {
		t.Fatalf("PruneUpdateDetail: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("expected exactly the terminal row pruned, got %d", pruned)
	}
	for _, nodeID := range []string{"node-1", "node-2"} {
		target, err := s.GetUpdateJobTarget(job.ID, nodeID)
		if err != nil || target == nil {
			t.Fatalf("expected protected row for %s to survive, err=%v", nodeID, err)
		}
	}
	if target, _ := s.GetUpdateJobTarget(job.ID, "node-3"); target != nil {
		t.Fatal("expected terminal row to be pruned")
	}
}

func TestMarkStalledObservationOnlyTouchesObservation(t *testing.T) {
	s := newStoreForTest(t)
	job := &models.UpdateJob{ID: "job-stall-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://example.com/agent", SHA256: "abc", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if _, err := s.CreateUpdateJobWithTargets(job, []string{"node-1"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.ApplyExecutionTransition(job.ID, "node-1", models.UpdateJobTargetStatusDownloading, "downloading", ""); err != nil {
		t.Fatalf("downloading: %v", err)
	}
	if _, err := s.db.Exec(`UPDATE update_job_targets SET last_progress_at = ?`, time.Now().Add(-2*time.Hour).Unix()); err != nil {
		t.Fatalf("age progress: %v", err)
	}
	stalled, err := s.MarkStalledUpdateObservations(30 * time.Minute)
	if err != nil {
		t.Fatalf("MarkStalledUpdateObservations: %v", err)
	}
	if stalled != 1 {
		t.Fatalf("expected 1 stalled target, got %d", stalled)
	}
	target, _ := s.GetUpdateJobTarget(job.ID, "node-1")
	if target.Observation != models.ObservationStalled {
		t.Fatalf("expected stalled observation, got %q", target.Observation)
	}
	if target.Status != models.UpdateJobTargetStatusDownloading {
		t.Fatalf("stalled observation must not change execution status, got %q", target.Status)
	}
}
