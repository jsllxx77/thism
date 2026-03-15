package collector

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/thism-dev/thism/internal/models"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestShortenID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc123def456789012345678", "abc123def456"},
		{"short", "short"},
		{"exactly12ch", "exactly12ch"},
		{"", ""},
	}
	for _, tt := range tests {
		got := shortenID(tt.input)
		if got != tt.expected {
			t.Errorf("shortenID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseDockerContainers(t *testing.T) {
	apiResponse := []dockerAPIContainer{
		{
			ID:     "abc123def456789012345678",
			Names:  []string{"/my-app"},
			Image:  "nginx:latest",
			State:  "running",
			Status: "Up 2 hours",
		},
		{
			ID:     "def456abc789012345678901",
			Names:  []string{"/redis-cache"},
			Image:  "redis:7",
			State:  "exited",
			Status: "Exited (0) 5 minutes ago",
		},
	}
	body, _ := json.Marshal(apiResponse)

	containers := parseDockerAPIResponse(body)
	if containers == nil {
		t.Fatal("expected non-nil containers")
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}

	// Verify first container
	c1 := containers[0]
	if c1.ID != "abc123def456" {
		t.Errorf("expected short ID 'abc123def456', got '%s'", c1.ID)
	}
	if c1.Name != "my-app" {
		t.Errorf("expected name 'my-app', got '%s'", c1.Name)
	}
	if c1.Image != "nginx:latest" {
		t.Errorf("expected image 'nginx:latest', got '%s'", c1.Image)
	}
	if c1.State != "running" {
		t.Errorf("expected state 'running', got '%s'", c1.State)
	}
	if c1.Status != "Up 2 hours" {
		t.Errorf("expected status 'Up 2 hours', got '%s'", c1.Status)
	}

	// Verify second container
	c2 := containers[1]
	if c2.Name != "redis-cache" {
		t.Errorf("expected name 'redis-cache', got '%s'", c2.Name)
	}
	if c2.State != "exited" {
		t.Errorf("expected state 'exited', got '%s'", c2.State)
	}
}

func TestParseDockerContainers_EmptyList(t *testing.T) {
	containers := parseDockerAPIResponse([]byte("[]"))
	if containers == nil {
		t.Fatal("expected non-nil (empty) slice, got nil")
	}
	if len(containers) != 0 {
		t.Fatalf("expected 0 containers, got %d", len(containers))
	}
}

func TestParseDockerContainers_InvalidJSON(t *testing.T) {
	containers := parseDockerAPIResponse([]byte("not json"))
	if containers != nil {
		t.Fatal("expected nil for invalid JSON")
	}
}

func TestParseDockerContainers_NoNames(t *testing.T) {
	apiResponse := []dockerAPIContainer{
		{
			ID:    "abc123def456789012345678",
			Names: []string{},
			Image: "busybox",
			State: "running",
		},
	}
	body, _ := json.Marshal(apiResponse)
	containers := parseDockerAPIResponse(body)
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}
	if containers[0].Name != "" {
		t.Errorf("expected empty name, got '%s'", containers[0].Name)
	}
}

func TestCollectDockerContainersUsesUnversionedEndpoint(t *testing.T) {
	originalClientFactory := dockerSocketClient
	t.Cleanup(func() {
		dockerSocketClient = originalClientFactory
	})

	dockerSocketClient = func() *http.Client {
		return &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/containers/json" {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body: io.NopCloser(strings.NewReader(
							`{"message":"client version 1.24 is too old. Minimum supported API version is 1.44"}`,
						)),
						Header: make(http.Header),
					}, nil
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"abc123def456789012345678","Names":["/my-app"],"Image":"nginx:latest","State":"running","Status":"Up 2 hours"}]`,
					)),
					Header: make(http.Header),
				}, nil
			}),
		}
	}

	containers, dockerAvailable, err := collectDockerContainers()
	if err != nil {
		t.Fatalf("collectDockerContainers returned unexpected error: %v", err)
	}
	if !dockerAvailable {
		t.Fatal("expected docker to be available when daemon accepts the unversioned endpoint")
	}
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}
	if containers[0].Name != "my-app" {
		t.Fatalf("expected container name my-app, got %q", containers[0].Name)
	}
}

// parseDockerAPIResponse is a helper to parse Docker API response
// body into our model. Extracted from collectDockerContainers for testability.
func parseDockerAPIResponse(body []byte) []models.DockerContainer {
	var apiContainers []dockerAPIContainer
	if err := json.Unmarshal(body, &apiContainers); err != nil {
		return nil
	}

	containers := make([]models.DockerContainer, 0, len(apiContainers))
	for _, c := range apiContainers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		containers = append(containers, models.DockerContainer{
			ID:     shortenID(c.ID),
			Name:   name,
			Image:  c.Image,
			State:  c.State,
			Status: c.Status,
		})
	}

	return containers
}
