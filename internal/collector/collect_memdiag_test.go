package collector

import (
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/thism-dev/thism/internal/models"
)

func BenchmarkCollectAlloc(b *testing.B) {
	c := NewWithInterval("ws://127.0.0.1:12026", "x", "mem-probe", "", DefaultReportInterval)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		payload, err := c.Collect()
		if err != nil {
			b.Fatal(err)
		}
		if payload == nil {
			b.Fatal("nil payload")
		}
	}
}

func BenchmarkCollectProcessesAlloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		procs, err := process.Processes()
		if err != nil {
			b.Fatal(err)
		}
		processSamples := make([]models.Process, 0, 64)
		for _, p := range procs {
			name, err := p.Name()
			if err != nil {
				continue
			}
			memInfo, err := p.MemoryInfo()
			if err != nil {
				continue
			}
			var rss uint64
			if memInfo != nil {
				rss = memInfo.RSS
			}
			if rss == 0 {
				continue
			}
			cpuPct, err := p.CPUPercent()
			if err != nil {
				continue
			}
			processSamples = append(processSamples, models.Process{
				PID:        p.Pid,
				Name:       name,
				CPUPercent: cpuPct,
				MemRSS:     rss,
			})
		}
		_ = selectTopProcesses(processSamples, 30)
	}
}

func BenchmarkCollectDiskAlloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		partitions, err := disk.Partitions(false)
		if err != nil {
			b.Fatal(err)
		}
		disks := make([]models.DiskStats, 0, len(partitions))
		for _, p := range partitions {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}
			disks = append(disks, models.DiskStats{
				Mount: p.Mountpoint,
				Used:  usage.Used,
				Total: usage.Total,
			})
		}
		_ = disks
	}
}

func BenchmarkCollectDockerAlloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		containers, dockerAvailable, err := collectDockerContainers()
		if err != nil {
			b.Fatal(err)
		}
		if dockerAvailable && containers == nil {
			b.Fatal("docker available with nil containers")
		}
	}
}

func BenchmarkCollectNetworkAlloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = collectNetworkStats()
	}
}

func BenchmarkRuntimeNumGoroutine(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = runtime.NumGoroutine()
	}
}
