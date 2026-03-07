package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/thism-dev/thism/frontend"
	"github.com/thism-dev/thism/internal/api"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/store"
)

func main() {
	port := flag.String("port", "12026", "HTTP port")
	dbPath := flag.String("db", "./thism.db", "SQLite database path")
	adminToken := flag.String("token", "", "Admin token for API auth")
	adminUser := flag.String("admin-user", "", "Admin username for login page authentication")
	adminPass := flag.String("admin-pass", "", "Admin password for login page authentication")
	flag.Parse()

	if *adminToken == "" {
		log.Fatal("--token is required")
	}
	if (*adminUser == "") != (*adminPass == "") {
		log.Fatal("--admin-user and --admin-pass must be provided together")
	}

	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	h := hub.New(s)
	go h.Run()

	router := api.NewRouterWithAuth(s, h, api.AuthConfig{
		AdminToken: *adminToken,
		Username:   *adminUser,
		Password:   *adminPass,
	}, frontend.Handler())
	log.Printf("ThisM server listening on :%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, router))
}
