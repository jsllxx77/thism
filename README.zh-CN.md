# ThisM

[English](README.md) | 简体中文

轻量级自托管服务器监控。单个二进制，零外部依赖。

## 亮点

- 单个 Go 服务端二进制，内嵌 React 前端
- 面向被监控节点的轻量 Linux agent
- 使用 SQLite，无需额外数据库
- 服务端内置 agent 安装脚本与升级清单分发能力
- 提供预构建 GHCR 镜像与 Docker Compose 部署方式

## 快速开始

### 一键 Docker Compose 安装

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/thism-dev/thism/main/deploy/install-compose.sh)
```

安装脚本会自动：

1. 创建部署目录
2. 下载 `compose.yaml` 和 `.env.example`
3. 首次运行时生成随机管理员密码和 API Token
4. 从 `ghcr.io/thism-dev/thism:latest` 启动 `thism-server`

前提条件：

- 目标主机已安装 Docker，并且可用 `docker compose` v2

安装完成后，请在浏览器中打开 `http://<服务器 IP 或域名>:8080`，并使用脚本输出的账号密码登录。如果你是在本机安装并且也在本机访问，`http://localhost:8080` 同样可用。

生成的凭据会保存在 `~/thism-deploy/.env`。这个文件包含 API Token 和 Web 管理员密码，应按敏感文件妥善保管。

### 手动 Docker Compose 部署

```bash
mkdir -p ~/thism-deploy
cd ~/thism-deploy
curl -fsSL https://raw.githubusercontent.com/thism-dev/thism/main/deploy/docker-compose.yml -o compose.yaml
curl -fsSL https://raw.githubusercontent.com/thism-dev/thism/main/deploy/.env.example -o .env

# 首次启动前请先编辑 .env
docker compose up -d
```

默认 compose 部署会把数据保存在 Docker 命名卷中，并将 Web 界面暴露在 `8080` 端口。

`.env` 文件里保存了 API Token 和 Web 登录凭据。请注意保密，并在需要保留原始凭据时做好备份。

### 从源码构建

前置条件：

- Go 1.24 或更高版本
- Node.js 和 npm，因为 `make build` 会先构建内嵌前端，再构建 Go 二进制

```bash
make build

./bin/thism-server --port 8080 --db ./thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

启动后，浏览器访问 `http://localhost:8080`，系统会跳转到 `/login`，使用配置的用户名和密码登录。

## 注册并安装 Agent

### 1. 注册节点

```bash
curl -X POST http://localhost:8080/api/nodes/register \
  -H "Authorization: Bearer your-admin-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "web-1"}'
# 返回: {"id":"...","token":"..."}
```

### 2. 手动启动 Agent

```bash
./bin/thism-agent --server ws://your-host:8080 --token NODE_TOKEN --name web-1
```

### 3. 或直接使用服务端提供的安装脚本

```bash
curl -fsSL -H "Authorization: Bearer NODE_TOKEN" \
  "http://your-host:8080/install.sh?name=web-1" | sudo bash
```

这个安装命令需要以 root 身份执行，因为它会把 `thism-agent` 安装到 `/usr/local/bin`，写入 `/etc/systemd/system/thism-agent.service`，并通过 `systemctl` 重启服务。

安装脚本会自动识别 `linux/amd64` 与 `linux/arm64`，将 `thism-agent` 安装到 `/usr/local/bin/thism-agent`，并写入 `systemd` 服务，便于后续自动重启与重连。

## Docker 镜像

默认发布镜像：

```bash
ghcr.io/thism-dev/thism:latest
```

也可以不使用 Compose，直接运行镜像：

```bash
docker run --name thism-server -p 8080:8080 \
  -v thism-data:/data \
  ghcr.io/thism-dev/thism:latest \
  --port 8080 --db /data/thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

## 发布流程

正式发布仅通过手动语义化标签触发：

1. 准备并合并可发布变更到 `main`。
2. 在本地创建语义化版本标签（例如：`v1.4.0`）。
3. 推送标签：`git push origin v1.4.0`。

Release 工作流只会在推送 `v*` 标签时执行，并发布：

- `ghcr.io/thism-dev/thism:v1.4.0`（正式 semver 标签）
- `ghcr.io/thism-dev/thism:sha-<shortsha>`（可追溯的不可变标签）
- `ghcr.io/thism-dev/thism:latest`（当前正式发布）

Docker 构建时会把统一构建元数据注入到二进制：

- `THISM_VERSION`：来自 git 标签（例如 `v1.4.0`）
- `THISM_COMMIT`：完整 commit SHA
- `THISM_BUILD_TIME`：UTC RFC3339 时间戳

开发构建 vs 正式发布：

- 开发构建（例如本地 `make build` 或未传发布参数的临时 Docker 构建）用于调试验证，版本元数据可能是默认值或非正式值。
- 正式发布是由上述标签触发工作流生成的不可变 semver 构建产物。

## 开发流程

### 本地快速开发

日常前端开发建议使用两个终端：

```bash
# 终端 1：后端 API / WebSocket
make dev-server TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass

# 终端 2：Vite 热更新
make dev-ui
```

然后打开 `http://localhost:5173`，前端修改会即时热更新。

### 验证嵌入式前端

```bash
make dev-restart TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass
```

这个命令会：

1. 构建前端
2. 重新构建内嵌静态资源的 `bin/thism-server`
3. 重启本地服务端

### 测试与构建

```bash
make test

cd frontend
npm ci
npm run lint
npm test
npm run build
```

## systemd

`deploy/` 下的服务文件是模板，启用前需要先替换 `ExecStart` 里的占位参数，并确保对应二进制、运行用户和工作目录已经准备好：

```bash
# 服务端
sudo cp deploy/thism-server.service /etc/systemd/system/

# 先把 YOUR_ADMIN_TOKEN / YOUR_ADMIN_USER / YOUR_ADMIN_PASSWORD 等占位符改成真实值，
# 并确保 `thism` 用户和 /var/lib/thism 目录已存在。
sudo systemctl daemon-reload
sudo systemctl enable --now thism-server

# Agent（部署在每台被监控机器上）
sudo cp deploy/thism-agent.service /etc/systemd/system/

# 启动前请先替换 YOUR_SERVER_HOST / YOUR_NODE_TOKEN / YOUR_NODE_NAME。
sudo systemctl daemon-reload
sudo systemctl enable --now thism-agent
```

## 从源码构建 Docker 镜像

```bash
docker build -t thism-server .
docker run -p 8080:8080 -v thism-data:/data thism-server \
  --port 8080 --db /data/thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

## 架构说明

- **thisM-server**：单个 Go 服务端二进制，内嵌 React UI，同时负责提供 agent 下载。
- **thisM-agent**：轻量级 Go 二进制，运行在每台被监控服务器上。
- **通信方式**：agent 通过 WebSocket 与 server 保持连接，并定期上报指标。
- **存储**：使用 SQLite，无需额外数据库服务。
- **部署方式**：支持源码构建、Docker 镜像和 Docker Compose。
