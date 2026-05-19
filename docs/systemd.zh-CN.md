# systemd 部署

[English](systemd.md) | 简体中文

`deploy/` 下的文件是用于手动主机部署的模板。两个 unit 文件都通过 `EnvironmentFile=` 加载凭据，**不会**把 token/密码放在 `ExecStart` 命令行——命令行参数会通过 `/proc/<pid>/cmdline` 暴露给本机所有用户，所以管理 token 和管理员密码必须避开命令行。

启用前请先确认：

- 已创建对应的 `/etc/default/thism-*` 文件，权限 `0600`
- 对应二进制文件已经存在
- 所需运行用户和工作目录已经创建

## 服务端 Unit

```bash
sudo cp deploy/thism-server.service /etc/systemd/system/

sudo install -m 0600 /dev/null /etc/default/thism-server
sudo tee /etc/default/thism-server >/dev/null <<EOF
PORT=8080
TOKEN=$(openssl rand -hex 32)
ADMIN_USER=admin
ADMIN_PASS=$(openssl rand -base64 24)
EOF

sudo useradd --system --home-dir /var/lib/thism --shell /usr/sbin/nologin thism 2>/dev/null || true
sudo install -d -o thism -g thism -m 0700 /var/lib/thism

sudo systemctl daemon-reload
sudo systemctl enable --now thism-server
```

请记录 `/etc/default/thism-server` 里生成的 `TOKEN` 和 `ADMIN_PASS`，登录 Web UI 与调用管理 API 都需要它们。

## Agent Unit

```bash
sudo cp deploy/thism-agent.service /etc/systemd/system/

sudo install -m 0600 /dev/null /etc/default/thism-agent
sudo tee /etc/default/thism-agent >/dev/null <<EOF
SERVER=ws://YOUR_SERVER_HOST:8080
TOKEN=YOUR_NODE_TOKEN
NAME=YOUR_NODE_NAME
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now thism-agent
```

`YOUR_NODE_TOKEN` 与 `YOUR_NODE_NAME` 来自 Web UI 的 `添加节点` 流程（或它生成的安装命令）。

## 为什么不把凭据放在 `ExecStart` 上？

`ExecStart=... --token YOUR_TOKEN ...` 会出现在 `/proc/<pid>/cmdline`，Linux 默认配置下该文件全局可读——任何本机用户（包括无特权服务账户）都能读到。`EnvironmentFile=` 加 `0600` 权限的 env 文件配合，才能把凭据限制为只有 root 和服务本身可见。
