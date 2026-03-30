# Release Flow

[简体中文](release.zh-CN.md)

Formal releases are tag-driven only:

1. Prepare and merge release-ready changes to `main`.
2. Create a semantic version tag locally, for example `v1.4.0`.
3. Push the tag with `git push origin v1.4.0`.

The release workflow runs only for pushed `v*` tags and publishes:

- `ghcr.io/thism-dev/thism:v1.4.0`
- `ghcr.io/thism-dev/thism:sha-<shortsha>`
- `ghcr.io/thism-dev/thism:latest`

Build metadata is injected into the binaries during Docker builds:

- `THISM_VERSION` from the git tag
- `THISM_COMMIT` from the full commit SHA
- `THISM_BUILD_TIME` in UTC RFC3339 format

Dev builds vs formal releases:

- Dev builds such as local `make build` or ad-hoc Docker builds are for testing and may report non-release version metadata.
- Formal releases are immutable, semver-tagged builds generated only by the tag-triggered release workflow.
