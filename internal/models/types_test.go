package models_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/thism-dev/thism/internal/models"
)

func TestMetricsPayloadJSON(t *testing.T) {
	payload := models.MetricsPayload{
		Type:          "metrics",
		TS:            1709500000,
		CPU:           23.5,
		UptimeSeconds: 3723,
		Load:          models.LoadStats{Load1: 0.42, Load5: 0.52, Load15: 0.62},
		CPUStats:      models.CPUStats{IOWaitPercent: 3.5, StealPercent: 1.25},
		Pressure:      models.PressureStats{CPUSomeAvg10: 0.7, MemorySomeAvg10: 1.1, MemoryFullAvg10: 0.2, IOSomeAvg10: 2.3, IOFullAvg10: 0.4},
		Swap:          models.SwapStats{Used: 512, Total: 2048, In: 128, Out: 256},
		OOMKills:      2,
		DiskHealth: []models.DiskHealthStats{
			{
				Name:          "nvme0n1",
				Path:          "/dev/nvme0n1",
				Type:          "nvme",
				Model:         "FastDisk",
				Serial:        "SN123",
				SizeBytes:     1024,
				Status:        models.DiskHealthStatusOK,
				SupportStatus: models.DiskHealthSupportSupported,
				Mounts: []models.DiskHealthMount{
					{Mount: "/", FSType: "ext4", ReadOnly: false},
				},
				TemperatureC:          float64Ptr(36),
				LifeUsedPercent:       float64Ptr(4),
				AvailableSparePercent: float64Ptr(95),
				MediaErrors:           0,
				ReallocatedSectors:    2,
				PendingSectors:        1,
				InterfaceCRCErrors:    3,
				UnsafeShutdowns:       2,
				PowerOnHours:          100,
			},
		},
		Mem: models.MemStats{Used: 2048, Total: 8192},
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
	if out.UptimeSeconds != 3723 {
		t.Errorf("expected uptime 3723, got %d", out.UptimeSeconds)
	}
	if out.Load.Load1 != 0.42 || out.Load.Load5 != 0.52 || out.Load.Load15 != 0.62 {
		t.Fatalf("expected load stats to round-trip, got %#v", out.Load)
	}
	if out.CPUStats.IOWaitPercent != 3.5 || out.CPUStats.StealPercent != 1.25 {
		t.Fatalf("expected cpu stats to round-trip, got %#v", out.CPUStats)
	}
	if out.Pressure.CPUSomeAvg10 != 0.7 || out.Pressure.MemorySomeAvg10 != 1.1 || out.Pressure.MemoryFullAvg10 != 0.2 || out.Pressure.IOSomeAvg10 != 2.3 || out.Pressure.IOFullAvg10 != 0.4 {
		t.Fatalf("expected pressure stats to round-trip, got %#v", out.Pressure)
	}
	if out.Swap.Used != 512 || out.Swap.Total != 2048 || out.Swap.In != 128 || out.Swap.Out != 256 {
		t.Fatalf("expected swap stats to round-trip, got %#v", out.Swap)
	}
	if out.OOMKills != 2 {
		t.Fatalf("expected oom kills to round-trip, got %d", out.OOMKills)
	}
	if len(out.DiskHealth) != 1 || out.DiskHealth[0].Name != "nvme0n1" || out.DiskHealth[0].Status != models.DiskHealthStatusOK {
		t.Fatalf("expected disk health to round-trip, got %#v", out.DiskHealth)
	}
	if out.DiskHealth[0].SupportStatus != models.DiskHealthSupportSupported {
		t.Fatalf("expected disk health support status to round-trip, got %#v", out.DiskHealth[0])
	}
	if len(out.DiskHealth[0].Mounts) != 1 || out.DiskHealth[0].Mounts[0].Mount != "/" || out.DiskHealth[0].Mounts[0].FSType != "ext4" || out.DiskHealth[0].Mounts[0].ReadOnly {
		t.Fatalf("expected disk mount observations to round-trip, got %#v", out.DiskHealth[0].Mounts)
	}
	if out.DiskHealth[0].TemperatureC == nil || *out.DiskHealth[0].TemperatureC != 36 {
		t.Fatalf("expected disk temperature to round-trip, got %#v", out.DiskHealth[0].TemperatureC)
	}
	if out.DiskHealth[0].ReallocatedSectors != 2 || out.DiskHealth[0].PendingSectors != 1 || out.DiskHealth[0].InterfaceCRCErrors != 3 {
		t.Fatalf("expected ATA counters to round-trip, got %#v", out.DiskHealth[0])
	}
}

func TestAggregateDiskTotalsDedupesSameDevice(t *testing.T) {
	used, total := models.AggregateDiskTotals([]models.DiskStats{
		{Mount: "/", Device: "/dev/sda2", Used: 90, Total: 200},
		{Mount: "/boot/efi", Device: "/dev/sda1", Used: 1, Total: 1},
		{Mount: "/opt/hermes-agent", Device: "/dev/sda2", Used: 90, Total: 200},
		{Mount: "/opt/hermes-script", Device: "/dev/sda2", Used: 90, Total: 200},
	})
	if used != 91 || total != 201 {
		t.Fatalf("expected bind mounts of the same device to count once, got used=%d total=%d", used, total)
	}
}

func TestAggregateDiskTotalsCountsMountsWithoutDevice(t *testing.T) {
	used, total := models.AggregateDiskTotals([]models.DiskStats{
		{Mount: "/", Used: 50, Total: 100},
		{Mount: "/data", Used: 20, Total: 40},
	})
	if used != 70 || total != 140 {
		t.Fatalf("expected legacy mounts without device to sum fully, got used=%d total=%d", used, total)
	}
}

func TestMetricsPayloadEmptyDiskHealthEncodesEmptyArray(t *testing.T) {
	payload := models.MetricsPayload{
		Type:       "metrics",
		TS:         1709500000,
		DiskHealth: []models.DiskHealthStats{},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(b), `"disk_health":[]`) {
		t.Fatalf("expected empty disk health to encode as [], got %s", b)
	}
}

func float64Ptr(value float64) *float64 {
	return &value
}
