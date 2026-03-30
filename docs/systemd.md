# systemd Deployment

[简体中文](systemd.zh-CN.md)

The files in `deploy/` are templates for manual host installs.

Before enabling them:

- Replace placeholder values in `ExecStart`
- Make sure the referenced binaries exist
- Create any required runtime user and working directory

## Server Unit

```bash
sudo cp deploy/thism-server.service /etc/systemd/system/

# Edit placeholders such as YOUR_ADMIN_TOKEN / YOUR_ADMIN_USER / YOUR_ADMIN_PASSWORD
# and ensure the `thism` user plus /var/lib/thism exist before starting.
sudo systemctl daemon-reload
sudo systemctl enable --now thism-server
```

## Agent Unit

```bash
sudo cp deploy/thism-agent.service /etc/systemd/system/

# Edit YOUR_SERVER_HOST / YOUR_NODE_TOKEN / YOUR_NODE_NAME before starting.
sudo systemctl daemon-reload
sudo systemctl enable --now thism-agent
```
