# Security Roadmap

[简体中文](security-roadmap.zh-CN.md)

Tracks the remaining hardening items from the November 2026 security review. Items here either need a code change, a release, or operator action.

## Operator follow-ups (this deployment)

- [ ] Move the release signing private key offline. The key currently lives at `/opt/thism/release.priv.b64` (mode 0600). GitHub Actions has a copy in encrypted secrets, but you should keep a second copy on a hardware token or encrypted USB and then `shred -u /opt/thism/release.priv.b64`. Without this, a server compromise compromises the signing key too.

## Outstanding High-severity items

No open high-severity code items remain from this review.

## Medium-severity backlog

- [x] Login failure messages distinguish "user not found" from "wrong password" — collapse to a generic `invalidCredentials`.
- [ ] Session lifetime is 30 days; shorten the cookie TTL and add a refresh-token flow (or session blacklist on rotation).
- [ ] Argon2id parameters are below OWASP recommendation; raise iterations and document the trade-off.
- [ ] Latency monitor HTTP probe target lacks SSRF protection — add a denylist for loopback, private RFC1918, link-local, and cloud-metadata addresses.
- [ ] Identifier-concatenated SQL in `internal/store/store.go:339` (`PRAGMA table_info(<table>)`) and `:363` (`ALTER TABLE ... ADD COLUMN ...`) — gate on a whitelist of identifiers used by migrations.
- [ ] Agent token rotation: tokens currently never expire. Add a TTL and a graceful-rotation flow.
- [ ] Agent WebSocket Origin check allows empty `Origin` — fine for non-browser clients but should be a separate code path from the dashboard upgrade.
- [ ] Dockerfile pins floating tags (`node:20-alpine`, `golang:1.24-alpine`, `alpine:3.19`). Pin `@sha256:...` digests and upgrade Alpine.
- [ ] `err.Error()` from internal errors is returned to API clients in ~50 places — wrap in generic error + correlation ID for logging.

## Low-severity / housekeeping

- [x] `/api/auth/login`-style endpoints should add a small constant-time delay regardless of result, to make timing-based username enumeration harder.
- [ ] Investigate adding `govulncheck` and `npm audit --omit=dev` as CI gates.
- [x] Add `.dockerignore` entries for `memory/`, `.agents/`, `.claude/`, `.worktrees/`, etc. to ensure no local state ever leaks into a container build.

## Already shipped

For history, the following audit findings were resolved in v0.6.0 – v0.6.2. See [CHANGELOG.md](../CHANGELOG.md) for the actual changes.

- Login endpoint now has per-IP and per-username throttling with fixed failure delay and short lockout.
- Cookie-authenticated admin state changes now require a double-submit CSRF token.
- Global security response headers are mounted by the API router.
- WebSocket handlers now set read limits plus ping/pong/read/write deadlines.
- JSON API request bodies are capped with `http.MaxBytesReader`, including unauthenticated login.
- Agent WebSocket authentication now prefers `Authorization: Bearer` over the query-string token fallback.
- Agent self-update downloads now refuse HTTP redirects.
- Agent HTTP self-update and WSS connections can pin the server certificate SPKI SHA-256 at build time via `SERVER_TLS_SPKI_SHA256`.
- `.dockerignore` excludes local runtime state, databases, secrets, dependency folders, and build output from Docker contexts.
- Removed hard-coded `thism2026` admin token from Makefile / dev systemd unit / CONTRIBUTING (v0.6.0).
- Ed25519-signed agent self-update with pinned public key at build time, `thism-sign` CLI, and signed agents shipped in the upstream GHCR image (v0.6.0 → v0.6.1).
- SQLite database (`thism.db` + `-wal` / `-shm` sidecars) chmod'd to 0600 on every open (v0.6.0).
- CI signing pipeline gated on `THISM_RELEASE_PUBLIC_KEY` / `THISM_RELEASE_PRIVATE_KEY` repository secrets; release fails fast when secrets are missing (v0.6.1).
- In-product install script switched from `--token` on `ExecStart=` to an `EnvironmentFile` (v0.6.1).
- Both server and agent read credentials from `THISM_*` env vars and the bundled units invoke the binaries with no flags — `/proc/<pid>/cmdline` carries no secrets (v0.6.2).
