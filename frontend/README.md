# ThisM Frontend

This directory contains the React + TypeScript frontend embedded into the ThisM server binary.

## Development

```bash
cd frontend
npm ci
npm run lint
npm test
npm run build
```

For a full local development loop, use the root-level workflow documented in `README.md`:

```bash
make dev-server TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass
make dev-ui
```

When you need to verify the embedded frontend inside the Go binary, run:

```bash
make dev-restart TOKEN=mytoken PORT=12026 ADMIN_USER=admin ADMIN_PASS=dev-pass
```
