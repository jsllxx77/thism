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

## 添加并安装 Agent

常规接入流程直接通过 Web 面板完成：

1. 打开 Web 界面，并以管理员身份登录。
2. 进入 `设置` 页面。
3. 在 `节点管理` 区域点击 `添加节点`。
4. 输入节点名称后，点击 `获取安装命令`。
5. 复制面板里显示的 `安装命令`。
6. 在目标 Linux 机器上以 `root` 身份执行该命令。

生成的命令会把 `thism-agent` 安装到 `/usr/local/bin`，写入 `systemd` 服务，并启动 agent。安装器支持 `linux/amd64` 和 `linux/arm64`。

如果节点已经存在，之后需要再次获取命令，可以进入 `设置` -> `节点管理`，在对应节点行点击 `获取脚本`。

## 更多文档

- [高级安装选项](docs/advanced-install.zh-CN.md)：从源码构建、直接运行已发布 Docker 镜像、或在本地构建 Docker 镜像
- [systemd 部署模板](docs/systemd.zh-CN.md)：使用仓库内置 unit 文件进行手动主机部署
- [开发流程](docs/development.zh-CN.md)：本地开发循环、前端验证方式与测试/构建命令
- [发布流程](docs/release.zh-CN.md)：标签驱动发布与镜像标签说明
- [架构说明](docs/architecture.zh-CN.md)：服务端、agent、通信、存储与部署模型概览
- [贡献指南](CONTRIBUTING.md)：仓库贡献约定
