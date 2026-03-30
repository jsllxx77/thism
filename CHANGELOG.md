# Changelog

All notable changes to this project will be documented in this file.

This file tracks release-facing changes for tagged versions and the upcoming `Unreleased` section.

## [Unreleased]

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
