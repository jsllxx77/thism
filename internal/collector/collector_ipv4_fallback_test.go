package collector

import (
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/gorilla/websocket"
)

type stubWebsocketConn struct {
	remoteAddr net.Addr
}

func (c *stubWebsocketConn) WriteMessage(_ int, _ []byte) error {
	return nil
}

func (c *stubWebsocketConn) ReadMessage() (int, []byte, error) {
	return websocket.TextMessage, nil, errors.New("not implemented")
}

func (c *stubWebsocketConn) Close() error {
	return nil
}

func (c *stubWebsocketConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func TestCollectorEnablesIPv4FallbackAfterIPv6Reset(t *testing.T) {
	c := NewWithInterval("wss://example.com", "token", "node", "", DefaultReportInterval)

	if modes := c.dialModes(); len(modes) != 1 || modes[0] != dialModeAuto {
		t.Fatalf("expected initial dial mode [auto], got %#v", modes)
	}

	c.noteConnectionError(
		dialModeAuto,
		&net.TCPAddr{IP: net.ParseIP("2606:4700:3031::6815:5969"), Port: 443},
		syscall.ECONNRESET,
	)

	if !c.preferIPv4Fallback {
		t.Fatal("expected IPv4 fallback to be enabled after IPv6 reset")
	}

	modes := c.dialModes()
	if len(modes) != 2 || modes[0] != dialModeIPv4 || modes[1] != dialModeAuto {
		t.Fatalf("expected fallback dial order [ipv4 auto], got %#v", modes)
	}
}

func TestCollectorDoesNotEnableIPv4FallbackForNonIPv6OrHandshakeErrors(t *testing.T) {
	tests := []struct {
		name string
		addr net.Addr
		err  error
	}{
		{
			name: "ipv4 reset",
			addr: &net.TCPAddr{IP: net.ParseIP("104.21.89.105"), Port: 443},
			err:  syscall.ECONNRESET,
		},
		{
			name: "ipv6 handshake",
			addr: &net.TCPAddr{IP: net.ParseIP("2606:4700:3031::6815:5969"), Port: 443},
			err:  errors.New("websocket: bad handshake"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewWithInterval("wss://example.com", "token", "node", "", DefaultReportInterval)

			c.noteConnectionError(dialModeAuto, tc.addr, tc.err)

			if c.preferIPv4Fallback {
				t.Fatalf("expected IPv4 fallback to remain disabled for %s", tc.name)
			}
		})
	}
}

func TestCollectorDialUsesIPv4FallbackBeforeAuto(t *testing.T) {
	c := NewWithInterval("wss://example.com", "token", "node", "", DefaultReportInterval)
	c.preferIPv4Fallback = true

	var gotModes []dialMode
	c.dialWebsocket = func(mode dialMode, targetURL string) (websocketConn, error) {
		gotModes = append(gotModes, mode)
		if mode == dialModeIPv4 {
			return nil, errors.New("ipv4 unavailable")
		}
		return &stubWebsocketConn{remoteAddr: &net.TCPAddr{IP: net.ParseIP("104.21.89.105"), Port: 443}}, nil
	}

	conn, mode, err := c.dialAgent("wss://example.com/ws/agent?token=token")
	if err != nil {
		t.Fatalf("dialAgent: %v", err)
	}
	if conn == nil {
		t.Fatal("expected a websocket connection")
	}
	if mode != dialModeAuto {
		t.Fatalf("expected auto mode to succeed after ipv4 fallback failure, got %v", mode)
	}
	if len(gotModes) != 2 || gotModes[0] != dialModeIPv4 || gotModes[1] != dialModeAuto {
		t.Fatalf("expected dial order [ipv4 auto], got %#v", gotModes)
	}
}
