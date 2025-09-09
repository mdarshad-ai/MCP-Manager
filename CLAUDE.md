# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

MCP Manager is a desktop application for managing local MCP (Model Context Protocol) servers. It provides installation, configuration, autostart, supervision, health monitoring, and client integration capabilities through an Electron-based UI and a Go daemon backend.

## Architecture

### Core Components
- **apps/desktop**: Electron + React UI with Vite, TypeScript, and Tailwind CSS
- **services/manager**: Go daemon (v1.22+) for server management, supervision, and health monitoring
- **packages/shared**: Shared schemas and types (registry schema, IPC contracts)
- **scripts/**: Development and packaging scripts
- **docs/**: Architecture decisions and specifications

### Runtime Directory Structure
```
~/.mcp/
  registry.json         # Server registry
  servers/<slug>/       # Individual server installations
  logs/<slug>.log       # Server logs
  secrets/              # OS keychain-backed secrets
  cache/                # Temporary cache
  clients/config.json   # Client configuration paths
```

## Development Commands

### Quick Start
```bash
npm ci                    # Install dependencies
npm run dev              # Start both daemon and UI (orchestrated)
npm run dev:web          # Start UI dev server only
npm run dev:manager      # Start Go daemon only
```

### Build & Package
```bash
npm run build            # Build both manager and renderer
npm run build:manager    # Build Go daemon to services/manager/bin/
npm run build:renderer   # Build React UI
npm run package:mac      # Package macOS app (requires electron-builder)
```

### Testing
```bash
npm test                 # Run workspace tests
go test ./...            # Run Go tests (in services/manager/)
npm -w apps/desktop run test  # Run UI tests with Vitest
```

### Code Quality
```bash
npm run lint             # Run Biome linter
npm run format           # Format code with Biome
biome check .            # Check code quality
biome format .           # Format files
```

## Key Technologies

### Frontend (apps/desktop)
- React 18 + TypeScript
- Vite for bundling and dev server
- Tailwind CSS for styling
- Vitest for testing
- Routes: `/dashboard`, `/install`, `/server/:slug`, `/clients`, `/settings`, `/logs/:slug`

### Backend (services/manager)
- Go 1.22+ daemon
- HTTP API on `127.0.0.1:38018`
- Internal packages:
  - `internal/registry`: Server registry management
  - `internal/supervisor`: Process supervision with restart policies
  - `internal/health`: MCP health checks and monitoring
  - `internal/clients`: Claude/Cursor config writers
  - `internal/autostart`: Login daemon integration
  - `internal/httpapi`: REST API server
  - `internal/logs`: Log rotation and management

## MCP Server Management

### Installation Methods
- **zip**: Extract and normalize to `servers/<slug>/`
- **git**: Clone repositories
- **npm**: Install Node.js packages
- **pip**: Install Python packages

### Client Integration
- **Claude Desktop**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Cursor**: `~/.cursor/mcp.json`
- **Continue**: Snippet export support

## Code Standards

### Formatting
- TypeScript/JavaScript: 2 spaces indentation (enforced by Biome)
- Go: Standard `gofmt` formatting
- Naming conventions:
  - camelCase: functions, variables
  - PascalCase: classes, components
  - kebab-case: directories, packages

### Testing
- UI: Vitest + React Testing Library
- Go: Standard `testing` package
- Test file naming:
  - TypeScript: `*.test.ts(x)`
  - Go: `*_test.go`
- Target ≥80% coverage

## Development Workflow

### Starting Development
1. Ensure Go 1.22+ is installed (`brew install go` on macOS)
2. Run `npm ci` to install dependencies
3. Use `npm run dev` to start both daemon and UI with hot reload
4. The dev script orchestrates startup, waits for daemon health, then starts UI

### Port Configuration
- Manager daemon: `127.0.0.1:38018` (loopback only for security)
- Vite dev server: Default port (usually 5173)

### Health Monitoring
The daemon provides health endpoints:
- `/healthz`: Basic health check
- Automatic retry with exponential backoff for failed servers
- Log rotation at 128MB per server, 1GB total

## Security Considerations
- All HTTP servers default to loopback-only (127.0.0.1)
- Secrets stored in OS keychain-backed vault
- Warnings displayed for unsigned/untrusted sources
- Never commit `.env` files or secrets
- Write only under `~/.mcp/` directory

## Troubleshooting

### Common Issues
- **Go not found**: Install Go 1.22+ with `brew install go`
- **Port conflict**: Ensure port 38018 is available for daemon
- **Permission errors**: Check write permissions for `~/.mcp/`

### Build Errors
- Run `npm run build:manager` separately to debug Go build issues
- Check Go module dependencies with `go mod tidy` in services/manager/
- For UI issues, check `npm -w apps/desktop run build` output

## Commit Guidelines
Use conventional commits:
- `feat:` New features
- `fix:` Bug fixes
- `chore:` Maintenance tasks
- `docs:` Documentation updates
- `refactor:` Code refactoring
- `test:` Test additions/changes