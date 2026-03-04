package hub_test

import (
	"testing"
	"time"

	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
)

func TestHubRegisterUnregister(t *testing.T) {
	h := hub.New(nil)
	go h.Run()

	// Give hub goroutine time to start
	time.Sleep(10 * time.Millisecond)

	h.Register("node-1", nil)
	time.Sleep(10 * time.Millisecond)

	if !h.IsOnline("node-1") {
		t.Error("expected node-1 to be online after register")
	}

	h.Unregister("node-1")
	time.Sleep(10 * time.Millisecond)

	if h.IsOnline("node-1") {
		t.Error("expected node-1 to be offline after unregister")
	}
}

func TestHubBroadcast(t *testing.T) {
	h := hub.New(nil)
	go h.Run()
	time.Sleep(10 * time.Millisecond)

	ch := make(chan models.WSMessage, 1)
	h.Subscribe(ch)

	h.Broadcast(models.WSMessage{Type: "test"})
	time.Sleep(10 * time.Millisecond)

	select {
	case msg := <-ch:
		if msg.Type != "test" {
			t.Errorf("expected type 'test', got '%s'", msg.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for broadcast message")
	}
}
