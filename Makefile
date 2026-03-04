.PHONY: build build-server build-agent build-agent-all build-frontend test clean

build: build-frontend build-server build-agent build-agent-all

build-frontend:
	cd frontend && npm install && npm run build

build-server: build-frontend
	go build -o bin/thism-server ./cmd/server

build-agent:
	go build -o bin/thism-agent ./cmd/agent

build-agent-all:
	GOOS=linux GOARCH=amd64 go build -o dist/thism-agent-linux-amd64 ./cmd/agent
	GOOS=linux GOARCH=arm64 go build -o dist/thism-agent-linux-arm64 ./cmd/agent

test:
	go test ./...

clean:
	rm -rf bin/ dist/ frontend/dist/
