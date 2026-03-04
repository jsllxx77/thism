.PHONY: build build-server build-agent build-frontend test clean

build: build-frontend build-server build-agent

build-frontend:
	cd frontend && npm install && npm run build

build-server: build-frontend
	go build -o bin/thism-server ./cmd/server

build-agent:
	go build -o bin/thism-agent ./cmd/agent

test:
	go test ./...

clean:
	rm -rf bin/ frontend/dist/
