package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/notify"
	"github.com/thism-dev/thism/internal/security"
	_ "modernc.org/sqlite"
)

// Store is the SQLite-backed data access layer.
type Store struct {
	db *sql.DB
}

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

const DefaultMetricsRetentionDays = 7

const metricsRetentionSettingKey = "metrics_retention_days"
const dashboardSettingsKey = "dashboard_settings"
const notificationSettingsKey = "notification_settings"
const adminSessionTTL = 30 * 24 * time.Hour
const sqliteBusyTimeout = 5000

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

type Metrics1mRow struct {
	NodeID       string
	TS           int64
	Samples      int64
	CPUAvg       float64
	CPUMax       float64
	MemUsedAvg   int64
	MemUsedMax   int64
	MemTotalMax  int64
	DiskUsedAvg  int64
	DiskUsedMax  int64
	DiskTotalMax int64
	NetRxMax     int64
	NetTxMax     int64
	UptimeMax    int64
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
	if err := s.configureConnectionPragmas(); err != nil {
		db.Close()
		return nil, fmt.Errorf("configure sqlite pragmas: %w", err)
	}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) configureConnectionPragmas() error {
	pragmas := []string{
		`PRAGMA journal_mode = WAL`,
		fmt.Sprintf(`PRAGMA busy_timeout = %d`, sqliteBusyTimeout),
		`PRAGMA synchronous = NORMAL`,
	}
	for _, pragma := range pragmas {
		if _, err := s.db.Exec(pragma); err != nil {
			return err
		}
	}
	return nil
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

CREATE TABLE IF NOT EXISTS metrics_1m (
    node_id            TEXT NOT NULL,
    ts                 INTEGER NOT NULL,
    samples            INTEGER NOT NULL DEFAULT 0,
    cpu_avg            REAL DEFAULT 0,
    cpu_max            REAL DEFAULT 0,
    mem_used_avg       INTEGER DEFAULT 0,
    mem_used_max       INTEGER DEFAULT 0,
    mem_total_max      INTEGER DEFAULT 0,
    disk_used_avg      INTEGER DEFAULT 0,
    disk_used_max      INTEGER DEFAULT 0,
    disk_total_max     INTEGER DEFAULT 0,
    net_rx_max         INTEGER DEFAULT 0,
    net_tx_max         INTEGER DEFAULT 0,
    uptime_seconds_max INTEGER DEFAULT 0,
    PRIMARY KEY (node_id, ts)
);

CREATE INDEX IF NOT EXISTS idx_metrics_1m_node_ts ON metrics_1m(node_id, ts);

CREATE TABLE IF NOT EXISTS processes (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL UNIQUE,
    ts      INTEGER NOT NULL,
    data    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS docker_containers (
    node_id          TEXT PRIMARY KEY,
    ts               INTEGER NOT NULL,
    docker_available INTEGER NOT NULL DEFAULT 0,
    data             TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS service_checks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id      TEXT NOT NULL,
    name         TEXT NOT NULL,
    status       TEXT DEFAULT 'unknown',
    last_checked INTEGER NOT NULL DEFAULT 0,
    UNIQUE(node_id, name)
);

CREATE TABLE IF NOT EXISTS monitor_items (
    id                    TEXT PRIMARY KEY,
    name                  TEXT NOT NULL,
    type                  TEXT NOT NULL,
    target                TEXT NOT NULL,
    interval_seconds      INTEGER NOT NULL,
    auto_assign_new_nodes INTEGER NOT NULL DEFAULT 1,
    created_at            INTEGER NOT NULL DEFAULT 0,
    updated_at            INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS monitor_item_nodes (
    monitor_id TEXT NOT NULL,
    node_id    TEXT NOT NULL,
    PRIMARY KEY (monitor_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_monitor_item_nodes_node_id ON monitor_item_nodes(node_id);

CREATE TABLE IF NOT EXISTS monitor_results (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id    TEXT NOT NULL,
    node_id       TEXT NOT NULL,
    ts            INTEGER NOT NULL,
    latency_ms    REAL,
    loss_percent  REAL,
    jitter_ms     REAL,
    success       INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_monitor_results_node_ts ON monitor_results(node_id, ts);
CREATE INDEX IF NOT EXISTS idx_monitor_results_monitor_node_ts ON monitor_results(monitor_id, node_id, ts);

CREATE TABLE IF NOT EXISTS latency_1m (
    node_id          TEXT NOT NULL,
    monitor_id       TEXT NOT NULL,
    ts               INTEGER NOT NULL,
    samples          INTEGER NOT NULL DEFAULT 0,
    success_samples  INTEGER NOT NULL DEFAULT 0,
    latency_ms_avg   REAL,
    loss_percent_avg REAL,
    jitter_ms_avg    REAL,
    error_message    TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (node_id, monitor_id, ts)
);

CREATE INDEX IF NOT EXISTS idx_latency_1m_node_ts ON latency_1m(node_id, ts);
CREATE INDEX IF NOT EXISTS idx_latency_1m_monitor_node_ts ON latency_1m(monitor_id, node_id, ts);

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

CREATE TABLE IF NOT EXISTS admin_sessions (
    session_id TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL DEFAULT 0,
    expires_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS alert_deliveries (
    node_id      TEXT NOT NULL,
    metric       TEXT NOT NULL,
    severity     TEXT NOT NULL,
    value        REAL NOT NULL DEFAULT 0,
    threshold    REAL NOT NULL DEFAULT 0,
    delivered_at INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (node_id, metric, severity)
);

CREATE TABLE IF NOT EXISTS recovery_deliveries (
    node_id      TEXT NOT NULL,
    metric       TEXT NOT NULL,
    delivered_at INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (node_id, metric)
);

CREATE TABLE IF NOT EXISTS recovery_states (
    node_id               TEXT NOT NULL,
    metric                TEXT NOT NULL,
    consecutive_successes INTEGER NOT NULL DEFAULT 0,
    last_observed_at      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (node_id, metric)
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
	if err := s.ensureColumn("monitor_results", "loss_percent", "REAL"); err != nil {
		return err
	}
	if err := s.ensureColumn("monitor_results", "jitter_ms", "REAL"); err != nil {
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

func defaultNotificationSettings() models.NotificationSettings {
	return models.NotificationSettings{
		Enabled:                             false,
		Channel:                             string(models.NotificationChannelTelegram),
		TelegramTargets:                     []models.TelegramTarget{},
		CPUWarningPercent:                   85,
		CPUCriticalPercent:                  95,
		MemWarningPercent:                   85,
		MemCriticalPercent:                  95,
		DiskWarningPercent:                  85,
		DiskCriticalPercent:                 95,
		CooldownMinutes:                     30,
		RecoverySuccessiveSamples:           3,
		RecoveryNotificationCooldownMinutes: 30,
		TimeZoneMode:                        models.NotificationTimeZoneModeSystem,
		TimeZone:                            "",
		DispatcherQueueCapacity:             models.DefaultDispatcherQueueCapacity,
		NotifyDispatcherDrops:               false,
	}
}

func defaultDashboardSettings() models.DashboardSettings {
	return models.DashboardSettings{
		ShowDashboardCardIP: true,
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func normalizeLatencyMonitorNodeIDs(nodeIDs []string) []string {
	clean := make([]string, 0, len(nodeIDs))
	seen := make(map[string]struct{}, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		normalized := strings.TrimSpace(nodeID)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		clean = append(clean, normalized)
	}
	return clean
}

func normalizeLatencyMonitor(monitor *models.LatencyMonitor) (*models.LatencyMonitor, error) {
	if monitor == nil {
		return nil, fmt.Errorf("latency monitor is nil")
	}
	normalized := *monitor
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.Target = strings.TrimSpace(normalized.Target)
	if normalized.ID == "" {
		return nil, fmt.Errorf("latency monitor id is required")
	}
	if normalized.Name == "" {
		return nil, fmt.Errorf("latency monitor name is required")
	}
	if normalized.Target == "" {
		return nil, fmt.Errorf("latency monitor target is required")
	}
	if normalized.IntervalSeconds <= 0 {
		return nil, fmt.Errorf("latency monitor interval_seconds must be greater than 0")
	}
	switch normalized.Type {
	case models.LatencyMonitorTypeICMP, models.LatencyMonitorTypeTCP, models.LatencyMonitorTypeHTTP:
	default:
		return nil, fmt.Errorf("invalid latency monitor type")
	}
	now := time.Now().Unix()
	if normalized.CreatedAt <= 0 {
		normalized.CreatedAt = now
	}
	if normalized.UpdatedAt <= 0 {
		normalized.UpdatedAt = now
	}
	normalized.AssignedNodeIDs = normalizeLatencyMonitorNodeIDs(normalized.AssignedNodeIDs)
	return &normalized, nil
}

func replaceLatencyMonitorNodes(tx *sql.Tx, monitorID string, nodeIDs []string) error {
	if _, err := tx.Exec(`DELETE FROM monitor_item_nodes WHERE monitor_id = ?`, monitorID); err != nil {
		return err
	}
	for _, nodeID := range normalizeLatencyMonitorNodeIDs(nodeIDs) {
		if _, err := tx.Exec(`INSERT INTO monitor_item_nodes (monitor_id, node_id) VALUES (?, ?)`, monitorID, nodeID); err != nil {
			return err
		}
	}
	return nil
}

func listLatencyMonitorNodeIDsWith(db queryRower, monitorID string) ([]string, error) {
	rows, err := db.Query(`
SELECT node_id
FROM monitor_item_nodes
WHERE monitor_id = ?
ORDER BY node_id
`, monitorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodeIDs []string
	for rows.Next() {
		var nodeID string
		if err := rows.Scan(&nodeID); err != nil {
			return nil, err
		}
		nodeIDs = append(nodeIDs, nodeID)
	}
	return nodeIDs, rows.Err()
}

type queryRower interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func (s *Store) populateLatencyMonitorAssignments(monitors []*models.LatencyMonitor) error {
	for _, monitor := range monitors {
		nodeIDs, err := listLatencyMonitorNodeIDsWith(s.db, monitor.ID)
		if err != nil {
			return err
		}
		monitor.AssignedNodeIDs = nodeIDs
		if monitor.AssignedNodeCount == 0 {
			monitor.AssignedNodeCount = len(nodeIDs)
		}
	}
	return nil
}

func (s *Store) CreateLatencyMonitor(monitor *models.LatencyMonitor, nodeIDs []string) error {
	normalized, err := normalizeLatencyMonitor(monitor)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
INSERT INTO monitor_items (id, name, type, target, interval_seconds, auto_assign_new_nodes, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, normalized.ID, normalized.Name, normalized.Type, normalized.Target, normalized.IntervalSeconds, boolToInt(normalized.AutoAssignNewNodes), normalized.CreatedAt, normalized.UpdatedAt)
	if err != nil {
		return err
	}
	if err := replaceLatencyMonitorNodes(tx, normalized.ID, nodeIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UpdateLatencyMonitor(monitor *models.LatencyMonitor, nodeIDs []string) error {
	normalized, err := normalizeLatencyMonitor(monitor)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE monitor_items
SET name = ?, type = ?, target = ?, interval_seconds = ?, auto_assign_new_nodes = ?, updated_at = ?
WHERE id = ?
`, normalized.Name, normalized.Type, normalized.Target, normalized.IntervalSeconds, boolToInt(normalized.AutoAssignNewNodes), normalized.UpdatedAt, normalized.ID)
	if err != nil {
		return err
	}
	if err := replaceLatencyMonitorNodes(tx, normalized.ID, nodeIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetLatencyMonitorByID(monitorID string) (*models.LatencyMonitor, error) {
	row := s.db.QueryRow(`
SELECT id, name, type, target, interval_seconds, auto_assign_new_nodes, created_at, updated_at
FROM monitor_items
WHERE id = ?
`, monitorID)

	var monitor models.LatencyMonitor
	var autoAssign int
	if err := row.Scan(&monitor.ID, &monitor.Name, &monitor.Type, &monitor.Target, &monitor.IntervalSeconds, &autoAssign, &monitor.CreatedAt, &monitor.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	monitor.AutoAssignNewNodes = autoAssign == 1
	nodeIDs, err := listLatencyMonitorNodeIDsWith(s.db, monitor.ID)
	if err != nil {
		return nil, err
	}
	monitor.AssignedNodeIDs = nodeIDs
	monitor.AssignedNodeCount = len(nodeIDs)
	return &monitor, nil
}

func (s *Store) ListLatencyMonitors() ([]*models.LatencyMonitor, error) {
	rows, err := s.db.Query(`
SELECT m.id, m.name, m.type, m.target, m.interval_seconds, m.auto_assign_new_nodes, m.created_at, m.updated_at, COUNT(n.node_id) AS assigned_node_count
FROM monitor_items m
LEFT JOIN monitor_item_nodes n ON n.monitor_id = m.id
GROUP BY m.id, m.name, m.type, m.target, m.interval_seconds, m.auto_assign_new_nodes, m.created_at, m.updated_at
ORDER BY m.name, m.id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []*models.LatencyMonitor
	for rows.Next() {
		var monitor models.LatencyMonitor
		var autoAssign int
		if err := rows.Scan(&monitor.ID, &monitor.Name, &monitor.Type, &monitor.Target, &monitor.IntervalSeconds, &autoAssign, &monitor.CreatedAt, &monitor.UpdatedAt, &monitor.AssignedNodeCount); err != nil {
			return nil, err
		}
		monitor.AutoAssignNewNodes = autoAssign == 1
		monitors = append(monitors, &monitor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := s.populateLatencyMonitorAssignments(monitors); err != nil {
		return nil, err
	}
	return monitors, nil
}

func (s *Store) ListLatencyMonitorsByNodeID(nodeID string) ([]*models.LatencyMonitor, error) {
	rows, err := s.db.Query(`
SELECT m.id, m.name, m.type, m.target, m.interval_seconds, m.auto_assign_new_nodes, m.created_at, m.updated_at
FROM monitor_items m
JOIN monitor_item_nodes n ON n.monitor_id = m.id
WHERE n.node_id = ?
ORDER BY m.name, m.id
`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []*models.LatencyMonitor
	for rows.Next() {
		var monitor models.LatencyMonitor
		var autoAssign int
		if err := rows.Scan(&monitor.ID, &monitor.Name, &monitor.Type, &monitor.Target, &monitor.IntervalSeconds, &autoAssign, &monitor.CreatedAt, &monitor.UpdatedAt); err != nil {
			return nil, err
		}
		monitor.AutoAssignNewNodes = autoAssign == 1
		monitors = append(monitors, &monitor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := s.populateLatencyMonitorAssignments(monitors); err != nil {
		return nil, err
	}
	return monitors, nil
}

func (s *Store) AssignAutoLatencyMonitorsToNode(nodeID string) error {
	if strings.TrimSpace(nodeID) == "" {
		return nil
	}
	_, err := s.db.Exec(`
INSERT OR IGNORE INTO monitor_item_nodes (monitor_id, node_id)
SELECT id, ?
FROM monitor_items
WHERE auto_assign_new_nodes = 1
`, nodeID)
	return err
}

func (s *Store) DeleteLatencyMonitor(monitorID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range []string{
		`DELETE FROM monitor_results WHERE monitor_id = ?`,
		`DELETE FROM latency_1m WHERE monitor_id = ?`,
		`DELETE FROM monitor_item_nodes WHERE monitor_id = ?`,
		`DELETE FROM monitor_items WHERE id = ?`,
	} {
		if _, err := tx.Exec(stmt, monitorID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) InsertLatencyResult(result *models.LatencyMonitorResult) error {
	if result == nil {
		return fmt.Errorf("latency monitor result is nil")
	}
	if strings.TrimSpace(result.MonitorID) == "" {
		return fmt.Errorf("latency monitor result monitor_id is required")
	}
	if strings.TrimSpace(result.NodeID) == "" {
		return fmt.Errorf("latency monitor result node_id is required")
	}
	_, err := s.db.Exec(`
INSERT INTO monitor_results (monitor_id, node_id, ts, latency_ms, loss_percent, jitter_ms, success, error_message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, result.MonitorID, result.NodeID, result.TS, result.LatencyMs, result.LossPercent, result.JitterMs, boolToInt(result.Success), result.ErrorMessage)
	return err
}

func (s *Store) QueryLatencyResultsByNodeID(nodeID string, from, to int64) ([]*models.LatencyMonitorResult, error) {
	rows, err := s.db.Query(`
SELECT monitor_id, node_id, ts, latency_ms, loss_percent, jitter_ms, success, error_message
FROM monitor_results
WHERE node_id = ? AND ts BETWEEN ? AND ?
ORDER BY ts, monitor_id
`, nodeID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.LatencyMonitorResult
	for rows.Next() {
		var result models.LatencyMonitorResult
		var latency sql.NullFloat64
		var loss sql.NullFloat64
		var jitter sql.NullFloat64
		var success int
		if err := rows.Scan(&result.MonitorID, &result.NodeID, &result.TS, &latency, &loss, &jitter, &success, &result.ErrorMessage); err != nil {
			return nil, err
		}
		if latency.Valid {
			value := latency.Float64
			result.LatencyMs = &value
		}
		if loss.Valid {
			value := loss.Float64
			result.LossPercent = &value
		}
		if jitter.Valid {
			value := jitter.Float64
			result.JitterMs = &value
		}
		result.Success = success == 1
		results = append(results, &result)
	}
	return results, rows.Err()
}

func (s *Store) QueryLatencyResultsByNodeID1m(nodeID string, from, to int64) ([]*models.LatencyMonitorResult, error) {
	from = (from / 60) * 60
	to = (to / 60) * 60
	rows, err := s.db.Query(`
SELECT monitor_id, node_id, ts, latency_ms_avg, loss_percent_avg, jitter_ms_avg, success_samples, error_message
FROM latency_1m
WHERE node_id = ? AND ts BETWEEN ? AND ?
ORDER BY ts, monitor_id
`, nodeID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.LatencyMonitorResult
	for rows.Next() {
		var result models.LatencyMonitorResult
		var latency sql.NullFloat64
		var loss sql.NullFloat64
		var jitter sql.NullFloat64
		var successSamples int
		if err := rows.Scan(&result.MonitorID, &result.NodeID, &result.TS, &latency, &loss, &jitter, &successSamples, &result.ErrorMessage); err != nil {
			return nil, err
		}
		if latency.Valid {
			value := latency.Float64
			result.LatencyMs = &value
		}
		if loss.Valid {
			value := loss.Float64
			result.LossPercent = &value
		}
		if jitter.Valid {
			value := jitter.Float64
			result.JitterMs = &value
		}
		result.Success = successSamples > 0
		results = append(results, &result)
	}
	return results, rows.Err()
}

func (s *Store) RollupLatencyResults1m(from, to int64) error {
	from = (from / 60) * 60
	if to < from {
		return nil
	}
	_, err := s.db.Exec(`
INSERT INTO latency_1m (
  node_id, monitor_id, ts, samples, success_samples,
  latency_ms_avg, loss_percent_avg, jitter_ms_avg, error_message
)
SELECT
  node_id,
  monitor_id,
  (ts / 60) * 60 AS minute_ts,
  COUNT(*) AS samples,
  SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS success_samples,
  AVG(latency_ms) AS latency_ms_avg,
  AVG(loss_percent) AS loss_percent_avg,
  AVG(jitter_ms) AS jitter_ms_avg,
  CASE
    WHEN SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) > 0 THEN ''
    ELSE COALESCE(MAX(NULLIF(error_message, '')), '')
  END AS error_message
FROM monitor_results
WHERE ts BETWEEN ? AND ?
GROUP BY node_id, monitor_id, minute_ts
ON CONFLICT(node_id, monitor_id, ts) DO UPDATE SET
  samples          = excluded.samples,
  success_samples  = excluded.success_samples,
  latency_ms_avg   = excluded.latency_ms_avg,
  loss_percent_avg = excluded.loss_percent_avg,
  jitter_ms_avg    = excluded.jitter_ms_avg,
  error_message    = excluded.error_message
`, from, to)
	return err
}

func (s *Store) NeedsLatencyResults1mRollup(nodeID string, from, to int64) (bool, error) {
	var rawCount int
	var rawMin sql.NullInt64
	var rawMax sql.NullInt64
	if err := s.db.QueryRow(`
SELECT COUNT(*), MIN(ts), MAX(ts)
FROM monitor_results
WHERE node_id = ? AND ts BETWEEN ? AND ?
`, nodeID, from, to).Scan(&rawCount, &rawMin, &rawMax); err != nil {
		return false, err
	}
	if rawCount == 0 || !rawMin.Valid || !rawMax.Valid {
		return false, nil
	}

	alignedFrom := (from / 60) * 60
	alignedTo := (to / 60) * 60
	var rollupCount int
	var rollupMin sql.NullInt64
	var rollupMax sql.NullInt64
	if err := s.db.QueryRow(`
SELECT COUNT(*), MIN(ts), MAX(ts)
FROM latency_1m
WHERE node_id = ? AND ts BETWEEN ? AND ?
`, nodeID, alignedFrom, alignedTo).Scan(&rollupCount, &rollupMin, &rollupMax); err != nil {
		return false, err
	}
	if rollupCount == 0 || !rollupMin.Valid || !rollupMax.Valid {
		return true, nil
	}

	expectedMin := (rawMin.Int64 / 60) * 60
	expectedMax := (rawMax.Int64 / 60) * 60
	return rollupMin.Int64 > expectedMin || rollupMax.Int64 < expectedMax, nil
}

func (s *Store) GetDashboardSettings() (models.DashboardSettings, error) {
	var raw string
	err := s.db.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, dashboardSettingsKey).Scan(&raw)
	if err == sql.ErrNoRows {
		return defaultDashboardSettings(), nil
	}
	if err != nil {
		return models.DashboardSettings{}, err
	}

	var decoded struct {
		ShowDashboardCardIP *bool `json:"show_dashboard_card_ip"`
	}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return defaultDashboardSettings(), nil
	}

	settings := defaultDashboardSettings()
	if decoded.ShowDashboardCardIP != nil {
		settings.ShowDashboardCardIP = *decoded.ShowDashboardCardIP
	}
	return settings, nil
}

func (s *Store) UpsertDashboardSettings(settings models.DashboardSettings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
INSERT INTO app_settings (key, value, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET
	value = excluded.value,
	updated_at = excluded.updated_at
`, dashboardSettingsKey, string(raw), time.Now().Unix())
	return err
}

func normalizeNotificationSettings(settings models.NotificationSettings) models.NotificationSettings {
	defaults := defaultNotificationSettings()
	if strings.TrimSpace(settings.Channel) == "" {
		settings.Channel = defaults.Channel
	}
	if settings.CPUWarningPercent <= 0 {
		settings.CPUWarningPercent = defaults.CPUWarningPercent
	}
	if settings.CPUCriticalPercent <= 0 {
		settings.CPUCriticalPercent = defaults.CPUCriticalPercent
	}
	if settings.MemWarningPercent <= 0 {
		settings.MemWarningPercent = defaults.MemWarningPercent
	}
	if settings.MemCriticalPercent <= 0 {
		settings.MemCriticalPercent = defaults.MemCriticalPercent
	}
	if settings.DiskWarningPercent <= 0 {
		settings.DiskWarningPercent = defaults.DiskWarningPercent
	}
	if settings.DiskCriticalPercent <= 0 {
		settings.DiskCriticalPercent = defaults.DiskCriticalPercent
	}
	if settings.CooldownMinutes <= 0 {
		settings.CooldownMinutes = defaults.CooldownMinutes
	}
	if settings.RecoverySuccessiveSamples <= 0 {
		settings.RecoverySuccessiveSamples = defaults.RecoverySuccessiveSamples
	}
	if settings.RecoveryNotificationCooldownMinutes <= 0 {
		settings.RecoveryNotificationCooldownMinutes = defaults.RecoveryNotificationCooldownMinutes
	}
	settings.TimeZoneMode = normalizeNotificationTimeZoneMode(settings.TimeZoneMode)
	settings.TimeZone = strings.TrimSpace(settings.TimeZone)
	if settings.TimeZoneMode != models.NotificationTimeZoneModeCustom {
		settings.TimeZone = ""
	}
	if settings.DispatcherQueueCapacity <= 0 {
		settings.DispatcherQueueCapacity = defaults.DispatcherQueueCapacity
	}
	cleanTargets := make([]models.TelegramTarget, 0, len(settings.TelegramTargets))
	for _, target := range settings.TelegramTargets {
		normalized := target.Normalized()
		if normalized.ChatID == "" {
			continue
		}
		cleanTargets = append(cleanTargets, normalized)
	}
	settings.TelegramTargets = cleanTargets
	normalizeNodeIDs := func(ids []string) []string {
		clean := make([]string, 0, len(ids))
		seen := make(map[string]struct{}, len(ids))
		for _, nodeID := range ids {
			normalized := strings.TrimSpace(nodeID)
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			clean = append(clean, normalized)
		}
		return clean
	}
	settings.EnabledNodeIDs = normalizeNodeIDs(settings.EnabledNodeIDs)
	settings.NodeScopeNodeIDs = normalizeNodeIDs(settings.NodeScopeNodeIDs)
	settings.NodeScopeMode = strings.TrimSpace(settings.NodeScopeMode)
	if settings.NodeScopeMode == "" {
		if len(settings.NodeScopeNodeIDs) > 0 {
			settings.NodeScopeMode = "include"
		} else if len(settings.EnabledNodeIDs) > 0 {
			settings.NodeScopeMode = "include"
			settings.NodeScopeNodeIDs = append([]string(nil), settings.EnabledNodeIDs...)
		} else {
			settings.NodeScopeMode = "all"
		}
	}
	if settings.NodeScopeMode != "all" && settings.NodeScopeMode != "include" && settings.NodeScopeMode != "exclude" {
		settings.NodeScopeMode = "all"
		settings.NodeScopeNodeIDs = nil
	}
	if settings.NodeScopeMode == "all" && len(settings.NodeScopeNodeIDs) == 0 && len(settings.EnabledNodeIDs) > 0 {
		settings.NodeScopeMode = "include"
		settings.NodeScopeNodeIDs = append([]string(nil), settings.EnabledNodeIDs...)
	}
	return settings
}

func normalizeNotificationTimeZoneMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case models.NotificationTimeZoneModeCustom:
		return models.NotificationTimeZoneModeCustom
	default:
		return models.NotificationTimeZoneModeSystem
	}
}

func (s *Store) GetNotificationSettings() (models.NotificationSettings, error) {
	var raw string
	err := s.db.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, notificationSettingsKey).Scan(&raw)
	if err == sql.ErrNoRows {
		return defaultNotificationSettings(), nil
	}
	if err != nil {
		return models.NotificationSettings{}, err
	}
	var settings models.NotificationSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return defaultNotificationSettings(), nil
	}
	return normalizeNotificationSettings(settings), nil
}

func (s *Store) UpsertNotificationSettings(settings models.NotificationSettings) error {
	settings = normalizeNotificationSettings(settings)
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
INSERT INTO app_settings (key, value, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET
	value = excluded.value,
	updated_at = excluded.updated_at
`, notificationSettingsKey, string(raw), time.Now().Unix())
	return err
}

func (s *Store) NotificationSettingsView(includeSecret bool) (models.NotificationSettingsView, error) {
	settings, err := s.GetNotificationSettings()
	if err != nil {
		return models.NotificationSettingsView{}, err
	}
	systemLocation := time.Now().Location()
	currentTime := time.Now()
	view := models.NotificationSettingsView{
		Enabled:                             settings.Enabled,
		Channel:                             settings.Channel,
		TelegramBotTokenSet:                 strings.TrimSpace(settings.TelegramBotToken) != "",
		TelegramTargets:                     settings.TelegramTargets,
		EnabledNodeIDs:                      settings.EnabledNodeIDs,
		NodeScopeMode:                       settings.NodeScopeMode,
		NodeScopeNodeIDs:                    settings.NodeScopeNodeIDs,
		CPUWarningPercent:                   settings.CPUWarningPercent,
		CPUCriticalPercent:                  settings.CPUCriticalPercent,
		MemWarningPercent:                   settings.MemWarningPercent,
		MemCriticalPercent:                  settings.MemCriticalPercent,
		DiskWarningPercent:                  settings.DiskWarningPercent,
		DiskCriticalPercent:                 settings.DiskCriticalPercent,
		CooldownMinutes:                     settings.CooldownMinutes,
		RecoverySuccessiveSamples:           settings.RecoverySuccessiveSamples,
		RecoveryNotificationCooldownMinutes: settings.RecoveryNotificationCooldownMinutes,
		NotifyNodeOffline:                   settings.NotifyNodeOffline,
		NotifyNodeOnline:                    settings.NotifyNodeOnline,
		TimeZoneMode:                        settings.TimeZoneMode,
		TimeZone:                            settings.TimeZone,
		SystemTimeZone:                      notify.LocationLabel(systemLocation, currentTime),
		EffectiveTimeZone:                   notify.LocationLabel(notify.ResolveLocation(settings, systemLocation), currentTime),
		NodeOfflineGraceMinutes:             settings.NodeOfflineGraceMinutes,
		DispatcherQueueCapacity:             settings.DispatcherQueueCapacity,
		NotifyDispatcherDrops:               settings.NotifyDispatcherDrops,
	}
	if includeSecret {
		view.TelegramBotToken = settings.TelegramBotToken
	}
	return view, nil
}

func (s *Store) ShouldSendAlert(nodeID, metric, severity string, cooldown time.Duration, now int64) (bool, error) {
	if cooldown <= 0 {
		return true, nil
	}
	var deliveredAt sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(delivered_at) FROM alert_deliveries WHERE node_id = ? AND metric = ?`, nodeID, metric).Scan(&deliveredAt)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !deliveredAt.Valid || deliveredAt.Int64 == 0 {
		return true, nil
	}
	return now-deliveredAt.Int64 >= int64(cooldown.Seconds()), nil
}

func (s *Store) RecordAlertDelivery(nodeID, metric, severity string, value, threshold float64, deliveredAt int64) error {
	_, err := s.db.Exec(`
INSERT INTO alert_deliveries (node_id, metric, severity, value, threshold, delivered_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(node_id, metric, severity) DO UPDATE SET
	value = excluded.value,
	threshold = excluded.threshold,
	delivered_at = excluded.delivered_at
`, nodeID, metric, severity, value, threshold, deliveredAt)
	return err
}

func (s *Store) HasActiveAlertDelivery(nodeID, metric string) (bool, error) {
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM alert_deliveries WHERE node_id = ? AND metric = ? LIMIT 1`, nodeID, metric).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) ClearAlertDelivery(nodeID, metric string) error {
	_, err := s.db.Exec(`DELETE FROM alert_deliveries WHERE node_id = ? AND metric = ?`, nodeID, metric)
	return err
}

func (s *Store) ShouldSendRecovery(nodeID, metric string, cooldown time.Duration, now int64) (bool, error) {
	if cooldown <= 0 {
		return true, nil
	}
	var deliveredAt sql.NullInt64
	err := s.db.QueryRow(`SELECT delivered_at FROM recovery_deliveries WHERE node_id = ? AND metric = ?`, nodeID, metric).Scan(&deliveredAt)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !deliveredAt.Valid || deliveredAt.Int64 == 0 {
		return true, nil
	}
	return now-deliveredAt.Int64 >= int64(cooldown.Seconds()), nil
}

func (s *Store) RecordRecoveryDelivery(nodeID, metric string, deliveredAt int64) error {
	_, err := s.db.Exec(`
INSERT INTO recovery_deliveries (node_id, metric, delivered_at)
VALUES (?, ?, ?)
ON CONFLICT(node_id, metric) DO UPDATE SET
	delivered_at = excluded.delivered_at
`, nodeID, metric, deliveredAt)
	return err
}

func (s *Store) IncrementRecoveryStreak(nodeID, metric string, observedAt int64) (int, error) {
	_, err := s.db.Exec(`
INSERT INTO recovery_states (node_id, metric, consecutive_successes, last_observed_at)
VALUES (?, ?, 1, ?)
ON CONFLICT(node_id, metric) DO UPDATE SET
	consecutive_successes = recovery_states.consecutive_successes + 1,
	last_observed_at = excluded.last_observed_at
`, nodeID, metric, observedAt)
	if err != nil {
		return 0, err
	}
	var count int
	if err := s.db.QueryRow(`SELECT consecutive_successes FROM recovery_states WHERE node_id = ? AND metric = ?`, nodeID, metric).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) ResetRecoveryState(nodeID, metric string) error {
	_, err := s.db.Exec(`DELETE FROM recovery_states WHERE node_id = ? AND metric = ?`, nodeID, metric)
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
	if security.NeedsPasswordHashUpgrade(password) {
		hashedPassword, err := security.HashPassword(password)
		if err != nil {
			return err
		}
		password = hashedPassword
	}

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

func (s *Store) CreateAdminSession(sessionID string, expiresAt int64) error {
	_, err := s.db.Exec(`
INSERT INTO admin_sessions (session_id, created_at, expires_at)
VALUES (?, ?, ?)
ON CONFLICT(session_id) DO UPDATE SET
	created_at = excluded.created_at,
	expires_at = excluded.expires_at
`, sessionID, time.Now().Unix(), expiresAt)
	return err
}

func (s *Store) HasAdminSession(sessionID string) (bool, error) {
	if strings.TrimSpace(sessionID) == "" {
		return false, nil
	}
	var expiresAt int64
	err := s.db.QueryRow(`SELECT expires_at FROM admin_sessions WHERE session_id = ?`, sessionID).Scan(&expiresAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if expiresAt > 0 && expiresAt < time.Now().Unix() {
		_, _ = s.db.Exec(`DELETE FROM admin_sessions WHERE session_id = ?`, sessionID)
		return false, nil
	}
	return true, nil
}

func (s *Store) DeleteAdminSession(sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM admin_sessions WHERE session_id = ?`, sessionID)
	return err
}

func (s *Store) CleanupExpiredAdminSessions() error {
	_, err := s.db.Exec(`DELETE FROM admin_sessions WHERE expires_at > 0 AND expires_at < ?`, time.Now().Unix())
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

func aggregateDiskTotals(disks []models.DiskStats) (uint64, uint64) {
	var used uint64
	var total uint64
	for _, disk := range disks {
		used += disk.Used
		total += disk.Total
	}
	return used, total
}

// Keep this lookup index-friendly: /api/nodes calls it for every node shown on
// dashboard/settings, so scanning the full metrics history or using window
// functions here quickly regresses page load time on real deployments.
func latestMetricsLookupQuery() string {
	return `
SELECT ts, cpu_percent, mem_used, mem_total, disk_used, disk_total, net_rx, net_tx, uptime_seconds
FROM metrics
WHERE node_id = ?
ORDER BY ts DESC, id DESC
LIMIT 1`
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
	return updateNodeMetadataWith(s.db, nodeID, ip, osName, arch, agentVersion, hardware, lastSeen)
}

func updateNodeMetadataWith(db sqlExecer, nodeID, ip, osName, arch, agentVersion string, hardware *models.NodeHardware, lastSeen int64) error {
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
	_, err = db.Exec(query, args...)
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
		`DELETE FROM metrics_1m WHERE node_id = ?`,
		`DELETE FROM monitor_results WHERE node_id = ?`,
		`DELETE FROM latency_1m WHERE node_id = ?`,
		`DELETE FROM monitor_item_nodes WHERE node_id = ?`,
		`DELETE FROM processes WHERE node_id = ?`,
		`DELETE FROM docker_containers WHERE node_id = ?`,
		`DELETE FROM service_checks WHERE node_id = ?`,
		`DELETE FROM update_job_targets WHERE node_id = ?`,
		`DELETE FROM alert_deliveries WHERE node_id = ?`,
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
	return insertMetricsWith(s.db, nodeID, m)
}

func insertMetricsWith(db sqlExecer, nodeID string, m *models.MetricsPayload) error {
	if m == nil {
		return fmt.Errorf("metrics payload is nil")
	}

	diskUsed, diskTotal := aggregateDiskTotals(m.Disk)
	_, err := db.Exec(`
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

func (s *Store) QueryMetrics1m(nodeID string, from, to int64) ([]*MetricsRow, error) {
	rows, err := s.db.Query(`
SELECT ts, cpu_avg, mem_used_avg, mem_total_max, disk_used_avg, disk_total_max, net_rx_max, net_tx_max, uptime_seconds_max
FROM metrics_1m WHERE node_id = ? AND ts BETWEEN ? AND ? ORDER BY ts`,
		nodeID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*MetricsRow
	for rows.Next() {
		var r MetricsRow
		var memUsedAvg int64
		var memTotalMax int64
		var diskUsedAvg int64
		var diskTotalMax int64
		var netRxMax int64
		var netTxMax int64
		var uptimeMax int64
		if err := rows.Scan(&r.TS, &r.CPU, &memUsedAvg, &memTotalMax, &diskUsedAvg, &diskTotalMax, &netRxMax, &netTxMax, &uptimeMax); err != nil {
			return nil, err
		}
		if memUsedAvg > 0 {
			r.MemUsed = uint64(memUsedAvg)
		}
		if memTotalMax > 0 {
			r.MemTotal = uint64(memTotalMax)
		}
		if diskUsedAvg > 0 {
			r.DiskUsed = uint64(diskUsedAvg)
		}
		if diskTotalMax > 0 {
			r.DiskTotal = uint64(diskTotalMax)
		}
		if netRxMax > 0 {
			r.NetRx = uint64(netRxMax)
		}
		if netTxMax > 0 {
			r.NetTx = uint64(netTxMax)
		}
		if uptimeMax > 0 {
			r.UptimeSeconds = uint64(uptimeMax)
		}
		result = append(result, &r)
	}
	return result, rows.Err()
}

// RollupMetrics1m aggregates raw metrics into 1-minute buckets over [from, to].
// from/to are unix timestamps in seconds.
func (s *Store) RollupMetrics1m(from, to int64) error {
	// Align to minute boundaries.
	from = (from / 60) * 60
	to = (to / 60) * 60
	if to < from {
		return nil
	}
	_, err := s.db.Exec(`
INSERT INTO metrics_1m (
  node_id, ts, samples,
  cpu_avg, cpu_max,
  mem_used_avg, mem_used_max, mem_total_max,
  disk_used_avg, disk_used_max, disk_total_max,
  net_rx_max, net_tx_max,
  uptime_seconds_max
)
SELECT
  node_id,
  (ts/60)*60 AS minute_ts,
  COUNT(*) AS samples,
  AVG(cpu_percent) AS cpu_avg,
  MAX(cpu_percent) AS cpu_max,
  CAST(AVG(mem_used) AS INTEGER) AS mem_used_avg,
  MAX(mem_used) AS mem_used_max,
  MAX(mem_total) AS mem_total_max,
  CAST(AVG(disk_used) AS INTEGER) AS disk_used_avg,
  MAX(disk_used) AS disk_used_max,
  MAX(disk_total) AS disk_total_max,
  MAX(net_rx) AS net_rx_max,
  MAX(net_tx) AS net_tx_max,
  MAX(uptime_seconds) AS uptime_seconds_max
FROM metrics
WHERE ts BETWEEN ? AND ?
GROUP BY node_id, minute_ts
ON CONFLICT(node_id, ts) DO UPDATE SET
  samples            = excluded.samples,
  cpu_avg            = excluded.cpu_avg,
  cpu_max            = excluded.cpu_max,
  mem_used_avg       = excluded.mem_used_avg,
  mem_used_max       = excluded.mem_used_max,
  mem_total_max      = excluded.mem_total_max,
  disk_used_avg      = excluded.disk_used_avg,
  disk_used_max      = excluded.disk_used_max,
  disk_total_max     = excluded.disk_total_max,
  net_rx_max         = excluded.net_rx_max,
  net_tx_max         = excluded.net_tx_max,
  uptime_seconds_max = excluded.uptime_seconds_max
`, from, to)
	return err
}

// LatestMetricsByNodeIDs returns the most recent metrics sample for each node ID.
func (s *Store) LatestMetricsByNodeIDs(nodeIDs []string) (map[string]*models.NodeMetricsSnapshot, error) {
	result := make(map[string]*models.NodeMetricsSnapshot, len(nodeIDs))
	seen := make(map[string]struct{}, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		nodeID = strings.TrimSpace(nodeID)
		if nodeID == "" {
			continue
		}
		if _, ok := seen[nodeID]; ok {
			continue
		}
		seen[nodeID] = struct{}{}
	}
	if len(seen) == 0 {
		return result, nil
	}

	for nodeID := range seen {
		var snapshot models.NodeMetricsSnapshot
		err := s.db.QueryRow(latestMetricsLookupQuery(), nodeID).Scan(
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

func (s *Store) ListNodesWithLatestMetrics() ([]*models.Node, error) {
	nodes, err := s.ListNodes()
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}

	nodeIDs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}

	latestMetrics, err := s.LatestMetricsByNodeIDs(nodeIDs)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		node.LatestMetrics = latestMetrics[node.ID]
	}
	return nodes, nil
}

// PruneOldMetrics deletes historical raw and rolled-up monitoring rows older than retentionDays days.
func (s *Store) PruneOldMetrics(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM metrics WHERE ts < ?`, cutoff); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM metrics_1m WHERE ts < ?`, cutoff); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM monitor_results WHERE ts < ?`, cutoff); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM latency_1m WHERE ts < ?`, cutoff); err != nil {
		return err
	}

	return tx.Commit()
}

// -------------------------------------------------------------------------
// Process operations
// -------------------------------------------------------------------------

// UpsertProcesses stores a pre-serialized JSON process list for a node,
// overwriting any existing record.
func (s *Store) UpsertProcesses(nodeID string, ts int64, data string) error {
	return upsertProcessesWith(s.db, nodeID, ts, data)
}

func upsertProcessesWith(db sqlExecer, nodeID string, ts int64, data string) error {
	_, err := db.Exec(`
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
// Docker operations
// -------------------------------------------------------------------------

// UpsertDockerContainers stores Docker availability and a pre-serialized JSON
// container list for a node, overwriting any existing record.
func (s *Store) UpsertDockerContainers(nodeID string, ts int64, dockerAvailable bool, data string) error {
	return upsertDockerContainersWith(s.db, nodeID, ts, dockerAvailable, data)
}

func upsertDockerContainersWith(db sqlExecer, nodeID string, ts int64, dockerAvailable bool, data string) error {
	if strings.TrimSpace(data) == "" {
		data = "[]"
	}

	available := 0
	if dockerAvailable {
		available = 1
	}

	_, err := db.Exec(`
		INSERT INTO docker_containers (node_id, ts, docker_available, data) VALUES (?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET ts=excluded.ts, docker_available=excluded.docker_available, data=excluded.data
	`, nodeID, ts, available, data)
	return err
}

// GetDockerContainers returns Docker availability and the JSON-encoded container
// list for a node. Returns (false, "[]", nil) when no data is found.
func (s *Store) GetDockerContainers(nodeID string) (bool, string, error) {
	var dockerAvailable int64
	var data string
	err := s.db.QueryRow(`SELECT docker_available, data FROM docker_containers WHERE node_id = ?`, nodeID).Scan(&dockerAvailable, &data)
	if err == sql.ErrNoRows {
		return false, "[]", nil
	}
	if err != nil {
		return false, "[]", err
	}
	return dockerAvailable != 0, data, nil
}

// -------------------------------------------------------------------------
// Service check operations
// -------------------------------------------------------------------------

// UpsertServiceCheck inserts or updates a service check result for a node.
func (s *Store) UpsertServiceCheck(nodeID, name, status string) error {
	return upsertServiceCheckWith(s.db, nodeID, name, status, time.Now().Unix())
}

func upsertServiceCheckWith(db sqlExecer, nodeID, name, status string, checkedAt int64) error {
	if checkedAt <= 0 {
		checkedAt = time.Now().Unix()
	}

	_, err := db.Exec(`
		INSERT INTO service_checks (node_id, name, status, last_checked) VALUES (?, ?, ?, ?)
		ON CONFLICT(node_id, name) DO UPDATE SET status=excluded.status, last_checked=excluded.last_checked
	`, nodeID, name, status, checkedAt)
	return err
}

func (s *Store) ApplyAgentSnapshot(nodeID string, payload *models.MetricsPayload, resolvedIP string, lastSeen int64) error {
	if payload == nil {
		return fmt.Errorf("metrics payload is nil")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := insertMetricsWith(tx, nodeID, payload); err != nil {
		return err
	}
	if err := updateNodeMetadataWith(tx, nodeID, resolvedIP, payload.OS, payload.Arch, payload.AgentVersion, payload.Hardware, lastSeen); err != nil {
		return err
	}
	if len(payload.Processes) > 0 {
		processesJSON, err := json.Marshal(payload.Processes)
		if err != nil {
			return err
		}
		if err := upsertProcessesWith(tx, nodeID, payload.TS, string(processesJSON)); err != nil {
			return err
		}
	}
	for _, service := range payload.Services {
		if err := upsertServiceCheckWith(tx, nodeID, service.Name, service.Status, payload.TS); err != nil {
			return err
		}
	}
	if payload.DockerAvailable != nil {
		dockerAvailable := *payload.DockerAvailable
		containers := payload.Containers
		if containers == nil || !dockerAvailable {
			containers = []models.DockerContainer{}
		}
		containersJSON, err := json.Marshal(containers)
		if err != nil {
			return err
		}
		if err := upsertDockerContainersWith(tx, nodeID, payload.TS, dockerAvailable, string(containersJSON)); err != nil {
			return err
		}
	}

	return tx.Commit()
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
