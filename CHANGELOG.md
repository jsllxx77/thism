# Changelog

All notable changes to this project will be documented in this file.

This file tracks release-facing changes for tagged versions and the upcoming `Unreleased` section.

## [Unreleased]

## [0.6.2] - 2026-05-20

### Security

- Both the server and the agent now accept their sensitive flags from `THISM_*` environment variables (`THISM_TOKEN`, `THISM_ADMIN_USER`, `THISM_ADMIN_PASS`, `THISM_AGENT_SERVER`, `THISM_AGENT_TOKEN`, `THISM_AGENT_NAME`). The bundled systemd templates and the in-product install script invoke the binaries with no flags at all, so the secrets never reach `/proc/<pid>/cmdline`. Existing deployments that still use `--token` on `ExecStart=` keep working unchanged; flags take precedence over env vars when both are set.

### Changed

- `deploy/thism-server.service` and `deploy/thism-agent.service` no longer place credential flags on `ExecStart=`; they rely on `EnvironmentFile=/etc/default/thism-*` providing `THISM_*` variables.
- The install script emitted by the panel writes `/etc/default/thism-agent` with `THISM_AGENT_*` keys and registers a unit whose `ExecStart=` is just `/usr/local/bin/thism-agent`.
- `docs/systemd.md` (en + zh) document the new flow, explain why `ExecStart=... --token ${TOKEN} ...` still leaks via `/proc/cmdline` even with `EnvironmentFile=`, and show how to migrate existing installs.

### Upgrading

- Server: replace `/etc/systemd/system/thism-server.service` with the new template and rewrite `/etc/default/thism-server` so each key is prefixed with `THISM_` (e.g. `TOKEN=` becomes `THISM_TOKEN=`). Then `systemctl daemon-reload` + `systemctl restart thism-server`.
- Agents: re-run the install command from the panel; the new script writes the new env file and unit. Agents that don't get re-installed keep working with their old `--token`-on-`ExecStart` unit (no functional regression, only the `/proc/cmdline` leak persists for them).

## [0.6.1] - 2026-05-20

### Fixed

- The upstream Docker image previously shipped unsigned agents with no pinned release public key, so every user who installed via the standard "Add Node → install command → bash" flow ended up with an agent that could install and connect but never self-update. The Docker build now consumes pre-signed agents produced by the release workflow (gated on the `THISM_RELEASE_PUBLIC_KEY` and `THISM_RELEASE_PRIVATE_KEY` repository secrets), and the workflow fails fast rather than publishing an unsigned image.
- The systemd unit emitted by the panel's install script no longer places the node token on the `ExecStart` command line. The install script now writes `/etc/default/thism-agent` (mode 0600) and the unit references it via `EnvironmentFile=`, so the token no longer leaks via `/proc/<pid>/cmdline`.

### Changed

- `.github/workflows/release.yml` now requires both signing secrets, builds and signs the agent binaries with `thism-sign` before invoking Docker Buildx, and attaches the signed binaries plus `.sig` sidecars to the GitHub Release.
- `Dockerfile` accepts `PREBUILT_AGENTS` and `RELEASE_PUBLIC_KEY` build args. CI uses `PREBUILT_AGENTS=1` to consume the pre-signed dist artifacts; local `docker build` defaults to compiling the agents in-container without a pinned key (self-update will fail closed, matching the v0.6.0 contract).

## [0.6.0] - 2026-05-20

### Security

- Agent self-update binaries are now verified with Ed25519 signatures alongside the existing SHA-256 hash; the verifier public key is pinned into the agent at build time via `-X github.com/thism-dev/thism/internal/security/release.PublicKeyBase64`, and agents without a pinned key now fail closed (refuse to apply any binary update). Includes a new `thism-sign` CLI for offline key generation and binary signing, and corresponding `make release-keygen` / `make sign-dist` targets.
- Removed the hard-coded `thism2026` development admin token from the Makefile and the dev systemd unit. The Makefile dev targets now require a `TOKEN=` value, and the dev systemd unit now requires `/etc/default/thism-dev-server` to be provided rather than falling back to a baked-in default.
- SQLite database file (and its `-wal` / `-shm` sidecars) are now created with mode `0600` on every store open, preventing other local users from reading admin password hashes, integration tokens, and node metadata.

### Added

- `signature` field on the agent self-update API (`/api/agent-updates`, `/api/agent-update-jobs`) and the auto-update manifest (`/api/agent-release`), populated from a sibling `<binary>.sig` file in the dist directory.
- Lazy mount and dedicated Monitoring tab placement for the Latency Monitors card so the Agent tab opens faster.

### Fixed

- Dashboard card rendering churn by removing the page-wide 1s refresh loop, memoizing node cards, and isolating live last-seen updates to the label itself
- Agent websocket overhead by stopping redundant latency monitor config resync on every metrics heartbeat
- Latest-metrics lookups for node listings by switching from per-node queries to a batched store query
- Settings tab startup cost by lazily mounting section panels, deferring tab-specific API requests until first open, and pausing alert dispatcher polling when the Alerts tab is hidden
- Node detail metric recomputation by deriving all CPU, memory, network, and disk chart series from a shared segmented pass instead of rebuilding each series separately
- Agent metrics collection latency by replacing the blocking 1-second CPU percentage probe with a non-blocking cumulative CPU time sampler
- Agent network topology refresh cost by caching detected local IP and selected non-loopback interfaces for a short TTL instead of re-enumerating them on every metrics report

## [0.4.0] - 2026-04-11

### Added

- Persistent `latency_1m` rollups for node-detail latency history, with automatic backfill on large-range requests

### Fixed

- Long-range latency chart rendering by switching 7d+ requests to 1-minute rollups, preserving lightweight front-end downsampling, and suppressing dense chart dots
- Frontend linting and agent release build reliability in CI
- GitHub Actions runtime compatibility for the release pipeline

## [0.3.0] - 2026-04-08

### Added

- Telegram notification delivery, test sends, notification node scoping, dispatcher diagnostics, and configurable dispatcher queue settings
- Dashboard card IP visibility controls, explicit sign-out, and a header shortcut back to the dashboard

### Fixed

- Agent Docker socket lifecycle and latest-metrics lookup performance
- Admin session persistence, notification settings behavior, login language toggle contrast, and mobile guest-mode navigation

### Documentation

- Split user-facing setup docs from contributor, release, architecture, and `systemd` references

## [0.2.0] - 2026-03-18

### Added

- Metrics retention controls, real node uptime, inbound and outbound traffic speed, Docker container detail views, and surfaced agent versions
- Unified runtime and release version metadata across builds and published artifacts

### Fixed

- CI and `dev-restart` stability
- Authentication hardening and general server reliability improvements

## [0.1.0] - 2026-03-08

### Added

- Initial public release of the ThisM server and Linux agent
- Docker Compose deployment assets, CI workflow, container release workflow, and development `systemd` service templates
