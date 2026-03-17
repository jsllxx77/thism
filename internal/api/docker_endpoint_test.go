package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

func TestDockerEndpointReturnsContainersWhenAvailable(t *testing.T) {
	s, _ := store.New(":memory:")
	defer s.Close()
	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, "admin-token", nil)
	server := newIPv4TestServer(router)
	defer server.Close()

	if err := s.UpsertNode(&models.Node{
		ID:        "node-1",
		Name:      "agent-node",
		Token:     "agent-token-1",
		CreatedAt: time.Now().Unix(),
	}); err != nil {
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

	dockerAvailable := true
	payload := models.MetricsPayload{
		Type:            "metrics",
		TS:              time.Now().Unix(),
		CPU:             10,
		Mem:             models.MemStats{Used: 1, Total: 2},
		DockerAvailable: &dockerAvailable,
		Containers: []models.DockerContainer{
			{
				ID:     "0123456789ab",
				Name:   "web",
				Image:  "nginx:alpine",
				State:  "running",
				Status: "Up 2 hours",
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := agentConn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write agent metrics: %v", err)
	}

	dockerURL := *baseURL
	dockerURL.Path = "/api/nodes/node-1/docker"
	dockerQuery := dockerURL.Query()
	dockerQuery.Set("token", "admin-token")
	dockerURL.RawQuery = dockerQuery.Encode()

	req, err := http.NewRequest(http.MethodGet, dockerURL.String(), nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer admin-token")

	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("request docker endpoint: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read docker endpoint response: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.StatusCode, string(body))
	}

	var out struct {
		DockerAvailable bool `json:"docker_available"`
		Containers      []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Image  string `json:"image"`
			State  string `json:"state"`
			Status string `json:"status"`
		} `json:"containers"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal docker endpoint: %v", err)
	}

	if !out.DockerAvailable {
		t.Fatalf("expected docker_available=true, got false: %s", string(body))
	}
	if len(out.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d: %s", len(out.Containers), string(body))
	}
	if out.Containers[0].Name != "web" {
		t.Fatalf("expected container name web, got %q", out.Containers[0].Name)
	}
	if out.Containers[0].Image != "nginx:alpine" {
		t.Fatalf("expected container image nginx:alpine, got %q", out.Containers[0].Image)
	}
}
