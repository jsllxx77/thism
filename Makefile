.PHONY: \
	build build-server build-agent build-agent-all build-frontend test clean \
	dev-ui dev-server dev-rebuild dev-restart

GO ?= /usr/local/go/bin/go
GOCACHE_DIR ?= /tmp/go-build
PORT ?= 12026
TOKEN ?= thism2026
ADMIN_USER ?=
ADMIN_PASS ?=
ADMIN_AUTH_ARGS := $(strip $(if $(ADMIN_USER),--admin-user $(ADMIN_USER),) $(if $(ADMIN_PASS),--admin-pass $(ADMIN_PASS),))

build: build-frontend build-server build-agent build-agent-all

build-frontend:
	cd frontend && npm ci && npm run build

build-server: build-frontend
	GOCACHE=$(GOCACHE_DIR) $(GO) build -o bin/thism-server ./cmd/server

build-agent:
	GOCACHE=$(GOCACHE_DIR) $(GO) build -o bin/thism-agent ./cmd/agent

build-agent-all:
	GOOS=linux GOARCH=amd64 GOCACHE=$(GOCACHE_DIR) $(GO) build -o dist/thism-agent-linux-amd64 ./cmd/agent
	GOOS=linux GOARCH=arm64 GOCACHE=$(GOCACHE_DIR) $(GO) build -o dist/thism-agent-linux-arm64 ./cmd/agent

dev-ui:
	cd frontend && npm run dev -- --host 0.0.0.0 --port 5173

dev-server:
	GOCACHE=$(GOCACHE_DIR) $(GO) run ./cmd/server --token $(TOKEN) --port $(PORT) $(ADMIN_AUTH_ARGS)

dev-rebuild:
	cd frontend && npm run build
	GOCACHE=$(GOCACHE_DIR) $(GO) build -o bin/thism-server ./cmd/server

dev-restart: dev-rebuild
	pkill -x thism-server || true
	nohup ./bin/thism-server --token $(TOKEN) --port $(PORT) $(ADMIN_AUTH_ARGS) >/tmp/thism-server.log 2>&1 & echo $$!
	@echo "Server restarted. Logs: /tmp/thism-server.log"

test:
	GOCACHE=$(GOCACHE_DIR) $(GO) test ./...

clean:
	rm -rf bin/ dist/ frontend/dist/
