package collector

import (
	"net"
	"testing"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
)

func TestCollectCachesNetworkTopologyBetweenSamples(t *testing.T) {
	originalNetInterfacesFunc := netInterfacesFunc
	originalIOCountersFunc := ioCountersFunc
	originalInterfaceAddrsFunc := interfaceAddrsFunc
	defer func() {
		netInterfacesFunc = originalNetInterfacesFunc
		ioCountersFunc = originalIOCountersFunc
		interfaceAddrsFunc = originalInterfaceAddrsFunc
	}()

	interfaceCalls := 0
	netInterfacesFunc = func() ([]net.Interface, error) {
		interfaceCalls += 1
		return []net.Interface{{Name: "eth0", Flags: net.FlagUp}}, nil
	}

	addrCalls := 0
	interfaceAddrsFunc = func() ([]net.Addr, error) {
		addrCalls += 1
		return []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.10"), Mask: net.CIDRMask(24, 32)}}, nil
	}

	ioCounterCalls := 0
	ioCountersFunc = func(pernic bool) ([]psnet.IOCountersStat, error) {
		ioCounterCalls += 1
		return []psnet.IOCountersStat{{Name: "eth0", BytesRecv: 10, BytesSent: 20}}, nil
	}

	currentTime := time.Unix(100, 0)
	collector := NewWithInterval("ws://localhost:9999", "token", "test", "", DefaultReportInterval)
	collector.now = func() time.Time { return currentTime }

	for i := 0; i < 2; i += 1 {
		payload, err := collector.Collect()
		if err != nil {
			t.Fatalf("Collect %d: %v", i+1, err)
		}
		if payload.IP != "192.0.2.10" {
			t.Fatalf("expected cached IP to be used, got %q", payload.IP)
		}
		currentTime = currentTime.Add(5 * time.Second)
	}

	if interfaceCalls != 1 {
		t.Fatalf("expected interface list to be refreshed once within cache window, got %d", interfaceCalls)
	}
	if addrCalls != 1 {
		t.Fatalf("expected local IP detection to be refreshed once within cache window, got %d", addrCalls)
	}
	if ioCounterCalls != 2 {
		t.Fatalf("expected io counters to remain uncached per collect, got %d", ioCounterCalls)
	}
}

func TestCollectRefreshesNetworkTopologyAfterCacheExpires(t *testing.T) {
	originalNetInterfacesFunc := netInterfacesFunc
	originalIOCountersFunc := ioCountersFunc
	originalInterfaceAddrsFunc := interfaceAddrsFunc
	defer func() {
		netInterfacesFunc = originalNetInterfacesFunc
		ioCountersFunc = originalIOCountersFunc
		interfaceAddrsFunc = originalInterfaceAddrsFunc
	}()

	interfaceCalls := 0
	netInterfacesFunc = func() ([]net.Interface, error) {
		interfaceCalls += 1
		return []net.Interface{{Name: "eth0", Flags: net.FlagUp}}, nil
	}
	addrCalls := 0
	interfaceAddrsFunc = func() ([]net.Addr, error) {
		addrCalls += 1
		return []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.20"), Mask: net.CIDRMask(24, 32)}}, nil
	}
	ioCountersFunc = func(pernic bool) ([]psnet.IOCountersStat, error) {
		return []psnet.IOCountersStat{{Name: "eth0", BytesRecv: 10, BytesSent: 20}}, nil
	}

	currentTime := time.Unix(200, 0)
	collector := NewWithInterval("ws://localhost:9999", "token", "test", "", DefaultReportInterval)
	collector.now = func() time.Time { return currentTime }

	if _, err := collector.Collect(); err != nil {
		t.Fatalf("Collect first: %v", err)
	}
	currentTime = currentTime.Add(networkTopologyCacheTTL + time.Second)
	if _, err := collector.Collect(); err != nil {
		t.Fatalf("Collect second: %v", err)
	}

	if interfaceCalls != 2 {
		t.Fatalf("expected interface cache to refresh after expiry, got %d calls", interfaceCalls)
	}
	if addrCalls != 2 {
		t.Fatalf("expected local IP cache to refresh after expiry, got %d calls", addrCalls)
	}
}
