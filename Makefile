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
DEV_SYSTEMD_SERVICE ?= thism-server.service
DEV_SYSTEMD_ENV_FILE ?= /etc/default/thism-dev-server

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
	@if command -v systemctl >/dev/null 2>&1 && [ "$$(id -u)" -eq 0 ] && systemctl cat $(DEV_SYSTEMD_SERVICE) >/dev/null 2>&1; then \
		tmp_file=$$(mktemp); \
		resolved_admin_user="$(ADMIN_USER)"; \
		resolved_admin_pass="$(ADMIN_PASS)"; \
		if [ -f $(DEV_SYSTEMD_ENV_FILE) ]; then \
			set -a; \
			. $(DEV_SYSTEMD_ENV_FILE); \
			set +a; \
			if [ -z "$$resolved_admin_user" ] && [ -n "$$ADMIN_USER" ]; then resolved_admin_user="$$ADMIN_USER"; fi; \
			if [ -z "$$resolved_admin_pass" ] && [ -n "$$ADMIN_PASS" ]; then resolved_admin_pass="$$ADMIN_PASS"; fi; \
		fi; \
		printf "TOKEN=%s\nPORT=%s\n" "$(TOKEN)" "$(PORT)" > "$$tmp_file"; \
		if [ -n "$$resolved_admin_user" ]; then printf "ADMIN_USER=%s\n" "$$resolved_admin_user" >> "$$tmp_file"; fi; \
		if [ -n "$$resolved_admin_pass" ]; then printf "ADMIN_PASS=%s\n" "$$resolved_admin_pass" >> "$$tmp_file"; fi; \
		install -m 600 "$$tmp_file" $(DEV_SYSTEMD_ENV_FILE); \
		rm -f "$$tmp_file"; \
		systemctl stop $(DEV_SYSTEMD_SERVICE) || true; \
		pkill -x thism-server || true; \
		sleep 1; \
		systemctl reset-failed $(DEV_SYSTEMD_SERVICE) || true; \
		systemctl start $(DEV_SYSTEMD_SERVICE); \
		systemctl --no-pager --lines=8 status $(DEV_SYSTEMD_SERVICE); \
	else \
		pkill -x thism-server || true; \
		nohup ./bin/thism-server --token $(TOKEN) --port $(PORT) $(ADMIN_AUTH_ARGS) >/tmp/thism-server.log 2>&1 & echo $$!; \
	fi
	@echo "Server restarted. Logs: /tmp/thism-server.log"

test:
	GOCACHE=$(GOCACHE_DIR) $(GO) test ./...

clean:
	rm -rf bin/ dist/ frontend/dist/
