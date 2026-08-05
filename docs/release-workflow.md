# Release Workflow (Official Tag Release)

[简体中文](release-workflow.zh-CN.md)

This is the end-to-end official release workflow (Path B) — the only path that
produces signed agent binaries that the fleet can self-update to. It is the
same flow used for v0.6.33: commit → bilingual release notes → tag → GitHub
Actions builds/signs/publishes → pull the image locally → agents auto-update.

## Prerequisites

- Local clone at `/opt/thism` with the GitHub remote configured.
- `gh` CLI authenticated with the `workflow` scope (`gh auth status`).
- Repository secrets configured (the release workflow fails fast without them):
  - `THISM_RELEASE_PUBLIC_KEY` — base64 Ed25519 public key (from `release.pub.b64`)
  - `THISM_RELEASE_PRIVATE_KEY` — base64 Ed25519 private key (from `release.priv.b64`)
- Local Docker deployment at `/opt/thism-deploy` (compose project `thism-deploy`).

## 1. Optional: local sanity check

Before committing, verify the change in a local build. Agents will **not**
auto-update from an unsigned local build (fail-closed), so this is safe:

```bash
cd /opt/thism
docker build -t thism:local --build-arg THISM_VERSION=vX.Y.Z-dev .

cd /opt/thism-deploy
cat > docker-compose.override.yml <<'EOF'
services:
  thism-server:
    image: thism:local
EOF
docker compose up -d
# ... test, then revert to the official image:
rm docker-compose.override.yml
docker compose pull && docker compose up -d
```

## 2. Commit and push the changes

```bash
cd /opt/thism
git add -A
git commit -m "Describe the change"
git push origin main
```

Also update `CHANGELOG.md`: move the change into a `## [X.Y.Z] - <date>`
section (or keep it under `[Unreleased]` until tagging).

## 3. Write bilingual release notes

The release workflow **fails fast** if `docs/releases/<tag>.md` is missing or
lacks both `## English` and `## 中文` sections.

```bash
cd /opt/thism
cp docs/releases/TEMPLATE.md docs/releases/vX.Y.Z.md
```

Fill in the template by hand:

- `## English` — short user-facing summary, upgrade notes if any.
- `## 中文` — a real Chinese user-facing summary (not translated commit titles).
- Keep the `### Assets` section as-is.

Commit and push:

```bash
git add docs/releases/vX.Y.Z.md
git commit -m "Add vX.Y.Z release notes"
git push origin main
```

## 4. Tag and push

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

The `Release Container` workflow (`.github/workflows/release.yml`) then runs
automatically and:

1. Builds signed agent binaries (`make build-sign-tool`, `build-agent-all`,
   `./bin/thism-sign sign-dist -dir dist`) using the repository secrets.
2. Verifies the signed dist artifacts.
3. Builds and pushes the multi-arch image (linux/amd64 + arm64) to GHCR:
   - `ghcr.io/jsllxx77/thism:vX.Y.Z`
   - `ghcr.io/jsllxx77/thism:sha-<12>` (commit short SHA)
   - `ghcr.io/jsllxx77/thism:latest`
4. Creates the GitHub Release with the bilingual notes and attaches the signed
   agent binaries + `.sig` + `.version` sidecars.

Typical duration: ~13 minutes.

## 5. Monitor the run

```bash
gh run list --workflow=release.yml --limit 1
gh run watch <run-id> --exit-status   # blocks until done; fails on error
gh release view vX.Y.Z                # confirm release + assets
```

## 6. Deploy locally

```bash
cd /opt/thism-deploy
rm -f docker-compose.override.yml      # in case a local-image override exists
docker compose pull
docker compose up -d
docker compose ps
docker compose logs --tail 20
```

The container runs as the same port (12026) and keeps the existing data
volume (`thism-deploy_thism-data`) and secrets — no data migration needed.

## 7. Verify the fleet auto-updates

Agents fetch the server-hosted manifest and self-update within ~30 minutes
(or shortly after their next reconnect). Verify via the API:

```bash
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:12026/api/nodes \
  | python3 -m json.tool | grep -E '"name"|"agent_version"'
```

All nodes should report the new `agent_version`. If a node is stuck, restart
its agent service or push an update job from the panel.

## 8. Rollback

If the release misbehaves:

1. Pin the previous image in `/opt/thism-deploy/.env`:
   `THISM_IMAGE=ghcr.io/jsllxx77/thism:v0.6.33`
2. `docker compose pull && docker compose up -d`
3. The DB is kept in the named volume; before major upgrades, back it up:

```bash
docker run --rm -v thism-deploy_thism-data:/data -v /opt:/backup \
  alpine:3.24 cp /data/thism.db /backup/thism.db.bak-pre-X.Y.Z
```

## Versioning

- Semver tags (`v<major>.<minor>.<patch>`), pushed only from `main`.
- The workflow only triggers on `v*` tags — do not push the tag before the
  release notes commit is on `main`, or the workflow will fail fast.
