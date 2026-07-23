package collector

import (
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func TestCollectHardwareProfile(t *testing.T) {
	originalCPUInfoFunc := cpuInfoFunc
	originalCPUCountsFunc := cpuCountsFunc
	originalVirtualMemoryFunc := virtualMemoryFunc
	originalHostInfoFunc := hostInfoFunc
	originalDiskPartitionsFunc := diskPartitionsFunc
	originalDiskUsageFunc := diskUsageFunc
	defer func() {
		cpuInfoFunc = originalCPUInfoFunc
		cpuCountsFunc = originalCPUCountsFunc
		virtualMemoryFunc = originalVirtualMemoryFunc
		hostInfoFunc = originalHostInfoFunc
		diskPartitionsFunc = originalDiskPartitionsFunc
		diskUsageFunc = originalDiskUsageFunc
	}()

	cpuInfoFunc = func() ([]cpu.InfoStat, error) {
		return []cpu.InfoStat{{ModelName: "AMD EPYC 7B13", Cores: 8}}, nil
	}
	cpuCountsFunc = func(logical bool) (int, error) {
		if logical {
			return 16, nil
		}
		return 8, nil
	}
	virtualMemoryFunc = func() (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Total: 34359738368}, nil
	}
	hostInfoFunc = func() (*host.InfoStat, error) {
		return &host.InfoStat{VirtualizationSystem: "kvm", VirtualizationRole: "guest"}, nil
	}
	diskPartitionsFunc = func(all bool) ([]disk.PartitionStat, error) {
		return []disk.PartitionStat{
			{Device: "/dev/sda2", Mountpoint: "/"},
			{Device: "/dev/sdb1", Mountpoint: "/data"},
		}, nil
	}
	diskUsageFunc = func(path string) (*disk.UsageStat, error) {
		if path == "/" {
			return &disk.UsageStat{Total: 107374182400}, nil
		}
		return &disk.UsageStat{Total: 214748364800}, nil
	}

	profile := collectHardwareProfile()
	if profile == nil {
		t.Fatal("expected hardware profile")
	}
	if profile.CPUModel != "AMD EPYC 7B13" {
		t.Fatalf("expected cpu model, got %q", profile.CPUModel)
	}
	if profile.CPUCores != 8 || profile.CPUThreads != 16 {
		t.Fatalf("expected cores/threads 8/16, got %d/%d", profile.CPUCores, profile.CPUThreads)
	}
	if profile.MemoryTotal != 34359738368 {
		t.Fatalf("expected memory total, got %d", profile.MemoryTotal)
	}
	if profile.DiskTotal != 322122547200 {
		t.Fatalf("expected aggregated disk total, got %d", profile.DiskTotal)
	}
	if profile.VirtualizationSystem != "kvm" || profile.VirtualizationRole != "guest" {
		t.Fatalf("expected virtualization kvm/guest, got %q/%q", profile.VirtualizationSystem, profile.VirtualizationRole)
	}
}

func TestCollectHardwareProfileDedupesBindMountsOfSameDevice(t *testing.T) {
	originalCPUInfoFunc := cpuInfoFunc
	originalCPUCountsFunc := cpuCountsFunc
	originalVirtualMemoryFunc := virtualMemoryFunc
	originalHostInfoFunc := hostInfoFunc
	originalDiskPartitionsFunc := diskPartitionsFunc
	originalDiskUsageFunc := diskUsageFunc
	defer func() {
		cpuInfoFunc = originalCPUInfoFunc
		cpuCountsFunc = originalCPUCountsFunc
		virtualMemoryFunc = originalVirtualMemoryFunc
		hostInfoFunc = originalHostInfoFunc
		diskPartitionsFunc = originalDiskPartitionsFunc
		diskUsageFunc = originalDiskUsageFunc
	}()

	cpuInfoFunc = func() ([]cpu.InfoStat, error) { return nil, errors.New("unused") }
	cpuCountsFunc = func(logical bool) (int, error) { return 0, errors.New("unused") }
	virtualMemoryFunc = func() (*mem.VirtualMemoryStat, error) { return nil, errors.New("unused") }
	hostInfoFunc = func() (*host.InfoStat, error) { return nil, errors.New("unused") }
	diskPartitionsFunc = func(all bool) ([]disk.PartitionStat, error) {
		return []disk.PartitionStat{
			{Device: "/dev/sda2", Mountpoint: "/"},
			{Device: "/dev/sda1", Mountpoint: "/boot/efi"},
			{Device: "/dev/sda2", Mountpoint: "/opt/hermes-agent"},
			{Device: "/dev/sda2", Mountpoint: "/opt/hermes-script"},
		}, nil
	}
	diskUsageFunc = func(path string) (*disk.UsageStat, error) {
		switch path {
		case "/":
			return &disk.UsageStat{Total: 209711013888}, nil
		case "/boot/efi":
			return &disk.UsageStat{Total: 536576000}, nil
		case "/opt/hermes-agent", "/opt/hermes-script":
			return &disk.UsageStat{Total: 209711013888}, nil
		default:
			return nil, errors.New("unexpected mount")
		}
	}

	profile := collectHardwareProfile()
	if profile == nil {
		t.Fatal("expected hardware profile")
	}
	const want = uint64(209711013888 + 536576000)
	if profile.DiskTotal != want {
		t.Fatalf("expected bind mounts of /dev/sda2 to count once (want %d), got %d", want, profile.DiskTotal)
	}
}

func TestCollectHardwareProfileReturnsPartialDataWhenSomeSourcesFail(t *testing.T) {
	originalCPUInfoFunc := cpuInfoFunc
	originalCPUCountsFunc := cpuCountsFunc
	originalVirtualMemoryFunc := virtualMemoryFunc
	originalHostInfoFunc := hostInfoFunc
	originalDiskPartitionsFunc := diskPartitionsFunc
	originalDiskUsageFunc := diskUsageFunc
	defer func() {
		cpuInfoFunc = originalCPUInfoFunc
		cpuCountsFunc = originalCPUCountsFunc
		virtualMemoryFunc = originalVirtualMemoryFunc
		hostInfoFunc = originalHostInfoFunc
		diskPartitionsFunc = originalDiskPartitionsFunc
		diskUsageFunc = originalDiskUsageFunc
	}()

	cpuInfoFunc = func() ([]cpu.InfoStat, error) { return nil, errors.New("cpu info unavailable") }
	cpuCountsFunc = func(logical bool) (int, error) { return 0, errors.New("counts unavailable") }
	virtualMemoryFunc = func() (*mem.VirtualMemoryStat, error) { return &mem.VirtualMemoryStat{Total: 1024}, nil }
	hostInfoFunc = func() (*host.InfoStat, error) { return nil, errors.New("host info unavailable") }
	diskPartitionsFunc = func(all bool) ([]disk.PartitionStat, error) { return nil, errors.New("disk partitions unavailable") }
	diskUsageFunc = func(path string) (*disk.UsageStat, error) { return nil, errors.New("disk usage unavailable") }

	profile := collectHardwareProfile()
	if profile == nil {
		t.Fatal("expected partial hardware profile")
	}
	if profile.MemoryTotal != 1024 {
		t.Fatalf("expected partial memory total, got %d", profile.MemoryTotal)
	}
}
