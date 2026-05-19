# systemd 部署

[English](systemd.md) | 简体中文

`deploy/` 下的文件是用于手动主机部署的模板。两个 unit 文件都通过 `EnvironmentFile=` 加载凭据，`ExecStart` **不**传任何敏感参数——两个二进制都从 `THISM_*` 环境变量读取。这样 token 和管理员密码既不在 unit 文件里，也不在 `/proc/<pid>/cmdline` 里。

启用前请先确认：

- 已创建对应的 `/etc/default/thism-*` 文件，权限 `0600`
- 对应二进制文件已经存在
- 所需运行用户和工作目录已经创建

## 服务端 Unit

```bash
sudo cp deploy/thism-server.service /etc/systemd/system/

sudo install -m 0600 /dev/null /etc/default/thism-server
sudo tee /etc/default/thism-server >/dev/null <<EOF
THISM_PORT=8080
THISM_TOKEN=$(openssl rand -hex 32)
THISM_ADMIN_USER=admin
THISM_ADMIN_PASS=$(openssl rand -base64 24)
EOF

sudo useradd --system --home-dir /var/lib/thism --shell /usr/sbin/nologin thism 2>/dev/null || true
sudo install -d -o thism -g thism -m 0700 /var/lib/thism

sudo systemctl daemon-reload
sudo systemctl enable --now thism-server
```

请记录 `/etc/default/thism-server` 里生成的 `THISM_TOKEN` 和 `THISM_ADMIN_PASS`，登录 Web UI 与调用管理 API 都需要它们。

## Agent Unit

```bash
sudo cp deploy/thism-agent.service /etc/systemd/system/

sudo install -m 0600 /dev/null /etc/default/thism-agent
sudo tee /etc/default/thism-agent >/dev/null <<EOF
THISM_AGENT_SERVER=ws://YOUR_SERVER_HOST:8080
THISM_AGENT_TOKEN=YOUR_NODE_TOKEN
THISM_AGENT_NAME=YOUR_NODE_NAME
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now thism-agent
```

`YOUR_NODE_TOKEN` 与 `YOUR_NODE_NAME` 来自 Web UI 的 `添加节点` 流程（或它生成的安装命令）。

## 为什么不把凭据放在 `ExecStart` 上？

即便配置了 `EnvironmentFile=`，如果 unit 写成 `ExecStart=... --token ${TOKEN} ...`，systemd 会在 exec **前**把 `${TOKEN}` 展开成真值传给 argv，于是 `/proc/<pid>/cmdline` 仍然能读到 token。Linux 默认配置下该文件全局可读——任何本机用户都能读到。改走 `THISM_*` 环境变量路径，二进制直接从进程环境拿凭据，命令行上彻底不出现。

两个二进制仍保留对应的 CLI flag（`--token`、`--admin-user` 等）用于向后兼容，但模板和系统内的安装脚本不再使用它们。
