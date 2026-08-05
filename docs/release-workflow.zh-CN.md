# 发布工作流（正式 tag 发布）

[English](release-workflow.md)

这是完整的正式发布工作流（路径 B）——唯一能产出"可被整个集群自动更新"的签名 agent 构建的路径。与 v0.6.33 实操流程一致：提交代码 → 双语 release notes → 打 tag → GitHub Actions 构建/签名/发布 → 本地拉取镜像 → agent 自动更新。

## 前置条件

- 本地仓库 `/opt/thism`，已配置 GitHub 远端。
- `gh` CLI 已认证且带 `workflow` 权限（`gh auth status`）。
- 仓库 Secrets 已配置（缺少时 release workflow 会直接失败）：
  - `THISM_RELEASE_PUBLIC_KEY` — base64 Ed25519 公钥（`release.pub.b64` 内容）
  - `THISM_RELEASE_PRIVATE_KEY` — base64 Ed25519 私钥（`release.priv.b64` 内容）
- 本地 Docker 部署目录 `/opt/thism-deploy`（compose 项目名 `thism-deploy`）。

## 1.（可选）本地构建验证

提交前先用本地构建验证改动。本地构建的 agent 二进制**没有官方签名，agent 会 fail-closed 拒绝自动更新**，所以随意测试不会影响线上集群：

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
# ... 验证完成后，切回官方镜像：
rm docker-compose.override.yml
docker compose pull && docker compose up -d
```

## 2. 提交并推送代码

```bash
cd /opt/thism
git add -A
git commit -m "描述本次改动"
git push origin main
```

同步更新 `CHANGELOG.md`：把改动移入 `## [X.Y.Z] - <日期>` 小节（或打 tag 前保留在 `[Unreleased]`）。

## 3. 编写双语 release notes

release workflow 的**硬性要求**：`docs/releases/<tag>.md` 必须存在，且必须同时包含 `## English` 和 `## 中文` 两个章节，否则直接失败。

```bash
cd /opt/thism
cp docs/releases/TEMPLATE.md docs/releases/vX.Y.Z.md
```

按模板手写填写：

- `## English` — 简短的面向用户的内容摘要；如有升级注意事项（Upgrade note）也写在这里。
- `## 中文` — 真正的中文用户摘要（不要直接复制英文 commit 标题）。
- `### Assets` / `### 发布产物` 段落保持模板原文。

提交并推送：

```bash
git add docs/releases/vX.Y.Z.md
git commit -m "Add vX.Y.Z release notes"
git push origin main
```

## 4. 打 tag 并推送

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

`Release Container` workflow（`.github/workflows/release.yml`）随即自动执行：

1. 用仓库 Secrets 构建签名 agent 二进制（`make build-sign-tool`、`build-agent-all`、`./bin/thism-sign sign-dist -dir dist`）。
2. 校验签名产物。
3. 构建并推送多架构镜像（linux/amd64 + arm64）到 GHCR：
   - `ghcr.io/jsllxx77/thism:vX.Y.Z`
   - `ghcr.io/jsllxx77/thism:sha-<12位短SHA>`
   - `ghcr.io/jsllxx77/thism:latest`
4. 创建 GitHub Release：附带双语 notes 和签名 agent 二进制 + `.sig` + `.version` sidecar。

典型耗时约 13 分钟。

## 5. 监控构建

```bash
gh run list --workflow=release.yml --limit 1
gh run watch <run-id> --exit-status   # 阻塞直到完成；失败时返回非零
gh release view vX.Y.Z                # 确认 Release 和资产
```

## 6. 本地部署

```bash
cd /opt/thism-deploy
rm -f docker-compose.override.yml      # 如有本地镜像 override 先删掉
docker compose pull
docker compose up -d
docker compose ps
docker compose logs --tail 20
```

容器保持相同端口（12026），数据卷（`thism-deploy_thism-data`）和 secrets 原样保留，无需迁移数据。

## 7. 验证集群自动更新

Agent 会拉取服务端托管的 manifest 并在约 30 分钟内自动更新（或下次重连后不久）。用 API 验证：

```bash
curl -s -H "Authorization: Bearer $TOKEN" http://127.0.0.1:12026/api/nodes \
  | python3 -m json.tool | grep -E '"name"|"agent_version"'
```

所有节点应显示新版本。若某节点卡住，重启该节点 agent 服务，或在面板手动下发更新任务。

## 8. 回滚

发布异常时：

1. 在 `/opt/thism-deploy/.env` 固定上一个镜像版本：
   `THISM_IMAGE=ghcr.io/jsllxx77/thism:v0.6.33`
2. `docker compose pull && docker compose up -d`
3. 数据库在命名卷里；重大升级前先备份：

```bash
docker run --rm -v thism-deploy_thism-data:/data -v /opt:/backup \
  alpine:3.24 cp /data/thism.db /backup/thism.db.bak-pre-X.Y.Z
```

## 版本号约定

- Semver tag（`v<主>.<次>.<补丁>`），只从 `main` 推送。
- workflow 只在 `v*` tag 上触发——release notes 提交到达 `main` 之前不要推 tag，否则 workflow 会快速失败。
