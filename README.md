# ThisM

English | [简体中文](README.zh-CN.md)

Lightweight self-hosted server monitoring. One binary, zero external dependencies.

## Highlights

- Single Go server binary with embedded React frontend
- Lightweight Linux agents for monitored nodes
- SQLite storage with no external database requirement
- Server-hosted agent install script and release manifest
- Prebuilt GHCR image plus Docker Compose deployment path

## Quick Start

### One-command Docker Compose install

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/thism-dev/thism/main/deploy/install-compose.sh)
```

The installer will:

1. Create a deployment directory
2. Download `compose.yaml` and `.env.example`
3. Generate a random admin password and API token on first run
4. Start `thism-server` from `ghcr.io/thism-dev/thism:latest`

Prerequisites:

- Docker with `docker compose` v2 available on the host

When it finishes, open `http://<server-ip-or-domain>:8080` from your browser and log in with the credentials printed by the installer. If you are running the installer on the same machine where you will open the browser, `http://localhost:8080` also works.

The generated credentials are stored in `~/thism-deploy/.env`. Treat that file as sensitive because it contains the API token and the web UI administrator password.

### Manual Docker Compose deployment

```bash
mkdir -p ~/thism-deploy
cd ~/thism-deploy
curl -fsSL https://raw.githubusercontent.com/thism-dev/thism/main/deploy/docker-compose.yml -o compose.yaml
curl -fsSL https://raw.githubusercontent.com/thism-dev/thism/main/deploy/.env.example -o .env

# edit .env before first start
docker compose up -d
```

The default compose deployment stores application data in a named Docker volume and publishes the web UI on port `8080`.

The `.env` file contains the API token and web login credentials. Keep it private and back it up if you want to preserve the generated secrets.

### Build from source

```bash
make build

./bin/thism-server --port 8080 --db ./thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

Open `http://localhost:8080` in your browser. You will be redirected to `/login` and authenticate with the configured username and password.

## Register and Install an Agent

### 1. Register a node

```bash
curl -X POST http://localhost:8080/api/nodes/register \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "web-1"}'
# Returns: {"id":"...","token":"..."}
```

### 2. Start the agent manually

```bash
./bin/thism-agent --server ws://your-host:8080 --token NODE_TOKEN --name web-1
```

### 3. Or install the agent from the server-hosted script

```bash
curl -fsSL "http://your-host:8080/install.sh?token=NODE_TOKEN&name=web-1" | bash
```

The install script detects `linux/amd64` and `linux/arm64`, installs `thism-agent` to `/usr/local/bin/thism-agent`, and writes a `systemd` unit that reconnects to the server after restart.

## Docker Image

Published runtime image:

```bash
ghcr.io/thism-dev/thism:latest
```

You can also run it directly without Compose:

```bash
docker run --name thism-server -p 8080:8080 \
  -v thism-data:/data \
  ghcr.io/thism-dev/thism:latest \
  --port 8080 --db /data/thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

## Development Workflow

### Fast local loop

Use two terminals for everyday frontend work:

```bash
# Terminal 1: backend API/ws
make dev-server TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass

# Terminal 2: Vite with HMR
make dev-ui
```

Open `http://localhost:5173`. Frontend changes hot-reload instantly.

### Verify the embedded frontend

```bash
make dev-restart TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass
```

This command:

1. Builds the frontend
2. Rebuilds `bin/thism-server` with embedded assets
3. Restarts the local server

### Test and build

```bash
make test

cd frontend
npm ci
npm run lint
npm test
npm run build
```

## systemd

The files in `deploy/` are templates. Before enabling them, replace the placeholder values in `ExecStart`, make sure the referenced binary exists, and create any required runtime user / working directory:

```bash
# Server
sudo cp deploy/thism-server.service /etc/systemd/system/

# Edit placeholders such as YOUR_ADMIN_TOKEN / YOUR_ADMIN_USER / YOUR_ADMIN_PASSWORD
# and ensure the `thism` user plus /var/lib/thism exist before starting.
sudo systemctl enable --now thism-server

# Agent (on each monitored machine)
sudo cp deploy/thism-agent.service /etc/systemd/system/

# Edit YOUR_SERVER_HOST / YOUR_NODE_TOKEN / YOUR_NODE_NAME before starting.
sudo systemctl enable --now thism-agent
```

## Build Docker Image from Source

```bash
docker build -t thism-server .
docker run -p 8080:8080 -v thism-data:/data thism-server \
  --port 8080 --db /data/thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

## Architecture

- **thisM-server**: Single Go binary with embedded React UI. Runs on your main server and hosts agent downloads.
- **thisM-agent**: Lightweight Go binary. Runs on each monitored server.
- **Communication**: Agent connects to server via WebSocket and pushes metrics every 5 seconds.
- **Storage**: SQLite with zero external database dependencies.
- **Deployment**: Source builds, Docker image, and Docker Compose are supported.
