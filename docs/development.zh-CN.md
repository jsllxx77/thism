# 开发流程

[English](development.md) | 简体中文

本页面向本地开发内嵌前端和 Go 服务端的贡献者。

## 本地快速开发

日常前端开发建议使用两个终端：

```bash
# 终端 1：后端 API / WebSocket
make dev-server TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass

# 终端 2：Vite 热更新
make dev-ui
```

然后打开 `http://localhost:5173`，前端修改会即时热更新。

## 验证嵌入式前端

```bash
make dev-restart TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass
```

这个命令会：

1. 构建前端
2. 重新构建内嵌静态资源的 `bin/thism-server`
3. 重启本地服务端

## 测试与构建

```bash
make test

cd frontend
npm ci
npm run lint
npm test
npm run build
```
