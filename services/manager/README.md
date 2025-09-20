# Manager Daemon

Single background process responsible for installers, process supervision, health monitoring, and client config writers.

- Install sources: zip, git, npm, pip (v0 target).
- Normalize installs to `~/.mcp/servers/<slug>/bin/<entrypoint>`.
- Supervision: autostart on login, restart policy with backoff, CPU/RAM metrics.
- Health: MCP initialize/ready + periodic ping.
- Clients: write configs for Claude Desktop and Cursor.
- Dev: run via `npm run dev:manager` (placeholder).
