package collector

import (
	"net"
	"testing"

	psnet "github.com/shirou/gopsutil/v3/net"
)

func TestCollectExcludesLoopbackTrafficFromNetworkTotals(t *testing.T) {
	originalIOCountersFunc := ioCountersFunc
	originalNetInterfacesFunc := netInterfacesFunc
	originalReadFileFunc := readFileFunc
	defer func() {
		ioCountersFunc = originalIOCountersFunc
		netInterfacesFunc = originalNetInterfacesFunc
		readFileFunc = originalReadFileFunc
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

	readFileFunc = func(path string) ([]byte, error) {
		if path == ipv4DefaultRoutePath {
			return []byte("Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\neth0\t00000000\t010011AC\t0003\t0\t0\t0\t00000000\t0\t0\t0\n"), nil
		}
		return nil, nil
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
