package models_test

import (
	"encoding/json"
	"testing"

	"github.com/thism-dev/thism/internal/models"
)

func TestMetricsPayloadJSON(t *testing.T) {
	payload := models.MetricsPayload{
		Type: "metrics",
		TS:   1709500000,
		CPU:  23.5,
		Mem:  models.MemStats{Used: 2048, Total: 8192},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var out models.MetricsPayload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if out.CPU != 23.5 {
		t.Errorf("expected CPU 23.5, got %v", out.CPU)
	}
}
