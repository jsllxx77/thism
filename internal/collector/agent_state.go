package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

// This file implements the agent-side of the control-plane concurrency
// protocol: durable local state (stream id, retired watermark, execution
// records), atomic persistence, crash recovery, sequence gating, JobID
// deduplication and the single serialized self-update executor shared by
// remote commands and auto-update. Invariants are documented in
// docs/concurrency-remediation-plan.md §7.

const (
	// agentStateRecordsWindow is the number of recent execution records kept
	// locally for replay and handshake backfill (plan: 1024).
	agentStateRecordsWindow = 1024
	// Stage lease defaults (plan §7.3.2): download, verify/write, restart.
	leaseDownload = 2 * time.Minute
	leaseVerify   = 1 * time.Minute
	leaseRestart  = 2 * time.Minute
	// agentStateFileName is the durable state file name next to the binary.
	agentStateFileName = ".thism-agent.state.json"
)

var (
	errSelfUpdateCancelled = errors.New("self update cancelled before commit point")
	errSelfUpdateBusy      = errors.New("another self update is already in progress")
)

type executionRecord struct {
	Key             string                       `json:"key"`
	Seq             int64                        `json:"seq"`
	JobID           string                       `json:"job_id"`
	Status          models.UpdateJobTargetStatus `json:"status"`
	Message         string                       `json:"message,omitempty"`
	ReportedVersion string                       `json:"reported_version,omitempty"`
	TargetSHA256    string                       `json:"target_sha256,omitempty"`
	UpdatedAt       int64                        `json:"updated_at"`
}

type inProgressState struct {
	Key            string                       `json:"key"`
	Seq            int64                        `json:"seq"`
	JobID          string                       `json:"job_id"`
	Stage          models.UpdateJobTargetStatus `json:"stage"`
	TargetVersion  string                       `json:"target_version,omitempty"`
	TargetSHA256   string                       `json:"target_sha256,omitempty"`
	StartedAt      int64                        `json:"started_at"`
	LastProgressAt int64                        `json:"last_progress_at"`
	CommitPoint    bool                         `json:"commit_point"`
	CancelIntent   bool                         `json:"cancel_intent,omitempty"`
}

// agentState is the durable local protocol state. It is persisted atomically
// (temp file + fsync + rename) so a partially written file can never be
// mistaken for a valid watermark.
type agentState struct {
	StreamID         string              `json:"stream_id,omitempty"`
	RetiredWatermark int64               `json:"retired_watermark"`
	InProgress       *inProgressState    `json:"in_progress,omitempty"`
	Records          []executionRecord   `json:"records,omitempty"`
	ControlState     models.ControlState `json:"control_state,omitempty"`
	ControlReason    string              `json:"control_reason,omitempty"`
}

func defaultAgentStatePath() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	resolvedPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = resolvedPath
	}
	return filepath.Join(filepath.Dir(exePath), agentStateFileName)
}

func (c *Collector) loadAgentState() {
	path := c.statePath
	if path == "" {
		path = defaultAgentStatePath()
	}
	c.statePath = path
	c.state = &agentState{}
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return // fresh install; stream binding happens at handshake
		}
		c.state.ControlState = models.ControlStateQuarantined
		c.state.ControlReason = "local state unreadable: " + err.Error()
		return
	}
	if err := json.Unmarshal(raw, c.state); err != nil {
		// Never auto-reset a corrupted watermark; fail closed.
		c.state = &agentState{}
		c.state.ControlState = models.ControlStateQuarantined
		c.state.ControlReason = "local state corrupted: " + err.Error()
		return
	}
	c.recoverInProgressState()
}

// recoverInProgressState resolves an in-progress task left by a crash,
// syscall.Exec failure or process restart:
//   - before the rename commit point: the update did not happen -> failed;
//   - after the commit point with matching executable SHA256: succeeded;
//   - after the commit point with unverifiable binary: unknown + blocked.
func (c *Collector) recoverInProgressState() {
	inProgress := c.state.InProgress
	if inProgress == nil {
		return
	}
	now := time.Now().Unix()
	if inProgress.CommitPoint {
		digest, err := c.currentExecutableSHA256()
		if err == nil && strings.EqualFold(strings.TrimSpace(digest), strings.TrimSpace(inProgress.TargetSHA256)) {
			c.state.Records = appendRecord(c.state.Records, executionRecord{
				Key: inProgress.Key, Seq: inProgress.Seq, JobID: inProgress.JobID,
				Status:       models.UpdateJobTargetStatusSucceeded,
				Message:      "recovered after restart: executable SHA256 matches target",
				TargetSHA256: inProgress.TargetSHA256, UpdatedAt: now,
			})
		} else {
			c.state.Records = appendRecord(c.state.Records, executionRecord{
				Key: inProgress.Key, Seq: inProgress.Seq, JobID: inProgress.JobID,
				Status:       models.UpdateJobTargetStatusUnknown,
				Message:      "post-commit restart with unverifiable binary; operator disposition required",
				TargetSHA256: inProgress.TargetSHA256, UpdatedAt: now,
			})
			c.state.ControlState = models.ControlStateBlocked
			c.state.ControlReason = "undisposed unknown execution state"
		}
	} else {
		status := models.UpdateJobTargetStatusFailed
		message := "interrupted before commit point"
		if inProgress.CancelIntent {
			status = models.UpdateJobTargetStatusCancelled
			message = "cancelled before commit point"
		}
		c.state.Records = appendRecord(c.state.Records, executionRecord{
			Key: inProgress.Key, Seq: inProgress.Seq, JobID: inProgress.JobID,
			Status:  status,
			Message: message, UpdatedAt: now,
		})
		_ = os.Remove(filepath.Join(filepath.Dir(c.statePath), ".thism-agent-update.tmp"))
	}
	if inProgress.Seq > c.state.RetiredWatermark {
		c.state.RetiredWatermark = inProgress.Seq
	}
	c.state.InProgress = nil
	_ = c.saveAgentState()
}

func appendRecord(records []executionRecord, record executionRecord) []executionRecord {
	for i := range records {
		if records[i].Key == record.Key {
			records[i] = record
			return records
		}
	}
	records = append(records, record)
	if len(records) > agentStateRecordsWindow {
		records = records[len(records)-agentStateRecordsWindow:]
	}
	return records
}

func findRecord(records []executionRecord, key string) *executionRecord {
	for i := range records {
		if records[i].Key == key {
			return &records[i]
		}
	}
	return nil
}

func (c *Collector) saveAgentState() error {
	path := c.statePath
	if path == "" {
		return nil
	}
	if c.state == nil {
		return nil
	}
	if c.state.ControlState == models.ControlStateQuarantined {
		// Do not overwrite the quarantined evidence file.
		return nil
	}
	raw, err := json.MarshalIndent(c.state, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func (c *Collector) handshakePayload() models.AgentHandshakePayload {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	payload := models.AgentHandshakePayload{
		StreamID:         c.state.StreamID,
		RetiredWatermark: c.state.RetiredWatermark,
		ControlState:     c.state.ControlState,
		ControlReason:    c.state.ControlReason,
	}
	if c.state.InProgress != nil {
		payload.InProgress = &models.AgentInProgressState{
			JobID:          c.state.InProgress.JobID,
			Seq:            c.state.InProgress.Seq,
			Stage:          c.state.InProgress.Stage,
			TargetVersion:  c.state.InProgress.TargetVersion,
			TargetSHA256:   c.state.InProgress.TargetSHA256,
			LastProgressAt: c.state.InProgress.LastProgressAt,
			CommitPoint:    c.state.InProgress.CommitPoint,
			CancelIntent:   c.state.InProgress.CancelIntent,
		}
	}
	if len(c.state.Records) > 0 {
		payload.Records = make([]models.AgentExecutionRecord, 0, len(c.state.Records))
		for _, record := range c.state.Records {
			payload.Records = append(payload.Records, models.AgentExecutionRecord{
				Seq:             record.Seq,
				JobID:           record.JobID,
				Status:          record.Status,
				Message:         record.Message,
				ReportedVersion: record.ReportedVersion,
				TargetSHA256:    record.TargetSHA256,
				UpdatedAt:       record.UpdatedAt,
			})
		}
	}
	return payload
}

// applyHandshakeAck adopts the server's stream binding and watermark. The
// watermark can only move forward. A stream mismatch or a server-side control
// state isolates the agent locally (fail closed).
func (c *Collector) applyHandshakeAck(ack models.AgentHandshakeAckPayload) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if ack.ControlState == models.ControlStateQuarantined {
		c.state.ControlState = models.ControlStateQuarantined
		c.state.ControlReason = "server quarantined node: " + ack.Message
		_ = c.saveAgentState()
		return
	}
	if ack.ControlState == models.ControlStateBlocked {
		c.state.ControlState = models.ControlStateBlocked
		c.state.ControlReason = "server control-blocked: " + ack.Message
		_ = c.saveAgentState()
		return
	}

	if strings.TrimSpace(ack.StreamID) != "" {
		if c.state.StreamID == "" {
			c.state.StreamID = ack.StreamID
		} else if c.state.StreamID != ack.StreamID {
			// Defense in depth: the server already quarantined the node.
			c.state.ControlState = models.ControlStateQuarantined
			c.state.ControlReason = "stream mismatch with server"
			_ = c.saveAgentState()
			return
		}
	}
	if ack.RetiredWatermark > c.state.RetiredWatermark {
		c.state.RetiredWatermark = ack.RetiredWatermark
	}
	if c.state.ControlState == models.ControlStateBlocked {
		// A server ack with control_state=none is the audited operator
		// disposition of the outstanding unknown state.
		c.state.ControlState = models.ControlStateNone
		c.state.ControlReason = ""
	}
	_ = c.saveAgentState()
}

func commandDedupKey(cmd models.AgentCommandPayload) string {
	if cmd.Seq > 0 {
		return fmt.Sprintf("seq:%d", cmd.Seq)
	}
	if cmd.JobID != "" && cmd.JobID != "auto-update" {
		return "job:" + cmd.JobID
	}
	return "auto:" + strings.ToLower(strings.TrimSpace(cmd.SHA256))
}

// beginUpdateTask runs the admission control for the single global update
// executor: control state, JobID/seq dedup replay, sequence gating, and the
// durable accepted+in_progress record written before any side effect.
// Return values: ok=false with a replay/expired/busy outcome for reporting.
func (c *Collector) beginUpdateTask(cmd models.AgentCommandPayload) (ok bool, replay *executionRecord, rejectCode models.CommandRejectCode, rejectMsg string) {
	key := commandDedupKey(cmd)
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	switch c.state.ControlState {
	case models.ControlStateQuarantined:
		return false, nil, models.CommandRejectControlQuarantined, "agent quarantined: " + c.state.ControlReason
	case models.ControlStateBlocked:
		return false, nil, models.CommandRejectControlBlocked, "agent control-blocked: " + c.state.ControlReason
	}
	if cmd.Seq > 0 && (strings.TrimSpace(c.state.StreamID) == "" || strings.TrimSpace(cmd.StreamID) != strings.TrimSpace(c.state.StreamID)) {
		c.state.ControlState = models.ControlStateQuarantined
		c.state.ControlReason = "command stream mismatch"
		_ = c.saveAgentState()
		return false, nil, models.CommandRejectControlQuarantined, "command stream mismatch"
	}

	if cmd.Seq > 0 {
		switch {
		case cmd.Seq < c.state.RetiredWatermark:
			return false, nil, models.CommandRejectExpired, fmt.Sprintf("seq %d already retired (watermark %d)", cmd.Seq, c.state.RetiredWatermark)
		case cmd.Seq == c.state.RetiredWatermark:
			// Replay of the retired sequence: report its recorded outcome.
			if record := findRecord(c.state.Records, key); record != nil {
				return false, record, "", ""
			}
			return false, nil, models.CommandRejectAlreadyProcessed, fmt.Sprintf("seq %d already processed", cmd.Seq)
		case cmd.Seq > c.state.RetiredWatermark+1:
			return false, nil, models.CommandRejectSequenceGap, fmt.Sprintf("expected seq %d", c.state.RetiredWatermark+1)
		}
	}

	// Dedup: a repeated JobID/seq replays the exact recorded state.
	if record := findRecord(c.state.Records, key); record != nil {
		return false, record, "", ""
	}
	if c.state.InProgress != nil {
		if c.state.InProgress.Key == key {
			return false, &executionRecord{
				Key: c.state.InProgress.Key, Seq: c.state.InProgress.Seq, JobID: c.state.InProgress.JobID,
				Status:          c.state.InProgress.Stage,
				Message:         "already in progress",
				ReportedVersion: c.state.InProgress.TargetVersion,
				TargetSHA256:    c.state.InProgress.TargetSHA256,
				UpdatedAt:       c.state.InProgress.LastProgressAt,
			}, "", ""
		}
		return false, nil, "", errSelfUpdateBusy.Error()
	}

	now := c.now().Unix()
	c.state.InProgress = &inProgressState{
		Key:            key,
		Seq:            cmd.Seq,
		JobID:          cmd.JobID,
		Stage:          models.UpdateJobTargetStatusAccepted,
		TargetVersion:  cmd.TargetVersion,
		TargetSHA256:   strings.ToLower(strings.TrimSpace(cmd.SHA256)),
		StartedAt:      now,
		LastProgressAt: now,
	}
	c.state.Records = appendRecord(c.state.Records, executionRecord{
		Key: key, Seq: cmd.Seq, JobID: cmd.JobID,
		Status:       models.UpdateJobTargetStatusAccepted,
		Message:      "accepted",
		TargetSHA256: strings.ToLower(strings.TrimSpace(cmd.SHA256)),
		UpdatedAt:    now,
	})
	if err := c.saveAgentState(); err != nil {
		log.Printf("collector: persist accepted state failed: %v", err)
	}
	return true, nil, "", ""
}

func (c *Collector) updateTaskProgress(stage models.UpdateJobTargetStatus, message string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.state.InProgress == nil {
		return
	}
	c.state.InProgress.Stage = stage
	c.state.InProgress.LastProgressAt = c.now().Unix()
	record := findRecord(c.state.Records, c.state.InProgress.Key)
	if record == nil {
		return
	}
	record.Status = stage
	record.Message = message
	record.UpdatedAt = c.now().Unix()
	if c.state.InProgress.TargetVersion != "" {
		record.ReportedVersion = c.state.InProgress.TargetVersion
	}
	_ = c.saveAgentState()
}

func (c *Collector) finishUpdateTask(status models.UpdateJobTargetStatus, message string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.state.InProgress == nil {
		return
	}
	inProgress := c.state.InProgress
	record := executionRecord{
		Key: inProgress.Key, Seq: inProgress.Seq, JobID: inProgress.JobID,
		Status: status, Message: message, TargetSHA256: inProgress.TargetSHA256,
		UpdatedAt: c.now().Unix(),
	}
	if status == models.UpdateJobTargetStatusCancelled {
		record.Message = "cancelled before commit point"
	} else if inProgress.TargetVersion != "" {
		record.ReportedVersion = inProgress.TargetVersion
	}
	c.state.Records = appendRecord(c.state.Records, record)
	if status == models.UpdateJobTargetStatusUnknown {
		c.state.ControlState = models.ControlStateBlocked
		c.state.ControlReason = "undisposed unknown execution state"
	}
	if inProgress.Seq > 0 && status != models.UpdateJobTargetStatusAccepted {
		if inProgress.Seq > c.state.RetiredWatermark {
			c.state.RetiredWatermark = inProgress.Seq
		}
	}
	c.state.InProgress = nil
	_ = c.saveAgentState()
}

// runUpdateTask is the single serialized update executor used by remote
// commands and by auto-update. It holds the task's cancel context (used by
// cancel intents), runs selfUpdateFunc and records the durable outcome.
func (c *Collector) runUpdateTask(cmd models.AgentCommandPayload, report func(models.UpdateJobTargetStatus, string, string) error) error {
	taskCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.taskMu.Lock()
	c.taskCancel = cancel
	c.taskCtx = taskCtx
	c.taskMu.Unlock()
	defer func() {
		c.taskMu.Lock()
		c.taskCancel = nil
		c.taskCtx = nil
		c.taskMu.Unlock()
	}()

	if err := c.selfUpdateFunc(cmd, report); err != nil {
		c.stateMu.Lock()
		stillActive := c.state.InProgress != nil
		c.stateMu.Unlock()
		switch {
		case errors.Is(err, errSelfUpdateCancelled):
			c.finishUpdateTask(models.UpdateJobTargetStatusCancelled, err.Error())
			_ = report(models.UpdateJobTargetStatusCancelled, "cancelled before commit point", cmd.TargetVersion)
		case errors.Is(err, context.Canceled):
			c.finishUpdateTask(models.UpdateJobTargetStatusCancelled, "cancelled")
			_ = report(models.UpdateJobTargetStatusCancelled, "cancelled before commit point", cmd.TargetVersion)
		case !stillActive:
			// The update function already recorded a durable terminal outcome
			// (e.g. post-commit unknown + control-blocked).
			return err
		default:
			// A clean pre-commit failure.
			c.finishUpdateTask(models.UpdateJobTargetStatusFailed, err.Error())
			_ = report(models.UpdateJobTargetStatusFailed, err.Error(), cmd.TargetVersion)
		}
		return err
	}
	// selfUpdateFunc returned without Exec'ing (only possible with a stubbed
	// implementation or when the process was replaced); record success.
	c.finishUpdateTask(models.UpdateJobTargetStatusSucceeded, "self update completed")
	return nil
}

// cancelTask requests cancellation of the in-flight task when its sequence
// matches. Returns true when a matching task existed and was cancelled.
func (c *Collector) cancelTask(seq int64, jobID string) bool {
	c.stateMu.Lock()
	inProgress := c.state.InProgress
	c.stateMu.Unlock()
	if inProgress == nil {
		return false
	}
	if seq > 0 && inProgress.Seq != seq {
		return false
	}
	if jobID != "" && inProgress.JobID != jobID {
		return false
	}
	c.taskMu.Lock()
	cancel := c.taskCancel
	c.taskMu.Unlock()
	if cancel != nil {
		cancel()
	}
	return true
}

// markCommitPoint durably records that os.Rename is the next operation. Once
// this is persisted, cancellation is refused and the update must continue.
func (c *Collector) markCommitPoint() error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.state.InProgress == nil {
		return fmt.Errorf("no in-progress task to commit")
	}
	c.state.InProgress.CommitPoint = true
	return c.saveAgentState()
}
