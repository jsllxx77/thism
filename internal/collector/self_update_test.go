package collector

import (
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/models"
)

type commandTestConn struct {
	remoteAddr net.Addr
	mu         sync.Mutex
	writes     [][]byte
}

func (c *commandTestConn) WriteMessage(_ int, data []byte) error {
	copied := make([]byte, len(data))
	copy(copied, data)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, copied)
	return nil
}

func (c *commandTestConn) snapshotWrites() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	snapshot := make([][]byte, len(c.writes))
	for index, raw := range c.writes {
		snapshot[index] = append([]byte(nil), raw...)
	}
	return snapshot
}

func (c *commandTestConn) ReadMessage() (int, []byte, error) {
	return websocket.TextMessage, nil, errors.New("not implemented")
}

func (c *commandTestConn) Close() error         { return nil }
func (c *commandTestConn) RemoteAddr() net.Addr { return c.remoteAddr }

func decodeStatuses(t *testing.T, writes [][]byte) []models.AgentCommandStatusPayload {
	t.Helper()
	statuses := make([]models.AgentCommandStatusPayload, 0, len(writes))
	for _, raw := range writes {
		var msg struct {
			Type    string                           `json:"type"`
			Payload models.AgentCommandStatusPayload `json:"payload"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal status message: %v", err)
		}
		if msg.Type == "agent_command_status" {
			statuses = append(statuses, msg.Payload)
		}
	}
	return statuses
}

func TestDispatchAgentCommandRunsSelfUpdateFlow(t *testing.T) {
	collector := NewWithInterval("ws://localhost:12026", "token", "node", "", DefaultReportInterval)
	collector.agentVersion = "1.0.0"
	collector.selfUpdateFunc = func(cmd models.AgentCommandPayload, report func(models.UpdateJobTargetStatus, string, string) error) error {
		if err := report(models.UpdateJobTargetStatusDownloading, "downloading", ""); err != nil {
			return err
		}
		if err := report(models.UpdateJobTargetStatusVerifying, "verifying", ""); err != nil {
			return err
		}
		if err := report(models.UpdateJobTargetStatusRestarting, "restarting", cmd.TargetVersion); err != nil {
			return err
		}
		return nil
	}
	conn := &commandTestConn{}
	var writeMu sync.Mutex

	collector.dispatchAgentCommand(models.AgentCommandPayload{
		JobID:         "job-1",
		Kind:          models.AgentCommandKindSelfUpdate,
		TargetVersion: "1.2.3",
		DownloadURL:   "https://updates.example/thism-agent",
		SHA256:        "abc123",
	}, conn, &writeMu)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		statuses := decodeStatuses(t, conn.snapshotWrites())
		if len(statuses) >= 4 {
			if statuses[0].Status != models.UpdateJobTargetStatusAccepted {
				t.Fatalf("expected first status accepted, got %q", statuses[0].Status)
			}
			if statuses[len(statuses)-1].Status != models.UpdateJobTargetStatusRestarting {
				t.Fatalf("expected last status restarting, got %q", statuses[len(statuses)-1].Status)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected staged status messages, got %d writes", len(conn.snapshotWrites()))
}

func TestDispatchAgentCommandRejectsConcurrentUpdates(t *testing.T) {
	collector := NewWithInterval("ws://localhost:12026", "token", "node", "", DefaultReportInterval)
	block := make(chan struct{})
	collector.selfUpdateFunc = func(cmd models.AgentCommandPayload, report func(models.UpdateJobTargetStatus, string, string) error) error {
		<-block
		return nil
	}
	conn := &commandTestConn{}
	var writeMu sync.Mutex

	collector.dispatchAgentCommand(models.AgentCommandPayload{JobID: "job-1", Kind: models.AgentCommandKindSelfUpdate}, conn, &writeMu)
	time.Sleep(20 * time.Millisecond)
	collector.dispatchAgentCommand(models.AgentCommandPayload{JobID: "job-2", Kind: models.AgentCommandKindSelfUpdate}, conn, &writeMu)
	close(block)

	statuses := decodeStatuses(t, conn.snapshotWrites())
	foundRejected := false
	for _, status := range statuses {
		if status.JobID == "job-2" && status.Status == models.UpdateJobTargetStatusFailed {
			foundRejected = true
		}
	}
	if !foundRejected {
		t.Fatalf("expected second concurrent update to be rejected, got %#v", statuses)
	}
}
