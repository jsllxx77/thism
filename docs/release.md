# Release Flow

[简体中文](release.zh-CN.md)

Formal releases are tag-driven only:

1. Prepare and merge release-ready changes to `main`.
2. Create a semantic version tag locally, for example `v1.4.0`.
3. Push the tag with `git push origin v1.4.0`.

The release workflow runs only for pushed `v*` tags and publishes:

- `ghcr.io/jsllxx77/thism:v1.4.0`
- `ghcr.io/jsllxx77/thism:sha-<shortsha>`
- `ghcr.io/jsllxx77/thism:latest`

Build metadata is injected into the binaries during Docker builds:

- `THISM_VERSION` from the git tag
- `THISM_COMMIT` from the full commit SHA
- `THISM_BUILD_TIME` in UTC RFC3339 format

Dev builds vs formal releases:

- Dev builds such as local `make build` or ad-hoc Docker builds are for testing and may report non-release version metadata.
- Formal releases are immutable, semver-tagged builds generated only by the tag-triggered release workflow.

## Signed Agent Updates

Starting with v0.6.0, the agent self-update channel requires every replacement binary to carry a valid Ed25519 signature in addition to the SHA-256 hash. Agents that were built without a pinned public key refuse to apply any update — they fail closed.

The upstream release workflow (`.github/workflows/release.yml`) signs every published agent binary automatically when the repository has the following secrets configured:

| Secret | Value |
|--------|-------|
| `THISM_RELEASE_PUBLIC_KEY` | base64 Ed25519 public key (the bytes from `release.pub.b64`) |
| `THISM_RELEASE_PRIVATE_KEY` | base64 Ed25519 private key (the bytes from `release.priv.b64`) |

With both secrets configured, every `v*` tag push produces a Docker image whose `dist/` already contains signed agents plus matching `.sig` sidecar files, and the same files are attached to the GitHub Release as assets. Downstream users running `ghcr.io/jsllxx77/thism:latest` get signed self-updates without any additional setup on their side.

If the repository runs the release workflow **without** the secrets configured, the workflow fails fast — it refuses to publish an unsigned release rather than silently shipping agents that cannot self-update.

### Forking the project

If you fork the project and publish your own GHCR image, run the steps below; the project image (`ghcr.io/jsllxx77/thism`) ships with this project's pinned key, and your fork cannot sign updates for that image without rebuilding the agent.

### One-time keypair setup

Generate the keypair **once**, on a trusted offline workstation. The private key signs every future agent binary; treat it like a code-signing certificate.

```bash
make release-keygen
# wrote public key  -> release.pub.b64
# wrote private key -> release.priv.b64 (mode 0600, keep offline)
```

After generation:

- Copy `release.priv.b64` to offline storage (hardware token, encrypted USB, password manager attachment).
- Shred the on-disk copy from the build host: `shred -u release.priv.b64`.
- Commit `release.pub.b64` to your fork if you want the public key tracked, or keep it alongside your release notes.

The `release.priv.b64` and `release.pub.b64` filenames are already in `.gitignore` for this repository.

### Building signed agents

Compile the agent with the matching public key baked in via ldflags. The Makefile reads `RELEASE_PUBLIC_KEY` and routes it into the verifier:

```bash
RELEASE_PUBLIC_KEY="$(cat release.pub.b64)" make build-agent-all
```

This produces `dist/thism-agent-linux-{amd64,arm64}` along with `*.version` sidecars.

### Signing the dist artifacts

Bring the private key online (or use a host with the key file mounted only for this step) and run:

```bash
make sign-dist
# expects release.priv.b64 in cwd, or THISM_RELEASE_PRIVATE_KEY env var
# writes dist/thism-agent-linux-amd64.sig and dist/thism-agent-linux-arm64.sig
```

The `.sig` files are hex-encoded Ed25519 signatures. The server's `/api/agent-release` manifest endpoint reads them automatically and exposes the value in the manifest JSON; agents fetch the manifest, verify, and only then write the new binary to disk.

If a `.sig` file is missing, the manifest returns an empty `signature` field and every up-to-date agent (one that has a pinned key) refuses the update.

### Manual update jobs from the API

The `/api/agent-updates` and `/api/agent-update-jobs` endpoints now require a `signature` field alongside `download_url`, `target_version`, and `sha256`. Requests missing the signature receive HTTP 400.

### Key rotation

Ed25519 public keys are pinned into the agent at build time, so rotation requires you to:

1. Generate a new keypair (`make release-keygen` with new file names).
2. Rebuild the agent with the new `RELEASE_PUBLIC_KEY` and ship it to every node through the **current** signing key (one final signed update under the old key carries the new key into the field).
3. Once every node reports the rotated build, retire the old private key.

There is intentionally no in-band rotation: an attacker that compromises the server should not be able to swap the trusted public key remotely.

### Failure modes you can rely on

| Condition | Agent behavior |
|-----------|----------------|
| No public key pinned at build time | Refuse every update (`ErrNoPublicKey`) |
| Signature field empty or absent | Refuse update (`ErrMissingSignature`) |
| Signature hex malformed or wrong length | Refuse update (`ErrInvalidSignature`) |
| Signature does not verify under pinned key | Refuse update (`ErrInvalidSignature`) |
| SHA-256 mismatch | Refuse update before signature is even checked |

All failures return the error to the server via the update job target status; the agent process keeps running on its current binary.
