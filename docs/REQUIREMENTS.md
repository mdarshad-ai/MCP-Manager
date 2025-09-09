# MCP Manager — Requirements (Imported)

This document mirrors the high-level requirements provided out-of-band to anchor implementation and review within the repository.

## Purpose
- Desktop application to manage local MCP servers: installation, configuration, autostart, supervision, health, and client integration.

## Architecture Overview
- Electron Shell (Chromium UI), Manager Daemon (installers/supervisor/health/clients), Central Directory `~/.mcp/`.

## Central Directory Layout
```
~/.mcp/
  registry.json
  servers/
    <slug>/
      manifest.json
      install/
      runtime/
      bin/<entrypoint>
  logs/<slug>.log
  secrets/
  cache/
```

## Registry Schema
- See `packages/shared/schema/registry.schema.json`.

## Core Features (v0 targets)
- Install: zip, git, npm, pip → normalized to `servers/<slug>/bin/<entrypoint>`.
- Supervision: login daemon, per-server autostart, restart with backoff.
- Health: MCP initialize→ready, periodic ping, metrics, log tailing.
- Clients: write configs for Claude Desktop and Cursor; Continue snippet export.

## UI Routes
`/dashboard`, `/install`, `/server/:slug`, `/clients`, `/settings`, `/logs/:slug`.

## Security
- Least-privilege permissions; OS keychain-backed secrets; warnings for unsigned sources; HTTP servers loopback-only by default.

## Acceptance Criteria (v0)
- Install from zip, git, npm, pip; autostart toggle; health display; client config writers; graceful shutdown and auto-restart.
