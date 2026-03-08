package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thism-dev/thism/internal/models"
	_ "modernc.org/sqlite"
)

// Store is the SQLite-backed data access layer.
type Store struct {
	db *sql.DB
}

const DefaultMetricsRetentionDays = 7

const metricsRetentionSettingKey = "metrics_retention_days"

var metricsRetentionOptions = []int{7, 30}

func MetricsRetentionOptions() []int {
	options := make([]int, len(metricsRetentionOptions))
	copy(options, metricsRetentionOptions)
	return options
}

func IsValidMetricsRetentionDays(days int) bool {
	for _, candidate := range metricsRetentionOptions {
		if candidate == days {
			return true
		}
	}
	return false
}

// MetricsRow is a flat struct representing a single metrics sample for API use.
type MetricsRow struct {
	TS            int64   `json:"ts"`
	CPU           float64 `json:"cpu"`
	MemUsed       uint64  `json:"mem_used"`
	MemTotal      uint64  `json:"mem_total"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	NetRx         uint64  `json:"net_rx"`
	NetTx         uint64  `json:"net_tx"`
	UptimeSeconds uint64  `json:"uptime_seconds,omitempty"`
}

// New opens (or creates) the SQLite database at the given path, runs migrations,
// and returns a ready-to-use Store.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// SQLite only supports a single writer at a time.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates all required tables if they do not already exist.
func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS nodes (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    token         TEXT NOT NULL UNIQUE,
    ip            TEXT DEFAULT '',
    os            TEXT DEFAULT '',
    arch          TEXT DEFAULT '',
    agent_version TEXT DEFAULT '',
    hardware_json TEXT DEFAULT '',
    created_at    INTEGER NOT NULL DEFAULT 0,
    last_seen     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS metrics (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id     TEXT NOT NULL,
    ts          INTEGER NOT NULL,
    cpu_percent REAL DEFAULT 0,
    mem_used    INTEGER DEFAULT 0,
    mem_total   INTEGER DEFAULT 0,
    disk_used   INTEGER DEFAULT 0,
    disk_total  INTEGER DEFAULT 0,
    net_rx         INTEGER DEFAULT 0,
    net_tx         INTEGER DEFAULT 0,
    uptime_seconds INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_metrics_node_ts ON metrics(node_id, ts);

CREATE TABLE IF NOT EXISTS processes (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL UNIQUE,
    ts      INTEGER NOT NULL,
    data    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS service_checks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id      TEXT NOT NULL,
    name         TEXT NOT NULL,
    status       TEXT DEFAULT 'unknown',
    last_checked INTEGER NOT NULL DEFAULT 0,
    UNIQUE(node_id, name)
);

CREATE TABLE IF NOT EXISTS admin_auth (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    username   TEXT NOT NULL,
    password   TEXT NOT NULL,
    updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS update_jobs (
    id             TEXT PRIMARY KEY,
    kind           TEXT NOT NULL,
    target_version TEXT NOT NULL,
    download_url   TEXT NOT NULL,
    sha256         TEXT NOT NULL,
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL DEFAULT 0,
    created_by     TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS update_job_targets (
    job_id           TEXT NOT NULL,
    node_id          TEXT NOT NULL,
    status           TEXT NOT NULL,
    message          TEXT NOT NULL DEFAULT '',
    updated_at       INTEGER NOT NULL DEFAULT 0,
    reported_version TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (job_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_update_job_targets_job_id ON update_job_targets(job_id);

CREATE TABLE IF NOT EXISTS app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at INTEGER NOT NULL DEFAULT 0
);
`)
	if err != nil {
		return err
	}

	if err := s.ensureColumn("nodes", "hardware_json", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("nodes", "agent_version", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("metrics", "uptime_seconds", "INTEGER DEFAULT 0"); err != nil {
		return err
	}
	return s.ensureColumn("update_jobs", "updated_at", "INTEGER NOT NULL DEFAULT 0")
}

func (s *Store) ensureColumn(table, column, definition string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = s.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

func (s *Store) GetMetricsRetentionDays() (int, error) {
	var raw string
	err := s.db.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, metricsRetentionSettingKey).Scan(&raw)
	if err == sql.ErrNoRows {
		return DefaultMetricsRetentionDays, nil
	}
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || !IsValidMetricsRetentionDays(value) {
		return DefaultMetricsRetentionDays, nil
	}
	return value, nil
}

func (s *Store) SetMetricsRetentionDays(days int) error {
	if !IsValidMetricsRetentionDays(days) {
		return fmt.Errorf("invalid metrics retention days")
	}
	_, err := s.db.Exec(`
INSERT INTO app_settings (key, value, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET
	value = excluded.value,
	updated_at = excluded.updated_at
`, metricsRetentionSettingKey, strconv.Itoa(days), time.Now().Unix())
	return err
}

// GetAdminAuth returns the persisted admin username/password pair.
// found is false when no persisted credentials exist yet.
func (s *Store) GetAdminAuth() (username, password string, found bool, err error) {
	row := s.db.QueryRow(`SELECT username, password FROM admin_auth WHERE id = 1`)

	if err := row.Scan(&username, &password); err != nil {
		if err == sql.ErrNoRows {
			return "", "", false, nil
		}
		return "", "", false, err
	}
	return username, password, true, nil
}

// UpsertAdminAuth persists admin login credentials.
func (s *Store) UpsertAdminAuth(username, password string) error {
	_, err := s.db.Exec(`
INSERT INTO admin_auth (id, username, password, updated_at)
VALUES (1, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	username   = excluded.username,
	password   = excluded.password,
	updated_at = excluded.updated_at
`, username, password, time.Now().Unix())
	return err
}

func aggregateUpdateJobStatus(targets []*models.UpdateJobTarget) models.UpdateJobStatus {
	if len(targets) == 0 {
		return models.UpdateJobStatusPending
	}

	pendingCount := 0
	activeCount := 0
	successCount := 0
	failureCount := 0

	for _, target := range targets {
		switch target.Status {
		case models.UpdateJobTargetStatusPending:
			pendingCount++
		case models.UpdateJobTargetStatusDispatched,
			models.UpdateJobTargetStatusAccepted,
			models.UpdateJobTargetStatusDownloading,
			models.UpdateJobTargetStatusVerifying,
			models.UpdateJobTargetStatusRestarting:
			activeCount++
		case models.UpdateJobTargetStatusSucceeded:
			successCount++
		case models.UpdateJobTargetStatusFailed,
			models.UpdateJobTargetStatusTimeout,
			models.UpdateJobTargetStatusOfflineSkipped:
			failureCount++
		default:
			failureCount++
		}
	}

	total := len(targets)
	if pendingCount == total {
		return models.UpdateJobStatusPending
	}
	if activeCount > 0 || pendingCount > 0 {
		return models.UpdateJobStatusRunning
	}
	if successCount == total {
		return models.UpdateJobStatusCompleted
	}
	if failureCount == total {
		return models.UpdateJobStatusFailed
	}
	if successCount > 0 && failureCount > 0 {
		return models.UpdateJobStatusPartialFailed
	}
	return models.UpdateJobStatusRunning
}

// -------------------------------------------------------------------------
// Update job operations
// -------------------------------------------------------------------------

func (s *Store) CreateUpdateJob(job *models.UpdateJob) error {
	if job == nil {
		return fmt.Errorf("nil update job")
	}
	createdAt := job.CreatedAt
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	updatedAt := job.UpdatedAt
	if updatedAt <= 0 {
		updatedAt = createdAt
	}
	_, err := s.db.Exec(`
INSERT INTO update_jobs (id, kind, target_version, download_url, sha256, created_at, updated_at, created_by, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`, job.ID, job.Kind, job.TargetVersion, job.DownloadURL, job.SHA256, createdAt, updatedAt, job.CreatedBy, job.Status)
	return err
}

func (s *Store) CreateUpdateJobTarget(target *models.UpdateJobTarget) error {
	if target == nil {
		return fmt.Errorf("nil update job target")
	}
	updatedAt := target.UpdatedAt
	if updatedAt <= 0 {
		updatedAt = time.Now().Unix()
	}
	_, err := s.db.Exec(`
INSERT INTO update_job_targets (job_id, node_id, status, message, updated_at, reported_version)
VALUES (?, ?, ?, ?, ?, ?)
`, target.JobID, target.NodeID, target.Status, target.Message, updatedAt, target.ReportedVersion)
	return err
}

func (s *Store) CreateUpdateJobTargets(jobID string, nodeIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	updatedAt := time.Now().Unix()
	for _, nodeID := range nodeIDs {
		_, err := tx.Exec(`
INSERT INTO update_job_targets (job_id, node_id, status, message, updated_at, reported_version)
VALUES (?, ?, ?, '', ?, '')
`, jobID, nodeID, models.UpdateJobTargetStatusPending, updatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetUpdateJob(jobID string) (*models.UpdateJob, error) {
	row := s.db.QueryRow(`
SELECT id, kind, target_version, download_url, sha256, created_at, updated_at, created_by, status
FROM update_jobs WHERE id = ?
`, jobID)

	var job models.UpdateJob
	if err := row.Scan(&job.ID, &job.Kind, &job.TargetVersion, &job.DownloadURL, &job.SHA256, &job.CreatedAt, &job.UpdatedAt, &job.CreatedBy, &job.Status); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (s *Store) ListUpdateJobTargets(jobID string) ([]*models.UpdateJobTarget, error) {
	rows, err := s.db.Query(`
SELECT job_id, node_id, status, message, updated_at, reported_version
FROM update_job_targets WHERE job_id = ? ORDER BY node_id
`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []*models.UpdateJobTarget
	for rows.Next() {
		var target models.UpdateJobTarget
		if err := rows.Scan(&target.JobID, &target.NodeID, &target.Status, &target.Message, &target.UpdatedAt, &target.ReportedVersion); err != nil {
			return nil, err
		}
		targets = append(targets, &target)
	}
	return targets, rows.Err()
}

func (s *Store) FinalizeUpdateJobsForNodeVersion(nodeID, version string) error {
	rows, err := s.db.Query(`
SELECT t.job_id
FROM update_job_targets t
JOIN update_jobs j ON j.id = t.job_id
WHERE t.node_id = ?
  AND j.target_version = ?
  AND t.status IN (?, ?, ?, ?, ?)
`, nodeID, version,
		models.UpdateJobTargetStatusDispatched,
		models.UpdateJobTargetStatusAccepted,
		models.UpdateJobTargetStatusDownloading,
		models.UpdateJobTargetStatusVerifying,
		models.UpdateJobTargetStatusRestarting,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var jobIDs []string
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			return err
		}
		jobIDs = append(jobIDs, jobID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, jobID := range jobIDs {
		if err := s.UpdateUpdateJobTargetStatus(jobID, nodeID, models.UpdateJobTargetStatusSucceeded, "agent reconnected with target version", version); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateUpdateJobTargetStatus(jobID, nodeID string, status models.UpdateJobTargetStatus, message, reportedVersion string) error {
	updatedAt := time.Now().Unix()
	_, err := s.db.Exec(`
UPDATE update_job_targets
SET status = ?,
    message = ?,
    reported_version = CASE WHEN ? != '' THEN ? ELSE reported_version END,
    updated_at = ?
WHERE job_id = ? AND node_id = ?
`, status, message, reportedVersion, reportedVersion, updatedAt, jobID, nodeID)
	if err != nil {
		return err
	}
	_, err = s.RecomputeUpdateJobStatus(jobID)
	return err
}

func (s *Store) RecomputeUpdateJobStatus(jobID string) (*models.UpdateJob, error) {
	targets, err := s.ListUpdateJobTargets(jobID)
	if err != nil {
		return nil, err
	}
	job, err := s.GetUpdateJob(jobID)
	if err != nil || job == nil {
		return job, err
	}
	if len(targets) == 0 {
		return job, nil
	}

	job.Status = aggregateUpdateJobStatus(targets)
	job.UpdatedAt = time.Now().Unix()
	_, err = s.db.Exec(`UPDATE update_jobs SET status = ?, updated_at = ? WHERE id = ?`, job.Status, job.UpdatedAt, job.ID)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) MarkRestartingUpdateJobTargetsSucceeded(nodeID, reportedVersion string) error {
	rows, err := s.db.Query(`SELECT job_id FROM update_job_targets WHERE node_id = ? AND status = ?`, nodeID, models.UpdateJobTargetStatusRestarting)
	if err != nil {
		return err
	}
	defer rows.Close()

	var jobIDs []string
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			return err
		}
		jobIDs = append(jobIDs, jobID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, jobID := range jobIDs {
		if err := s.UpdateUpdateJobTargetStatus(jobID, nodeID, models.UpdateJobTargetStatusSucceeded, "agent reconnected after restart", reportedVersion); err != nil {
			return err
		}
	}
	return nil
}

// -------------------------------------------------------------------------
// Node operations
// -------------------------------------------------------------------------

func encodeHardware(hardware *models.NodeHardware) (string, error) {
	if hardware == nil || hardware.IsEmpty() {
		return "", nil
	}

	raw, err := json.Marshal(hardware)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeHardware(raw string) *models.NodeHardware {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var hardware models.NodeHardware
	if err := json.Unmarshal([]byte(raw), &hardware); err != nil {
		return nil
	}
	if hardware.IsEmpty() {
		return nil
	}
	return &hardware
}

// UpsertNode inserts or updates a node record.
func (s *Store) UpsertNode(node *models.Node) error {
	hardwareJSON, err := encodeHardware(node.Hardware)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
INSERT INTO nodes (id, name, token, ip, os, arch, agent_version, hardware_json, created_at, last_seen)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name          = excluded.name,
    token         = excluded.token,
    ip            = excluded.ip,
    os            = excluded.os,
    arch          = excluded.arch,
    agent_version = CASE WHEN excluded.agent_version != '' THEN excluded.agent_version ELSE nodes.agent_version END,
    hardware_json = CASE WHEN excluded.hardware_json != '' THEN excluded.hardware_json ELSE nodes.hardware_json END,
    last_seen     = excluded.last_seen
`,
		node.ID, node.Name, node.Token, node.IP, node.OS, node.Arch, node.AgentVersion, hardwareJSON,
		node.CreatedAt, node.LastSeen,
	)
	return err
}

// UpdateLastSeen updates the last_seen timestamp for the given node.
func (s *Store) UpdateLastSeen(nodeID string) error {
	_, err := s.db.Exec(`UPDATE nodes SET last_seen = ? WHERE id = ?`, time.Now().Unix(), nodeID)
	return err
}

// UpdateNodeMetadata updates node network/system metadata from live agent signals.
// Empty values are ignored so we never overwrite existing data with blanks.
func (s *Store) UpdateNodeMetadata(nodeID, ip, osName, arch, agentVersion string, hardware *models.NodeHardware, lastSeen int64) error {
	updates := make([]string, 0, 4)
	args := make([]any, 0, 6)
	hardwareJSON, err := encodeHardware(hardware)
	if err != nil {
		return err
	}

	if strings.TrimSpace(ip) != "" {
		updates = append(updates, "ip = ?")
		args = append(args, ip)
	}
	if strings.TrimSpace(osName) != "" {
		updates = append(updates, "os = ?")
		args = append(args, osName)
	}
	if strings.TrimSpace(arch) != "" {
		updates = append(updates, "arch = ?")
		args = append(args, arch)
	}
	if strings.TrimSpace(agentVersion) != "" {
		updates = append(updates, "agent_version = ?")
		args = append(args, agentVersion)
	}
	if hardwareJSON != "" {
		updates = append(updates, "hardware_json = ?")
		args = append(args, hardwareJSON)
	}

	if lastSeen <= 0 {
		lastSeen = time.Now().Unix()
	}
	updates = append(updates, "last_seen = ?")
	args = append(args, lastSeen)
	args = append(args, nodeID)

	query := "UPDATE nodes SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err = s.db.Exec(query, args...)
	return err
}

// GetNodeByToken returns the node with the given token, or (nil, nil) if not found.
func (s *Store) GetNodeByToken(token string) (*models.Node, error) {
	row := s.db.QueryRow(`
SELECT id, name, token, ip, os, arch, agent_version, hardware_json, created_at, last_seen
FROM nodes WHERE token = ?`, token)

	var hardwareJSON string
	var n models.Node
	err := row.Scan(&n.ID, &n.Name, &n.Token, &n.IP, &n.OS, &n.Arch, &n.AgentVersion,
		&hardwareJSON, &n.CreatedAt, &n.LastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Hardware = decodeHardware(hardwareJSON)
	return &n, nil
}

// GetNodeByID returns the node with the given ID, or (nil, nil) if not found.
func (s *Store) GetNodeByID(id string) (*models.Node, error) {
	row := s.db.QueryRow(`
SELECT id, name, token, ip, os, arch, agent_version, hardware_json, created_at, last_seen
FROM nodes WHERE id = ?`, id)

	var hardwareJSON string
	var n models.Node
	err := row.Scan(&n.ID, &n.Name, &n.Token, &n.IP, &n.OS, &n.Arch, &n.AgentVersion,
		&hardwareJSON, &n.CreatedAt, &n.LastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Hardware = decodeHardware(hardwareJSON)
	return &n, nil
}

// ListNodes returns all registered nodes.
func (s *Store) ListNodes() ([]*models.Node, error) {
	rows, err := s.db.Query(`
SELECT id, name, token, ip, os, arch, agent_version, hardware_json, created_at, last_seen
FROM nodes ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*models.Node
	for rows.Next() {
		var n models.Node
		var hardwareJSON string
		if err := rows.Scan(&n.ID, &n.Name, &n.Token, &n.IP, &n.OS, &n.Arch, &n.AgentVersion,
			&hardwareJSON, &n.CreatedAt, &n.LastSeen); err != nil {
			return nil, err
		}
		n.Hardware = decodeHardware(hardwareJSON)
		nodes = append(nodes, &n)
	}
	return nodes, rows.Err()
}

// RenameNode updates only a node's display name.
func (s *Store) RenameNode(nodeID, name string) error {
	_, err := s.db.Exec(`UPDATE nodes SET name = ? WHERE id = ?`, name, nodeID)
	return err
}

// DeleteNode removes a node and its related telemetry rows.
func (s *Store) DeleteNode(nodeID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		`DELETE FROM metrics WHERE node_id = ?`,
		`DELETE FROM processes WHERE node_id = ?`,
		`DELETE FROM service_checks WHERE node_id = ?`,
		`DELETE FROM update_job_targets WHERE node_id = ?`,
		`DELETE FROM nodes WHERE id = ?`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt, nodeID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// -------------------------------------------------------------------------
// Metrics operations
// -------------------------------------------------------------------------

// InsertMetrics inserts a single metrics sample for a node.
// Disk partitions are aggregated into a single used/total pair.
func (s *Store) InsertMetrics(nodeID string, m *models.MetricsPayload) error {
	var diskUsed, diskTotal uint64
	for _, d := range m.Disk {
		diskUsed += d.Used
		diskTotal += d.Total
	}

	_, err := s.db.Exec(`
INSERT INTO metrics (node_id, ts, cpu_percent, mem_used, mem_total, disk_used, disk_total, net_rx, net_tx, uptime_seconds)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nodeID, m.TS, m.CPU,
		m.Mem.Used, m.Mem.Total,
		diskUsed, diskTotal,
		m.Net.RxBytes, m.Net.TxBytes, m.UptimeSeconds,
	)
	return err
}

// QueryMetrics returns metrics rows for a node within the given time range,
// ordered by ascending timestamp.
func (s *Store) QueryMetrics(nodeID string, from, to int64) ([]*MetricsRow, error) {
	rows, err := s.db.Query(`
SELECT ts, cpu_percent, mem_used, mem_total, disk_used, disk_total, net_rx, net_tx, uptime_seconds
FROM metrics WHERE node_id = ? AND ts BETWEEN ? AND ? ORDER BY ts`,
		nodeID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*MetricsRow
	for rows.Next() {
		var r MetricsRow
		if err := rows.Scan(&r.TS, &r.CPU, &r.MemUsed, &r.MemTotal, &r.DiskUsed, &r.DiskTotal, &r.NetRx, &r.NetTx, &r.UptimeSeconds); err != nil {
			return nil, err
		}
		result = append(result, &r)
	}
	return result, rows.Err()
}

// LatestMetricsByNodeIDs returns the most recent metrics sample for each node ID.
func (s *Store) LatestMetricsByNodeIDs(nodeIDs []string) (map[string]*models.NodeMetricsSnapshot, error) {
	result := make(map[string]*models.NodeMetricsSnapshot, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		var snapshot models.NodeMetricsSnapshot
		err := s.db.QueryRow(`
SELECT ts, cpu_percent, mem_used, mem_total, disk_used, disk_total, net_rx, net_tx, uptime_seconds
FROM metrics
WHERE node_id = ?
ORDER BY ts DESC, id DESC
LIMIT 1`, nodeID).Scan(
			&snapshot.TS,
			&snapshot.CPU,
			&snapshot.MemUsed,
			&snapshot.MemTotal,
			&snapshot.DiskUsed,
			&snapshot.DiskTotal,
			&snapshot.NetRx,
			&snapshot.NetTx,
			&snapshot.UptimeSeconds,
		)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		result[nodeID] = &snapshot
	}

	return result, nil
}

// PruneOldMetrics deletes metrics rows older than retentionDays days.
func (s *Store) PruneOldMetrics(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	_, err := s.db.Exec(`DELETE FROM metrics WHERE ts < ?`, cutoff)
	return err
}

// -------------------------------------------------------------------------
// Process operations
// -------------------------------------------------------------------------

// UpsertProcesses stores a pre-serialized JSON process list for a node,
// overwriting any existing record.
func (s *Store) UpsertProcesses(nodeID string, ts int64, data string) error {
	_, err := s.db.Exec(`
		INSERT INTO processes (node_id, ts, data) VALUES (?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET ts=excluded.ts, data=excluded.data
	`, nodeID, ts, data)
	return err
}

// GetProcesses returns the JSON-encoded process list for a node.
// Returns "[]" when no data is found.
func (s *Store) GetProcesses(nodeID string) (string, error) {
	var data string
	err := s.db.QueryRow(`SELECT data FROM processes WHERE node_id = ?`, nodeID).Scan(&data)
	if err == sql.ErrNoRows {
		return "[]", nil
	}
	if err != nil {
		return "[]", err
	}
	return data, nil
}

// -------------------------------------------------------------------------
// Service check operations
// -------------------------------------------------------------------------

// UpsertServiceCheck inserts or updates a service check result for a node.
func (s *Store) UpsertServiceCheck(nodeID, name, status string) error {
	_, err := s.db.Exec(`
		INSERT INTO service_checks (node_id, name, status, last_checked) VALUES (?, ?, ?, ?)
		ON CONFLICT(node_id, name) DO UPDATE SET status=excluded.status, last_checked=excluded.last_checked
	`, nodeID, name, status, time.Now().Unix())
	return err
}

// GetServiceChecks returns all service checks for a node as a slice of maps.
func (s *Store) GetServiceChecks(nodeID string) ([]map[string]any, error) {
	rows, err := s.db.Query(`
SELECT name, status, last_checked FROM service_checks WHERE node_id = ? ORDER BY name`,
		nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var name, status string
		var lastChecked int64
		if err := rows.Scan(&name, &status, &lastChecked); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{
			"name":         name,
			"status":       status,
			"last_checked": lastChecked,
		})
	}
	return result, rows.Err()
}
