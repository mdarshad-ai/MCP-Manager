# MCP Manager Development Guidelines

## Overview

MCP Manager is a desktop application for managing local Machine Learning Control Panel (MCP) servers. It serves as a comprehensive management layer for AI/ML services, handling installation, configuration, monitoring, and client integration.

### Key Objectives
- Provide reliable server management and monitoring
- Ensure secure and isolated server operations
- Enable seamless integration with AI development tools
- Maintain high code quality and test coverage

## Project Architecture & Structure

### Core Components
- `apps/desktop`: Electron + React UI application
  - React routes: `/dashboard`, `/install`, `/server/:slug`, `/clients`, `/settings`, `/logs/:slug`
  - Built with Vite, Tailwind CSS, and shadcn/ui (New York theme)
  - Entry: `electron-main.ts` with preload scripts

- `services/manager`: Go-based Manager Daemon
  - Core packages: registry, health, supervisor, logs, clients, autostart, httpapi
  - Local HTTP API on `127.0.0.1:38018`
  - Entry: `cmd/manager/main.go`

- `packages/shared`: Cross-cutting types and utilities
  - Registry schema definitions
  - IPC contracts and type definitions
  - Shared constants and configurations

### Runtime Data Structure
```
~/.mcp/
  registry.json      # Server definitions and state
  servers/
    <slug>/         # Per-server isolation
      manifest.json  # Installation metadata
      install/      # Source files
      runtime/      # Working directory
      bin/          # Normalized executables
  logs/             # Per-server logs
    <slug>.log      # 128MB per file
  secrets/          # OS keychain integration
  cache/            # Downloads and build artifacts
  clients/          # Client config state
```

## Development Workflow

### Build & Development Commands
- `npm ci` or `pnpm i`: Install dependencies (use pnpm for better perf)
- Development mode:
  - `npm run dev:desktop`: Electron + Vite with hot reload
  - `npm run dev:manager`: Go daemon in watch mode
- Build and package:
  - `npm run build`: Full build (UI + daemon)
  - `npm run package:mac`: macOS app with DMG
  - `npm run package:win`: Windows executable
  - `npm run package:linux`: Linux AppImage
- Testing:
  - `npm test`: Run all unit tests
  - `npm run e2e`: Playwright UI tests
  - Individual test suites:
    - UI: `cd apps/desktop && npm test`
    - Manager: `cd services/manager && go test ./...`

### Environment Setup
- Node.js 20+ and Go 1.22+ required
- macOS: Xcode Command Line Tools needed for launchd integration
- Windows/Linux: See platform-specific notes in BUILD.md

## Coding Standards & Patterns

### TypeScript (UI)
- Use TypeScript for all new code
- Indentation: 2 spaces
- Naming:
  - Components: PascalCase (`ServerDetails.tsx`)
  - Hooks: camelCase with `use` prefix (`useHealthStatus.ts`)
  - Utils: camelCase (`formatUptime.ts`)
  - Types/Interfaces: PascalCase (`ServerConfig.ts`)
- Component structure:
  - Props interface at top
  - Hooks before render
  - Extract complex logic to custom hooks
  - Prefer composition over inheritance

### Go (Manager Daemon)
- Follow standard Go conventions
- Indentation: tabs (gofmt)
- Package organization:
  - One package per core feature
  - Internal packages under `internal/`
  - Tests alongside source files
- Error handling:
  - Return errors rather than panic
  - Use error wrapping with context
  - Include stack traces for unexpected errors
- Concurrency:
  - Clear ownership of goroutines
  - Use context for cancellation
  - Protect shared state with mutex

### Testing Guidelines
- UI Testing:
  - Unit: Vitest + React Testing Library
  - Component tests mirror component structure
  - E2E: Playwright for critical flows
  - Test IDs prefixed with feature name
  - Snapshot tests discouraged

- Go Testing:
  - Standard `testing` package
  - Table-driven tests preferred
  - Mock external services via interfaces
  - Separate integration tests with build tags
  - Coverage target: ≥80%

- Test file naming:
  - TypeScript: `*.test.ts(x)`
  - Go: `*_test.go`
  - E2E: `*.spec.ts`

### Performance & Security
- UI performance:
  - Memoize expensive computations
  - Virtualize long lists
  - Code-split routes
  - Preload critical resources

- Security:
  - HTTP server binds to loopback only
  - Validate all external input
  - Use OS keychain for secrets
  - Warn on unsigned sources
  - Regular dependency audits

### Error Handling
- UI errors:
  - Toast notifications for transient errors
  - Full error pages for critical failures
  - Retry with exponential backoff
  - Clear recovery actions

- Daemon errors:
  - Structured logging with levels
  - Error context preservation
  - Graceful degradation
  - Auto-restart critical services

## Version Control & PRs

### Branch Strategy
- Main branch: `main`
- Feature branches: `feature/short-name`
- Fix branches: `fix/issue-desc`

### Commit Messages
- Follow Conventional Commits
- Format: `type(scope): message`
- Types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`
- Include issue refs: `(#123)`

### Pull Requests
Required sections:
- Summary of changes
- Linked issues (`Closes #123`)
- Testing steps
- Screenshots (UI changes)
- Migration notes
- Security implications

## Client Integration

### Supported Clients
- Claude Desktop
  - Config: `~/Library/Application Support/Claude/claude_desktop_config.json`
  - Features: server selection, autoconnect
  
- Cursor
  - Config: `~/.cursor/mcp.json`
  - Features: multiple server support, path detection

### Integration Points
- Config writers in `internal/clients/`
- Automatic path detection
- Config diffing and backup
- Validation before write

## Documentation

### Required Documentation
- README.md: Project overview
- DEVELOPMENT.md: Setup guide
- API.md: HTTP API spec
- SECURITY.md: Security model
- CONTRIBUTING.md: Contribution guide

### Code Documentation
- Go: Package docs and exported symbols
- TypeScript: JSDoc for public APIs
- Comments explain "why" not "what"
- Keep docs close to code

## Debugging

### UI Debugging
- React DevTools
- Network tab for API calls
- Redux DevTools (if added)
- Console logging levels

### Daemon Debugging
- Log levels: debug, info, warn, error
- HTTP API debugging endpoints
- Performance profiling
- Memory leak detection

### Known Issues
- Document workarounds in docs/KNOWN_ISSUES.md
- Include version information
- Link related issues

## Platform Support

### macOS
- Primary development target
- Uses launchd for autostart
- Native keychain integration

### Windows (Planned)
- Services for daemon
- Windows Credential Manager
- NTFS permissions

### Linux (Planned)
- systemd integration
- Secret Service API
- AppImage distribution
