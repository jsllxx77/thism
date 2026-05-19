# Security Policy

English | [简体中文](SECURITY.zh-CN.md)

## Supported Scope

Security reports are welcome for:

- the Go server
- the agent
- authentication and authorization flows
- installer and update paths
- Docker deployment assets in this repository

## Reporting a Vulnerability

Please do **not** open a public GitHub issue for undisclosed vulnerabilities.

Instead, use one of these private channels if available:

- GitHub Security Advisories / private vulnerability reporting
- a private contact method listed on the repository owner or organization profile

When reporting, include:

- affected version or commit
- deployment method
- reproduction steps
- impact assessment
- any proof-of-concept material needed to reproduce safely

## Response Goals

- Acknowledge valid reports as quickly as practical
- Reproduce and assess impact
- Prepare and publish a fix when confirmed
- Credit reporters when appropriate and desired

## Update Integrity

The agent self-update channel is a high-value target — a compromised update path equals remote code execution on every monitored node. ThisM defends it with the following measures:

- **Ed25519 signatures, fail closed.** Every agent built with a pinned release public key requires a valid Ed25519 signature on the replacement binary in addition to the SHA-256 hash. Agents without a pinned key refuse every update. The pinned key is set at compile time via ldflags and cannot be rotated remotely; rotation requires a signed update under the **current** key that carries a new agent build embedding the **new** public key.
- **Private key is offline.** The signing private key is never persisted on the server. The server only sees SHA-256, download URL, target version, and the *hex signature value*. It can dispatch updates but cannot forge them.
- **Server-side validation.** The `/api/agent-updates`, `/api/agent-update-jobs`, and `/api/agent-release` endpoints require a non-empty `signature` field; requests without it return HTTP 400. The manifest endpoint reads `<binary>.sig` sidecar files from disk and exposes the value to agents.
- **Manifest source check.** The agent rejects download URLs that do not point at the configured server host before fetching the binary.

For the operator-facing workflow (key generation, building signed agents, signing dist artifacts, key rotation), see [Release flow](docs/release.md#signed-agent-updates).

## Operator Hardening Notes

- Do **not** put admin tokens, admin passwords, or node tokens on the systemd `ExecStart` command line. Use `EnvironmentFile=` referencing a 0600-owned file (`/etc/default/thism-server`, `/etc/default/thism-agent`). The bundled templates in `deploy/` follow this pattern.
- The SQLite database (`thism.db` + `-wal` / `-shm` sidecars) contains admin password hashes and integration tokens. Recent server builds chmod these files to 0600 on every open; verify the on-disk permissions match if you ever copy the database manually.
- Avoid committing `release.priv.b64` to any repository or backup that is not protected at the same level as a code-signing certificate. The repository `.gitignore` already excludes it; do not override that locally.

