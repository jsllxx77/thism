# Architecture Overview

[简体中文](architecture.zh-CN.md)

- **thism-server**: single Go binary with the embedded React UI. Runs on the main server and hosts agent downloads.
- **thism-agent**: lightweight Go binary that runs on each monitored server.
- **Communication**: the agent connects to the server over WebSocket and pushes metrics every 5 seconds.
- **Storage**: SQLite, with no external database dependency.
- **Deployment**: source builds, Docker image, and Docker Compose are supported.
