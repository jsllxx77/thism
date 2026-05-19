# systemd Deployment

[简体中文](systemd.zh-CN.md)

The files in `deploy/` are templates for manual host installs. Both unit files load their secrets from an `EnvironmentFile=` instead of placing them on the `ExecStart` command line — command-line arguments are exposed to every local user via `/proc/<pid>/cmdline`, which is why tokens and admin passwords must not live there.

Before enabling them:

- Create the matching `/etc/default/thism-*` file with mode `0600`
- Make sure the referenced binaries exist
- Create any required runtime user and working directory

## Server Unit

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

Record the generated `TOKEN` and `ADMIN_PASS` from `/etc/default/thism-server`; you will need them to log into the web UI and to call the admin API.

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

`YOUR_NODE_TOKEN` and `YOUR_NODE_NAME` come from the `Add Node` flow in the web UI (or the install command it generates).

## Why not put secrets on `ExecStart`?

`ExecStart=... --token YOUR_TOKEN ...` ends up in `/proc/<pid>/cmdline`, which is world-readable on a default Linux install. Any local user — including unprivileged service accounts — can read it. The `EnvironmentFile=` pattern, combined with `0600` ownership on the env file, keeps the secret accessible only to root and to the service itself.
