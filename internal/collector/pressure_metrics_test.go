package collector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shirou/gopsutil/v3/mem"
)

func TestParsePressureFileUsesAvg10Values(t *testing.T) {
	stats, err := parsePressureFile([]byte("some avg10=1.23 avg60=0.50 avg300=0.10 total=123\nfull avg10=0.45 avg60=0.20 avg300=0.05 total=45\n"))
	if err != nil {
		t.Fatalf("parsePressureFile: %v", err)
	}

	if stats.SomeAvg10 != 1.23 {
		t.Fatalf("expected some avg10 1.23, got %.2f", stats.SomeAvg10)
	}
	if stats.FullAvg10 != 0.45 {
		t.Fatalf("expected full avg10 0.45, got %.2f", stats.FullAvg10)
	}
}

func TestCollectPressureStatsReadsLinuxPressureFiles(t *testing.T) {
	originalReadFileFunc := readFileFunc
	defer func() {
		readFileFunc = originalReadFileFunc
	}()

	files := map[string][]byte{
		"/proc/pressure/cpu":    []byte("some avg10=2.00 avg60=1.00 avg300=0.50 total=200\n"),
		"/proc/pressure/memory": []byte("some avg10=3.00 avg60=1.00 avg300=0.50 total=300\nfull avg10=0.75 avg60=0.20 avg300=0.10 total=75\n"),
		"/proc/pressure/io":     []byte("some avg10=4.00 avg60=1.00 avg300=0.50 total=400\nfull avg10=1.25 avg60=0.20 avg300=0.10 total=125\n"),
	}
	readFileFunc = func(path string) ([]byte, error) {
		data, ok := files[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		return data, nil
	}

	stats := collectPressureStats()
	if stats.CPUSomeAvg10 != 2 || stats.MemorySomeAvg10 != 3 || stats.MemoryFullAvg10 != 0.75 || stats.IOSomeAvg10 != 4 || stats.IOFullAvg10 != 1.25 {
		t.Fatalf("unexpected pressure stats: %#v", stats)
	}
}

func TestCollectSwapStatsUsesGopsutilSwapCounters(t *testing.T) {
	originalSwapMemoryFunc := swapMemoryFunc
	defer func() {
		swapMemoryFunc = originalSwapMemoryFunc
	}()

	swapMemoryFunc = func() (*mem.SwapMemoryStat, error) {
		return &mem.SwapMemoryStat{Used: 2048, Total: 8192, Sin: 128, Sout: 256}, nil
	}

	stats := collectSwapStats()
	if stats.Used != 2048 || stats.Total != 8192 || stats.In != 128 || stats.Out != 256 {
		t.Fatalf("unexpected swap stats: %#v", stats)
	}
}

func TestCollectOOMKillsCountsKernelLogEvents(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "kern.log")
	if err := os.WriteFile(logPath, []byte("kernel: Out of memory: Killed process 111 (redis)\nkernel: oom-kill:constraint=CONSTRAINT_NONE\nkernel: normal line\n"), 0o600); err != nil {
		t.Fatalf("write kernel log: %v", err)
	}

	if got := countOOMKillEvents([]string{filepath.Join(tempDir, "missing"), logPath}); got != 2 {
		t.Fatalf("expected 2 oom events, got %d", got)
	}
}
