# 高级安装选项

[English](advanced-install.md) | 简体中文

如果你只想尽快部署，请优先看根目录的 [README](../README.zh-CN.md)。本页补充手动和面向贡献者的安装方式。

## 从源码构建

前置条件：

- Go 1.24 或更高版本
- Node.js 和 npm，因为 `make build` 会先构建内嵌前端，再构建 Go 二进制

```bash
make build

./bin/thism-server --port 8080 --db ./thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

启动后，浏览器访问 `http://localhost:8080`，系统会跳转到 `/login`，使用配置的用户名和密码登录。

## 直接运行已发布 Docker 镜像

默认发布镜像：

```bash
ghcr.io/thism-dev/thism:latest
```

无需 Compose，直接运行：

```bash
docker run --name thism-server -p 8080:8080 \
  -v thism-data:/data \
  ghcr.io/thism-dev/thism:latest \
  --port 8080 --db /data/thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

## 从源码构建 Docker 镜像

```bash
docker build -t thism-server .
docker run -p 8080:8080 -v thism-data:/data thism-server \
  --port 8080 --db /data/thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```
