package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/thism-dev/thism/internal/models"
	_ "modernc.org/sqlite"
)

// Store is the SQLite-backed data access layer.
type Store struct {
	db *sql.DB
}

// MetricsRow is a flat struct representing a single metrics sample for API use.
type MetricsRow struct {
	TS       int64   `json:"ts"`
	CPU      float64 `json:"cpu"`
	MemUsed  uint64  `json:"mem_used"`
	MemTotal uint64  `json:"mem_total"`
	DiskUsed uint64  `json:"disk_used"`
	DiskTotal uint64 `json:"disk_total"`
	NetRx    uint64  `json:"net_rx"`
	NetTx    uint64  `json:"net_tx"`
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
// modernc.org/sqlite supports multiple statements separated by semicolons in a
// single Exec call.
func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS nodes (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    token      TEXT NOT NULL UNIQUE,
    ip         TEXT,
    os         TEXT,
    arch       TEXT,
    created_at INTEGER NOT NULL DEFAULT 0,
    last_seen  INTEGER NOT NULL DEFAULT 0,
    online     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS metrics (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id    TEXT    NOT NULL,
    ts         INTEGER NOT NULL,
    cpu        REAL    NOT NULL DEFAULT 0,
    mem_used   INTEGER NOT NULL DEFAULT 0,
    mem_total  INTEGER NOT NULL DEFAULT 0,
    disk_used  INTEGER NOT NULL DEFAULT 0,
    disk_total INTEGER NOT NULL DEFAULT 0,
    net_rx     INTEGER NOT NULL DEFAULT 0,
    net_tx     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_metrics_node_ts ON metrics (node_id, ts);

CREATE TABLE IF NOT EXISTS processes (
    node_id TEXT PRIMARY KEY,
    data    TEXT NOT NULL DEFAULT '[]',
    updated INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS service_checks (
    node_id TEXT    NOT NULL,
    name    TEXT    NOT NULL,
    status  TEXT    NOT NULL DEFAULT '',
    checked INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (node_id, name)
);
`)
	return err
}

// -------------------------------------------------------------------------
// Node operations
// -------------------------------------------------------------------------

// UpsertNode inserts or updates a node record.
func (s *Store) UpsertNode(node *models.Node) error {
	online := 0
	if node.Online {
		online = 1
	}
	_, err := s.db.Exec(`
INSERT INTO nodes (id, name, token, ip, os, arch, created_at, last_seen, online)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name       = excluded.name,
    token      = excluded.token,
    ip         = excluded.ip,
    os         = excluded.os,
    arch       = excluded.arch,
    last_seen  = excluded.last_seen,
    online     = excluded.online
`,
		node.ID, node.Name, node.Token, node.IP, node.OS, node.Arch,
		node.CreatedAt, node.LastSeen, online,
	)
	return err
}

// GetNodeByToken returns the node with the given token, or (nil, nil) if not found.
func (s *Store) GetNodeByToken(token string) (*models.Node, error) {
	row := s.db.QueryRow(`
SELECT id, name, token, ip, os, arch, created_at, last_seen, online
FROM nodes WHERE token = ?`, token)

	var n models.Node
	var online int
	err := row.Scan(&n.ID, &n.Name, &n.Token, &n.IP, &n.OS, &n.Arch,
		&n.CreatedAt, &n.LastSeen, &online)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Online = online != 0
	return &n, nil
}

// GetNodeByID returns the node with the given ID, or (nil, nil) if not found.
func (s *Store) GetNodeByID(id string) (*models.Node, error) {
	row := s.db.QueryRow(`
SELECT id, name, token, ip, os, arch, created_at, last_seen, online
FROM nodes WHERE id = ?`, id)

	var n models.Node
	var online int
	err := row.Scan(&n.ID, &n.Name, &n.Token, &n.IP, &n.OS, &n.Arch,
		&n.CreatedAt, &n.LastSeen, &online)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Online = online != 0
	return &n, nil
}

// ListNodes returns all registered nodes.
func (s *Store) ListNodes() ([]*models.Node, error) {
	rows, err := s.db.Query(`
SELECT id, name, token, ip, os, arch, created_at, last_seen, online
FROM nodes ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*models.Node
	for rows.Next() {
		var n models.Node
		var online int
		if err := rows.Scan(&n.ID, &n.Name, &n.Token, &n.IP, &n.OS, &n.Arch,
			&n.CreatedAt, &n.LastSeen, &online); err != nil {
			return nil, err
		}
		n.Online = online != 0
		nodes = append(nodes, &n)
	}
	return nodes, rows.Err()
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
INSERT INTO metrics (node_id, ts, cpu, mem_used, mem_total, disk_used, disk_total, net_rx, net_tx)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nodeID, m.TS, m.CPU,
		m.Mem.Used, m.Mem.Total,
		diskUsed, diskTotal,
		m.Net.RxBytes, m.Net.TxBytes,
	)
	return err
}

// QueryMetrics returns metrics rows for a node within the given time range,
// ordered by ascending timestamp.
func (s *Store) QueryMetrics(nodeID string, from, to int64) ([]*MetricsRow, error) {
	rows, err := s.db.Query(`
SELECT ts, cpu, mem_used, mem_total, disk_used, disk_total, net_rx, net_tx
FROM metrics
WHERE node_id = ? AND ts >= ? AND ts <= ?
ORDER BY ts ASC`,
		nodeID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*MetricsRow
	for rows.Next() {
		var r MetricsRow
		if err := rows.Scan(&r.TS, &r.CPU, &r.MemUsed, &r.MemTotal,
			&r.DiskUsed, &r.DiskTotal, &r.NetRx, &r.NetTx); err != nil {
			return nil, err
		}
		result = append(result, &r)
	}
	return result, rows.Err()
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

// UpsertProcesses stores the process list for a node as a JSON blob,
// overwriting any existing record.
func (s *Store) UpsertProcesses(nodeID string, processes []models.Process) error {
	data, err := json.Marshal(processes)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
INSERT INTO processes (node_id, data, updated) VALUES (?, ?, ?)
ON CONFLICT(node_id) DO UPDATE SET data = excluded.data, updated = excluded.updated`,
		nodeID, string(data), time.Now().Unix(),
	)
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
func (s *Store) UpsertServiceCheck(nodeID string, svc models.Service) error {
	_, err := s.db.Exec(`
INSERT INTO service_checks (node_id, name, status, checked) VALUES (?, ?, ?, ?)
ON CONFLICT(node_id, name) DO UPDATE SET status = excluded.status, checked = excluded.checked`,
		nodeID, svc.Name, svc.Status, time.Now().Unix(),
	)
	return err
}

// GetServiceChecks returns all service checks for a node as a slice of maps.
func (s *Store) GetServiceChecks(nodeID string) ([]map[string]any, error) {
	rows, err := s.db.Query(`
SELECT name, status, checked FROM service_checks WHERE node_id = ? ORDER BY name`,
		nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var name, status string
		var checked int64
		if err := rows.Scan(&name, &status, &checked); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{
			"name":    name,
			"status":  status,
			"checked": checked,
		})
	}
	return result, rows.Err()
}
