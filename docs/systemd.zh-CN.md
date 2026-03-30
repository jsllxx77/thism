# systemd 部署

[English](systemd.md) | 简体中文

`deploy/` 下的文件是用于手动主机部署的模板。

启用前请先确认：

- 已把 `ExecStart` 中的占位符替换为真实值
- 对应二进制文件已经存在
- 所需运行用户和工作目录已经创建

## 服务端 Unit

```bash
sudo cp deploy/thism-server.service /etc/systemd/system/

# 先把 YOUR_ADMIN_TOKEN / YOUR_ADMIN_USER / YOUR_ADMIN_PASSWORD 等占位符改成真实值，
# 并确保 `thism` 用户和 /var/lib/thism 目录已存在。
sudo systemctl daemon-reload
sudo systemctl enable --now thism-server
```

## Agent Unit

```bash
sudo cp deploy/thism-agent.service /etc/systemd/system/

# 启动前请先替换 YOUR_SERVER_HOST / YOUR_NODE_TOKEN / YOUR_NODE_NAME。
sudo systemctl daemon-reload
sudo systemctl enable --now thism-agent
```
