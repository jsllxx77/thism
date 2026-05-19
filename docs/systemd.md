# systemd Deployment

[简体中文](systemd.zh-CN.md)

The files in `deploy/` are templates for manual host installs. Both unit files load their credentials from an `EnvironmentFile=` and pass **no** sensitive flags on `ExecStart` — both binaries read `THISM_*` environment variables. This keeps tokens and admin passwords out of both the unit file on disk and `/proc/<pid>/cmdline`.

Before enabling them:

- Create the matching `/etc/default/thism-*` file with mode `0600`
- Make sure the referenced binaries exist
- Create any required runtime user and working directory

## Server Unit

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

Record the generated `THISM_TOKEN` and `THISM_ADMIN_PASS` from `/etc/default/thism-server`; you will need them to log into the web UI and to call the admin API.

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

`YOUR_NODE_TOKEN` and `YOUR_NODE_NAME` come from the `Add Node` flow in the web UI (or the install command it generates).

## Why not put secrets on `ExecStart`?

Even with `EnvironmentFile=`, a unit body of `ExecStart=... --token ${TOKEN} ...` causes systemd to expand `${TOKEN}` **before** exec, so the actual argv contains the literal secret. That argv is then visible via `/proc/<pid>/cmdline`, which is world-readable on a default Linux install. The `THISM_*` env-var read path lets the binary pick up credentials from its environment without ever putting them on the command line.

Both binaries accept the equivalent CLI flags (`--token`, `--admin-user`, etc.) for backward compatibility, but the templates and the in-product install script no longer use them.

