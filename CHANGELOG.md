# Changelog

All notable changes to this project will be documented in this file.

This file tracks release-facing changes for tagged versions and the upcoming `Unreleased` section.

## [Unreleased]

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
