# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN mkdir -p /out/dist \
	&& GOCACHE=/tmp/go-build go build -o /out/thism-server ./cmd/server \
	&& GOOS=linux GOARCH=amd64 GOCACHE=/tmp/go-build go build -o /out/dist/thism-agent-linux-amd64 ./cmd/agent \
	&& GOOS=linux GOARCH=arm64 GOCACHE=/tmp/go-build go build -o /out/dist/thism-agent-linux-arm64 ./cmd/agent

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
