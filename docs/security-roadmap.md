# Security Roadmap

[简体中文](security-roadmap.zh-CN.md)

Tracks the remaining hardening items from the November 2026 security review. Items here either need a code change, a release, or operator action.

## Operator follow-ups (this deployment)

- [ ] Move the release signing private key offline. The key currently lives at `/opt/thism/release.priv.b64` (mode 0600). GitHub Actions has a copy in encrypted secrets, but you should keep a second copy on a hardware token or encrypted USB and then `shred -u /opt/thism/release.priv.b64`. Without this, a server compromise compromises the signing key too.

## Outstanding High-severity items

### Auth / web surface

- [ ] **Login endpoint rate limiting.** `handlePasswordLogin` accepts unlimited attempts with no IP- or username-keyed throttling. Add a per-IP and per-username token bucket plus a fixed failure delay (~250 ms) and N-strikes lockout. Files: `internal/api/api.go:1366-1397`.
- [ ] **CSRF token enforcement.** Cookie auth currently relies on `SameSite=Lax` alone. Add a double-submit CSRF token check on every state-changing route (POST/PUT/DELETE under `/api/`). Files: `internal/api/api.go` global router setup.
- [ ] **Global secure response headers.** No `X-Content-Type-Options`, `X-Frame-Options`/`frame-ancestors`, `Content-Security-Policy`, `Strict-Transport-Security`, or `Referrer-Policy` middleware is registered. Add a `secureHeaders` middleware mounted globally. Files: `internal/api/api.go` router setup; `cmd/server/main.go` HTTP server config.

### Resource exhaustion

- [ ] **WebSocket `SetReadLimit` + ping/pong deadlines.** Neither `handleAgentWS` nor `handleDashboardWS` calls `conn.SetReadLimit` or sets read/write deadlines. A misbehaving (or compromised) peer can send GB-scale frames or hold a half-open connection forever. Files: `internal/api/api.go` WebSocket handlers near lines 2615 and 2944.
- [ ] **HTTP body size limit (`MaxBytesReader`).** 11 handlers call `json.NewDecoder(r.Body).Decode(...)` without wrapping in `http.MaxBytesReader`, including the unauthenticated `/api/auth/login`. Add a global request body limit middleware. Files: search `json.NewDecoder` in `internal/api/api.go`.

### Agent ↔ server protocol

- [ ] **Agent token in WebSocket query string.** `handleAgentWS` reads the node token from `?token=` in the URL, which proxies and access logs commonly retain. Promote the `Authorization: Bearer` fallback (already present) to the primary path and deprecate the query-string variant. Files: `internal/api/api.go:2615-2641`.
- [ ] **Agent self-update redirect bypass.** The agent's `http.Client` has no custom `CheckRedirect`, so a hostile (or compromised) download URL can chain redirects to an arbitrary host before `validateSelfUpdateSource` would catch it on the original request. Set `CheckRedirect: http.ErrUseLastResponse`. Files: `internal/collector/collector.go:78` and the download path around line 982.
- [ ] **Agent TLS certificate pinning.** A valid CA-signed certificate from any CA can MITM the agent. Pin the server's SPKI hash at build time and verify via a custom `DialContext`. Files: `internal/collector/collector.go` HTTP/WebSocket client setup.

## Medium-severity backlog

- [ ] Login failure messages distinguish "user not found" from "wrong password" — collapse to a generic `invalidCredentials`.
- [ ] Session lifetime is 30 days; shorten the cookie TTL and add a refresh-token flow (or session blacklist on rotation).
- [ ] Argon2id parameters are below OWASP recommendation; raise iterations and document the trade-off.
- [ ] Latency monitor HTTP probe target lacks SSRF protection — add a denylist for loopback, private RFC1918, link-local, and cloud-metadata addresses.
- [ ] Identifier-concatenated SQL in `internal/store/store.go:339` (`PRAGMA table_info(<table>)`) and `:363` (`ALTER TABLE ... ADD COLUMN ...`) — gate on a whitelist of identifiers used by migrations.
- [ ] Agent token rotation: tokens currently never expire. Add a TTL and a graceful-rotation flow.
- [ ] Agent WebSocket Origin check allows empty `Origin` — fine for non-browser clients but should be a separate code path from the dashboard upgrade.
- [ ] Dockerfile pins floating tags (`node:20-alpine`, `golang:1.24-alpine`, `alpine:3.19`). Pin `@sha256:...` digests and upgrade Alpine.
- [ ] `err.Error()` from internal errors is returned to API clients in ~50 places — wrap in generic error + correlation ID for logging.

## Low-severity / housekeeping

- [ ] `/api/auth/login`-style endpoints should add a small constant-time delay regardless of result, to make timing-based username enumeration harder.
- [ ] Investigate adding `govulncheck` and `npm audit --omit=dev` as CI gates.
- [ ] Add `.dockerignore` entries for `memory/`, `.agents/`, `.claude/`, `.worktrees/`, etc. to ensure no local state ever leaks into a container build.

## Already shipped

For history, the following audit findings were resolved in v0.6.0 – v0.6.2. See [CHANGELOG.md](../CHANGELOG.md) for the actual changes.

- Removed hard-coded `thism2026` admin token from Makefile / dev systemd unit / CONTRIBUTING (v0.6.0).
- Ed25519-signed agent self-update with pinned public key at build time, `thism-sign` CLI, and signed agents shipped in the upstream GHCR image (v0.6.0 → v0.6.1).
- SQLite database (`thism.db` + `-wal` / `-shm` sidecars) chmod'd to 0600 on every open (v0.6.0).
- CI signing pipeline gated on `THISM_RELEASE_PUBLIC_KEY` / `THISM_RELEASE_PRIVATE_KEY` repository secrets; release fails fast when secrets are missing (v0.6.1).
- In-product install script switched from `--token` on `ExecStart=` to an `EnvironmentFile` (v0.6.1).
- Both server and agent read credentials from `THISM_*` env vars and the bundled units invoke the binaries with no flags — `/proc/<pid>/cmdline` carries no secrets (v0.6.2).
