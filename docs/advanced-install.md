# Advanced Installation

[简体中文](advanced-install.zh-CN.md)

Use the root [README](../README.md) for the fastest path. This page covers manual and contributor-oriented install paths.

## Build From Source

Prerequisites:

- Go 1.26.4 or newer
- Node.js and npm, because `make build` compiles the embedded frontend before building the Go binaries

```bash
make build
mkdir -p geo

THISM_TOKEN=your-admin-token \
THISM_ADMIN_USER=admin \
THISM_ADMIN_PASS=strong-password \
THISM_GEOIP_DIR="$PWD/geo" \
./bin/thism-server --port 8080 --db ./thism.db
```

Open `http://localhost:8080` in your browser. You will be redirected to `/login` and authenticate with the configured username and password.

## Run the Published Docker Image Directly

Published runtime image:

```bash
ghcr.io/jsllxx77/thism:latest
```

Run it without Compose:

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

## Build the Docker Image From Source

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
