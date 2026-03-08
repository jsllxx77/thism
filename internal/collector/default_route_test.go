package collector

import (
	"fmt"
	"net"
	"testing"

	psnet "github.com/shirou/gopsutil/v3/net"
)

func TestParseIPv4DefaultRouteInterfaceNames(t *testing.T) {
	raw := []byte("Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\neth0\t00000000\t010011AC\t0003\t0\t0\t0\t00000000\t0\t0\t0\ndocker0\t000011AC\t00000000\t0001\t0\t0\t0\t0000FFFF\t0\t0\t0\n")

	names := parseIPv4DefaultRouteInterfaceNames(raw)
	if len(names) != 1 {
		t.Fatalf("expected exactly one IPv4 default route interface, got %d", len(names))
	}
	if _, ok := names["eth0"]; !ok {
		t.Fatalf("expected eth0 to be selected as IPv4 default route interface")
	}
}

func TestParseIPv6DefaultRouteInterfaceNames(t *testing.T) {
	raw := []byte("00000000000000000000000000000000 00 00000000000000000000000000000000 00 20010db8000000000000000000000001 00000064 00000000 00000000 00000003 eth1\n20010db8000000000000000000000000 40 00000000000000000000000000000000 00 00000000000000000000000000000000 00000100 00000000 00000000 00000001 eth0\n")

	names := parseIPv6DefaultRouteInterfaceNames(raw)
	if len(names) != 1 {
		t.Fatalf("expected exactly one IPv6 default route interface, got %d", len(names))
	}
	if _, ok := names["eth1"]; !ok {
		t.Fatalf("expected eth1 to be selected as IPv6 default route interface")
	}
}

func TestCollectUsesDefaultIPv4AndIPv6EgressInterfacesOnly(t *testing.T) {
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
			{Name: "eth1", BytesRecv: 2200, BytesSent: 4400},
			{Name: "docker0", BytesRecv: 9000, BytesSent: 11000},
			{Name: "veth123", BytesRecv: 7000, BytesSent: 8000},
			{Name: "lo", BytesRecv: 5000, BytesSent: 6000},
		}, nil
	}

	netInterfacesFunc = func() ([]net.Interface, error) {
		return []net.Interface{
			{Name: "eth0", Flags: net.FlagUp},
			{Name: "eth1", Flags: net.FlagUp},
			{Name: "docker0", Flags: net.FlagUp},
			{Name: "veth123", Flags: net.FlagUp},
			{Name: "lo", Flags: net.FlagLoopback | net.FlagUp},
		}, nil
	}

	readFileFunc = func(path string) ([]byte, error) {
		switch path {
		case ipv4DefaultRoutePath:
			return []byte("Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\neth0\t00000000\t010011AC\t0003\t0\t0\t0\t00000000\t0\t0\t0\n"), nil
		case ipv6DefaultRoutePath:
			return []byte("00000000000000000000000000000000 00 00000000000000000000000000000000 00 20010db8000000000000000000000001 00000064 00000000 00000000 00000003 eth1\n"), nil
		default:
			return nil, fmt.Errorf("unexpected path %s", path)
		}
	}

	collector := New("ws://localhost:9999", "token", "test", "")
	payload, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if payload.Net.RxBytes != 3400 {
		t.Fatalf("expected only default IPv4+IPv6 ingress totals, got %d", payload.Net.RxBytes)
	}
	if payload.Net.TxBytes != 7800 {
		t.Fatalf("expected only default IPv4+IPv6 egress totals, got %d", payload.Net.TxBytes)
	}
}

func TestCollectDoesNotDoubleCountSharedDefaultInterface(t *testing.T) {
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
			{Name: "docker0", BytesRecv: 9000, BytesSent: 11000},
		}, nil
	}

	netInterfacesFunc = func() ([]net.Interface, error) {
		return []net.Interface{
			{Name: "eth0", Flags: net.FlagUp},
			{Name: "docker0", Flags: net.FlagUp},
		}, nil
	}

	readFileFunc = func(path string) ([]byte, error) {
		switch path {
		case ipv4DefaultRoutePath:
			return []byte("Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\neth0\t00000000\t010011AC\t0003\t0\t0\t0\t00000000\t0\t0\t0\n"), nil
		case ipv6DefaultRoutePath:
			return []byte("00000000000000000000000000000000 00 00000000000000000000000000000000 00 20010db8000000000000000000000001 00000064 00000000 00000000 00000003 eth0\n"), nil
		default:
			return nil, fmt.Errorf("unexpected path %s", path)
		}
	}

	collector := New("ws://localhost:9999", "token", "test", "")
	payload, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if payload.Net.RxBytes != 1200 {
		t.Fatalf("expected shared default interface to be counted once, got %d", payload.Net.RxBytes)
	}
	if payload.Net.TxBytes != 3400 {
		t.Fatalf("expected shared default interface to be counted once, got %d", payload.Net.TxBytes)
	}
}
