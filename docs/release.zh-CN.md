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

## Agent 更新签名（v0.6.0 起）

从 v0.6.0 开始，agent 自更新通道要求每个替换二进制都附带有效的 Ed25519 签名（与 SHA-256 校验和并存）。**没有内嵌公钥的 agent 会拒绝任何更新（fail closed）**。

上游 release 工作流（`.github/workflows/release.yml`）会在 repository 配置了下列 secret 后**自动**为每次发布签名：

| Secret 名 | 值 |
|----------|----|
| `THISM_RELEASE_PUBLIC_KEY` | base64 Ed25519 公钥（`release.pub.b64` 内容）|
| `THISM_RELEASE_PRIVATE_KEY` | base64 Ed25519 私钥（`release.priv.b64` 内容）|

配置好两个 secret 后，每次推送 `v*` tag 都会构建出 `dist/` 中已包含签名 agent 和 `.sig` 文件的 Docker 镜像，并把同一组文件作为 release 资产附加到 GitHub Release。下游用户拉 `ghcr.io/thism-dev/thism:latest` 即可获得带签名校验的自更新，无需额外配置。

若 release 工作流**未**配置这两个 secret，CI 会直接失败，避免静默地发布一份永远无法自更新的 agent。

### Fork 项目

如果你 fork 项目并发布自己的 GHCR 镜像，下面的步骤是必做的。官方公开镜像 `ghcr.io/thism-dev/thism` 出厂带上游项目自己的固定公钥，fork 无法在不重新构建 agent 的前提下为它签发更新。

### 一次性密钥生成

在**可信的离线工作机**上生成一次密钥对。私钥用于签发所有未来的 agent 二进制，请像对待代码签名证书一样保管。

```bash
make release-keygen
# wrote public key  -> release.pub.b64
# wrote private key -> release.priv.b64 (mode 0600, keep offline)
```

生成完后：

- 把 `release.priv.b64` 复制到离线介质（硬件密钥、加密 U 盘、密码管理器附件）。
- 在构建机上销毁本地副本：`shred -u release.priv.b64`。
- `release.pub.b64` 可以提交到你的 fork（如果想把公钥纳入版本控制），或与发布说明一起存档。

仓库的 `.gitignore` 已经默认排除 `release.priv.b64` 和 `release.pub.b64`。

### 构建带签名校验的 agent

通过 ldflags 把对应公钥嵌入 agent。Makefile 会读取 `RELEASE_PUBLIC_KEY` 并注入到验证器：

```bash
RELEASE_PUBLIC_KEY="$(cat release.pub.b64)" make build-agent-all
```

产物：`dist/thism-agent-linux-{amd64,arm64}` 以及对应的 `*.version` 文件。

### 签名 dist 产物

把私钥临时上线（或在一台仅在签名步骤挂载密钥文件的主机上），执行：

```bash
make sign-dist
# 默认从当前目录读取 release.priv.b64，或读取 THISM_RELEASE_PRIVATE_KEY 环境变量
# 产物：dist/thism-agent-linux-amd64.sig、dist/thism-agent-linux-arm64.sig
```

`.sig` 是十六进制编码的 Ed25519 签名。Server 的 `/api/agent-release` manifest 接口会自动读取并把签名值放进 JSON 响应；agent 拉到 manifest 后先校验签名，通过才会把新二进制落盘。

如果 `.sig` 文件缺失，manifest 返回空 `signature` 字段，所有已嵌入公钥的 agent 会拒绝该次更新。

### 手动从 API 推送更新

`/api/agent-updates` 和 `/api/agent-update-jobs` 现在要求请求体除 `download_url`、`target_version`、`sha256` 外，必须包含 `signature` 字段。缺失签名的请求返回 HTTP 400。

### 公钥轮换

Ed25519 公钥在 agent 构建时被烧入二进制，所以轮换流程是：

1. 生成新密钥对（重新执行 `make release-keygen`，换文件名）。
2. 用新的 `RELEASE_PUBLIC_KEY` 重新构建 agent，并通过**当前正在使用的私钥**对它签名后下发到所有节点（这一次签名的二进制把新公钥送到现场）。
3. 等所有节点都升级到新公钥的构建后，作废旧私钥。

这里**故意没有"在线轮换"**：server 被攻陷的攻击者不应该能远程换信任的公钥。

### 你可以依赖的失败模式

| 情况 | Agent 行为 |
|------|-----------|
| 构建时未嵌入公钥 | 拒绝任何更新（`ErrNoPublicKey`）|
| signature 字段为空或缺失 | 拒绝更新（`ErrMissingSignature`）|
| signature hex 格式错误或长度错误 | 拒绝更新（`ErrInvalidSignature`）|
| signature 在固定公钥下校验失败 | 拒绝更新（`ErrInvalidSignature`）|
| SHA-256 不匹配 | 在签名校验之前就拒绝 |

所有失败都通过 update job target status 上报给 server，agent 进程继续运行当前二进制。
