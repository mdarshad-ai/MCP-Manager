# Development Guide

Prerequisites
- Node 20+, pnpm or npm, Go 1.22+.
- macOS (v0 target) with Xcode Command Line Tools for launchd.

Workspace
- Root scripts:
  - `npm run lint` — Biome check.
  - `npm run dev:desktop` — starts Vite (configure deps).
  - `npm run dev:manager` — `go run ./services/manager/cmd/manager`.

Desktop (Electron + React)
- App code under `apps/desktop/src`. Tailwind configured; shadcn/ui components can be added once dependencies are installed.
- Pages: Dashboard, Install, Server Details, Clients, Logs, Settings.

Manager (Go)
- Entry: `services/manager/cmd/manager/main.go` — starts HTTP API on `127.0.0.1:38018`.
- Packages: `internal/registry`, `internal/health`, `internal/supervisor`, `internal/logs`, `internal/clients`, `internal/autostart`, `internal/httpapi`.
- Tests: run `go test ./...` in `services/manager`.

Config/Paths
- Registry: `~/.mcp/registry.json` (see `packages/shared/schema/registry.schema.json`).
- Logs: `~/.mcp/logs/` with hybrid rotation (128MB per server, 1GB total).
- Clients: Claude `~/Library/Application Support/Claude/claude_desktop_config.json`, Cursor `~/.cursor/mcp.json`. Store remembered paths under `~/.mcp/clients/config.json`.

Next Steps
- Wire real supervisor start/stop/restart and health ping loop.
- Implement client writers and UI diff/confirm flows.
- Add Electron packaging and preload, IPC bridge to Go API.

Packaging (scaffold)
- electron-builder config at `electron-builder.json` with app root `apps/desktop`.
- Scripts (run from repo root):
  - Build Go daemon: `npm run build:manager`
  - Build UI: `npm run build:renderer`
  - Package macOS app: `npm run package:mac`
    - Requires: `electron-builder` available (`npm i -g electron-builder` or `npx electron-builder`).
    - Output: `dist/mac/MCP Manager.app` and DMG/ZIP under `dist/`.
