# ThisM

English | [简体中文](README.zh-CN.md)

Lightweight self-hosted server monitoring. One binary, zero external dependencies.

## Preview

<p>
  <img src="docs/assets/previews/dashboard.png" alt="ThisM dashboard showing node health cards and resource usage" width="100%">
</p>

<p>
  <img src="docs/assets/previews/reports.png" alt="ThisM availability report with SLA charts and node rankings" width="49%">
  <img src="docs/assets/previews/dashboard-admin-table.png" alt="ThisM administrator node inventory table with node versions and redacted IP addresses" width="49%">
</p>

<p>
  <img src="docs/assets/previews/node-detail-metrics.png" alt="ThisM administrator node detail view with hardware profile and metric charts" width="49%">
  <img src="docs/assets/previews/node-detail-runtime.png" alt="ThisM administrator node detail view with latency monitors and top processes" width="49%">
</p>

<p>
  <img src="docs/assets/previews/settings-monitoring.png" alt="ThisM monitoring settings with retention, dashboard visibility, and latency monitors" width="49%">
  <img src="docs/assets/previews/settings-node-management.png" alt="ThisM node management settings with enrollment and node actions" width="49%">
</p>

## Highlights

- Single Go server binary with embedded React frontend
- Lightweight Linux agents for monitored nodes
- SQLite storage with no external database requirement
- Server-hosted agent install script and release manifest
- Ed25519-signed agent self-updates (fail-closed when the public key is missing)
- Node tags, tag filtering, and SLA-style availability reports
- Built-in ICMP, TCP, and HTTP latency monitoring from selected nodes
- Configurable metrics retention, defaulting to 30 days with longer reporting options
- Runtime shadcn/ui theme packages and full frontend skin packages installable from GitHub
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

## Themes and Frontend Skins

The current ThisM appearance system has three layers:

- Built-in themes are `Classic`, `Ocean`, and `Graphite`. They share the same dashboard layout and card geometry while keeping different color, density, and control treatments.
- Runtime theme packages keep the built-in React frontend and replace its shadcn/ui semantic tokens, card/panel radius, density, typography, surfaces, navigation treatment, and shadows.
- Frontend skin packages install a complete alternative frontend as a zip archive with its own `index.html`, CSS, and JavaScript.

Install them from the web UI:

1. Open `Settings`.
2. Open `Appearance`.
3. Use `Theme System` for theme JSON files or GitHub theme repositories.
4. Use `Frontend Skins` for skin zip files or GitHub skin repositories.

Runtime theme packages are stored in the browser's local storage. They are best for changing how the bundled shadcn/ui dashboard looks without replacing the application. Frontend skins are stored by the server and are the right option when you need a fully custom UI.

Example runtime theme repository:

```text
https://github.com/jsllxx77/thism-shadcn-operations-theme
```

Paste that URL into `Settings` -> `Appearance` -> `Theme System` -> `GitHub theme repository` to install the Shadcn Operations theme. ThisM loads the latest release theme asset when one exists, otherwise it falls back to a recognized theme JSON file in the repository.

### Build a Theme Package

A theme package is a JSON file with `type: "thism-theme"` and `version: 1`. The `id` must use lowercase letters, numbers, and hyphens after normalization, and cannot be `classic`, `ocean`, or `graphite`.

The GitHub importer accepts a direct raw/blob/release URL, or a repository URL. Repository imports look for the latest release asset first, then these repository paths: `thism-theme.json`, `.thism-theme.json`, `theme.json`, `themes/thism-theme.json`, and `themes/theme.json`. Release assets are accepted when named `thism-theme.json`, `theme.json`, or `*.thism-theme.json`.

Minimal package:

```json
{
  "type": "thism-theme",
  "version": 1,
  "id": "shadcn-operations",
  "name": "Shadcn Operations",
  "description": "Neutral shadcn/ui operations dashboard theme.",
  "accent": "#18181b",
  "tokens": {
    "light": {
      "background": "240 6% 96%",
      "foreground": "240 10% 3.9%",
      "card": "240 6% 99%",
      "card-foreground": "240 10% 3.9%",
      "primary": "240 5.9% 10%",
      "primary-foreground": "0 0% 98%",
      "border": "240 6% 84%",
      "input": "240 6% 84%",
      "ring": "240 5.9% 10%"
    },
    "dark": {
      "background": "240 10% 3.9%",
      "foreground": "0 0% 98%",
      "card": "240 7% 7%",
      "card-foreground": "0 0% 98%",
      "primary": "0 0% 98%",
      "primary-foreground": "240 5.9% 10%",
      "border": "240 3.7% 15.9%",
      "input": "240 3.7% 15.9%",
      "ring": "240 4.9% 83.9%"
    }
  },
  "appearance": {
    "radius": "0.625rem",
    "cardRadius": "0.75rem",
    "panelRadius": "0.75rem",
    "controlRadius": "0.5rem",
    "density": "compact",
    "surface": "solid",
    "background": "solid",
    "navigation": "solid",
    "cardPadding": "0.875rem",
    "panelPadding": "1rem",
    "fontFamily": "\"Inter\", \"Fira Sans\", \"Segoe UI\", sans-serif",
    "monoFontFamily": "\"JetBrains Mono\", \"Fira Code\", \"SFMono-Regular\", monospace",
    "shadow": "none"
  }
}
```

The validator requires the core light and dark tokens shown above. Full shadcn/ui-compatible themes can also include optional tokens such as `secondary`, `muted`, `accent`, `destructive`, `popover`, `chart-1` through `chart-5`, and `sidebar-*`.

`appearance` supports these runtime fields:

- `radius`, `cardRadius`, `panelRadius`, `controlRadius`, `cardPadding`, and `panelPadding` as CSS lengths such as `0.75rem`.
- `fontFamily` and `monoFontFamily` as safe font-family strings.
- `shadow` as a safe CSS shadow string.
- `density`: `compact`, `comfortable`, or `spacious`.
- `surface`: `solid`, `glass`, or `command`.
- `background`: `solid`, `grid`, or `mesh`.
- `navigation`: `solid`, `floating`, or `transparent`.

To publish one from a repository:

```bash
mkdir thism-theme
cd thism-theme
$EDITOR thism-theme.json
git init
git add thism-theme.json
git commit -m "Add thisM theme"
gh repo create <owner>/<repo> --public --source . --remote origin --push
gh release create v1.0.0 thism-theme.json --title v1.0.0 --notes "Initial thisM theme"
```

For a complete repository example, see `https://github.com/jsllxx77/thism-shadcn-operations-theme`. It includes:

- `thism-theme.json` for direct ThisM import
- `registry-item.json` for shadcn registry-compatible distribution
- `styles/shadcn-theme.css` and `styles/shadcn-v4.css` for shadcn projects
- release assets that ThisM can discover from the repository URL

### Build a Frontend Skin Package

A frontend skin package is a `.zip` archive with `thism-frontend-skin.json` at the archive root. The skin ID must use lowercase letters, numbers, and hyphens, and cannot be `classic`. The entry file must be HTML. Archives are limited to 32 MiB compressed, 96 MiB extracted, and 2048 files.

Manifest example:

```json
{
  "type": "thism-frontend-skin",
  "version": 1,
  "id": "ops-console",
  "name": "Ops Console",
  "description": "Custom thisM frontend skin.",
  "entry": "index.html",
  "apiVersion": "thism.v1",
  "assets": ["assets/app.css", "assets/app.js"],
  "preview": "preview.png"
}
```

Recommended archive layout:

```text
thism-frontend-skin.json
index.html
assets/app.css
assets/app.js
preview.png
```

Package and publish:

```bash
zip -r ops-console.thism-frontend-skin.zip thism-frontend-skin.json index.html assets preview.png
gh release create v1.0.0 ops-console.thism-frontend-skin.zip --title v1.0.0 --notes "Initial thisM frontend skin"
```

The GitHub importer accepts a direct raw/release URL, or a repository URL. Repository imports look for the latest release asset first, then `thism-frontend-skin.zip`, `frontend-skin.zip`, and `skins/thism-frontend-skin.zip`. Release assets are accepted when named `thism-frontend-skin.zip` or `*.thism-frontend-skin.zip`.

Installed skins are stored in the server's frontend skin directory. By default this is `frontend-skins` beside the database path, or `./frontend-skins` when the database path is empty or `:memory:`. Override it with `THISM_FRONTEND_SKINS_DIR` or `thism-server --frontend-skins-dir`.

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
