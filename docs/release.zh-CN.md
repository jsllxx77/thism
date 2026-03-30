# 发布流程

[English](release.md) | 简体中文

正式发布仅通过手动语义化标签触发：

1. 准备并合并可发布变更到 `main`。
2. 在本地创建语义化版本标签，例如 `v1.4.0`。
3. 执行 `git push origin v1.4.0` 推送标签。

Release 工作流只会在推送 `v*` 标签时执行，并发布：

- `ghcr.io/thism-dev/thism:v1.4.0`
- `ghcr.io/thism-dev/thism:sha-<shortsha>`
- `ghcr.io/thism-dev/thism:latest`

Docker 构建时会把统一构建元数据注入到二进制：

- `THISM_VERSION` 来自 git 标签
- `THISM_COMMIT` 是完整 commit SHA
- `THISM_BUILD_TIME` 为 UTC RFC3339 时间戳

开发构建 vs 正式发布：

- 开发构建，例如本地 `make build` 或临时 Docker 构建，仅用于测试，版本元数据可能不是正式发布值。
- 正式发布是仅由标签触发工作流生成的不可变 semver 构建产物。
