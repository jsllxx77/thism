package collector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

func newStateOnlyCollector(t *testing.T) *Collector {
	t.Helper()
	return newTestCollector(t, "http://localhost:12026", "token", "node", "", DefaultReportInterval)
}

func TestAgentStateRoundTripPersistsStreamWatermarkAndRecords(t *testing.T) {
	collector := newStateOnlyCollector(t)
	collector.stateMu.Lock()
	collector.state.StreamID = "stream-test"
	collector.state.RetiredWatermark = 7
	collector.state.Records = []executionRecord{{
		Key: "seq:7", Seq: 7, JobID: "job-7", Status: models.UpdateJobTargetStatusSucceeded, UpdatedAt: time.Now().Unix(),
	}}
	if err := collector.saveAgentState(); err != nil {
		collector.stateMu.Unlock()
		t.Fatalf("saveAgentState: %v", err)
	}
	statePath := collector.statePath
	collector.stateMu.Unlock()

	other := NewWithInterval("http://localhost:12026", "token", "node", "", DefaultReportInterval)
	other.statePath = statePath
	other.loadAgentState()
	if other.state.ControlState != models.ControlStateNone {
		t.Fatalf("expected healthy state, got %q", other.state.ControlState)
	}
	if other.state.StreamID != "stream-test" || other.state.RetiredWatermark != 7 {
		t.Fatalf("state did not round-trip: %#v", other.state)
	}
	if record := findRecord(other.state.Records, "seq:7"); record == nil || record.Status != models.UpdateJobTargetStatusSucceeded {
		t.Fatalf("terminal record did not round-trip: %#v", other.state.Records)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(statePath), "agent-state.json.tmp")); !os.IsNotExist(err) {
		t.Fatalf("temporary state file remains: %v", err)
	}
}

func TestAgentSequenceGateRejectsFutureAndReplaysRetiredCommand(t *testing.T) {
	collector := newStateOnlyCollector(t)
	collector.stateMu.Lock()
	collector.state.StreamID = "stream-test"
	collector.state.RetiredWatermark = 0
	collector.stateMu.Unlock()

	future := models.AgentCommandPayload{JobID: "job-2", Kind: models.AgentCommandKindSelfUpdate, StreamID: "stream-test", Seq: 2}
	ok, replay, code, message := collector.beginUpdateTask(future)
	if ok || replay != nil || code != models.CommandRejectSequenceGap {
		t.Fatalf("expected sequence_gap, got ok=%v replay=%#v code=%q message=%q", ok, replay, code, message)
	}

	first := models.AgentCommandPayload{JobID: "job-1", Kind: models.AgentCommandKindSelfUpdate, StreamID: "stream-test", Seq: 1}
	ok, replay, code, message = collector.beginUpdateTask(first)
	if !ok || replay != nil || code != "" || message != "" {
		t.Fatalf("expected seq 1 admission, got ok=%v replay=%#v code=%q message=%q", ok, replay, code, message)
	}
	collector.finishUpdateTask(models.UpdateJobTargetStatusSucceeded, "done")

	ok, replay, code, _ = collector.beginUpdateTask(first)
	if ok || replay == nil || replay.Status != models.UpdateJobTargetStatusSucceeded || code != "" {
		t.Fatalf("expected exact terminal replay, got ok=%v replay=%#v code=%q", ok, replay, code)
	}
}

func TestAgentRejectsMismatchedCommandStream(t *testing.T) {
	collector := newStateOnlyCollector(t)
	collector.stateMu.Lock()
	collector.state.StreamID = "stream-current"
	collector.stateMu.Unlock()
	cmd := models.AgentCommandPayload{JobID: "job-stream-mismatch", Kind: models.AgentCommandKindSelfUpdate, StreamID: "stream-other", Seq: 1}
	ok, replay, code, _ := collector.beginUpdateTask(cmd)
	if ok || replay != nil || code != models.CommandRejectControlQuarantined {
		t.Fatalf("expected stream mismatch quarantine, got ok=%v replay=%#v code=%q", ok, replay, code)
	}
	collector.stateMu.Lock()
	defer collector.stateMu.Unlock()
	if collector.state.ControlState != models.ControlStateQuarantined {
		t.Fatalf("expected local quarantine, got %q", collector.state.ControlState)
	}
}

func TestAgentJobIDDedupPreventsSecondExecution(t *testing.T) {
	collector := newStateOnlyCollector(t)
	cmd := models.AgentCommandPayload{JobID: "job-dedup", Kind: models.AgentCommandKindSelfUpdate, SHA256: "abc"}
	ok, _, _, _ := collector.beginUpdateTask(cmd)
	if !ok {
		t.Fatal("expected first command admission")
	}
	collector.finishUpdateTask(models.UpdateJobTargetStatusFailed, "checksum mismatch")

	ok, replay, code, _ := collector.beginUpdateTask(cmd)
	if ok || replay == nil || replay.Status != models.UpdateJobTargetStatusFailed || code != "" {
		t.Fatalf("expected JobID replay, got ok=%v replay=%#v code=%q", ok, replay, code)
	}
}

func TestAgentRecoveryUsesExecutableEvidenceAfterCommitPoint(t *testing.T) {
	collector := newStateOnlyCollector(t)
	digest, err := collector.currentExecutableSHA256()
	if err != nil {
		t.Fatalf("currentExecutableSHA256: %v", err)
	}
	statePath := collector.statePath
	state := agentState{
		StreamID:         "stream-test",
		RetiredWatermark: 2,
		InProgress: &inProgressState{
			Key: "seq:3", Seq: 3, JobID: "job-3", Stage: models.UpdateJobTargetStatusRestarting,
			TargetSHA256: digest, CommitPoint: true, LastProgressAt: time.Now().Unix(),
		},
	}
	raw, _ := json.Marshal(state)
	if err := os.WriteFile(statePath, raw, 0600); err != nil {
		t.Fatalf("write recovery state: %v", err)
	}

	recovered := NewWithInterval("http://localhost:12026", "token", "node", "", DefaultReportInterval)
	recovered.statePath = statePath
	recovered.loadAgentState()
	if recovered.state.InProgress != nil {
		t.Fatal("expected in-progress state to be resolved")
	}
	if recovered.state.RetiredWatermark != 3 {
		t.Fatalf("expected watermark 3 after evidence recovery, got %d", recovered.state.RetiredWatermark)
	}
	record := findRecord(recovered.state.Records, "seq:3")
	if record == nil || record.Status != models.UpdateJobTargetStatusSucceeded {
		t.Fatalf("expected recovered success record, got %#v", recovered.state.Records)
	}
	if recovered.state.ControlState != models.ControlStateNone {
		t.Fatalf("matching executable evidence must not block control, got %q", recovered.state.ControlState)
	}
}

func TestAgentRecoveryBlocksWhenPostCommitEvidenceDoesNotMatch(t *testing.T) {
	collector := newStateOnlyCollector(t)
	statePath := collector.statePath
	state := agentState{
		StreamID: "stream-test",
		InProgress: &inProgressState{
			Key: "seq:4", Seq: 4, JobID: "job-4", Stage: models.UpdateJobTargetStatusRestarting,
			TargetSHA256: "definitely-not-the-current-binary", CommitPoint: true,
		},
	}
	raw, _ := json.Marshal(state)
	if err := os.WriteFile(statePath, raw, 0600); err != nil {
		t.Fatalf("write recovery state: %v", err)
	}

	recovered := NewWithInterval("http://localhost:12026", "token", "node", "", DefaultReportInterval)
	recovered.statePath = statePath
	recovered.loadAgentState()
	if recovered.state.ControlState != models.ControlStateBlocked {
		t.Fatalf("expected control-blocked after unverifiable commit, got %q", recovered.state.ControlState)
	}
	record := findRecord(recovered.state.Records, "seq:4")
	if record == nil || record.Status != models.UpdateJobTargetStatusUnknown {
		t.Fatalf("expected unknown recovery record, got %#v", recovered.state.Records)
	}
}

func TestAgentHandshakeAckClearsDisposedBlockedStateButNotQuarantine(t *testing.T) {
	collector := newStateOnlyCollector(t)
	collector.stateMu.Lock()
	collector.state.ControlState = models.ControlStateBlocked
	collector.state.ControlReason = "old unknown"
	collector.state.StreamID = "stream-test"
	collector.stateMu.Unlock()
	collector.applyHandshakeAck(models.AgentHandshakeAckPayload{StreamID: "stream-test", ControlState: models.ControlStateNone})
	collector.stateMu.Lock()
	if collector.state.ControlState != models.ControlStateNone {
		t.Fatalf("expected blocked state to clear after server disposition, got %q", collector.state.ControlState)
	}
	collector.state.ControlState = models.ControlStateQuarantined
	collector.state.ControlReason = "corrupt state"
	collector.stateMu.Unlock()
	collector.applyHandshakeAck(models.AgentHandshakeAckPayload{StreamID: "stream-test", ControlState: models.ControlStateNone})
	collector.stateMu.Lock()
	defer collector.stateMu.Unlock()
	if collector.state.ControlState != models.ControlStateQuarantined {
		t.Fatalf("quarantine must not clear automatically, got %q", collector.state.ControlState)
	}
}

func TestAgentCancelIntentPersistsBeforeCommitAndIsRefusedAfterCommit(t *testing.T) {
	collector := newStateOnlyCollector(t)
	collector.stateMu.Lock()
	collector.state.StreamID = "stream-test"
	collector.stateMu.Unlock()
	cmd := models.AgentCommandPayload{JobID: "job-cancel", Kind: models.AgentCommandKindSelfUpdate, StreamID: "stream-test", Seq: 1}
	ok, _, _, _ := collector.beginUpdateTask(cmd)
	if !ok {
		t.Fatal("expected command admission")
	}
	conn := &commandTestConn{}
	var writeMu sync.Mutex
	collector.handleCancelRequest(models.AgentCommandCancelPayload{JobID: cmd.JobID, Seq: cmd.Seq}, conn, &writeMu)
	collector.stateMu.Lock()
	if collector.state.InProgress == nil || !collector.state.InProgress.CancelIntent {
		collector.stateMu.Unlock()
		t.Fatal("expected cancel intent to be durable before commit")
	}
	collector.stateMu.Unlock()

	collector.stateMu.Lock()
	collector.state.InProgress.CommitPoint = true
	collector.state.InProgress.CancelIntent = false
	_ = collector.saveAgentState()
	collector.stateMu.Unlock()
	collector.handleCancelRequest(models.AgentCommandCancelPayload{JobID: cmd.JobID, Seq: cmd.Seq}, conn, &writeMu)
	collector.stateMu.Lock()
	if collector.state.InProgress.CancelIntent {
		collector.stateMu.Unlock()
		t.Fatal("cancel intent must not be set after commit point")
	}
	collector.stateMu.Unlock()
}
