package store

import (
	"path/filepath"
	"testing"
)

func TestNewConfiguresSQLitePragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "thism.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	var journalMode string
	if err := s.db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected journal_mode=wal, got %q", journalMode)
	}

	var busyTimeout int
	if err := s.db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout < 5000 {
		t.Fatalf("expected busy_timeout >= 5000ms, got %d", busyTimeout)
	}

	var synchronous int
	if err := s.db.QueryRow(`PRAGMA synchronous`).Scan(&synchronous); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}
	if synchronous != 1 {
		t.Fatalf("expected synchronous=NORMAL (1), got %d", synchronous)
	}
}
