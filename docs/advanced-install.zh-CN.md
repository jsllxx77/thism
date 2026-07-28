# 高级安装选项

[English](advanced-install.md) | 简体中文

如果你只想尽快部署，请优先看根目录的 [README](../README.zh-CN.md)。本页补充手动和面向贡献者的安装方式。

## 从源码构建

前置条件：

- Go 1.26.4 或更高版本
- Node.js 和 npm，因为 `make build` 会先构建内嵌前端，再构建 Go 二进制

```bash
make build
mkdir -p geo

THISM_TOKEN=your-admin-token \
THISM_ADMIN_USER=admin \
THISM_ADMIN_PASS=strong-password \
THISM_GEOIP_DIR="$PWD/geo" \
./bin/thism-server --port 8080 --db ./thism.db
```

启动后，浏览器访问 `http://localhost:8080`，系统会跳转到 `/login`，使用配置的用户名和密码登录。

## 直接运行已发布 Docker 镜像

默认发布镜像：

```bash
ghcr.io/jsllxx77/thism:latest
```

无需 Compose，直接运行：

```bash
mkdir -p secrets
chmod 700 secrets
printf '%s\n' 'your-admin-token' > secrets/thism_token
printf '%s\n' 'admin' > secrets/thism_admin_user
printf '%s\n' 'strong-password' > secrets/thism_admin_pass
chmod 444 secrets/thism_*

IMAGE=ghcr.io/jsllxx77/thism:latest
GEO_UID="$(docker run --rm --entrypoint id "$IMAGE" -u)"
GEO_GID="$(docker run --rm --entrypoint id "$IMAGE" -g)"
sudo install -d -m 0755 -o "$GEO_UID" -g "$GEO_GID" /var/lib/thism/geo

docker run --name thism-server -p 8080:8080 \
  -v thism-data:/data \
  -v /var/lib/thism/geo:/geo \
  --mount type=bind,src="$PWD/secrets/thism_token",dst=/run/secrets/thism_token,readonly \
  --mount type=bind,src="$PWD/secrets/thism_admin_user",dst=/run/secrets/thism_admin_user,readonly \
  --mount type=bind,src="$PWD/secrets/thism_admin_pass",dst=/run/secrets/thism_admin_pass,readonly \
  -e THISM_PORT=8080 \
  -e THISM_DB=/data/thism.db \
  -e THISM_TOKEN_FILE=/run/secrets/thism_token \
  -e THISM_ADMIN_USER_FILE=/run/secrets/thism_admin_user \
  -e THISM_ADMIN_PASS_FILE=/run/secrets/thism_admin_pass \
  -e THISM_GEOIP_DIR=/geo \
  "$IMAGE"
```

## 从源码构建 Docker 镜像

```bash
docker build -t thism-server .
IMAGE=thism-server
mkdir -p secrets
chmod 700 secrets
printf '%s\n' 'your-admin-token' > secrets/thism_token
printf '%s\n' 'admin' > secrets/thism_admin_user
printf '%s\n' 'strong-password' > secrets/thism_admin_pass
chmod 444 secrets/thism_*

GEO_UID="$(docker run --rm --entrypoint id "$IMAGE" -u)"
GEO_GID="$(docker run --rm --entrypoint id "$IMAGE" -g)"
sudo install -d -m 0755 -o "$GEO_UID" -g "$GEO_GID" /var/lib/thism/geo

docker run -p 8080:8080 \
  -v thism-data:/data \
  -v /var/lib/thism/geo:/geo \
  --mount type=bind,src="$PWD/secrets/thism_token",dst=/run/secrets/thism_token,readonly \
  --mount type=bind,src="$PWD/secrets/thism_admin_user",dst=/run/secrets/thism_admin_user,readonly \
  --mount type=bind,src="$PWD/secrets/thism_admin_pass",dst=/run/secrets/thism_admin_pass,readonly \
  -e THISM_PORT=8080 \
  -e THISM_DB=/data/thism.db \
  -e THISM_TOKEN_FILE=/run/secrets/thism_token \
  -e THISM_ADMIN_USER_FILE=/run/secrets/thism_admin_user \
  -e THISM_ADMIN_PASS_FILE=/run/secrets/thism_admin_pass \
  -e THISM_GEOIP_DIR=/geo \
  "$IMAGE"
```
