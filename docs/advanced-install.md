# Advanced Installation

[简体中文](advanced-install.zh-CN.md)

Use the root [README](../README.md) for the fastest path. This page covers manual and contributor-oriented install paths.

## Build From Source

Prerequisites:

- Go 1.24 or newer
- Node.js and npm, because `make build` compiles the embedded frontend before building the Go binaries

```bash
make build

./bin/thism-server --port 8080 --db ./thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

Open `http://localhost:8080` in your browser. You will be redirected to `/login` and authenticate with the configured username and password.

## Run the Published Docker Image Directly

Published runtime image:

```bash
ghcr.io/thism-dev/thism:latest
```

Run it without Compose:

```bash
docker run --name thism-server -p 8080:8080 \
  -v thism-data:/data \
  ghcr.io/thism-dev/thism:latest \
  --port 8080 --db /data/thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```

## Build the Docker Image From Source

```bash
docker build -t thism-server .
docker run -p 8080:8080 -v thism-data:/data thism-server \
  --port 8080 --db /data/thism.db --token your-admin-token \
  --admin-user admin --admin-pass strong-password
```
