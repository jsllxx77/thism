# ThisM

English | [简体中文](README.zh-CN.md)

Lightweight self-hosted server monitoring. One binary, zero external dependencies.

## Highlights

- Single Go server binary with embedded React frontend
- Lightweight Linux agents for monitored nodes
- SQLite storage with no external database requirement
- Server-hosted agent install script and release manifest
- Built-in ICMP, TCP, and HTTP latency monitoring from selected nodes
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

## Add and Install an Agent

Use the web console for the normal enrollment flow:

1. Open the web UI and sign in as an administrator.
2. Go to `Settings`.
3. In the `Node Management` section, click `Add Node`.
4. Enter the node name and click `Get install command`.
5. Copy the `Install Command` shown by the panel.
6. Run that command as `root` on the target Linux machine.

The generated command installs `thism-agent` into `/usr/local/bin`, writes a `systemd` unit, and starts the service. The installer supports `linux/amd64` and `linux/arm64`.

If the node already exists and you need the command again, open `Settings` -> `Node Management` and click `Get Script` on that node row.

## Latency Monitoring

ThisM can run active latency checks from your agents and plot the results on the node detail page.

Current monitor types:

- `ICMP`
- `TCP`
- `HTTP`

To configure a monitor:

1. Open `Settings`.
2. Switch to the `Agent` section.
3. Open `Latency Monitors`.
4. Create a monitor, choose the target, interval, and nodes that should run it.
5. Open a node detail page to view the latency chart for the monitors assigned to that node.

## More Documentation

- [Advanced installation options](docs/advanced-install.md): build from source, run the published Docker image directly, or build the Docker image locally
- [systemd deployment templates](docs/systemd.md): use the bundled unit files for manual host installs
- [Development workflow](docs/development.md): local contributor loop, frontend validation, and test/build commands
- [Release flow](docs/release.md): tag-driven release process and published image tags
- [Architecture overview](docs/architecture.md): server, agent, transport, storage, and deployment model
- [Contributing](CONTRIBUTING.md): repository contribution guidelines
