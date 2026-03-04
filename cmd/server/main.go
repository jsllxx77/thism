package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/store"
)

func main() {
	port := flag.String("port", "8080", "HTTP port")
	dbPath := flag.String("db", "./thism.db", "SQLite database path")
	adminToken := flag.String("token", "", "Admin token for API auth")
	flag.Parse()

	if *adminToken == "" {
		log.Fatal("--token is required")
	}

	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	h := hub.New(s)
	go h.Run()

	router := api.NewRouter(s, h, *adminToken, nil)
	log.Printf("ThisM server listening on :%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, router))
}
