# MCP Manager

MCP Manager is a robust desktop application designed to manage local Machine Learning Control Panel (MCP) servers. It provides a comprehensive solution for installing, configuring, monitoring, and integrating various AI/ML services.

## Features

### Server Management
- Multi-source installation support (zip, git, npm, pip)
- Real-time health monitoring with metrics
- Automatic restart with intelligent backoff
- Centralized log management and rotation
- Secure credential handling via OS keychain

### Client Integration
- Seamless integration with AI development tools:
  - Claude Desktop
  - Cursor
  - More planned
- Automatic configuration management
- Smart path detection and validation
- Configuration backup and diff support

### User Interface
- Modern Electron-based desktop application
- Clean, intuitive dashboard for server monitoring
- Detailed server health and metrics views
- Comprehensive logging interface
- Easy-to-use installation wizard

## Getting Started

### Prerequisites
- Node.js 20+
- Go 1.22+ (for development)
- macOS, Windows, or Linux
- For macOS: Xcode Command Line Tools

### Quick Start
```bash
# Install dependencies
npm ci  # or pnpm i for better performance

# Start development environment
npm run dev:desktop  # UI with hot reload
npm run dev:manager  # Manager daemon

# Build for production
npm run build        # Full build
npm run package:mac  # Package for macOS
```

### Directory Structure
```
~/.mcp/              # Runtime directory
  registry.json      # Server definitions
  servers/<slug>/    # Isolated server instances
  logs/              # Centralized logs
  secrets/           # Secure storage
  cache/             # Build cache
```

## Development

See [AGENTS.md](AGENTS.md) for detailed development guidelines.

### Key Commands
- `npm run dev:desktop` - Start UI development
- `npm run dev:manager` - Start daemon development
- `npm test` - Run all tests
- `npm run e2e` - Run E2E tests
- `npm run build` - Build all components

### Architecture
- `apps/desktop`: Electron + React UI
- `services/manager`: Go-based daemon
- `packages/shared`: Shared types and utilities

## Security

- All secrets stored in OS keychain
- HTTP servers bound to loopback only
- Sandboxed server installations
- Regular security audits

## Contributing

1. Check existing issues or create a new one
2. Fork the repository
3. Create a feature branch (`git checkout -b feature/amazing-feature`)
4. Commit changes (`git commit -m 'feat: add amazing feature'`)
5. Push to branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for detailed guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- File an issue for bugs or feature requests
- Check [Known Issues](docs/KNOWN_ISSUES.md) for common problems
- Review documentation in `docs/` for detailed guides
