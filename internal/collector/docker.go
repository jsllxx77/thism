package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/thism-dev/thism/internal/models"
)

const (
	dockerSocketPath  = "/var/run/docker.sock"
	dockerListTimeout = 5 * time.Second
)

// dockerAPIContainer mirrors the relevant fields from Docker's /containers/json response.
type dockerAPIContainer struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
}

// dockerSocketClient is the HTTP client used to talk to Docker via unix socket.
// Overridable for tests.
var dockerSocketClient = newDockerUnixClient

func newDockerUnixClient() *http.Client {
	return &http.Client{
		Timeout: dockerListTimeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", dockerSocketPath, dockerListTimeout)
			},
		},
	}
}

// collectDockerContainers queries the local Docker daemon via /var/run/docker.sock
// and returns the container list. If Docker is not available or permission is denied,
// it returns (nil, false, nil) — no error, just docker_available=false.
func collectDockerContainers() ([]models.DockerContainer, bool, error) {
	client := dockerSocketClient()

	// Use the unversioned endpoint so newer daemons with higher minimum API
	// versions do not reject the request as "client version too old".
	url := fmt.Sprintf("http://localhost/containers/json?all=true")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, false, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		// Docker not installed, socket missing, or permission denied — not an error.
		return nil, false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, nil
	}

	var apiContainers []dockerAPIContainer
	if err := json.NewDecoder(resp.Body).Decode(&apiContainers); err != nil {
		return nil, false, nil
	}

	containers := make([]models.DockerContainer, 0, len(apiContainers))
	for _, c := range apiContainers {
		name := ""
		if len(c.Names) > 0 {
			// Docker prefixes container names with "/".
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

	return containers, true, nil
}

// shortenID returns the first 12 characters of a Docker container ID,
// matching the short-ID convention used by `docker ps`.
func shortenID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
