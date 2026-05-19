# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binaries
FROM golang:1.24-alpine AS builder
WORKDIR /app
ARG THISM_VERSION=dev
ARG THISM_COMMIT=unknown
ARG THISM_BUILD_TIME=unknown
# When PREBUILT_AGENTS=1 (CI release path) the workflow has already built
# and signed dist/thism-agent-linux-* outside Docker, with the official
# release public key baked in. The Docker build just consumes them.
# When PREBUILT_AGENTS=0 (default, local dev) the agents are built inside
# this stage without a pinned public key — self-update will fail closed.
ARG PREBUILT_AGENTS=0
ARG RELEASE_PUBLIC_KEY=
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN set -eux; \
    LDFLAGS="-s -w -X github.com/thism-dev/thism/internal/version.Version=${THISM_VERSION} -X github.com/thism-dev/thism/internal/version.Commit=${THISM_COMMIT} -X github.com/thism-dev/thism/internal/version.BuildTime=${THISM_BUILD_TIME}"; \
    mkdir -p /out/dist; \
    GOCACHE=/tmp/go-build go build -ldflags "${LDFLAGS}" -o /out/thism-server ./cmd/server; \
    if [ "${PREBUILT_AGENTS}" = "1" ]; then \
        cp dist/thism-agent-linux-amd64 /out/dist/; \
        cp dist/thism-agent-linux-arm64 /out/dist/; \
        cp dist/thism-agent-linux-amd64.version /out/dist/; \
        cp dist/thism-agent-linux-arm64.version /out/dist/; \
        if [ -f dist/thism-agent-linux-amd64.sig ]; then cp dist/thism-agent-linux-amd64.sig /out/dist/; fi; \
        if [ -f dist/thism-agent-linux-arm64.sig ]; then cp dist/thism-agent-linux-arm64.sig /out/dist/; fi; \
    else \
        AGENT_LDFLAGS="${LDFLAGS}"; \
        if [ -n "${RELEASE_PUBLIC_KEY}" ]; then \
            AGENT_LDFLAGS="${AGENT_LDFLAGS} -X github.com/thism-dev/thism/internal/security/release.PublicKeyBase64=${RELEASE_PUBLIC_KEY}"; \
        fi; \
        GOOS=linux GOARCH=amd64 GOCACHE=/tmp/go-build go build -ldflags "${AGENT_LDFLAGS}" -o /out/dist/thism-agent-linux-amd64 ./cmd/agent; \
        GOOS=linux GOARCH=arm64 GOCACHE=/tmp/go-build go build -ldflags "${AGENT_LDFLAGS}" -o /out/dist/thism-agent-linux-arm64 ./cmd/agent; \
        printf "%s\n" "${THISM_VERSION}" > /out/dist/thism-agent-linux-amd64.version; \
        printf "%s\n" "${THISM_VERSION}" > /out/dist/thism-agent-linux-arm64.version; \
    fi

# Stage 3: Minimal runtime image
FROM alpine:3.19
RUN adduser -D -h /opt/thism thism \
	&& mkdir -p /opt/thism/dist /data \
	&& chown -R thism:thism /opt/thism /data
WORKDIR /opt/thism
COPY --from=builder /out/thism-server ./thism-server
COPY --from=builder /out/dist ./dist
USER thism
EXPOSE 8080
ENTRYPOINT ["./thism-server"]
