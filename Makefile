.PHONY: \
	build build-server build-agent build-agent-all build-frontend test clean \
	dev-ui dev-server dev-rebuild dev-restart \
	build-sign-tool release-keygen sign-dist

GO ?= go
GO_PACKAGES := $(shell $(GO) list ./... | grep -v '/frontend/node_modules/')
GOCACHE_DIR ?= /tmp/go-build
PORT ?= 12026
TOKEN ?=
ADMIN_USER ?=
ADMIN_PASS ?=
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X github.com/thism-dev/thism/internal/version.Version=$(VERSION) -X github.com/thism-dev/thism/internal/version.Commit=$(COMMIT) -X github.com/thism-dev/thism/internal/version.BuildTime=$(BUILD_TIME)
SERVER_TLS_SPKI_SHA256 ?=
AGENT_LDFLAGS := $(LDFLAGS)$(if $(RELEASE_PUBLIC_KEY), -X github.com/thism-dev/thism/internal/security/release.PublicKeyBase64=$(RELEASE_PUBLIC_KEY),)$(if $(SERVER_TLS_SPKI_SHA256), -X github.com/thism-dev/thism/internal/collector.ServerTLSSPKISHA256Base64=$(SERVER_TLS_SPKI_SHA256),)
ADMIN_AUTH_ARGS := $(strip $(if $(ADMIN_USER),--admin-user $(ADMIN_USER),) $(if $(ADMIN_PASS),--admin-pass $(ADMIN_PASS),))
DEV_SYSTEMD_SERVICE ?= thism-server.service
DEV_SYSTEMD_ENV_FILE ?= /etc/default/thism-dev-server

build: build-frontend build-server build-agent build-agent-all

build-frontend:
	cd frontend && npm ci && npm run build

build-server: build-frontend
	GOCACHE=$(GOCACHE_DIR) $(GO) build -ldflags "$(LDFLAGS)" -o bin/thism-server ./cmd/server

build-agent:
	GOCACHE=$(GOCACHE_DIR) $(GO) build -ldflags "$(AGENT_LDFLAGS)" -o bin/thism-agent ./cmd/agent
	printf "%s\n" "$(VERSION)" > bin/thism-agent.version

build-agent-all:
	GOOS=linux GOARCH=amd64 GOCACHE=$(GOCACHE_DIR) $(GO) build -ldflags "$(AGENT_LDFLAGS)" -o dist/thism-agent-linux-amd64 ./cmd/agent
	printf "%s\n" "$(VERSION)" > dist/thism-agent-linux-amd64.version
	GOOS=linux GOARCH=arm64 GOCACHE=$(GOCACHE_DIR) $(GO) build -ldflags "$(AGENT_LDFLAGS)" -o dist/thism-agent-linux-arm64 ./cmd/agent
	printf "%s\n" "$(VERSION)" > dist/thism-agent-linux-arm64.version

dev-ui:
	cd frontend && npm run dev -- --host 0.0.0.0 --port 5173

dev-server:
	@if [ -z "$(TOKEN)" ]; then echo "TOKEN is required (e.g. TOKEN=\"\$$(openssl rand -hex 32)\" make dev-server)"; exit 1; fi
	GOCACHE=$(GOCACHE_DIR) $(GO) run -ldflags "$(LDFLAGS)" ./cmd/server --token $(TOKEN) --port $(PORT) $(ADMIN_AUTH_ARGS)

dev-rebuild:
	cd frontend && npm run build
	GOCACHE=$(GOCACHE_DIR) $(GO) build -ldflags "$(LDFLAGS)" -o bin/thism-server ./cmd/server

dev-restart: dev-rebuild
	@if [ -z "$(TOKEN)" ]; then echo "TOKEN is required (e.g. TOKEN=\"\$$(openssl rand -hex 32)\" make dev-restart)"; exit 1; fi
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
	GOCACHE=$(GOCACHE_DIR) $(GO) test $(GO_PACKAGES)

# Release signing -----------------------------------------------------------
# RELEASE_PUBLIC_KEY=<base64> is the Ed25519 public key the agent should pin.
# Pass it to build-agent-all so the verifier baked into the binary uses your
# key. Generate keys once via `make release-keygen` and store the private key
# offline (NEVER commit it).
RELEASE_PUBLIC_KEY ?=
RELEASE_PRIV_FILE  ?= release.priv.b64
RELEASE_PUB_FILE   ?= release.pub.b64

build-sign-tool:
	GOCACHE=$(GOCACHE_DIR) $(GO) build -o bin/thism-sign ./cmd/thism-sign

release-keygen: build-sign-tool
	@if [ -f $(RELEASE_PRIV_FILE) ]; then echo "$(RELEASE_PRIV_FILE) already exists; refusing to overwrite"; exit 1; fi
	./bin/thism-sign keygen -out-pub $(RELEASE_PUB_FILE) -out-priv $(RELEASE_PRIV_FILE)
	@echo
	@echo "Add to your build command:"
	@echo "  RELEASE_PUBLIC_KEY=\"\$$(cat $(RELEASE_PUB_FILE))\" make build-agent-all"

sign-dist: build-sign-tool
	@if [ ! -f $(RELEASE_PRIV_FILE) ] && [ -z "$$THISM_RELEASE_PRIVATE_KEY" ]; then \
		echo "private key required: provide $(RELEASE_PRIV_FILE) or set THISM_RELEASE_PRIVATE_KEY"; exit 1; \
	fi
	@if [ -f $(RELEASE_PRIV_FILE) ]; then \
		./bin/thism-sign sign-dist -priv $(RELEASE_PRIV_FILE) -dir dist; \
	else \
		./bin/thism-sign sign-dist -dir dist; \
	fi

clean:
	rm -rf bin/ dist/ frontend/dist/
