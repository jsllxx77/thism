# Unified Versioning Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Unify ThisM versioning so server, agent, release artifacts, and upgrade metadata all use one build-injected version model based on manual semver tags for releases and Git-describe output for development builds.

**Architecture:** Introduce a shared Go version package populated via linker flags, wire all binaries and release metadata to it, and update workflows so Git tags drive formal releases. Replace checksum-prefix pseudo-versions in the agent release manifest and install flow with the injected semantic version while keeping SHA256 verification intact.

**Tech Stack:** Go, React/Vite, Make, Docker, GitHub Actions, shell.

---

### Task 1: Capture the current versioning baseline

**Files:**
- Read: `cmd/agent/main.go`
- Read: `internal/collector/collector.go`
- Read: `internal/api/api.go`
- Read: `internal/api/agent_release_test.go`
- Read: `Makefile`
- Read: `.github/workflows/release.yml`

**Step 1: Run the current agent release test**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/api -run 'TestAgentReleaseManifest|TestInstallScriptUsesTempBinarySwap'`

Expected: PASS on the current checksum-prefix behavior.

**Step 2: Record current version sources**

Confirm from code that:

- `cmd/agent/main.go` hard-codes `agentVersion = "dev"`
- `internal/api/api.go` derives `target_version` from SHA256 prefix
- `Makefile` does not inject linker version metadata
- `.github/workflows/release.yml` publishes images but does not inject a unified runtime version

### Task 2: Add the shared version package

**Files:**
- Create: `internal/version/version.go`
- Create: `internal/version/version_test.go`

**Step 1: Write the failing test**

Add tests covering:

- default `Version == "dev"`
- default `Commit == ""`
- default `BuildTime == ""`

**Step 2: Run the test to confirm red/green baseline**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/version`

Expected before implementation: FAIL because the package does not exist.

**Step 3: Add minimal implementation**

Create a package exposing:

- `var Version = "dev"`
- `var Commit = ""`
- `var BuildTime = ""`

**Step 4: Re-run the test**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/version`

Expected: PASS.

### Task 3: Inject build metadata in local builds

**Files:**
- Modify: `Makefile`

**Step 1: Add computed build metadata variables**

Add Make variables that evaluate:

- `VERSION` from `git describe --tags --dirty --always`
- `COMMIT` from `git rev-parse --short HEAD`
- `BUILD_TIME` from UTC `date`
- `LDFLAGS` with `-X github.com/thism-dev/thism/internal/version.Version=...` and peers

**Step 2: Apply the linker flags to every Go build**

Update:

- `build-server`
- `build-agent`
- `build-agent-all`
- `dev-server`
- `dev-rebuild`

so each Go invocation passes `-ldflags "$(LDFLAGS)"`.

**Step 3: Run a focused build check**

Run:

`make build-agent`

Expected: PASS with no Make syntax errors.

### Task 4: Wire server and agent binaries to the shared version source

**Files:**
- Modify: `cmd/agent/main.go`
- Modify: `cmd/server/main.go`
- Modify: `internal/collector/collector.go`
- Modify: `internal/collector/self_update_test.go`
- Modify: `internal/collector/auto_update_test.go`

**Step 1: Write failing tests or update existing ones**

Add / update tests so they expect the collector and self-update flow to use a caller-provided injected version value rather than relying on a handwritten `"dev"` constant.

**Step 2: Run the focused collector tests and confirm failure**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/collector -run 'Test.*Update|Test.*Version'`

Expected: FAIL due to old version wiring assumptions.

**Step 3: Replace hard-coded version literals**

- Remove `var agentVersion = "dev"` from `cmd/agent/main.go`
- Read from `internal/version.Version`
- Ensure the server can also expose the same shared version metadata later without duplicating constants

**Step 4: Re-run the focused collector tests**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/collector -run 'Test.*Update|Test.*Version'`

Expected: PASS.

### Task 5: Switch the agent release manifest and install script to real versions

**Files:**
- Modify: `internal/api/api.go`
- Modify: `internal/api/agent_release_test.go`
- Modify: `internal/api/api_test.go`

**Step 1: Write the failing tests**

Update tests so they expect:

- `/api/agent-release` to return the injected version string as `target_version`
- the install script to persist that version string to `VERSION_FILE`
- no checksum-prefix-derived version naming

**Step 2: Run the focused API tests and confirm failure**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/api -run 'TestAgentReleaseManifest|TestInstallScriptUsesTempBinarySwap'`

Expected: FAIL because current behavior still uses SHA256 prefixes.

**Step 3: Implement the minimal change**

- In `handleAgentRelease`, set `target_version` from `internal/version.Version`
- Keep `sha256` generation unchanged
- In the generated install script, write the same `TARGET_VERSION` string to `.thism-agent.version`

**Step 4: Re-run the focused API tests**

Run:

`GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/api -run 'TestAgentReleaseManifest|TestInstallScriptUsesTempBinarySwap'`

Expected: PASS.

### Task 6: Expose version metadata to the frontend and display it

**Files:**
- Modify: `internal/api/api.go`
- Modify: `internal/api/api_test.go`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/pages/Settings.tsx`
- Create: `frontend/src/pages/settings-version.test.tsx`

**Step 1: Write the failing API test**

Add a backend test for a lightweight endpoint such as `GET /api/meta/version` returning:

- `version`
- `commit`
- `build_time`

**Step 2: Write the failing frontend test**

Add a Settings page test expecting a small version section to render the backend-provided version string.

**Step 3: Run the focused tests and confirm failure**

Run:

- `GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/api -run 'Test.*Version'`
- `cd frontend && npm test -- settings-version.test.tsx`

Expected: FAIL because neither the endpoint nor the UI exists yet.

**Step 4: Implement minimal API and UI wiring**

- Add the backend route and handler
- Add a typed client method in `frontend/src/lib/api.ts`
- Fetch and render the version in `frontend/src/pages/Settings.tsx`

**Step 5: Re-run the focused tests**

Run:

- `GOCACHE=/tmp/go-build /usr/local/go/bin/go test ./internal/api -run 'Test.*Version'`
- `cd frontend && npm test -- settings-version.test.tsx`

Expected: PASS.

### Task 7: Inject version metadata in Docker and release builds

**Files:**
- Modify: `Dockerfile`
- Modify: `.github/workflows/release.yml`

**Step 1: Add Docker build args**

Add:

- `ARG THISM_VERSION`
- `ARG THISM_COMMIT`
- `ARG THISM_BUILD_TIME`

and pass them into Go builds via `-ldflags`.

**Step 2: Restrict release publishing to manual semver tags**

Adjust the release workflow so formal publishing is driven by pushes to `v*` tags.

**Step 3: Publish semver-oriented image tags**

Ensure the workflow emits:

- semver tag, e.g. `v0.1.1`
- `sha-<shortsha>`
- `latest`

and injects tag metadata into Docker build args.

**Step 4: Validate workflow syntax**

Run:

`python3 - <<'PY'\nimport yaml, pathlib\nfor path in [pathlib.Path('.github/workflows/release.yml')]:\n    yaml.safe_load(path.read_text())\nprint('ok')\nPY`

Expected: `ok`

### Task 8: Update release documentation

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`

**Step 1: Document the version model**

Add concise release notes explaining:

- formal releases require manual `vX.Y.Z` tags
- development builds show Git-describe versions
- release images inherit the tag version
- agent upgrades now use formal version strings

**Step 2: Add the release command sequence**

Document a minimal release flow such as:

```bash
git checkout main
git pull
git tag v0.1.1
git push origin v0.1.1
```

### Task 9: Final verification and restart

**Step 1: Run backend tests**

Run:

`make test`

Expected: PASS.

**Step 2: Run frontend checks**

Run:

- `cd frontend && npm run lint`
- `cd frontend && npm test`

Expected: PASS.

**Step 3: Mandatory embedded-asset rebuild**

Run:

`make dev-restart TOKEN=thism2026 PORT=12026`

Expected: PASS and local service restart.

**Step 4: Manual release smoke checklist**

Verify:

- local `make build-agent` embeds a Git-describe version
- a future `vX.Y.Z` tag build would inject the pure semver value
- Settings renders version metadata
- `/api/agent-release` returns a semantic `target_version`
- `.thism-agent.version` is documented and expected to contain a real version string
