# 安全工作待办

[English](security-roadmap.md) | 简体中文

记录 2026 年安全审查后还未完成的加固项。条目要么需要代码改动，要么需要发版，要么需要运维动作。

## 部署运维（本环境）

- [ ] 把发布签名私钥转移到离线介质。私钥当前位于 `/opt/thism/release.priv.b64`（0600 权限）。GitHub Actions 已经在加密 secrets 里存了一份，建议你在硬件密钥或加密 U 盘上再保存一份后执行 `shred -u /opt/thism/release.priv.b64`。否则服务器被攻陷 = 签名私钥被攻陷。

## High 级未完成项

### 认证 / Web 接口

- [ ] **登录端点限速**。`handlePasswordLogin` 没有任何按 IP / 按用户名的限流，可无限尝试。需要加 token bucket + 固定延迟（~250 ms）+ N 次失败锁定。文件：`internal/api/api.go:1366-1397`。
- [ ] **CSRF token 校验**。当前仅依赖 `SameSite=Lax`，所有 `/api/` 下的 POST/PUT/DELETE 路由都该加 double-submit CSRF token 校验。文件：`internal/api/api.go` 路由部分。
- [ ] **全局安全响应头**。未注册 `X-Content-Type-Options`、`X-Frame-Options`/`frame-ancestors`、`Content-Security-Policy`、`Strict-Transport-Security`、`Referrer-Policy` 等中间件。新增 `secureHeaders` middleware 全局挂上。文件：`internal/api/api.go` 路由 + `cmd/server/main.go` HTTP server 配置。

### 资源耗尽

- [ ] **WebSocket `SetReadLimit` + ping/pong deadlines**。`handleAgentWS` 和 `handleDashboardWS` 都没设 `conn.SetReadLimit`，也没读写 deadline。一个行为异常（或被攻陷）的对端可以发 GB 级帧或者长时间挂半开连接。文件：`internal/api/api.go` 第 2615 行附近和 2944 行附近。
- [ ] **HTTP body 大小限制（`MaxBytesReader`）**。11 个 handler 用 `json.NewDecoder(r.Body).Decode(...)` 而没有 `http.MaxBytesReader` 包裹，**含未鉴权的 `/api/auth/login`**。加一个全局请求体大小限制 middleware。文件：在 `internal/api/api.go` 搜 `json.NewDecoder`。

### Agent ↔ Server 协议

- [ ] **Agent token 在 WebSocket query string 里**。`handleAgentWS` 从 `?token=` 取 node token，反代和访问日志通常会保留 URL。把已经存在的 `Authorization: Bearer` 回退路径提升为主路径，弃用 query string 模式。文件：`internal/api/api.go:2615-2641`。
- [ ] **Agent 自更新 redirect 绕过**。Agent 的 `http.Client` 没设自定义 `CheckRedirect`，恶意（或被攻陷）的下载 URL 可以通过链式跳转到任意主机，绕过 `validateSelfUpdateSource` 的来源校验。设置 `CheckRedirect: http.ErrUseLastResponse`。文件：`internal/collector/collector.go:78` 和下载逻辑（约 982 行）。
- [ ] **Agent TLS 证书 pinning**。任何受信 CA 签发的证书都能 MITM agent。在编译时固化 server SPKI 哈希，通过自定义 `DialContext` 校验。文件：`internal/collector/collector.go` HTTP/WebSocket client 初始化处。

## Medium 级 backlog

- [ ] 登录失败信息能区分"用户不存在"和"密码错误" —— 改为统一返回 `invalidCredentials`。
- [ ] Session TTL 是 30 天，应缩短并加 refresh token 流程（或在轮换时把旧 session 拉黑）。
- [ ] Argon2id 参数低于 OWASP 推荐，提高迭代次数并记录权衡。
- [ ] Latency monitor HTTP 探测目标缺 SSRF 防护，加 denylist：loopback、RFC1918 私有段、link-local、云元数据地址。
- [ ] `internal/store/store.go:339` 的 `PRAGMA table_info(<table>)` 和 `:363` 的 `ALTER TABLE ... ADD COLUMN ...` 用了字符串拼接 SQL identifier，需用白名单约束。
- [ ] Agent token 永不过期。加 TTL + 优雅轮换流程。
- [ ] Agent WebSocket Origin 校验允许空 Origin，非浏览器 client 这样设是合理的，但应该和 dashboard 升级路径走不同分支。
- [ ] Dockerfile 用浮动 tag（`node:20-alpine`、`golang:1.24-alpine`、`alpine:3.19`），应 pin `@sha256:...` digest 并升级 Alpine。
- [ ] `err.Error()` 直接进 API 响应大约 50 处，应该包成 generic error + correlation ID（详细错误进日志）。

## Low 级 / 杂项

- [ ] `/api/auth/login` 类端点无论成功失败都加固定常量时间延迟，让用户名枚举的时序攻击更难。
- [ ] CI 加 `govulncheck` + `npm audit --omit=dev` 作为门禁。
- [ ] `.dockerignore` 补充 `memory/`、`.agents/`、`.claude/`、`.worktrees/` 等本地状态目录，避免泄进镜像。

## 已完成（历史）

以下审查发现已在 v0.6.0 – v0.6.2 解决。具体改动见 [CHANGELOG.md](../CHANGELOG.md)。

- 移除 Makefile / dev systemd unit / CONTRIBUTING 中硬编码的 `thism2026` admin token（v0.6.0）。
- Agent 自更新二进制加 Ed25519 签名校验，公钥编译时固化，`thism-sign` CLI，上游 GHCR 镜像携带签名 agent（v0.6.0 → v0.6.1）。
- 每次打开数据库时把 `thism.db`（含 `-wal`/`-shm` 兄弟文件）chmod 0600（v0.6.0）。
- CI 签名流水线依赖 `THISM_RELEASE_PUBLIC_KEY` / `THISM_RELEASE_PRIVATE_KEY` 仓库 secret；缺 secret 时 release 直接失败（v0.6.1）。
- 面板内置 install 脚本从 `--token` on `ExecStart=` 改为 `EnvironmentFile`（v0.6.1）。
- Server 与 agent 都改为从 `THISM_*` 环境变量读凭据，systemd unit 调用二进制不传任何 flag —— `/proc/<pid>/cmdline` 不再含敏感数据（v0.6.2）。
