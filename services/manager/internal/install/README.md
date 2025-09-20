# MCP Manager Installation System

This directory contains a comprehensive installation system for MCP (Model Context Protocol) servers that supports multiple installation sources with detailed progress tracking, authentication, dependency management, and automatic server registration.

## Architecture Overview

### Core Components

1. **Source-Specific Installers**
   - `git.go` - Git repository installation with authentication support
   - `npm.go` - NPM package installation with multiple package managers
   - `pip.go` - Python package installation with virtual environments

2. **Job Management System**
   - `jobs.go` - Advanced job tracking with detailed progress reporting
   - `installers.go` - Concrete installer implementations with job integration

3. **Registry Integration**
   - `registry_integration.go` - Server registration and manifest creation
   - Automatic server entry creation in the MCP registry

4. **HTTP API Integration**
   - Enhanced API endpoints in `../httpapi/install_jobs.go`
   - Backward compatibility with existing installation API

### Installation Flow

1. **Job Creation** - Create installation job with source-specific options
2. **Validation** - Validate source accessibility and prerequisites
3. **Installation** - Execute installation with progress tracking
4. **Configuration** - Set up runtime environment and entry points
5. **Registration** - Register server in MCP registry
6. **Finalization** - Create manifests and validate installation

## Features

### Git Installation (`git.go`)
- **Authentication**: SSH keys, GitHub/GitLab tokens, basic auth
- **Repository Options**: Specific branches, tags, commits, shallow clones
- **Submodules**: Recursive cloning support
- **Post-Install**: Custom commands after cloning
- **Runtime Detection**: Automatic detection of Node.js, Python, Go, Rust
- **Dependency Installation**: Automatic dependency installation based on runtime

### NPM Installation (`npm.go`)
- **Package Manager Support**: npm, yarn, pnpm with auto-detection
- **Authentication**: Registry tokens, custom registries
- **Installation Options**: Global/local, production/development dependencies
- **Entry Point Detection**: Automatic detection from package.json bin/main fields
- **Environment Setup**: NODE_PATH and module resolution
- **Package Information**: Full package metadata extraction

### Pip Installation (`pip.go`)
- **Virtual Environments**: Automatic venv creation and management
- **Package Managers**: pip, pipenv, poetry support
- **Isolated Installation**: pipx support for system-wide isolation
- **Entry Points**: Console scripts and module detection
- **Python Versions**: Version validation and compatibility checking
- **Requirements**: Support for requirements.txt and setup.py

### Job Management (`jobs.go`)
- **Detailed Progress**: Stage-based progress with percentage completion
- **Logging System**: Structured logging with levels and timestamps  
- **Real-time Updates**: Live progress and log streaming
- **Cancellation**: Graceful job cancellation support
- **Persistence**: Job status persistence and recovery
- **Cleanup**: Automatic cleanup of old completed jobs

## API Endpoints

### Advanced Installation API

#### Start Installation
```http
POST /v1/install/start
Content-Type: application/json

{
  "type": "git|npm|pip",
  "uri": "source-uri",
  "slug": "server-name",
  "options": {
    // Source-specific options
  }
}
```

#### Monitor Job Progress
```http
GET /v1/install/logs?id={jobId}
```

#### Finalize Installation
```http
POST /v1/install/finalize?id={jobId}
```

#### List All Jobs
```http
GET /v1/install/list
```

### Installation Options

#### Git Options
```json
{
  "branch": "main",
  "token": "github_token",
  "sshKey": "/path/to/key",
  "postInstall": ["npm install", "npm run build"],
  "environment": {"NODE_ENV": "production"}
}
```

#### NPM Options
```json
{
  "version": "^1.0.0",
  "preferManager": "pnpm",
  "registry": "https://registry.npmjs.org/",
  "token": "npm_token",
  "development": false
}
```

#### Pip Options
```json
{
  "version": ">=1.0.0",
  "useVenv": true,
  "pythonVersion": "3.9",
  "extras": ["dev", "test"],
  "indexUrl": "https://pypi.org/simple/"
}
```

## Directory Structure

After installation, each MCP server is organized under `~/.mcp/servers/{slug}/`:

```
~/.mcp/servers/{slug}/
├── install/          # Original source code/files
├── runtime/          # Runtime dependencies (node_modules, venv, etc.)
├── bin/              # Executable scripts
└── manifest.json     # Installation metadata
```

## Integration with MCP Registry

The installation system automatically integrates with the MCP server registry:

1. **Server Registration**: Automatically creates registry entries
2. **Entry Point Configuration**: Sets up proper command and arguments
3. **Runtime Metadata**: Records runtime type and package manager
4. **Health Monitoring**: Configures health check parameters
5. **Client Integration**: Prepares for MCP client configuration

## Error Handling and Recovery

- **Validation Failures**: Pre-installation validation prevents failed installs
- **Installation Failures**: Automatic cleanup of partial installations
- **Network Issues**: Retry logic for network-dependent operations
- **Authentication Errors**: Clear error messages for auth failures
- **Dependency Conflicts**: Isolated environments prevent conflicts

## Monitoring and Logging

- **Structured Logging**: JSON-formatted logs with metadata
- **Progress Tracking**: Real-time progress updates with percentages  
- **Error Reporting**: Detailed error messages with context
- **Performance Metrics**: Installation timing and resource usage
- **Audit Trail**: Complete history of installation operations

## Extension Points

The system is designed for extensibility:

1. **New Source Types**: Implement the `Installer` interface
2. **Custom Validators**: Add validation logic for new sources
3. **Authentication Methods**: Extend authentication support
4. **Post-Install Hooks**: Add custom post-installation logic
5. **Registry Integrations**: Support additional registry formats

## Usage Examples

### Installing from Git
```bash
curl -X POST http://localhost:8080/v1/install/start \
  -H "Content-Type: application/json" \
  -d '{
    "type": "git",
    "uri": "https://github.com/anthropics/mcp-server-example",
    "slug": "example-server",
    "options": {
      "branch": "main",
      "postInstall": ["npm install"]
    }
  }'
```

### Installing from NPM
```bash
curl -X POST http://localhost:8080/v1/install/start \
  -H "Content-Type: application/json" \
  -d '{
    "type": "npm",
    "uri": "@anthropic/mcp-server-example", 
    "slug": "npm-example",
    "options": {
      "version": "latest",
      "preferManager": "npm"
    }
  }'
```

### Installing from PyPI
```bash
curl -X POST http://localhost:8080/v1/install/start \
  -H "Content-Type: application/json" \
  -d '{
    "type": "pip",
    "uri": "anthropic-mcp-server",
    "slug": "python-example",
    "options": {
      "useVenv": true,
      "pythonVersion": "3.9"
    }
  }'
```

## Development and Testing

For development and testing of the installation system:

1. **Unit Tests**: Test individual installer components
2. **Integration Tests**: Test full installation workflows  
3. **Mock Runners**: Use mock command runners for testing
4. **Local Testing**: Test with local repositories and packages
5. **Error Scenarios**: Test failure cases and recovery

## Security Considerations

- **Authentication**: Secure handling of tokens and credentials
- **Code Execution**: Sandboxed execution of post-install commands
- **File Permissions**: Proper file and directory permissions
- **Network Security**: HTTPS-only for package downloads
- **Input Validation**: Sanitization of all user inputs

## Performance Optimizations

- **Parallel Processing**: Concurrent job execution
- **Caching**: Package and dependency caching
- **Incremental Updates**: Only update changed components
- **Resource Limits**: Memory and CPU usage controls
- **Cleanup**: Automatic cleanup of temporary files

This installation system provides a robust, secure, and user-friendly way to install MCP servers from various sources while maintaining compatibility with the existing MCP ecosystem.