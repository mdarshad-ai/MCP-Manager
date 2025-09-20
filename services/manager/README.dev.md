# Manager Daemon (Go) — Developer Notes

This is a scaffold. Planned packages:

- `internal/registry`: load/save `~/.mcp/registry.json`, validate against schema.
- `internal/supervisor`: start/stop processes, restart/backoff, resource sampling.
- `internal/health`: MCP initialize→initialized readiness, periodic pings.
- `internal/clients`: writers for Claude Desktop, Cursor, and adapters (Cursor CLI, Claude Code, Codex CLI, Gemini CLI) with prompt-once path memory.
- `internal/logs`: tail processes, rotate per-server (128MB) and global cap (1GB).
- `internal/autostart`: macOS launchd login item.

Entry point: `cmd/manager/main.go`.
