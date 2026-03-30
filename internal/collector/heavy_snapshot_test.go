package collector

import (
	"errors"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/thism-dev/thism/internal/models"
)

func TestCollectThrottlesHeavySnapshots(t *testing.T) {
	originalCPUPercentFunc := cpuPercentFunc
	originalVirtualMemoryFunc := virtualMemoryFunc
	originalHostInfoFunc := hostInfoFunc
	originalDiskPartitionsFunc := diskPartitionsFunc
	originalIOCountersFunc := ioCountersFunc
	originalReadFileFunc := readFileFunc
	originalCollectProcessSamplesFunc := collectProcessSamplesFunc
	originalCollectDockerContainersFunc := collectDockerContainersFunc
	defer func() {
		cpuPercentFunc = originalCPUPercentFunc
		virtualMemoryFunc = originalVirtualMemoryFunc
		hostInfoFunc = originalHostInfoFunc
		diskPartitionsFunc = originalDiskPartitionsFunc
		ioCountersFunc = originalIOCountersFunc
		readFileFunc = originalReadFileFunc
		collectProcessSamplesFunc = originalCollectProcessSamplesFunc
		collectDockerContainersFunc = originalCollectDockerContainersFunc
	}()

	cpuPercentFunc = func(time.Duration, bool) ([]float64, error) {
		return []float64{25}, nil
	}
	virtualMemoryFunc = func() (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Used: 1024, Total: 4096}, nil
	}
	hostInfoFunc = func() (*host.InfoStat, error) {
		return &host.InfoStat{Uptime: 3600}, nil
	}
	diskPartitionsFunc = func(bool) ([]disk.PartitionStat, error) {
		return nil, nil
	}
	ioCountersFunc = func(bool) ([]psnet.IOCountersStat, error) {
		return nil, nil
	}
	readFileFunc = func(string) ([]byte, error) {
		return nil, errors.New("no route file")
	}

	processCalls := 0
	collectProcessSamplesFunc = func() ([]models.Process, error) {
		processCalls += 1
		return []models.Process{{PID: 1, Name: "api", CPUPercent: 1.5, MemRSS: 1024}}, nil
	}

	dockerCalls := 0
	collectDockerContainersFunc = func() ([]models.DockerContainer, bool, error) {
		dockerCalls += 1
		return []models.DockerContainer{{ID: "abcdef123456", Name: "web", Image: "nginx:latest"}}, true, nil
	}

	currentTime := time.Unix(1000, 0)
	c := New("ws://localhost:9999", "token", "test", "")
	c.now = func() time.Time { return currentTime }
	c.heavySnapshotInterval = time.Minute
	c.hardwareProfile = &models.NodeHardware{CPUModel: "Test CPU"}

	first, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect first: %v", err)
	}
	if first.Hardware == nil || first.Hardware.CPUModel != "Test CPU" {
		t.Fatalf("expected first report to include hardware snapshot, got %#v", first.Hardware)
	}
	if len(first.Processes) != 1 || first.Processes[0].Name != "api" {
		t.Fatalf("expected first report to include processes snapshot, got %#v", first.Processes)
	}
	if first.DockerAvailable == nil || !*first.DockerAvailable {
		t.Fatalf("expected first report to include docker availability, got %#v", first.DockerAvailable)
	}

	currentTime = currentTime.Add(5 * time.Second)
	second, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect second: %v", err)
	}
	if second.Hardware != nil {
		t.Fatalf("expected light report to omit hardware snapshot, got %#v", second.Hardware)
	}
	if second.Processes != nil {
		t.Fatalf("expected light report to omit processes snapshot, got %#v", second.Processes)
	}
	if second.DockerAvailable != nil {
		t.Fatalf("expected light report to omit docker snapshot, got %#v", second.DockerAvailable)
	}
	if processCalls != 1 || dockerCalls != 1 {
		t.Fatalf("expected heavy collectors to run once before interval, got processCalls=%d dockerCalls=%d", processCalls, dockerCalls)
	}

	currentTime = currentTime.Add(time.Minute)
	third, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect third: %v", err)
	}
	if third.Hardware != nil {
		t.Fatalf("expected later heavy report to omit static hardware snapshot after first send, got %#v", third.Hardware)
	}
	if len(third.Processes) != 1 || third.Processes[0].Name != "api" {
		t.Fatalf("expected heavy report after interval to include processes snapshot, got %#v", third.Processes)
	}
	if third.DockerAvailable == nil || !*third.DockerAvailable {
		t.Fatalf("expected heavy report after interval to include docker availability, got %#v", third.DockerAvailable)
	}
	if processCalls != 2 || dockerCalls != 2 {
		t.Fatalf("expected heavy collectors to run again after interval, got processCalls=%d dockerCalls=%d", processCalls, dockerCalls)
	}
}
