package collector

import (
	"math"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

func TestCollectUsesNonBlockingCPUSamples(t *testing.T) {
	originalCPUTimesFunc := cpuTimesFunc
	originalCPUPercentFunc := cpuPercentFunc
	defer func() {
		cpuTimesFunc = originalCPUTimesFunc
		cpuPercentFunc = originalCPUPercentFunc
	}()

	samples := [][]cpu.TimesStat{
		{{User: 10, System: 10, Idle: 80}},
		{{User: 25, System: 25, Idle: 150}},
	}
	cpuTimesFunc = func(percpu bool) ([]cpu.TimesStat, error) {
		if percpu {
			t.Fatalf("expected aggregate cpu times sample")
		}
		if len(samples) == 0 {
			t.Fatalf("cpuTimesFunc called more than expected")
		}
		sample := samples[0]
		samples = samples[1:]
		return sample, nil
	}
	cpuPercentFunc = func(time.Duration, bool) ([]float64, error) {
		t.Fatal("cpu.Percent should not be used for metrics collection")
		return nil, nil
	}

	collector := NewWithInterval("ws://localhost:9999", "token", "test", "", DefaultReportInterval)
	payload, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	const wantCPU = 30.0
	if math.Abs(payload.CPU-wantCPU) > 0.001 {
		t.Fatalf("expected cpu %.3f, got %.3f", wantCPU, payload.CPU)
	}
}
