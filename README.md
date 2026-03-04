# ThisM

Lightweight self-hosted server monitoring. One binary, zero external dependencies.

## Quick Start

### 1. Start the server

```bash
./thism-server --port 8080 --db ./data.db --token your-admin-token
```

Open http://localhost:8080 in your browser.

### 2. Register a node

```bash
curl -X POST http://localhost:8080/api/nodes/register \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "web-1"}'
# Returns: {"id":"...","token":"..."}
```

### 3. Start the agent on the monitored server

```bash
./thism-agent --server ws://your-host:8080 --token NODE_TOKEN --name web-1
```

## Build from Source

```bash
# Build everything (frontend + server + agent)
make build

# Binaries will be in bin/
./bin/thism-server --token mytoken
./bin/thism-agent --server ws://localhost:8080 --token NODE_TOKEN --name myserver
```

## systemd

Copy the service files from `deploy/`:

```bash
# Server
sudo cp deploy/thism-server.service /etc/systemd/system/
sudo systemctl enable --now thism-server

# Agent (on each monitored machine)
sudo cp deploy/thism-agent.service /etc/systemd/system/
sudo systemctl enable --now thism-agent
```

## Docker

```bash
docker build -t thism-server .
docker run -p 8080:8080 -v ./data:/data -e ADMIN_TOKEN=yourtoken thism-server
```

## Architecture

- **thisM-server**: Single Go binary with embedded React UI. Runs on your main server.
- **thisM-agent**: Lightweight Go binary. Runs on each monitored server.
- **Communication**: Agent connects to server via WebSocket and pushes metrics every 5 seconds.
- **Storage**: SQLite — zero external database dependencies.
