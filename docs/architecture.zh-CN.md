# 架构说明

[English](architecture.md) | 简体中文

- **thism-server**：单个 Go 服务端二进制，内嵌 React UI，同时负责提供 agent 下载。
- **thism-agent**：轻量级 Go 二进制，运行在每台被监控服务器上。
- **通信方式**：agent 通过 WebSocket 与 server 保持连接，并每 5 秒上报一次指标。
- **存储**：使用 SQLite，无需额外数据库依赖。
- **部署方式**：支持源码构建、Docker 镜像和 Docker Compose。
