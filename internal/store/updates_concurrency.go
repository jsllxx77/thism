package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

// This file implements the control-plane concurrency protocol: per-node
// command streams, monotonic sequence allocation, cancel tombstones, the
// two-dimensional delivery/execution state model and handshake
// reconciliation. All invariants here are documented in
// docs/concurrency-remediation-plan.md.

const (
	// CommandStreamDetailRetentionDays bounds the retention of detailed
	// update target rows.
	CommandStreamDetailRetentionDays = 180
	// CommandStreamDetailRowsPerNode bounds the number of detailed rows kept
	// per node (plan: 1024).
	CommandStreamDetailRowsPerNode = 1024
	// ControlAuditRetentionDays bounds control audit rows.
	ControlAuditRetentionDays = 365
	// ControlAuditRowsPerNode bounds audit rows per node.
	ControlAuditRowsPerNode = 2048
)

func newStreamID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("stream-%d", time.Now().UnixNano())
	}
	return "stream-" + hex.EncodeToString(raw)
}

func auditTx(tx *sql.Tx, nodeID, event, detail string) error {
	_, err := tx.Exec(`INSERT INTO control_audit_log (node_id, event, detail, created_at) VALUES (?, ?, ?, ?)`,
		nodeID, event, detail, time.Now().Unix())
	return err
}

// AuditNodeEvent records a control-plane audit event for a node.
func (s *Store) AuditNodeEvent(nodeID, event, detail string) error {
	_, err := s.db.Exec(`INSERT INTO control_audit_log (node_id, event, detail, created_at) VALUES (?, ?, ?, ?)`,
		nodeID, event, detail, time.Now().Unix())
	return err
}

// ListControlAudit returns the most recent audit events for a node.
func (s *Store) ListControlAudit(nodeID string, limit int) ([]models.AuditEvent, error) {
	if limit <= 0 || limit > ControlAuditRowsPerNode {
		limit = ControlAuditRowsPerNode
	}
	rows, err := s.db.Query(`SELECT id, node_id, event, detail, created_at FROM control_audit_log WHERE node_id = ? ORDER BY id DESC LIMIT ?`, nodeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.AuditEvent
	for rows.Next() {
		var event models.AuditEvent
		if err := rows.Scan(&event.ID, &event.NodeID, &event.Event, &event.Detail, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func ensureStreamTx(tx *sql.Tx, nodeID string) (*models.CommandStream, error) {
	stream := &models.CommandStream{NodeID: nodeID, StreamID: newStreamID()}
	err := tx.QueryRow(`SELECT stream_id, next_seq, retired_watermark, control_state, control_reason, updated_at FROM node_command_streams WHERE node_id = ?`, nodeID).
		Scan(&stream.StreamID, &stream.NextSeq, &stream.RetiredWatermark, &stream.ControlState, &stream.ControlReason, &stream.UpdatedAt)
	if err == nil {
		if stream.NextSeq <= 0 {
			stream.NextSeq = stream.RetiredWatermark + 1
			if _, updateErr := tx.Exec(`UPDATE node_command_streams SET next_seq = ?, updated_at = ? WHERE node_id = ?`, stream.NextSeq, time.Now().Unix(), nodeID); updateErr != nil {
				return nil, updateErr
			}
		}
		return stream, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	now := time.Now().Unix()
	stream.NextSeq = 1
	if _, err := tx.Exec(`INSERT INTO node_command_streams (node_id, stream_id, next_seq, retired_watermark, control_state, control_reason, updated_at) VALUES (?, ?, 1, 0, '', '', ?)`,
		nodeID, stream.StreamID, now); err != nil {
		return nil, err
	}
	if err := auditTx(tx, nodeID, "stream-created", "new command stream bound: "+stream.StreamID); err != nil {
		return nil, err
	}
	return stream, nil
}

// EnsureCommandStream returns the node's command stream, creating it with a
// fresh stream id and an audit event when it does not exist.
func (s *Store) EnsureCommandStream(nodeID string) (*models.CommandStream, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stream, err := ensureStreamTx(tx, nodeID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return stream, nil
}

// GetCommandStream returns the node's command stream or nil when absent.
func (s *Store) GetCommandStream(nodeID string) (*models.CommandStream, error) {
	var stream models.CommandStream
	err := s.db.QueryRow(`SELECT node_id, stream_id, next_seq, retired_watermark, control_state, control_reason, updated_at FROM node_command_streams WHERE node_id = ?`, nodeID).
		Scan(&stream.NodeID, &stream.StreamID, &stream.NextSeq, &stream.RetiredWatermark, &stream.ControlState, &stream.ControlReason, &stream.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &stream, nil
}

// SetNodeControlState quarantines or unblocks a node's control plane and
// records an audit event.
func (s *Store) SetNodeControlState(nodeID string, state models.ControlState, reason string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE node_command_streams SET control_state = ?, control_reason = ?, updated_at = ? WHERE node_id = ?`,
		state, reason, time.Now().Unix(), nodeID); err != nil {
		return err
	}
	if err := auditTx(tx, nodeID, "control-state", fmt.Sprintf("%s: %s", state, reason)); err != nil {
		return err
	}
	return tx.Commit()
}

// ReconcileCommandStream implements the handshake reconciliation rules:
//   - the agent reports its stream id and retired watermark;
//   - a matching stream whose server-side next_seq fell behind the agent
//     watermark (e.g. after a database restore) is transactionally advanced
//     and audited as rollback-reconciled;
//   - a missing server stream is re-created from the agent's id (audited);
//   - a stream id mismatch quarantines the node (fail closed);
//   - a quarantined/blocked node stays isolated.
func (s *Store) ReconcileCommandStream(nodeID, agentStreamID string, agentWatermark int64) (*models.CommandStream, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stream, err := ensureStreamTx(tx, nodeID)
	if err != nil {
		return nil, err
	}

	if stream.ControlState != models.ControlStateNone {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return stream, nil
	}

	if agentStreamID != "" && agentStreamID != stream.StreamID {
		// Mismatch: fail closed and isolate the node. Never reset the agent.
		if _, err := tx.Exec(`UPDATE node_command_streams SET control_state = ?, control_reason = ?, updated_at = ? WHERE node_id = ?`,
			models.ControlStateQuarantined, "stream mismatch: agent has "+agentStreamID+", server has "+stream.StreamID, time.Now().Unix(), nodeID); err != nil {
			return nil, err
		}
		if err := auditTx(tx, nodeID, "stream-mismatch", "agent stream "+agentStreamID+" does not match server stream "+stream.StreamID+"; node quarantined"); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		stream.ControlState = models.ControlStateQuarantined
		stream.ControlReason = "stream mismatch"
		return stream, nil
	}

	if agentStreamID != "" && agentStreamID == stream.StreamID && stream.NextSeq <= agentWatermark {
		// Database rollback: the agent has retired sequences the server no
		// longer believes exist. Advance next_seq transactionally; never
		// reuse sequences.
		from := stream.NextSeq
		stream.NextSeq = agentWatermark + 1
		if _, err := tx.Exec(`UPDATE node_command_streams SET next_seq = ?, updated_at = ? WHERE node_id = ?`,
			stream.NextSeq, time.Now().Unix(), nodeID); err != nil {
			return nil, err
		}
		detail := fmt.Sprintf("advanced next_seq %d -> %d from agent watermark %d", from, stream.NextSeq, agentWatermark)
		if err := auditTx(tx, nodeID, "rollback-reconciled", detail); err != nil {
			return nil, err
		}
	}

	if agentStreamID == "" {
		// Fresh agent (no local stream yet): bind the server stream id.
		if err := auditTx(tx, nodeID, "stream-bound", "agent adopted server stream "+stream.StreamID); err != nil {
			return nil, err
		}
	}

	// Persist the greatest trusted watermark from either side. The server
	// computes its own contiguous terminal/tombstone prefix so a node is
	// never sent seq N while seq N-1 is already retired server-side.
	serverWatermark, err := retiredWatermarkFrom(tx, nodeID)
	if err != nil {
		return nil, err
	}
	retiredWatermark := stream.RetiredWatermark
	if agentWatermark > retiredWatermark {
		retiredWatermark = agentWatermark
	}
	if serverWatermark > retiredWatermark {
		retiredWatermark = serverWatermark
	}
	if retiredWatermark > stream.RetiredWatermark {
		if _, err := tx.Exec(`UPDATE node_command_streams SET retired_watermark = ?, updated_at = ? WHERE node_id = ?`,
			retiredWatermark, time.Now().Unix(), nodeID); err != nil {
			return nil, err
		}
		stream.RetiredWatermark = retiredWatermark
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return stream, nil
}

// CreateUpdateJobWithTargets creates a job and its targets in a single
// transaction, allocating one monotonic sequence per node from its command
// stream. Allocation and creation cannot be split: a rolled-back transaction
// never leaves an allocated-but-jobless sequence.
func (s *Store) CreateUpdateJobWithTargets(job *models.UpdateJob, nodeIDs []string) ([]models.UpdateJobTarget, error) {
	if job == nil {
		return nil, fmt.Errorf("nil update job")
	}
	createdAt := job.CreatedAt
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	updatedAt := job.UpdatedAt
	if updatedAt <= 0 {
		updatedAt = createdAt
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
INSERT INTO update_jobs (id, kind, target_version, download_url, sha256, signature, created_at, updated_at, created_by, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, job.ID, job.Kind, job.TargetVersion, job.DownloadURL, job.SHA256, job.Signature, createdAt, updatedAt, job.CreatedBy, job.Status); err != nil {
		return nil, err
	}

	targets := make([]models.UpdateJobTarget, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		stream, err := ensureStreamTx(tx, nodeID)
		if err != nil {
			return nil, err
		}
		seq := stream.NextSeq
		stream.NextSeq = seq + 1
		if _, err := tx.Exec(`UPDATE node_command_streams SET next_seq = ?, updated_at = ? WHERE node_id = ?`,
			stream.NextSeq, time.Now().Unix(), nodeID); err != nil {
			return nil, err
		}
		target := models.UpdateJobTarget{
			JobID:          job.ID,
			NodeID:         nodeID,
			Status:         models.UpdateJobTargetStatusPending,
			UpdatedAt:      updatedAt,
			Seq:            seq,
			StreamID:       stream.StreamID,
			DeliveryState:  models.DeliveryStatePending,
			TargetSHA256:   job.SHA256,
			LastProgressAt: updatedAt,
		}
		if _, err := tx.Exec(`
INSERT INTO update_job_targets (job_id, node_id, status, message, updated_at, reported_version, seq, stream_id, delivery_state, target_sha256, last_progress_at)
VALUES (?, ?, ?, '', ?, '', ?, ?, ?, ?, ?)
`, target.JobID, target.NodeID, target.Status, target.UpdatedAt, target.Seq, target.StreamID, target.DeliveryState, target.TargetSHA256, target.LastProgressAt); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return targets, nil
}

// RecordDelivery writes the server-owned delivery fact for a target. It never
// touches the agent-owned execution status.
func (s *Store) RecordDelivery(jobID, nodeID string, state models.DeliveryState, errMsg string, generation int64) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`
UPDATE update_job_targets
SET delivery_state = ?, delivery_error = ?, delivery_attempted_at = ?, delivery_generation = ?,
    delivery_attempts = delivery_attempts + CASE WHEN ? = 'pending' OR ? = 'sending' THEN 1 ELSE 0 END,
    updated_at = ?
WHERE job_id = ? AND node_id = ?
`, state, errMsg, now, generation, state, state, now, jobID, nodeID)
	return err
}

// executionTransitions is the legal monotonic execution state machine. The
// agent owns this dimension; the server only commits transitions that appear
// here. Same-state reports are idempotent. Terminal states are sticky.
var executionTransitions = map[models.UpdateJobTargetStatus][]models.UpdateJobTargetStatus{
	models.UpdateJobTargetStatusPending: {
		models.UpdateJobTargetStatusDispatched,
		models.UpdateJobTargetStatusAccepted,
		models.UpdateJobTargetStatusDownloading,
		models.UpdateJobTargetStatusVerifying,
		models.UpdateJobTargetStatusRestarting,
		models.UpdateJobTargetStatusSucceeded,
		models.UpdateJobTargetStatusFailed,
		models.UpdateJobTargetStatusUnknown,
		models.UpdateJobTargetStatusTimeout,
		models.UpdateJobTargetStatusCancelled,
		models.UpdateJobTargetStatusOfflineSkipped,
	},
	models.UpdateJobTargetStatusDispatched: {
		models.UpdateJobTargetStatusAccepted,
		models.UpdateJobTargetStatusDownloading,
		models.UpdateJobTargetStatusVerifying,
		models.UpdateJobTargetStatusRestarting,
		models.UpdateJobTargetStatusSucceeded,
		models.UpdateJobTargetStatusFailed,
		models.UpdateJobTargetStatusUnknown,
		models.UpdateJobTargetStatusTimeout,
		models.UpdateJobTargetStatusCancelled,
	},
	models.UpdateJobTargetStatusAccepted: {
		models.UpdateJobTargetStatusDownloading,
		models.UpdateJobTargetStatusSucceeded,
		models.UpdateJobTargetStatusFailed,
		models.UpdateJobTargetStatusUnknown,
		models.UpdateJobTargetStatusTimeout,
		models.UpdateJobTargetStatusCancelled,
	},
	models.UpdateJobTargetStatusDownloading: {
		models.UpdateJobTargetStatusVerifying,
		models.UpdateJobTargetStatusSucceeded,
		models.UpdateJobTargetStatusFailed,
		models.UpdateJobTargetStatusUnknown,
		models.UpdateJobTargetStatusTimeout,
		models.UpdateJobTargetStatusCancelled,
	},
	models.UpdateJobTargetStatusVerifying: {
		models.UpdateJobTargetStatusRestarting,
		models.UpdateJobTargetStatusSucceeded,
		models.UpdateJobTargetStatusFailed,
		models.UpdateJobTargetStatusUnknown,
		models.UpdateJobTargetStatusTimeout,
		models.UpdateJobTargetStatusCancelled,
	},
	models.UpdateJobTargetStatusRestarting: {
		models.UpdateJobTargetStatusSucceeded,
		models.UpdateJobTargetStatusFailed,
		models.UpdateJobTargetStatusUnknown,
		models.UpdateJobTargetStatusTimeout,
		models.UpdateJobTargetStatusCancelled,
	},
	// Sticky terminal states.
	models.UpdateJobTargetStatusSucceeded:      {},
	models.UpdateJobTargetStatusFailed:         {},
	models.UpdateJobTargetStatusUnknown:        {},
	models.UpdateJobTargetStatusTimeout:        {},
	models.UpdateJobTargetStatusCancelled:      {},
	models.UpdateJobTargetStatusOfflineSkipped: {},
}

// terminalExecutionStatuses are sticky terminal execution states. Only these
// (or a same-seq cancel tombstone) may retire a sequence.
func terminalExecutionStatuses() map[models.UpdateJobTargetStatus]bool {
	return map[models.UpdateJobTargetStatus]bool{
		models.UpdateJobTargetStatusSucceeded:      true,
		models.UpdateJobTargetStatusFailed:         true,
		models.UpdateJobTargetStatusUnknown:        true,
		models.UpdateJobTargetStatusTimeout:        true,
		models.UpdateJobTargetStatusCancelled:      true,
		models.UpdateJobTargetStatusOfflineSkipped: true,
	}
}

func inProgressExecutionStatus(status models.UpdateJobTargetStatus) bool {
	switch status {
	case models.UpdateJobTargetStatusAccepted,
		models.UpdateJobTargetStatusDownloading,
		models.UpdateJobTargetStatusVerifying,
		models.UpdateJobTargetStatusRestarting:
		return true
	}
	return false
}

// ApplyExecutionTransition is the CAS/legal-transition entry point for the
// agent-owned execution dimension. The state check, transition and job
// aggregation happen in one transaction. It returns whether the transition
// was committed (a rejected transition is not an error).
func (s *Store) ApplyExecutionTransition(jobID, nodeID string, status models.UpdateJobTargetStatus, message, reportedVersion string) (bool, error) {
	now := time.Now().Unix()
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var current models.UpdateJobTargetStatus
	var seq int64
	err = tx.QueryRow(`SELECT status, seq FROM update_job_targets WHERE job_id = ? AND node_id = ?`, jobID, nodeID).
		Scan(&current, &seq)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if current != status {
		allowed := false
		for _, candidate := range executionTransitions[current] {
			if candidate == status {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, nil
		}
	}

	lastProgressAt := ""
	if inProgressExecutionStatus(status) {
		lastProgressAt = fmt.Sprintf("%d", now)
	}
	cancelledAt := ""
	if status == models.UpdateJobTargetStatusCancelled {
		cancelledAt = fmt.Sprintf("%d", now)
	}
	if _, err := tx.Exec(`
UPDATE update_job_targets
SET status = ?, message = ?,
    reported_version = CASE WHEN ? != '' THEN ? ELSE reported_version END,
    updated_at = ?,
    last_progress_at = CASE WHEN ? != '' THEN ? ELSE last_progress_at END,
    cancelled_at = CASE WHEN ? != '' THEN ? ELSE cancelled_at END
WHERE job_id = ? AND node_id = ?
`, status, message, reportedVersion, reportedVersion, now, lastProgressAt, lastProgressAt, cancelledAt, cancelledAt, jobID, nodeID); err != nil {
		return false, err
	}

	// An undisposed unknown blocks the node's control plane (fail closed).
	if status == models.UpdateJobTargetStatusUnknown && seq > 0 {
		if _, err := tx.Exec(`UPDATE node_command_streams SET control_state = ?, control_reason = ?, updated_at = ? WHERE node_id = ?`,
			models.ControlStateBlocked, "undisposed unknown execution state", now, nodeID); err != nil {
			return false, err
		}
		if err := auditTx(tx, nodeID, "unknown-execution", fmt.Sprintf("job %s seq %d reported unknown; node control-blocked", jobID, seq)); err != nil {
			return false, err
		}
	}
	if terminalExecutionStatuses()[status] && seq > 0 {
		if err := advanceRetiredWatermarkTx(tx, nodeID, now); err != nil {
			return false, err
		}
	}

	if err := recomputeUpdateJobStatusTx(tx, jobID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// ApplyHandshakeBackfill applies agent-verified execution facts to targets
// after a handshake. Facts can only move execution forward (monotonic) and
// only when the agent's seq matches the target's seq. It never infers success
// from missing data.
func (s *Store) ApplyHandshakeBackfill(nodeID string, inProgress *models.AgentInProgressState, records []models.AgentExecutionRecord) error {
	now := time.Now().Unix()
	for _, record := range records {
		if record.Seq <= 0 || record.JobID == "" {
			continue
		}
		var current models.UpdateJobTargetStatus
		var targetSeq int64
		err := s.db.QueryRow(`SELECT status, seq FROM update_job_targets WHERE job_id = ? AND node_id = ?`, record.JobID, nodeID).
			Scan(&current, &targetSeq)
		if err == sql.ErrNoRows {
			// The job record no longer exists on the server (e.g. it was
			// pruned or the database was restored). We cannot reconstruct the
			// command, so we do not fabricate facts.
			continue
		}
		if err != nil {
			return err
		}
		if targetSeq != record.Seq {
			continue
		}
		status := record.Status
		allowed := false
		if current == status {
			allowed = true
		} else {
			for _, candidate := range executionTransitions[current] {
				if candidate == status {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			continue
		}
		if _, err := s.db.Exec(`UPDATE update_job_targets SET status = ?, message = ?, reported_version = ?, updated_at = ?, last_progress_at = ? WHERE job_id = ? AND node_id = ?`,
			status, record.Message, record.ReportedVersion, now, record.UpdatedAt, record.JobID, nodeID); err != nil {
			return err
		}
		if status == models.UpdateJobTargetStatusUnknown {
			if _, err := s.db.Exec(`UPDATE node_command_streams SET control_state = ?, control_reason = ?, updated_at = ? WHERE node_id = ?`,
				models.ControlStateBlocked, "undisposed unknown execution state (handshake backfill)", now, nodeID); err != nil {
				return err
			}
			_ = s.AuditNodeEvent(nodeID, "unknown-execution", fmt.Sprintf("job %s seq %d backfilled unknown from agent handshake; node control-blocked", record.JobID, record.Seq))
		}
		_, _ = s.RecomputeUpdateJobStatus(record.JobID)
	}

	if inProgress != nil && inProgress.Seq > 0 && inProgress.JobID != "" {
		var current models.UpdateJobTargetStatus
		var targetSeq int64
		err := s.db.QueryRow(`SELECT status, seq FROM update_job_targets WHERE job_id = ? AND node_id = ?`, inProgress.JobID, nodeID).
			Scan(&current, &targetSeq)
		if err == nil && targetSeq == inProgress.Seq {
			allowed := false
			if current == inProgress.Stage {
				allowed = true
			} else {
				for _, candidate := range executionTransitions[current] {
					if candidate == inProgress.Stage {
						allowed = true
						break
					}
				}
			}
			if allowed {
				if _, err := s.db.Exec(`UPDATE update_job_targets SET status = ?, last_progress_at = ? WHERE job_id = ? AND node_id = ?`,
					inProgress.Stage, inProgress.LastProgressAt, inProgress.JobID, nodeID); err != nil {
					return err
				}
				_, _ = s.RecomputeUpdateJobStatus(inProgress.JobID)
			}
		}
	}
	return nil
}

// CreateCancelTombstone writes the persistent cancel intent for an assigned
// sequence and marks the target. Cancelling never deletes the assignment; the
// tombstone closes the sequence.
func (s *Store) CreateCancelTombstone(jobID, nodeID string, seq int64) error {
	now := time.Now().Unix()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT OR IGNORE INTO update_cancel_tombstones (node_id, seq, job_id, created_at) VALUES (?, ?, ?, ?)`,
		nodeID, seq, jobID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE update_job_targets SET cancel_intent_at = ? WHERE job_id = ? AND node_id = ?`,
		now, jobID, nodeID); err != nil {
		return err
	}
	if err := advanceRetiredWatermarkTx(tx, nodeID, now); err != nil {
		return err
	}
	if err := auditTx(tx, nodeID, "cancel-tombstone", fmt.Sprintf("job %s seq %d cancel intent recorded", jobID, seq)); err != nil {
		return err
	}
	return tx.Commit()
}

type storeQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// retiredWatermarkFrom returns the highest contiguous sequence for which the
// server has a terminal execution fact or a cancel tombstone. A later terminal
// sequence cannot skip an earlier non-terminal sequence.
func retiredWatermarkFrom(q storeQueryer, nodeID string) (int64, error) {
	rows, err := q.Query(`
SELECT seq FROM update_job_targets
WHERE node_id = ? AND seq > 0 AND status IN (?, ?, ?, ?, ?, ?)
UNION
SELECT seq FROM update_cancel_tombstones WHERE node_id = ? AND seq > 0
UNION
SELECT seq FROM update_sequence_retirements WHERE node_id = ? AND seq > 0
ORDER BY seq ASC
`, nodeID,
		models.UpdateJobTargetStatusSucceeded,
		models.UpdateJobTargetStatusFailed,
		models.UpdateJobTargetStatusUnknown,
		models.UpdateJobTargetStatusTimeout,
		models.UpdateJobTargetStatusCancelled,
		models.UpdateJobTargetStatusOfflineSkipped,
		nodeID,
		nodeID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	watermark := int64(0)
	expected := int64(1)
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err != nil {
			return 0, err
		}
		if seq == expected {
			watermark = seq
			expected++
			continue
		}
		if seq > expected {
			break
		}
	}
	return watermark, rows.Err()
}

func advanceRetiredWatermarkTx(tx *sql.Tx, nodeID string, now int64) error {
	watermark, err := retiredWatermarkFrom(tx, nodeID)
	if err != nil {
		return err
	}
	if watermark == 0 {
		return nil
	}
	_, err = tx.Exec(`
UPDATE node_command_streams
SET retired_watermark = CASE WHEN retired_watermark < ? THEN ? ELSE retired_watermark END,
    updated_at = ?
WHERE node_id = ?
`, watermark, watermark, now, nodeID)
	return err
}

// RetiredWatermarkForNode returns the highest contiguous sequence retired for
// a node: terminal execution states and cancel tombstones.
func (s *Store) RetiredWatermarkForNode(nodeID string) (int64, error) {
	return retiredWatermarkFrom(s.db, nodeID)
}

// PendingDeliveriesForNode returns commands whose delivery has not concluded
// (pending, sending, sent or send_failed) and whose execution has not started,
// with seq greater than the given watermark, ordered by seq. The caller sends
// them in order; the agent enforces seq = retired_watermark + 1.
func (s *Store) PendingDeliveriesForNode(nodeID string, afterWatermark int64, limit int) ([]*models.UpdateJobTarget, error) {
	if limit <= 0 {
		limit = 64
	}
	rows, err := s.db.Query(`
SELECT t.job_id, t.node_id, t.status, t.message, t.updated_at, t.reported_version,
       t.seq, t.stream_id, t.delivery_state, t.delivery_error, t.delivery_attempts,
       t.delivery_attempted_at, t.delivery_generation, t.target_sha256, t.observation,
       t.cancel_intent_at, t.cancelled_at, t.last_progress_at
FROM update_job_targets t
WHERE t.node_id = ?
  AND t.seq > ?
  AND t.status IN (?, ?)
  AND t.delivery_state IN (?, ?, ?, ?)
ORDER BY t.seq ASC
LIMIT ?
`, nodeID, afterWatermark,
		models.UpdateJobTargetStatusPending,
		models.UpdateJobTargetStatusDispatched,
		models.DeliveryStatePending,
		models.DeliveryStateSending,
		models.DeliveryStateSent,
		models.DeliveryStateSendFailed,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []*models.UpdateJobTarget
	for rows.Next() {
		target := &models.UpdateJobTarget{}
		if err := rows.Scan(&target.JobID, &target.NodeID, &target.Status, &target.Message, &target.UpdatedAt, &target.ReportedVersion,
			&target.Seq, &target.StreamID, &target.DeliveryState, &target.DeliveryError, &target.DeliveryAttempts,
			&target.DeliveryAttemptedAt, &target.DeliveryGeneration, &target.TargetSHA256, &target.Observation,
			&target.CancelIntentAt, &target.CancelledAt, &target.LastProgressAt); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

// GetUpdateJobTarget returns a single target with all protocol fields.
func (s *Store) GetUpdateJobTarget(jobID, nodeID string) (*models.UpdateJobTarget, error) {
	row := s.db.QueryRow(`
SELECT job_id, node_id, status, message, updated_at, reported_version,
       seq, stream_id, delivery_state, delivery_error, delivery_attempts,
       delivery_attempted_at, delivery_generation, target_sha256, observation,
       cancel_intent_at, cancelled_at, last_progress_at
FROM update_job_targets WHERE job_id = ? AND node_id = ?
`, jobID, nodeID)
	target := &models.UpdateJobTarget{}
	if err := row.Scan(&target.JobID, &target.NodeID, &target.Status, &target.Message, &target.UpdatedAt, &target.ReportedVersion,
		&target.Seq, &target.StreamID, &target.DeliveryState, &target.DeliveryError, &target.DeliveryAttempts,
		&target.DeliveryAttemptedAt, &target.DeliveryGeneration, &target.TargetSHA256, &target.Observation,
		&target.CancelIntentAt, &target.CancelledAt, &target.LastProgressAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return target, nil
}

// ResolveUnknownTarget is the audited operator disposition of an unknown
// execution state. The resolution must be a terminal state and requires an
// evidence string (artifact SHA256 or operator note). After resolution the
// node's control plane is unblocked if no other undisposed unknown remains.
func (s *Store) ResolveUnknownTarget(jobID, nodeID string, resolution models.UpdateJobTargetStatus, evidence, reason string) (bool, error) {
	if resolution != models.UpdateJobTargetStatusSucceeded &&
		resolution != models.UpdateJobTargetStatusFailed &&
		resolution != models.UpdateJobTargetStatusCancelled {
		return false, fmt.Errorf("invalid unknown resolution %q: must be succeeded, failed or cancelled", resolution)
	}
	if strings.TrimSpace(evidence) == "" && strings.TrimSpace(reason) == "" {
		return false, fmt.Errorf("unknown resolution requires evidence or a reason")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var current models.UpdateJobTargetStatus
	if err := tx.QueryRow(`SELECT status FROM update_job_targets WHERE job_id = ? AND node_id = ?`, jobID, nodeID).Scan(&current); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if current != models.UpdateJobTargetStatusUnknown {
		return false, nil
	}
	if _, err := tx.Exec(`UPDATE update_job_targets SET status = ?, message = ?, updated_at = ? WHERE job_id = ? AND node_id = ?`,
		resolution, reason, time.Now().Unix(), jobID, nodeID); err != nil {
		return false, err
	}
	detail := fmt.Sprintf("job %s resolved unknown -> %s (evidence: %s, reason: %s)", jobID, resolution, evidence, reason)
	if err := auditTx(tx, nodeID, "unknown-disposition", detail); err != nil {
		return false, err
	}

	// Unblock only when no other undisposed unknown remains for the node.
	var remaining sql.NullInt64
	if err := tx.QueryRow(`SELECT COUNT(*) FROM update_job_targets WHERE node_id = ? AND status = ?`, nodeID, models.UpdateJobTargetStatusUnknown).Scan(&remaining); err != nil {
		return false, err
	}
	if !remaining.Valid || remaining.Int64 == 0 {
		if _, err := tx.Exec(`UPDATE node_command_streams SET control_state = ?, control_reason = ?, updated_at = ? WHERE node_id = ?`,
			models.ControlStateNone, "", time.Now().Unix(), nodeID); err != nil {
			return false, err
		}
		if err := auditTx(tx, nodeID, "control-state", "control-blocked cleared after unknown disposition"); err != nil {
			return false, err
		}
	}
	if err := recomputeUpdateJobStatusTx(tx, jobID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// MarkStalledUpdateObservations flags in-progress executions whose progress
// lease has expired. It only writes the observation dimension; the agent owns
// the execution transition.
func (s *Store) MarkStalledUpdateObservations(staleAfter time.Duration) (int, error) {
	cutoff := time.Now().Add(-staleAfter).Unix()
	result, err := s.db.Exec(`
UPDATE update_job_targets
SET observation = ?
WHERE observation != ?
  AND status IN (?, ?, ?, ?)
  AND last_progress_at > 0
  AND last_progress_at < ?
`, models.ObservationStalled, models.ObservationStalled,
		models.UpdateJobTargetStatusAccepted,
		models.UpdateJobTargetStatusDownloading,
		models.UpdateJobTargetStatusVerifying,
		models.UpdateJobTargetStatusRestarting,
		cutoff)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	return int(affected), err
}

type pruneCandidate struct {
	NodeID string
	JobID  string
	Seq    int64
}

// pruneTargetRow deletes a detail row and closes its sequence with a cancel
// tombstone so the agent's retired watermark can fast-forward over it and no
// permanent sequence gap appears.
func (s *Store) pruneTargetRow(candidate pruneCandidate) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if candidate.Seq > 0 {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO update_sequence_retirements (node_id, seq, job_id, reason, created_at) VALUES (?, ?, ?, ?, ?)`,
			candidate.NodeID, candidate.Seq, candidate.JobID, "detail-pruned", time.Now().Unix()); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM update_job_targets WHERE job_id = ? AND node_id = ?`, candidate.JobID, candidate.NodeID); err != nil {
		return err
	}
	if err := advanceRetiredWatermarkTx(tx, candidate.NodeID, time.Now().Unix()); err != nil {
		return err
	}
	return tx.Commit()
}

// PruneUpdateDetail implements bounded detail retention: at most
// CommandStreamDetailRowsPerNode rows and CommandStreamDetailRetentionDays of
// age per node. In-progress, unknown and audit records are never silently
// deleted.
func (s *Store) PruneUpdateDetail() (int, error) {
	now := time.Now().Unix()
	cutoff := now - CommandStreamDetailRetentionDays*24*3600
	pruned := 0

	// Age-based pruning, excluding protected rows (in-progress, unknown).
	rows, err := s.db.Query(`
SELECT node_id, job_id, seq FROM update_job_targets
WHERE updated_at < ?
  AND status NOT IN (?, ?, ?, ?, ?)
`, cutoff,
		models.UpdateJobTargetStatusAccepted,
		models.UpdateJobTargetStatusDownloading,
		models.UpdateJobTargetStatusVerifying,
		models.UpdateJobTargetStatusRestarting,
		models.UpdateJobTargetStatusUnknown)
	if err != nil {
		return pruned, err
	}
	var aged []pruneCandidate
	for rows.Next() {
		var candidate pruneCandidate
		if err := rows.Scan(&candidate.NodeID, &candidate.JobID, &candidate.Seq); err != nil {
			rows.Close()
			return pruned, err
		}
		aged = append(aged, candidate)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return pruned, err
	}
	for _, candidate := range aged {
		if err := s.pruneTargetRow(candidate); err != nil {
			return pruned, err
		}
		pruned++
	}

	// Per-node row cap, keeping the newest rows and protected rows.
	capRows, err := s.db.Query(`
SELECT node_id FROM update_job_targets GROUP BY node_id
HAVING COUNT(*) > ?
`, CommandStreamDetailRowsPerNode)
	if err != nil {
		return pruned, err
	}
	var nodeIDs []string
	for capRows.Next() {
		var nodeID string
		if err := capRows.Scan(&nodeID); err != nil {
			capRows.Close()
			return pruned, err
		}
		nodeIDs = append(nodeIDs, nodeID)
	}
	capRows.Close()
	if err := capRows.Err(); err != nil {
		return pruned, err
	}

	for _, nodeID := range nodeIDs {
		excess, err := s.db.Query(`
SELECT job_id, seq FROM update_job_targets
WHERE node_id = ?
  AND status NOT IN (?, ?, ?, ?, ?)
ORDER BY updated_at ASC
`, nodeID,
			models.UpdateJobTargetStatusAccepted,
			models.UpdateJobTargetStatusDownloading,
			models.UpdateJobTargetStatusVerifying,
			models.UpdateJobTargetStatusRestarting,
			models.UpdateJobTargetStatusUnknown)
		if err != nil {
			return pruned, err
		}
		var candidates []pruneCandidate
		for excess.Next() {
			var candidate pruneCandidate
			candidate.NodeID = nodeID
			if err := excess.Scan(&candidate.JobID, &candidate.Seq); err != nil {
				excess.Close()
				return pruned, err
			}
			candidates = append(candidates, candidate)
		}
		excess.Close()
		if err := excess.Err(); err != nil {
			return pruned, err
		}
		// Keep the newest CommandStreamDetailRowsPerNode rows.
		toDelete := len(candidates) - CommandStreamDetailRowsPerNode
		for i := 0; i < toDelete; i++ {
			if err := s.pruneTargetRow(candidates[i]); err != nil {
				return pruned, err
			}
			pruned++
		}
	}

	// Bounded audit retention.
	if _, err := s.db.Exec(`DELETE FROM control_audit_log WHERE created_at < ?`, now-ControlAuditRetentionDays*24*3600); err != nil {
		return pruned, err
	}
	auditRows, err := s.db.Query(`SELECT node_id FROM control_audit_log GROUP BY node_id HAVING COUNT(*) > ?`, ControlAuditRowsPerNode)
	if err != nil {
		return pruned, err
	}
	var auditNodeIDs []string
	for auditRows.Next() {
		var nodeID string
		if err := auditRows.Scan(&nodeID); err != nil {
			auditRows.Close()
			return pruned, err
		}
		auditNodeIDs = append(auditNodeIDs, nodeID)
	}
	auditRows.Close()
	if err := auditRows.Err(); err != nil {
		return pruned, err
	}
	for _, nodeID := range auditNodeIDs {
		if _, err := s.db.Exec(`
DELETE FROM control_audit_log WHERE id IN (
  SELECT id FROM control_audit_log WHERE node_id = ? ORDER BY id ASC LIMIT ?
)`, nodeID, ControlAuditRowsPerNode); err != nil {
			return pruned, err
		}
	}
	return pruned, nil
}
