package collector

import (
	"net"
	"testing"

	psnet "github.com/shirou/gopsutil/v3/net"
)

func TestCollectExcludesLoopbackTrafficFromNetworkTotals(t *testing.T) {
	originalIOCountersFunc := ioCountersFunc
	originalNetInterfacesFunc := netInterfacesFunc
	defer func() {
		ioCountersFunc = originalIOCountersFunc
		netInterfacesFunc = originalNetInterfacesFunc
	}()

	ioCountersFunc = func(pernic bool) ([]psnet.IOCountersStat, error) {
		if !pernic {
			t.Fatalf("expected per-interface network counters")
		}
		return []psnet.IOCountersStat{
			{Name: "eth0", BytesRecv: 1200, BytesSent: 3400},
			{Name: "lo", BytesRecv: 9000, BytesSent: 11000},
		}, nil
	}

	netInterfacesFunc = func() ([]net.Interface, error) {
		return []net.Interface{
			{Name: "eth0", Flags: net.FlagUp},
			{Name: "lo", Flags: net.FlagLoopback | net.FlagUp},
		}, nil
	}

	collector := New("ws://localhost:9999", "token", "test", "")
	payload, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if payload.Net.RxBytes != 1200 {
		t.Fatalf("expected non-loopback rx bytes only, got %d", payload.Net.RxBytes)
	}
	if payload.Net.TxBytes != 3400 {
		t.Fatalf("expected non-loopback tx bytes only, got %d", payload.Net.TxBytes)
	}
}
