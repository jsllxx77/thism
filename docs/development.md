# Development Workflow

[简体中文](development.zh-CN.md)

This page is for contributors working on the embedded frontend and Go server locally.

## Fast Local Loop

Use two terminals for everyday frontend work:

```bash
# Terminal 1: backend API/ws
make dev-server TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass

# Terminal 2: Vite with HMR
make dev-ui
```

Open `http://localhost:5173`. Frontend changes hot-reload instantly.

## Verify the Embedded Frontend

```bash
make dev-restart TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass
```

This command:

1. Builds the frontend
2. Rebuilds `bin/thism-server` with embedded assets
3. Restarts the local server

## Test and Build

```bash
make test

cd frontend
npm ci
npm run lint
npm test
npm run build
```
