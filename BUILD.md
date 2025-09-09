# MCP Manager - Unified Build System

This project treats the entire MCP Manager as a single cohesive application, regardless of the multiple languages and technologies used (Go backend, TypeScript frontend, Electron wrapper).

## Quick Start

```bash
# Install dependencies
make install  # or: npm run setup

# Development mode (starts everything)
make dev      # or: npm run dev

# Build everything
make build    # or: npm run build

# Package for distribution
make package  # or: npm run package
```

## Available Commands

### Using Make (Recommended)

```bash
make help              # Show all available commands
make dev               # Start development mode
make dev-electron      # Start with Electron
make build             # Build entire application
make build-backend     # Build Go backend only
make build-frontend    # Build TypeScript frontend only
make package           # Package for current platform
make package-mac       # Package for macOS
make package-win       # Package for Windows
make package-linux     # Package for Linux
make test              # Run all tests
make lint              # Run linters
make clean             # Clean build artifacts
```

### Using npm scripts

```bash
npm run dev            # Start unified development mode
npm run dev:electron   # Start with Electron
npm run dev:backend    # Start backend only
npm run dev:frontend   # Start frontend only

npm run build          # Build everything
npm run build:backend  # Build backend only
npm run build:frontend # Build frontend only
npm run build:quick    # Build without cleaning
npm run build:prod     # Production build

npm run package        # Package for current platform
npm run package:mac    # Package for macOS
npm run package:win    # Package for Windows
npm run package:linux  # Package for Linux
npm run package:all    # Package for all platforms

npm run test           # Run all tests
npm run lint           # Run all linters
npm run format         # Format all code
npm run clean          # Clean build artifacts
```

## Architecture

The MCP Manager consists of:

1. **Go Backend** (`services/manager/`)
   - HTTP API server on port 38018
   - Manages MCP servers
   - Handles health monitoring and logging

2. **TypeScript Frontend** (`apps/desktop/`)
   - React-based UI
   - Vite development server on port 5173
   - shadcn/ui components with New York theme

3. **Electron Wrapper** (optional)
   - Desktop application packaging
   - Cross-platform distribution

## Development Workflow

### Standard Development Mode

```bash
# This starts both backend and frontend with hot-reload
npm run dev
```

This will:
1. Start the Go daemon (backend API)
2. Wait for daemon health check
3. Start Vite dev server (frontend)
4. Display URLs for both services

### Electron Development

```bash
npm run dev:electron
```

This additionally:
1. Waits for Vite to be ready
2. Launches Electron with the dev server

### Independent Development

```bash
# Backend only
npm run dev:backend

# Frontend only (requires backend running separately)
npm run dev:frontend
```

## Building for Production

### Complete Build

```bash
npm run build
```

This will:
1. Clean previous artifacts
2. Build Go backend (optimized binary)
3. Build TypeScript frontend (production bundle)
4. Collect artifacts in `build/` directory

### Platform-Specific Packages

```bash
# Current platform
npm run package

# Specific platforms
npm run package:mac
npm run package:win
npm run package:linux

# All platforms
npm run package:all
```

## Project Structure

```
MCP-Manager/
├── apps/
│   └── desktop/          # Frontend application
│       ├── src/          # React/TypeScript source
│       ├── dist/         # Build output
│       └── package.json
├── services/
│   └── manager/          # Backend service
│       ├── cmd/manager/  # Main entry point
│       ├── internal/     # Internal packages
│       ├── bin/          # Compiled binaries
│       └── go.mod
├── scripts/
│   ├── build.mjs         # Unified build script
│   ├── dev-unified.mjs   # Unified dev script
│   └── dev.mjs           # Legacy dev script
├── build/                # Unified build output
├── dist/                 # Distribution packages
├── Makefile              # Cross-platform make commands
└── package.json          # Root package configuration
```

## Prerequisites

- **Node.js** 18+ and npm
- **Go** 1.21+
- **Make** (optional, for Makefile usage)

## Testing

```bash
# Run all tests
npm run test

# Backend tests only
npm run test:backend

# Frontend tests only
npm run test:frontend
```

## Code Quality

```bash
# Lint all code
npm run lint

# Format all code
npm run format

# Backend specific
npm run lint:backend
npm run format:backend

# Frontend specific
npm run lint:frontend
npm run format:frontend
```

## Cleaning

```bash
# Clean build artifacts
npm run clean
# or
make clean

# Deep clean (including node_modules)
make deep-clean
```

## Environment Variables

The application uses these default ports:
- Backend API: `38018`
- Frontend Dev: `5173`

These can be configured in:
- Backend: `services/manager/internal/settings/store.go`
- Frontend: `apps/desktop/vite.config.ts`

## Troubleshooting

### Port Already in Use

If you see port conflicts:
```bash
# Kill processes on specific ports
lsof -ti:38018 | xargs kill -9  # Backend
lsof -ti:5173 | xargs kill -9   # Frontend
```

### Go Not Found

Install Go from https://golang.org/dl/ or:
```bash
# macOS
brew install go

# Linux
sudo apt install golang-go  # Debian/Ubuntu
sudo dnf install golang      # Fedora
```

### Build Failures

1. Clean and rebuild:
   ```bash
   npm run clean
   npm run build
   ```

2. Check dependencies:
   ```bash
   make check-deps
   ```

3. Reinstall dependencies:
   ```bash
   npm run setup
   ```

## Contributing

1. Use the unified build system for consistency
2. Test with `npm run test` before committing
3. Format code with `npm run format`
4. Use the Makefile for cross-platform compatibility

## License

See LICENSE file in the root directory.