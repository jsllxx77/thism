# Unified Versioning Design

## Goal

Unify ThisM server, agent, release artifacts, and upgrade metadata behind one version model so development builds show a descriptive Git-based version and formal releases show a manually-created semver tag such as `v0.1.1`.

## Context

- The repository currently has one formal Git tag: `v0.1.0`.
- `main` already contains additional unreleased commits beyond that tag.
- The agent currently defaults to a hard-coded runtime version of `"dev"`.
- The server-hosted agent release manifest currently uses the agent binary checksum prefix as `target_version`.
- Docker publishing is driven by GitHub Actions, but version identity is not injected consistently into Go binaries or surfaced to the UI.

## Non-Goals

This design does not introduce:

- Automatic semver bumping on each merge to `main`
- Release note generation
- Multi-channel release streams such as `beta` / `rc`
- Signed Git tags or signed binaries
- A requirement to tag every commit

## Chosen Approach

Use **manual semver tags for formal releases** and **Git describe for development builds**.

### Formal release model

- A formal release exists only when a maintainer manually creates a Git tag in the format `vMAJOR.MINOR.PATCH`.
- GitHub release publishing is triggered by pushing that tag.
- The tag string is the authoritative formal version for:
  - server binary
  - agent binary
  - Docker image tags
  - server-hosted agent release manifest
  - any version information shown in the UI

### Development build model

- Untagged builds on `main` or local branches are development builds.
- Their displayed version comes from `git describe --tags --dirty --always`.
- Example: `v0.1.0-11-gd00d980`
- If Git metadata is unavailable, the build falls back to `dev`.

## Why this approach

- It keeps release control explicit and predictable.
- It avoids meaningless semver bumps on every commit.
- It makes every runtime surface tell the same version story.
- It removes the current mismatch where the agent upgrade "version" is really a checksum prefix.
- It fits the current project scale and release cadence better than automatic bumping.

## Version Source of Truth

### Shared runtime package

Add a small shared Go package, `internal/version`, exposing:

- `Version`
- `Commit`
- `BuildTime`

The package ships with safe defaults:

- `Version = "dev"`
- `Commit = ""`
- `BuildTime = ""`

These values are injected at build time with `-ldflags -X ...`.

### Build-time injection rules

- Local builds use Git-derived values:
  - `Version = git describe --tags --dirty --always`
  - `Commit = git rev-parse --short HEAD`
  - `BuildTime = UTC timestamp`
- Tag builds in GitHub Actions inject:
  - `Version = github.ref_name`
  - `Commit = tag commit short SHA`
  - `BuildTime = workflow build timestamp`

## Runtime Behavior

### Server

- The server binary reports the injected version from `internal/version`.
- The server exposes version metadata to the frontend through a lightweight API endpoint such as `/api/meta/version`.

### Agent

- The agent binary reports the injected version from `internal/version`.
- The collector no longer starts from a handwritten `"dev"` literal in `cmd/agent/main.go`.
- The persisted `.thism-agent.version` file stores the injected semver / Git-describe version string instead of a checksum-derived pseudo-version.

### Agent release manifest

- `/api/agent-release` continues to return:
  - `download_url`
  - `sha256`
  - `check_interval_seconds`
- `target_version` changes from "checksum prefix" to the real injected agent version.
- Checksum verification remains mandatory and unchanged.

## Release and Packaging Flow

### GitHub Actions

- CI continues to validate code on pushes and pull requests.
- Release publishing is driven by pushing `v*` tags.
- The release workflow builds Docker images using the tag version as build metadata.

### Docker image tags

On tag push such as `v0.1.1`, publish:

- `ghcr.io/<repo>:v0.1.1`
- `ghcr.io/<repo>:sha-<shortsha>`
- `ghcr.io/<repo>:latest`

`latest` means "latest formal release", not "latest commit on main".

### Docker build metadata

The Dockerfile should accept build args for:

- `THISM_VERSION`
- `THISM_COMMIT`
- `THISM_BUILD_TIME`

Those values are forwarded into Go builds via `-ldflags`.

## UI and API Exposure

### API

Add a minimal endpoint returning shared version metadata, for example:

```json
{
  "version": "v0.1.1",
  "commit": "abc1234",
  "build_time": "2026-03-18T04:00:00Z"
}
```

### Frontend

- Surface version metadata in Settings or another stable admin-visible location.
- Development builds may show Git-describe values; formal releases show the tag.
- No UI feature should infer release status from checksum prefixes anymore.

## Compatibility

### Existing agents

- Older installed agents may continue reporting older checksum-style `reported_version` values until they are upgraded.
- New agents will report semver / Git-describe values.
- Update job bookkeeping must tolerate a mixed fleet during transition.

### Existing release semantics

- `v0.1.0` remains the current formal baseline.
- The first release after this change can be `v0.1.1` or another manually chosen semver tag.
- No automatic migration of old database rows is required for correctness.

## Testing Strategy

### Unit / integration

- Verify shared version package defaults and injected values.
- Verify agent release manifest returns the real version string.
- Verify install script persists the real version string to `.thism-agent.version`.
- Verify the collector reports injected version metadata.
- Verify the new version API endpoint response.

### Build verification

- Local `make build` / `make dev-restart` should inject Git-describe output.
- Tag-driven release builds should inject pure semver tag values.
- Docker image builds should preserve version metadata in both server and bundled agent binaries.

## Rollout

1. Add shared version package and wire Go binaries to it.
2. Update Makefile and Dockerfile to inject build metadata.
3. Update release workflow to inject tag metadata.
4. Switch agent release manifest and install script from checksum-prefix versioning to shared versioning.
5. Add version API and UI display.
6. Update README / release docs.
7. Create a new tag and validate end-to-end release behavior.

## Success Criteria

- A manually tagged release such as `v0.1.1` produces:
  - server version `v0.1.1`
  - agent version `v0.1.1`
  - release manifest `target_version = v0.1.1`
  - Docker image tags including `v0.1.1` and `latest`
- An untagged local build displays a Git-describe development version.
- No runtime component uses checksum prefixes as its human-facing version string.
