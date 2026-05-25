# 安全工作待办

[English](security-roadmap.md) | 简体中文

记录 2026 年安全审查后还未完成的加固项。条目要么需要代码改动，要么需要发版，要么需要运维动作。

## 部署运维（本环境）

- [ ] 把发布签名私钥转移到离线介质。私钥当前位于 `/opt/thism/release.priv.b64`（0600 权限）。GitHub Actions 已经在加密 secrets 里存了一份，建议你在硬件密钥或加密 U 盘上再保存一份后执行 `shred -u /opt/thism/release.priv.b64`。否则服务器被攻陷 = 签名私钥被攻陷。

## High 级未完成项

本轮审查已无 High 级未完成代码项。

## Medium 级 backlog

- [x] 登录失败信息能区分"用户不存在"和"密码错误" —— 改为统一返回 `invalidCredentials`。
- [ ] Session TTL 是 30 天，应缩短并加 refresh token 流程（或在轮换时把旧 session 拉黑）。
- [ ] Argon2id 参数低于 OWASP 推荐，提高迭代次数并记录权衡。
- [ ] Latency monitor HTTP 探测目标缺 SSRF 防护，加 denylist：loopback、RFC1918 私有段、link-local、云元数据地址。
- [ ] `internal/store/store.go:339` 的 `PRAGMA table_info(<table>)` 和 `:363` 的 `ALTER TABLE ... ADD COLUMN ...` 用了字符串拼接 SQL identifier，需用白名单约束。
- [ ] Agent token 永不过期。加 TTL + 优雅轮换流程。
- [ ] Agent WebSocket Origin 校验允许空 Origin，非浏览器 client 这样设是合理的，但应该和 dashboard 升级路径走不同分支。
- [ ] Dockerfile 用浮动 tag（`node:20-alpine`、`golang:1.24-alpine`、`alpine:3.19`），应 pin `@sha256:...` digest 并升级 Alpine。
- [ ] `err.Error()` 直接进 API 响应大约 50 处，应该包成 generic error + correlation ID（详细错误进日志）。

## Low 级 / 杂项

- [x] `/api/auth/login` 类端点无论成功失败都加固定常量时间延迟，让用户名枚举的时序攻击更难。
- [ ] CI 加 `govulncheck` + `npm audit --omit=dev` 作为门禁。
- [x] `.dockerignore` 补充 `memory/`、`.agents/`、`.claude/`、`.worktrees/` 等本地状态目录，避免泄进镜像。

## 已完成（历史）

以下审查发现已在 v0.6.0 – v0.6.2 解决。具体改动见 [CHANGELOG.md](../CHANGELOG.md)。

- 登录端点已加按 IP / 用户名的失败限流、固定失败延迟和短期锁定。
- 管理员 cookie 会话的状态变更已加 double-submit CSRF token。
- API router 已全局挂载安全响应头。
- WebSocket handler 已设置 read limit、ping/pong/read/write deadline。
- JSON API 请求体已用 `http.MaxBytesReader` 限制大小，包含未鉴权登录接口。
- Agent WebSocket 认证现在优先使用 `Authorization: Bearer`，query token 仅保留兼容回退。
- Agent 自更新下载现在拒绝 HTTP redirect。
- Agent HTTP 自更新与 WSS 连接支持通过 `SERVER_TLS_SPKI_SHA256` 在编译期固化 server 证书 SPKI SHA-256 pin。
- `.dockerignore` 已排除本地运行状态、数据库、密钥、依赖目录和构建产物，避免进入 Docker context。
- 移除 Makefile / dev systemd unit / CONTRIBUTING 中硬编码的 `thism2026` admin token（v0.6.0）。
- Agent 自更新二进制加 Ed25519 签名校验，公钥编译时固化，`thism-sign` CLI，上游 GHCR 镜像携带签名 agent（v0.6.0 → v0.6.1）。
- 每次打开数据库时把 `thism.db`（含 `-wal`/`-shm` 兄弟文件）chmod 0600（v0.6.0）。
- CI 签名流水线依赖 `THISM_RELEASE_PUBLIC_KEY` / `THISM_RELEASE_PRIVATE_KEY` 仓库 secret；缺 secret 时 release 直接失败（v0.6.1）。
- 面板内置 install 脚本从 `--token` on `ExecStart=` 改为 `EnvironmentFile`（v0.6.1）。
- Server 与 agent 都改为从 `THISM_*` 环境变量读凭据，systemd unit 调用二进制不传任何 flag —— `/proc/<pid>/cmdline` 不再含敏感数据（v0.6.2）。
