package main

import (
	"flag"
	"log"

	"github.com/thism-dev/thism/internal/collector"
)

func main() {
	serverURL := flag.String("server", "", "Server WebSocket URL (e.g. ws://host:8080)")
	token := flag.String("token", "", "Node token")
	name := flag.String("name", "", "Node display name")
	flag.Parse()

	if *serverURL == "" || *token == "" || *name == "" {
		log.Fatal("--server, --token, and --name are required")
	}

	c := collector.New(*serverURL, *token, *name)
	c.Run()
}
