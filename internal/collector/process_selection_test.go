package collector

import (
	"testing"

	"github.com/thism-dev/thism/internal/models"
)

func TestSelectTopProcessesFiltersAndSorts(t *testing.T) {
	input := []models.Process{
		{PID: 10, Name: "kernel-helper", CPUPercent: 80, MemRSS: 0},
		{PID: 20, Name: "db", CPUPercent: 5, MemRSS: 2048},
		{PID: 21, Name: "api", CPUPercent: 9, MemRSS: 1024},
		{PID: 22, Name: "worker", CPUPercent: 9, MemRSS: 4096},
		{PID: 23, Name: "idle", CPUPercent: 0.1, MemRSS: 512},
	}

	got := selectTopProcesses(input, 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 processes, got %d", len(got))
	}
	if got[0].Name != "worker" {
		t.Fatalf("expected top process worker, got %s", got[0].Name)
	}
	if got[1].Name != "api" {
		t.Fatalf("expected second process api, got %s", got[1].Name)
	}
	if got[2].Name != "db" {
		t.Fatalf("expected third process db, got %s", got[2].Name)
	}
}

func TestSelectTopProcessesHandlesEmptyAndNonPositiveLimit(t *testing.T) {
	if got := selectTopProcesses(nil, 30); len(got) != 0 {
		t.Fatalf("expected empty result for nil input, got %d entries", len(got))
	}

	if got := selectTopProcesses([]models.Process{{PID: 1, Name: "a", CPUPercent: 1, MemRSS: 1}}, 0); len(got) != 0 {
		t.Fatalf("expected empty result for zero limit, got %d entries", len(got))
	}
}
