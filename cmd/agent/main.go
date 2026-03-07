package main

import (
	"flag"
	"log"

	"github.com/thism-dev/thism/internal/collector"
)

var agentVersion = "dev"

func main() {
	serverURL := flag.String("server", "", "Server WebSocket URL (e.g. ws://host:8080)")
	token := flag.String("token", "", "Node token")
	name := flag.String("name", "", "Node display name")
	nodeIP := flag.String("ip", "", "Optional override for reported node IP (e.g. public IP)")
	reportInterval := flag.Duration("interval", collector.DefaultReportInterval, "Metrics reporting interval (e.g. 5s, 30s, 1m)")
	flag.Parse()

	if *serverURL == "" || *token == "" || *name == "" {
		log.Fatal("--server, --token, and --name are required")
	}
	if *reportInterval <= 0 {
		log.Fatal("--interval must be greater than 0")
	}

	c := collector.NewWithInterval(*serverURL, *token, *name, *nodeIP, *reportInterval)
	c.SetAgentVersion(agentVersion)
	c.Run()
}
