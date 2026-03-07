# Contributing to ThisM

Thanks for taking the time to contribute.

## Before You Start

- Open an issue first for large changes, breaking changes, or UI redesigns.
- Keep changes focused and easy to review.
- Do not mix refactors, formatting-only edits, and feature work in the same PR.

## Development Setup

### Backend

```bash
make test
```

### Frontend

```bash
cd frontend
npm install
npm run lint
npm test
npm run build
```

### Full local verification

```bash
make dev-restart TOKEN=thism2026 PORT=12026 ADMIN_USER=admin ADMIN_PASS=thism-admin-2026
```

This rebuilds the frontend, rebuilds the Go server binary, and restarts the local development runtime.

## Pull Request Guidelines

- Add or update tests when changing behavior.
- Update documentation when user-facing behavior changes.
- Keep commit history readable.
- Include a short summary, testing notes, and screenshots for UI changes.

## Coding Expectations

- Prefer small, direct fixes over broad rewrites.
- Preserve existing architecture unless there is a clear reason to change it.
- Avoid adding dependencies without justification.
- Do not commit secrets, local databases, or generated binaries.

## Reporting Bugs

When filing a bug, please include:

- What you expected to happen
- What actually happened
- How to reproduce it
- Logs, screenshots, or traces when relevant
- Version, platform, and deployment method
