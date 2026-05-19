package main

import (
	"flag"
	"log"
	"os"

	"github.com/thism-dev/thism/internal/collector"
	sharedversion "github.com/thism-dev/thism/internal/version"
)

// envOr returns the named environment variable when set, otherwise the
// provided fallback. Used so flag defaults can read sensitive values from
// the environment, keeping them off /proc/<pid>/cmdline.
func envOr(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return fallback
}

func main() {
	serverURL := flag.String("server", os.Getenv("THISM_AGENT_SERVER"), "Server WebSocket URL, e.g. wss://host (env: THISM_AGENT_SERVER)")
	token := flag.String("token", os.Getenv("THISM_AGENT_TOKEN"), "Node token (env: THISM_AGENT_TOKEN)")
	name := flag.String("name", os.Getenv("THISM_AGENT_NAME"), "Node display name (env: THISM_AGENT_NAME)")
	nodeIP := flag.String("ip", envOr("THISM_AGENT_IP", ""), "Optional override for reported node IP, e.g. public IP")
	reportInterval := flag.Duration("interval", collector.DefaultReportInterval, "Metrics reporting interval (e.g. 5s, 30s, 1m)")
	flag.Parse()

	if *serverURL == "" || *token == "" || *name == "" {
		log.Fatal("server, token, and name are required: pass via --server/--token/--name flags or THISM_AGENT_SERVER/THISM_AGENT_TOKEN/THISM_AGENT_NAME env vars")
	}
	if *reportInterval <= 0 {
		log.Fatal("--interval must be greater than 0")
	}

	c := collector.NewWithInterval(*serverURL, *token, *name, *nodeIP, *reportInterval)
	c.SetAgentVersion(runtimeAgentVersion())
	c.Run()
}

func runtimeAgentVersion() string {
	return sharedversion.Version
}
