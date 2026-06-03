# ThisM

English | [简体中文](README.zh-CN.md)

Lightweight self-hosted server monitoring. One binary, zero external dependencies.

## Highlights

- Single Go server binary with embedded React frontend
- Lightweight Linux agents for monitored nodes
- SQLite storage with no external database requirement
- Server-hosted agent install script and release manifest
- Ed25519-signed agent self-updates (fail-closed when the public key is missing)
- Node tags, tag filtering, and SLA-style availability reports
- Built-in ICMP, TCP, and HTTP latency monitoring from selected nodes
- Configurable metrics retention, defaulting to 30 days with longer reporting options
- Prebuilt GHCR image plus Docker Compose deployment path

## Quick Start

### One-command Docker Compose install

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/jsllxx77/thism/main/deploy/install-compose.sh)
```

The installer will:

1. Create a deployment directory
2. Download `compose.yaml` and `.env.example`
3. Generate a random admin password and API token on first run
4. Start `thism-server` from `ghcr.io/jsllxx77/thism:latest`

Prerequisites:

- Docker with `docker compose` v2 available on the host

When it finishes, open `http://<server-ip-or-domain>:8080` from your browser and log in with the credentials printed by the installer. If you are running the installer on the same machine where you will open the browser, `http://localhost:8080` also works.

The generated credentials are stored in `~/thism-deploy/.env`. Treat that file as sensitive because it contains the API token and the web UI administrator password.

To uninstall the server from the host:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/jsllxx77/thism/main/deploy/uninstall-server.sh)
```

The uninstall script stops the server and removes the local service/deployment files. It preserves Docker volume data and `/var/lib/thism` by default. To remove stored server data as well, run it with `THISM_REMOVE_DATA=1`. Agents installed on monitored hosts are not removed automatically; run the agent uninstall script on each host if needed.

### Manual Docker Compose deployment

```bash
mkdir -p ~/thism-deploy
cd ~/thism-deploy
curl -fsSL https://raw.githubusercontent.com/jsllxx77/thism/main/deploy/docker-compose.yml -o compose.yaml
curl -fsSL https://raw.githubusercontent.com/jsllxx77/thism/main/deploy/.env.example -o .env

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

## Node Tags and Reports

Use tags to organize nodes by environment, region, workload, or any other operator-owned grouping.

To edit tags:

1. Open `Settings`.
2. In the `Node Management` section, click `Edit tags` on a node row.
3. Enter comma-separated tags such as `prod, hk, database`.
4. Save the node.

Tags are normalized to lowercase so filters treat `Prod`, `prod`, and `PROD` as the same tag.

The `Reports` page summarizes availability and latency for the selected time window. It includes:

- `24h`, `7d`, and `30d` report ranges
- Tag filtering
- Average availability, nodes below 99%, total offline time, and highest p95 latency
- Availability ranking, offline impact, and SLA distribution charts
- Node-level SLA rows with samples, outages, p95 latency, and last seen status

Availability reports are computed from retained metrics and latency samples. If historical data has already been pruned, older report windows may contain less evidence than the selected range implies.

To uninstall an agent from a monitored Linux host:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/jsllxx77/thism/main/deploy/uninstall-agent.sh)
```

This removes the local `systemd` service, environment file, agent binary, and version file. It does not delete the node record from the ThisM server. If you no longer want the node listed in the panel, open `Settings` -> `Node Management` and delete it there as well.

## Latency Monitoring

ThisM can run active latency checks from your agents and plot the results on the node detail page.

Current monitor types:

- `ICMP`
- `TCP`
- `HTTP`

To configure a monitor:

1. Open `Settings`.
2. Switch to the `Monitoring` section.
3. Open `Latency Monitors`.
4. Create a monitor, choose the target, interval, and nodes that should run it.
5. Open a node detail page to view the latency chart for the monitors assigned to that node.

## Metrics Retention

Metrics retention controls how long historical metrics and latency samples stay on the server. The default is `30 days`; available options are `30`, `90`, `180`, and `365` days.

To change retention:

1. Open `Settings`.
2. Switch to the `Monitoring` section.
3. Open `Metrics Retention`.
4. Choose the retention period and save.

Changes apply immediately and prune metric rows older than the selected period. Reports and long-range node detail charts depend on this retained history.

## Releases and Update Integrity

Starting with v0.6.0, ThisM agents verify every self-update binary with an Ed25519 signature in addition to the SHA-256 hash. Agents built without a pinned release public key refuse to apply any update (fail closed).

If you only run the upstream Docker image, no extra setup is needed; the upstream agents ship with the project's pinned public key.

If you publish your own builds (fork, internal mirror, self-hosted distribution), you must:

1. Generate a release keypair offline (`make release-keygen`) and keep the private key off the server.
2. Build agents with the public key baked in (`RELEASE_PUBLIC_KEY="$(cat release.pub.b64)" make build-agent-all`).
3. Sign the produced binaries (`make sign-dist`) so the manifest endpoint can serve the `.sig` sidecar files.

See [Release flow](docs/release.md) for the full workflow, key rotation, and failure modes.

## More Documentation

- [Advanced installation options](docs/advanced-install.md): build from source, run the published Docker image directly, or build the Docker image locally
- [systemd deployment templates](docs/systemd.md): use the bundled unit files for manual host installs
- [Development workflow](docs/development.md): local contributor loop, frontend validation, and test/build commands
- [Release flow](docs/release.md): tag-driven release process and published image tags
- [Security roadmap](docs/security-roadmap.md): outstanding hardening work and history of what shipped in 0.6.x
- [Architecture overview](docs/architecture.md): server, agent, transport, storage, and deployment model
- [Contributing](CONTRIBUTING.md): repository contribution guidelines
