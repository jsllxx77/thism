package store

import (
	"strings"
	"testing"
)

func TestLatestMetricsLookupQueryPlanAvoidsTempSort(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if _, err := s.db.Exec(`
INSERT INTO nodes (id, name, token, created_at, last_seen)
VALUES ('node-1', 'alpha', 'token-1', 1700000000, 1700000000)`); err != nil {
		t.Fatalf("insert node: %v", err)
	}
	if _, err := s.db.Exec(`
INSERT INTO metrics (node_id, ts, cpu_percent)
VALUES
	('node-1', 1700000010, 10.0),
	('node-1', 1700000020, 20.0)`); err != nil {
		t.Fatalf("insert metrics: %v", err)
	}

	rows, err := s.db.Query(`EXPLAIN QUERY PLAN `+latestMetricsLookupQuery(), "node-1")
	if err != nil {
		t.Fatalf("explain query plan: %v", err)
	}
	defer rows.Close()

	var details []string
	for rows.Next() {
		var id int
		var parent int
		var notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan query plan row: %v", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate query plan rows: %v", err)
	}

	for _, detail := range details {
		if strings.Contains(detail, "USE TEMP B-TREE") {
			t.Fatalf("expected latest metrics lookup to avoid temp sorting, got plan: %v", details)
		}
	}
}

func TestLatestMetricsBatchLookupQueryPlanAvoidsTempSort(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if _, err := s.db.Exec(`
INSERT INTO nodes (id, name, token, created_at, last_seen)
VALUES
	('node-1', 'alpha', 'token-1', 1700000000, 1700000000),
	('node-2', 'beta', 'token-2', 1700000000, 1700000000)`); err != nil {
		t.Fatalf("insert nodes: %v", err)
	}
	if _, err := s.db.Exec(`
INSERT INTO metrics (node_id, ts, cpu_percent)
VALUES
	('node-1', 1700000010, 10.0),
	('node-1', 1700000020, 20.0),
	('node-2', 1700000015, 30.0),
	('node-2', 1700000025, 40.0)`); err != nil {
		t.Fatalf("insert metrics: %v", err)
	}

	rows, err := s.db.Query(`EXPLAIN QUERY PLAN `+latestMetricsBatchLookupQuery(2), "node-1", "node-2")
	if err != nil {
		t.Fatalf("explain batch query plan: %v", err)
	}
	defer rows.Close()

	var details []string
	for rows.Next() {
		var id int
		var parent int
		var notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan query plan row: %v", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate batch query plan rows: %v", err)
	}

	for _, detail := range details {
		if strings.Contains(detail, "USE TEMP B-TREE") {
			t.Fatalf("expected latest metrics batch lookup to avoid temp sorting, got plan: %v", details)
		}
	}
}
