package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func waitForOnline(t *testing.T, h *hub.Hub, nodeID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h.IsOnline(nodeID) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("node %s did not become online in time", nodeID)
}

func TestCreateBatchSelfUpdateDispatchesToOnlineAgent(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	err := s.UpsertNode(&models.Node{ID: "node-1", Name: "agent-node", Token: "agent-token-1", CreatedAt: time.Now().Unix()})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	agentURL := *baseURL
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	agentQuery := agentURL.Query()
	agentQuery.Set("token", "agent-token-1")
	agentURL.RawQuery = agentQuery.Encode()

	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer agentConn.Close()
	waitForOnline(t, h, "node-1")

	requestBody := map[string]any{
		"node_ids":       []string{"node-1"},
		"target_version": "1.2.3",
		"download_url":   "https://updates.example/thism-agent",
		"sha256":         "abc123",
	}
	raw, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/agent-updates", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 when creating update job, got %d: %s", resp.Code, resp.Body.String())
	}

	_ = agentConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg struct {
		Type    string                     `json:"type"`
		Payload models.AgentCommandPayload `json:"payload"`
	}
	if err := agentConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read agent command: %v", err)
	}
	if msg.Type != "agent_command" {
		t.Fatalf("expected agent_command message, got %q", msg.Type)
	}
	if msg.Payload.Kind != models.AgentCommandKindSelfUpdate {
		t.Fatalf("expected self_update kind, got %q", msg.Payload.Kind)
	}
	if msg.Payload.TargetVersion != "1.2.3" {
		t.Fatalf("expected target version 1.2.3, got %q", msg.Payload.TargetVersion)
	}

	var body struct {
		Job     models.UpdateJob         `json:"job"`
		Targets []models.UpdateJobTarget `json:"targets"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Job.Status != models.UpdateJobStatusRunning {
		t.Fatalf("expected running job status, got %q", body.Job.Status)
	}
	if len(body.Targets) != 1 || body.Targets[0].Status != models.UpdateJobTargetStatusDispatched {
		t.Fatalf("expected dispatched target, got %#v", body.Targets)
	}
}

func TestCreateBatchSelfUpdateMarksOfflineNodesSkipped(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)

	if err := s.UpsertNode(&models.Node{ID: "node-offline", Name: "offline", Token: "token-offline", CreatedAt: time.Now().Unix()}); err != nil {
		t.Fatalf("seed node: %v", err)
	}

	requestBody := map[string]any{
		"node_ids":       []string{"node-offline"},
		"target_version": "1.2.3",
		"download_url":   "https://updates.example/thism-agent",
		"sha256":         "abc123",
	}
	raw, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/agent-updates", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 when creating update job, got %d: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Job     models.UpdateJob         `json:"job"`
		Targets []models.UpdateJobTarget `json:"targets"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Job.Status != models.UpdateJobStatusFailed {
		t.Fatalf("expected failed job status for offline-only batch, got %q", body.Job.Status)
	}
	if len(body.Targets) != 1 || body.Targets[0].Status != models.UpdateJobTargetStatusOfflineSkipped {
		t.Fatalf("expected offline_skipped target, got %#v", body.Targets)
	}
}

func TestAgentCommandStatusUpdatesJobTargetState(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	err := s.UpsertNode(&models.Node{ID: "node-1", Name: "agent-node", Token: "agent-token-1", CreatedAt: time.Now().Unix()})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	agentURL := *baseURL
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	agentQuery := agentURL.Query()
	agentQuery.Set("token", "agent-token-1")
	agentURL.RawQuery = agentQuery.Encode()

	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer agentConn.Close()
	waitForOnline(t, h, "node-1")

	job := &models.UpdateJob{ID: "job-status-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://updates.example/thism-agent", SHA256: "abc123", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if err := s.CreateUpdateJob(job); err != nil {
		t.Fatalf("CreateUpdateJob: %v", err)
	}
	if err := s.CreateUpdateJobTargets(job.ID, []string{"node-1"}); err != nil {
		t.Fatalf("CreateUpdateJobTargets: %v", err)
	}
	if err := s.UpdateUpdateJobTargetStatus(job.ID, "node-1", models.UpdateJobTargetStatusDispatched, "sent", ""); err != nil {
		t.Fatalf("UpdateUpdateJobTargetStatus: %v", err)
	}

	statusMessage := models.WSMessage{
		Type: "agent_command_status",
		Payload: models.AgentCommandStatusPayload{
			JobID:           job.ID,
			Status:          models.UpdateJobTargetStatusAccepted,
			Message:         "accepted",
			ReportedVersion: "1.2.2",
		},
	}
	if err := agentConn.WriteJSON(statusMessage); err != nil {
		t.Fatalf("write status message: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		targets, err := s.ListUpdateJobTargets(job.ID)
		if err != nil {
			t.Fatalf("ListUpdateJobTargets: %v", err)
		}
		if len(targets) == 1 && targets[0].Status == models.UpdateJobTargetStatusAccepted {
			if targets[0].ReportedVersion != "1.2.2" {
				t.Fatalf("expected reported version to persist, got %q", targets[0].ReportedVersion)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for agent status update")
}

func TestAgentMetricsFinalizeUpdateJobWhenTargetVersionReconnects(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	if err := s.UpsertNode(&models.Node{ID: "node-1", Name: "agent-node", Token: "agent-token-1", CreatedAt: time.Now().Unix()}); err != nil {
		t.Fatalf("seed node: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	agentURL := *baseURL
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	agentQuery := agentURL.Query()
	agentQuery.Set("token", "agent-token-1")
	agentURL.RawQuery = agentQuery.Encode()

	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer agentConn.Close()
	waitForOnline(t, h, "node-1")

	job := &models.UpdateJob{ID: "job-finish-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://updates.example/thism-agent", SHA256: "abc123", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if err := s.CreateUpdateJob(job); err != nil {
		t.Fatalf("CreateUpdateJob: %v", err)
	}
	if err := s.CreateUpdateJobTargets(job.ID, []string{"node-1"}); err != nil {
		t.Fatalf("CreateUpdateJobTargets: %v", err)
	}
	if err := s.UpdateUpdateJobTargetStatus(job.ID, "node-1", models.UpdateJobTargetStatusRestarting, "restarting", "1.2.2"); err != nil {
		t.Fatalf("UpdateUpdateJobTargetStatus: %v", err)
	}

	metrics := models.MetricsPayload{Type: "metrics", TS: time.Now().Unix(), CPU: 10, OS: "linux", Arch: "amd64", AgentVersion: "1.2.3", Mem: models.MemStats{Used: 1, Total: 2}}
	if err := agentConn.WriteJSON(metrics); err != nil {
		t.Fatalf("write metrics payload: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		targets, err := s.ListUpdateJobTargets(job.ID)
		if err != nil {
			t.Fatalf("ListUpdateJobTargets: %v", err)
		}
		if len(targets) == 1 && targets[0].Status == models.UpdateJobTargetStatusSucceeded {
			if targets[0].ReportedVersion != "1.2.3" {
				t.Fatalf("expected reported version 1.2.3, got %q", targets[0].ReportedVersion)
			}
			storedJob, err := s.GetUpdateJob(job.ID)
			if err != nil {
				t.Fatalf("GetUpdateJob: %v", err)
			}
			if storedJob.Status != models.UpdateJobStatusCompleted {
				t.Fatalf("expected completed job status, got %q", storedJob.Status)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for version-based job finalization")
}

func TestMetricsPayloadFinalizesUpdateJobWhenVersionMatches(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	if err := s.UpsertNode(&models.Node{ID: "node-1", Name: "agent-node", Token: "agent-token-1", CreatedAt: time.Now().Unix()}); err != nil {
		t.Fatalf("seed node: %v", err)
	}
	job := &models.UpdateJob{ID: "job-finalize-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://updates.example/thism-agent", SHA256: "abc123", CreatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusPending}
	if err := s.CreateUpdateJob(job); err != nil {
		t.Fatalf("CreateUpdateJob: %v", err)
	}
	if err := s.CreateUpdateJobTargets(job.ID, []string{"node-1"}); err != nil {
		t.Fatalf("CreateUpdateJobTargets: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	agentURL := *baseURL
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	agentQuery := agentURL.Query()
	agentQuery.Set("token", "agent-token-1")
	agentURL.RawQuery = agentQuery.Encode()

	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer agentConn.Close()
	waitForOnline(t, h, "node-1")
	if err := s.UpdateUpdateJobTargetStatus(job.ID, "node-1", models.UpdateJobTargetStatusRestarting, "restarting", ""); err != nil {
		t.Fatalf("UpdateUpdateJobTargetStatus: %v", err)
	}

	payload := models.MetricsPayload{Type: "metrics", TS: time.Now().Unix(), CPU: 10, AgentVersion: "1.2.3", Mem: models.MemStats{Used: 1, Total: 2}}
	if err := agentConn.WriteJSON(payload); err != nil {
		t.Fatalf("write metrics payload: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		targets, err := s.ListUpdateJobTargets(job.ID)
		if err != nil {
			t.Fatalf("ListUpdateJobTargets: %v", err)
		}
		if len(targets) == 1 && targets[0].Status == models.UpdateJobTargetStatusSucceeded {
			storedJob, err := s.GetUpdateJob(job.ID)
			if err != nil {
				t.Fatalf("GetUpdateJob: %v", err)
			}
			if storedJob.Status != models.UpdateJobStatusCompleted {
				t.Fatalf("expected completed job after matching version reconnect, got %q", storedJob.Status)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for update job finalization")
}

func TestAgentMetricsReconnectMarksRestartingUpdateSucceeded(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	if err := s.UpsertNode(&models.Node{ID: "node-1", Name: "agent-node", Token: "agent-token-1", CreatedAt: time.Now().Unix()}); err != nil {
		t.Fatalf("seed node: %v", err)
	}
	job := &models.UpdateJob{ID: "job-restart-1", Kind: models.AgentCommandKindSelfUpdate, TargetVersion: "1.2.3", DownloadURL: "https://updates.example/thism-agent", SHA256: "abc123", CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix(), CreatedBy: "admin", Status: models.UpdateJobStatusRunning}
	if err := s.CreateUpdateJob(job); err != nil {
		t.Fatalf("CreateUpdateJob: %v", err)
	}
	if err := s.CreateUpdateJobTargets(job.ID, []string{"node-1"}); err != nil {
		t.Fatalf("CreateUpdateJobTargets: %v", err)
	}
	if err := s.UpdateUpdateJobTargetStatus(job.ID, "node-1", models.UpdateJobTargetStatusRestarting, "restarting agent", "1.2.3"); err != nil {
		t.Fatalf("UpdateUpdateJobTargetStatus: %v", err)
	}

	baseURL, _ := url.Parse(server.URL)
	agentURL := *baseURL
	agentURL.Scheme = "ws"
	agentURL.Path = "/ws/agent"
	query := agentURL.Query()
	query.Set("token", "agent-token-1")
	agentURL.RawQuery = query.Encode()
	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL.String(), http.Header{})
	if err != nil {
		t.Fatalf("dial agent websocket: %v", err)
	}
	defer agentConn.Close()
	waitForOnline(t, h, "node-1")

	payload := models.MetricsPayload{Type: "metrics", TS: time.Now().Unix(), CPU: 5, IP: "10.0.0.5", OS: "linux", Arch: "amd64", AgentVersion: "1.2.3", Mem: models.MemStats{Used: 1, Total: 2}}
	if err := agentConn.WriteJSON(payload); err != nil {
		t.Fatalf("write metrics payload: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		targets, err := s.ListUpdateJobTargets(job.ID)
		if err != nil {
			t.Fatalf("ListUpdateJobTargets: %v", err)
		}
		if len(targets) == 1 && targets[0].Status == models.UpdateJobTargetStatusSucceeded {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for restarting target to become succeeded")
}
