package hub_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
)

// blockingSocket blocks writes until released, to exercise the linearization
// of SendToAgent against connection replacement.
type blockingSocket struct {
	mu      sync.Mutex
	blocked bool
	release chan struct{}
	closed  bool
	writes  []models.WSMessage
}

func (s *blockingSocket) WriteJSON(v any) error {
	s.mu.Lock()
	if s.blocked {
		s.mu.Unlock()
		<-s.release
		s.mu.Lock()
	}
	if s.closed {
		s.mu.Unlock()
		return errors.New("socket closed")
	}
	s.writes = append(s.writes, v.(models.WSMessage))
	s.mu.Unlock()
	return nil
}

func (s *blockingSocket) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return nil
}

// TestHubSendLinearizedAgainstReplacement verifies that a send admitted while
// a connection is current either completes on that writer or fails explicitly
// when the connection is replaced; it never lands on the stale socket after
// replacement.
func TestHubSendLinearizedAgainstReplacement(t *testing.T) {
	h := hub.New(nil)
	go h.Run()
	defer h.Close()
	time.Sleep(10 * time.Millisecond)

	stale := &blockingSocket{release: make(chan struct{})}
	h.Register("node-1", stale)
	time.Sleep(10 * time.Millisecond)

	sendDone := make(chan error, 1)
	go func() {
		sendDone <- h.SendToAgent("node-1", models.WSMessage{Type: "agent_command"})
	}()

	// Let the send start on the stale socket, then replace the connection.
	time.Sleep(30 * time.Millisecond)
	current := &blockingSocket{release: make(chan struct{})}
	h.Register("node-1", current)
	time.Sleep(10 * time.Millisecond)

	// Release the in-flight write; it must still have been admitted to the
	// then-current writer (linearization point before replacement).
	close(stale.release)
	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatalf("send admitted before replacement must succeed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("send did not complete after release")
	}

	// After replacement, sends must go to the new writer only.
	if err := h.SendToAgent("node-1", models.WSMessage{Type: "agent_command"}); err != nil {
		t.Fatalf("send to current: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if len(stale.writes) != 1 {
		t.Fatalf("stale socket received %d writes, want exactly the pre-replacement one", len(stale.writes))
	}
	if len(current.writes) != 1 {
		t.Fatalf("current socket received %d writes, want 1", len(current.writes))
	}
}

// TestHubSendToFencedGenerationFails verifies that once a connection is fenced
// by replacement, commands fail explicitly instead of reaching the stale
// writer.
func TestHubSendToFencedGenerationFails(t *testing.T) {
	h := hub.New(nil)
	go h.Run()
	defer h.Close()
	time.Sleep(10 * time.Millisecond)

	stale := &stubAgentSocket{}
	current := &stubAgentSocket{}
	h.Register("node-1", stale)
	time.Sleep(10 * time.Millisecond)
	h.Register("node-1", current)
	time.Sleep(10 * time.Millisecond)

	if err := h.SendToAgent("node-1", models.WSMessage{Type: "agent_command"}); err != nil {
		t.Fatalf("send to current generation: %v", err)
	}
	if len(current.messages) != 1 {
		t.Fatalf("expected current socket to receive the command, got %d", len(current.messages))
	}
	if len(stale.messages) != 0 {
		t.Fatalf("expected stale socket to receive nothing, got %#v", stale.messages)
	}

	// The stale generation must never be accepted as current again.
	if h.IsCurrent("node-1", 1) {
		t.Fatal("expected generation 1 to be stale")
	}
	if !h.IsCurrent("node-1", 2) {
		t.Fatal("expected generation 2 to be current")
	}
}

// TestHubHandshakeGating verifies that control commands are gated on the
// handshake completing on the current connection.
func TestHubHandshakeGating(t *testing.T) {
	h := hub.New(nil)
	go h.Run()
	defer h.Close()
	time.Sleep(10 * time.Millisecond)

	conn := &stubAgentSocket{}
	gen := h.Register("node-1", conn)
	time.Sleep(10 * time.Millisecond)

	if h.HandshakeReady("node-1") {
		t.Fatal("expected handshake to be pending after register")
	}
	if !h.CompleteHandshake("node-1", gen) {
		t.Fatal("expected handshake completion to succeed")
	}
	if !h.HandshakeReady("node-1") {
		t.Fatal("expected handshake to be ready after completion")
	}

	// A stale generation cannot complete a handshake.
	replacement := &stubAgentSocket{}
	h.Register("node-1", replacement)
	time.Sleep(10 * time.Millisecond)
	if h.CompleteHandshake("node-1", gen) {
		t.Fatal("expected stale generation handshake completion to fail")
	}
	if h.HandshakeReady("node-1") {
		t.Fatal("expected replacement connection to start without handshake")
	}
	h.UnregisterGeneration("node-1", h.CurrentGeneration("node-1"))
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && h.IsOnline("node-1") {
		time.Sleep(5 * time.Millisecond)
	}
	if h.IsOnline("node-1") {
		t.Fatal("expected current generation to unregister by generation identity")
	}
}

// TestHubCloseLifecycle verifies that every entry point returns quickly after
// Close and the event loop exits.
func TestHubCloseLifecycle(t *testing.T) {
	h := hub.New(nil)
	go h.Run()
	time.Sleep(10 * time.Millisecond)

	conn := &stubAgentSocket{}
	gen := h.Register("node-1", conn)
	time.Sleep(10 * time.Millisecond)
	if gen == 0 {
		t.Fatal("expected valid generation before close")
	}

	h.Close()
	h.Close() // idempotent

	if err := h.SendToAgent("node-1", models.WSMessage{Type: "agent_command"}); !errors.Is(err, hub.ErrHubClosed) {
		t.Fatalf("expected ErrHubClosed after close, got %v", err)
	}
	if h.IsOnline("node-1") {
		t.Fatal("expected node to be offline after close")
	}
	if gen := h.Register("node-2", &stubAgentSocket{}); gen != 0 {
		t.Fatalf("expected register after close to return 0, got %d", gen)
	}
	if h.IsCurrent("node-1", gen) {
		t.Fatal("expected IsCurrent to be false after close")
	}
	// Broadcast after close must not panic or block.
	done := make(chan struct{})
	go func() {
		h.Broadcast(models.WSMessage{Type: "test"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Broadcast blocked after close")
	}
}
