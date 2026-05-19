# 安全策略

[English](SECURITY.md) | 简体中文

## 受支持范围

欢迎提交以下范围的安全问题：

- Go 服务端
- agent
- 认证与授权流程
- 安装器与升级路径
- 本仓库中的 Docker 部署资产

## 报告漏洞

对于尚未公开的漏洞，请**不要**提交公开 GitHub Issue。

如可用，请通过以下任一私密渠道报告：

- GitHub Security Advisories / 私密漏洞报告
- 仓库所有者或组织主页上列出的私下联系方式

提交报告时，请尽量包含：

- 受影响版本或 commit
- 部署方式
- 复现步骤
- 影响评估
- 安全复现所需的 PoC 材料

## 响应目标

- 尽快确认有效报告
- 复现并评估影响
- 在确认问题后准备并发布修复
- 在合适且报告者愿意的情况下给予致谢

## 更新完整性

Agent 自更新通道是高价值攻击面——任何能下发任意更新的 server 都等价于全部被监控节点的远程代码执行。ThisM 通过以下手段防护：

- **Ed25519 签名，fail-closed**。构建时嵌入发布公钥的 agent 在替换二进制时**强制**校验 Ed25519 签名（与 SHA-256 校验并存）。未嵌入公钥的 agent 拒绝任何更新。公钥在编译期通过 ldflags 烧入，无法远程轮换；轮换流程需要用**当前**私钥签发一次"携带新公钥"的更新，再退役旧私钥。
- **私钥离线保管**。签名私钥**不**会在 server 上落盘。server 只持有 SHA-256、下载 URL、目标版本号、十六进制签名值——可下发更新但无法伪造签名。
- **Server 端验证**。`/api/agent-updates`、`/api/agent-update-jobs`、`/api/agent-release` 接口要求 `signature` 字段非空，否则返回 HTTP 400。Manifest 接口自动从磁盘读取 `<binary>.sig` sidecar 文件并放进响应。
- **下载源校验**。Agent 在下载前会校验下载 URL 是否指向已配置的 server host，避免被引流到第三方主机。

发布密钥生成、构建带签名的 agent、签名 dist 产物、密钥轮换等运维流程见 [发布流程](docs/release.zh-CN.md#agent-更新签名-v060-起)。

## 部署加固提示

- **不要**把 admin token、admin password、node token 写进 systemd `ExecStart` 命令行。改用 `EnvironmentFile=` 引用一份 0600 文件（例如 `/etc/default/thism-server`、`/etc/default/thism-agent`），`deploy/` 下的模板已按此模式给出。
- SQLite 数据库（`thism.db` + `-wal` / `-shm`）包含 admin 密码哈希、集成 token。当前 server 构建在每次打开数据库时都会 chmod 0600；如果你手动复制过数据库文件，请确认权限一致。
- 任何不在"代码签名证书级别"保护下的仓库或备份**都不要**收纳 `release.priv.b64`。本仓库 `.gitignore` 已默认排除，请勿本地覆盖。
