# External MCP Server Implementation Plan

## Project Overview

This document outlines the comprehensive implementation plan for adding external MCP server support (like Notion MCP, Slack MCP, GitHub MCP) to MCP Manager. The plan is designed for agentic implementation with clear phases, dependencies, and deliverables.

## Current Status Assessment

### ✅ Already Implemented (30% Complete)
- **Registry Types**: `ExternalInfo` struct with provider, API endpoint, credentials
- **Runtime Support**: `"external"` runtime kind in Go types
- **Health Monitoring**: Complete `ExternalHealthChecker` with provider endpoints
- **Provider Endpoints**: Built-in health check URLs for major services

### ❌ Missing Components (70% Remaining)
- JSON Schema validation for external servers
- Backend API integration for external server management
- Frontend UI components and flows
- Credential storage and management
- Installation workflows for external services

## System Architecture Design

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React)                        │
├─────────────────────────────────────────────────────────────┤
│ • ExternalServerForm       • CredentialManager             │
│ • ExternalServerList       • ProviderTemplates             │
│ • HealthDashboard          • ConnectionWizard              │
└─────────────────────┬───────────────────────────────────────┘
                      │ HTTP API
┌─────────────────────┴───────────────────────────────────────┐
│                  Go Backend (HTTP API)                     │
├─────────────────────────────────────────────────────────────┤
│ • ExternalServerController  • CredentialVault              │
│ • ExternalHealthManager     • ProviderRegistry             │
│ • WebhookHandler            • ConfigWriter                 │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────┴───────────────────────────────────────┐
│                    Data Layer                              │
├─────────────────────────────────────────────────────────────┤
│ • Registry (JSON)           • Settings Store               │
│ • Health Monitoring         • Log Management               │
│ • OS Keychain Integration   • Client Configs               │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Plan

### Phase 1: Schema & Foundation (2-3 days)
**Priority: High | Dependencies: None**

#### 1.1 Update JSON Schema
- **File**: `packages/shared/schema/registry.schema.json`
- **Tasks**:
  - Add `"external"` to runtime kinds enum
  - Define external server properties validation
  - Add provider-specific credential schemas
  - Include webhook configuration options

#### 1.2 Extend Go Types
- **Files**: 
  - `services/manager/internal/registry/types.go`
  - `services/manager/internal/settings/store.go`
- **Tasks**:
  - Add external server configuration methods
  - Extend settings with external service preferences
  - Add credential storage interfaces

#### 1.3 Create Provider Registry
- **File**: `services/manager/internal/providers/registry.go` (new)
- **Tasks**:
  - Define provider templates (Notion, Slack, GitHub, etc.)
  - Implement credential requirements mapping
  - Add provider-specific health check logic

### Phase 2: Backend API Implementation (3-4 days)
**Priority: High | Dependencies: Phase 1**

#### 2.1 External Server Controller
- **File**: `services/manager/internal/httpapi/external.go` (new)
- **Endpoints**:
  ```go
  POST   /v1/external/servers          // Create external server
  GET    /v1/external/servers          // List external servers
  GET    /v1/external/servers/{slug}   // Get external server details
  PUT    /v1/external/servers/{slug}   // Update external server
  DELETE /v1/external/servers/{slug}   // Remove external server
  POST   /v1/external/servers/{slug}/test // Test connection
  ```

#### 2.2 Credential Management API
- **File**: `services/manager/internal/httpapi/credentials.go` (new)
- **Endpoints**:
  ```go
  POST   /v1/credentials              // Store credentials securely
  GET    /v1/credentials/{provider}   // Get credential requirements
  PUT    /v1/credentials/{provider}   // Update credentials
  DELETE /v1/credentials/{provider}   // Remove credentials
  POST   /v1/credentials/validate     // Validate credentials
  ```

#### 2.3 Provider Templates API
- **File**: `services/manager/internal/httpapi/providers.go` (new)
- **Endpoints**:
  ```go
  GET    /v1/providers                // List available providers
  GET    /v1/providers/{name}         // Get provider template
  POST   /v1/providers/{name}/setup   // Initialize provider setup
  ```

#### 2.4 Integrate External Health Monitoring
- **Files**: 
  - `services/manager/internal/health/monitor.go`
  - `services/manager/cmd/manager/main.go`
- **Tasks**:
  - Integrate `ExternalHealthChecker` into main monitoring loop
  - Add external server health endpoints to HTTP API
  - Implement automatic credential refresh logic

### Phase 3: Credential & Security Layer (2-3 days)
**Priority: High | Dependencies: Phase 2**

#### 3.1 Secure Credential Storage
- **File**: `services/manager/internal/vault/keychain.go` (new)
- **Tasks**:
  - Implement OS keychain integration (macOS Keychain, Windows Credential Manager)
  - Add encryption for stored credentials
  - Implement credential expiration and refresh logic

#### 3.2 OAuth2 Flow Support
- **File**: `services/manager/internal/auth/oauth2.go` (new)
- **Tasks**:
  - Implement OAuth2 authorization code flow
  - Add token refresh mechanisms
  - Support for provider-specific auth flows

#### 3.3 Webhook Handling
- **File**: `services/manager/internal/webhooks/handler.go` (new)
- **Tasks**:
  - HTTP server for receiving webhooks
  - Webhook signature verification
  - Event routing to appropriate handlers

### Phase 4: Frontend Implementation (4-5 days)
**Priority: High | Dependencies: Phase 2**

#### 4.1 External Server Management UI
- **Files**: 
  - `apps/desktop/src/pages/ExternalServers.tsx` (new)
  - `apps/desktop/src/components/ExternalServerCard.tsx` (new)
- **Components**:
  - External server list with status indicators
  - Add/Edit external server forms
  - Connection testing interface
  - Health monitoring dashboard

#### 4.2 Provider Setup Wizard
- **Files**: 
  - `apps/desktop/src/components/ProviderWizard.tsx` (new)
  - `apps/desktop/src/components/providers/` (new directory)
- **Components**:
  - Multi-step setup wizard
  - Provider-specific configuration forms
  - Credential input with validation
  - Connection testing and verification

#### 4.3 Credential Management Interface
- **File**: `apps/desktop/src/pages/Credentials.tsx` (new)
- **Components**:
  - Secure credential storage interface
  - OAuth2 authorization flows
  - Credential status and expiration monitoring
  - Credential testing and validation

#### 4.4 Update Navigation & Routing
- **Files**: 
  - `apps/desktop/src/App.tsx`
  - `apps/desktop/src/components/Sidebar.tsx`
- **Tasks**:
  - Add external servers navigation
  - Update routing for new pages
  - Add external server status to sidebar

### Phase 5: Integration & Polish (2-3 days)
**Priority: Medium | Dependencies: Phase 4**

#### 5.1 Client Configuration Updates
- **Files**: 
  - `services/manager/internal/clients/writers.go`
  - `services/manager/internal/clients/detect.go`
- **Tasks**:
  - Update Claude Desktop config writer for external servers
  - Add Cursor integration for external MCPs
  - Update Continue.dev configuration support

#### 5.2 Installation Workflow Integration
- **Files**: 
  - `services/manager/internal/install/external.go` (new)
  - `apps/desktop/src/pages/Install.tsx`
- **Tasks**:
  - Add external server installation to main install flow
  - Update install UI with external server options
  - Integrate with existing validation system

#### 5.3 Logging & Monitoring
- **Files**: 
  - `services/manager/internal/logs/external.go` (new)
  - `apps/desktop/src/pages/Logs.tsx`
- **Tasks**:
  - Add external server logging support
  - Update log viewer for external server logs
  - Add external server metrics to dashboard

### Phase 6: Testing & Documentation (1-2 days)
**Priority: Medium | Dependencies: Phase 5**

#### 6.1 Test Coverage
- **Files**: Various `*_test.go` files
- **Tasks**:
  - Unit tests for external server management
  - Integration tests for credential flows
  - End-to-end tests for provider setup

#### 6.2 Documentation Updates
- **Files**: 
  - `docs/external-servers.md` (new)
  - `README.md` updates
- **Tasks**:
  - External server setup guides
  - Provider-specific documentation
  - API documentation updates

## Data Models & API Contracts

### Registry Extension
```go
type Server struct {
    // ... existing fields
    External *ExternalInfo `json:"external,omitempty"`
}

type ExternalInfo struct {
    Provider       string                 `json:"provider"`
    DisplayName    string                 `json:"displayName"`
    APIEndpoint    string                 `json:"apiEndpoint"`
    HealthEndpoint string                 `json:"healthEndpoint,omitempty"`
    AuthType       string                 `json:"authType"` // "api_key", "oauth2", "basic"
    CredentialRef  string                 `json:"credentialRef"`
    Config         map[string]interface{} `json:"config,omitempty"`
    WebhookURL     string                 `json:"webhookUrl,omitempty"`
    Status         ExternalStatus         `json:"status"`
    LastSync       *time.Time             `json:"lastSync,omitempty"`
}

type ExternalStatus struct {
    State        string    `json:"state"` // "connected", "disconnected", "error", "setup_required"
    Message      string    `json:"message,omitempty"`
    LastChecked  time.Time `json:"lastChecked"`
    ResponseTime int64     `json:"responseTime"` // milliseconds
}
```

### API Request/Response Models
```go
// Create External Server Request
type CreateExternalServerRequest struct {
    Name        string                 `json:"name"`
    Provider    string                 `json:"provider"`
    Config      map[string]interface{} `json:"config"`
    Credentials map[string]string      `json:"credentials"`
}

// Provider Template Response
type ProviderTemplate struct {
    Name           string                    `json:"name"`
    DisplayName    string                    `json:"displayName"`
    Description    string                    `json:"description"`
    AuthType       string                    `json:"authType"`
    HealthEndpoint string                    `json:"healthEndpoint"`
    ConfigSchema   map[string]interface{}    `json:"configSchema"`
    CredentialReqs []CredentialRequirement   `json:"credentialRequirements"`
}

type CredentialRequirement struct {
    Key         string `json:"key"`
    Label       string `json:"label"`
    Type        string `json:"type"` // "text", "password", "url"
    Required    bool   `json:"required"`
    Description string `json:"description,omitempty"`
}
```

## Provider Templates

### Built-in Providers
```go
var BuiltinProviders = map[string]ProviderTemplate{
    "notion": {
        Name:           "notion",
        DisplayName:    "Notion",
        Description:    "Access and manage Notion databases and pages",
        AuthType:       "api_key",
        HealthEndpoint: "https://api.notion.com/v1/users/me",
        CredentialReqs: []CredentialRequirement{
            {Key: "api_key", Label: "Integration Token", Type: "password", Required: true},
        },
    },
    "slack": {
        Name:           "slack",
        DisplayName:    "Slack",
        Description:    "Interact with Slack channels and messages",
        AuthType:       "oauth2",
        HealthEndpoint: "https://slack.com/api/auth.test",
        CredentialReqs: []CredentialRequirement{
            {Key: "bot_token", Label: "Bot User OAuth Token", Type: "password", Required: true},
        },
    },
    "github": {
        Name:           "github",
        DisplayName:    "GitHub",
        Description:    "Access GitHub repositories and issues",
        AuthType:       "api_key",
        HealthEndpoint: "https://api.github.com/user",
        CredentialReqs: []CredentialRequirement{
            {Key: "token", Label: "Personal Access Token", Type: "password", Required: true},
        },
    },
}
```

## Frontend Component Architecture

### Page Components
```typescript
// External Servers Management
<ExternalServers>
  <ExternalServerList>
    <ExternalServerCard /> // Per server
  </ExternalServerList>
  <AddExternalServerButton />
</ExternalServers>

// Provider Setup Wizard
<ProviderWizard>
  <ProviderSelection />
  <CredentialConfiguration />
  <ConnectionTesting />
  <Summary />
</ProviderWizard>

// Credential Management
<CredentialManager>
  <CredentialList />
  <CredentialForm />
  <CredentialStatus />
</CredentialManager>
```

### State Management
```typescript
// External Server State
interface ExternalServerState {
  servers: ExternalServer[]
  providers: ProviderTemplate[]
  credentials: CredentialStatus[]
  loading: boolean
  error: string | null
}

// Provider Setup State
interface ProviderSetupState {
  selectedProvider: string | null
  configValues: Record<string, any>
  credentials: Record<string, string>
  currentStep: number
  isValid: boolean
  testing: boolean
  connected: boolean
}
```

## Security Considerations

### Credential Storage
- Use OS keychain for secure credential storage
- Encrypt credentials at rest with AES-256
- Never log or transmit credentials in plain text
- Implement credential rotation and expiration

### API Security
- HTTPS-only for all external API communications
- Validate all external server configurations
- Sanitize user inputs and configuration values
- Implement rate limiting for external API calls

### Webhook Security
- Verify webhook signatures when available
- Use HTTPS-only endpoints for webhooks
- Implement webhook replay attack prevention
- Validate webhook payloads against expected schemas

## Error Handling & Monitoring

### Health Check Scenarios
- **Healthy**: External service responding correctly
- **Degraded**: Slow response times or intermittent errors
- **Unhealthy**: Service unavailable or authentication failed
- **Setup Required**: Missing or invalid credentials

### Error Categories
- **Configuration Errors**: Invalid settings or missing required fields
- **Authentication Errors**: Invalid credentials or expired tokens
- **Network Errors**: Connection timeouts or DNS resolution failures
- **API Errors**: Service-specific errors or rate limiting

### Monitoring & Alerts
- Track external service response times and availability
- Log failed authentication attempts and configuration errors
- Monitor credential expiration and refresh status
- Alert on extended service unavailability

## Performance Considerations

### Caching Strategy
- Cache provider health check results for 30 seconds
- Cache credential validation results for 5 minutes
- Implement provider configuration template caching

### Rate Limiting
- Respect external service rate limits
- Implement exponential backoff for failed requests
- Queue health checks to avoid overwhelming services

### Resource Management
- Limit concurrent external API requests
- Implement timeout controls for all external calls
- Clean up unused webhook endpoints and credentials

## Migration Strategy

### Existing Installation Compatibility
- Maintain backward compatibility with existing registry format
- Add migration logic for upgrading registry schema
- Provide rollback capability for failed migrations

### Data Migration
- Preserve existing server configurations
- Migrate settings to new format incrementally
- Validate data integrity during migration process

## Success Criteria

### Functional Requirements
- [ ] Users can add external MCP servers through UI
- [ ] Credential management is secure and user-friendly
- [ ] Health monitoring works for external services
- [ ] Client configurations are updated automatically
- [ ] External servers appear in dashboard alongside local servers

### Technical Requirements
- [ ] All API endpoints return appropriate status codes and error messages
- [ ] External server health checks complete within 10 seconds
- [ ] Credential storage uses OS keychain securely
- [ ] UI is responsive and provides clear feedback during operations
- [ ] External server configurations persist across application restarts

### User Experience Requirements
- [ ] Setup wizard completes in under 3 minutes for major providers
- [ ] Clear error messages guide users through troubleshooting
- [ ] External servers integrate seamlessly with existing workflow
- [ ] Health status is clearly indicated in all relevant UI components

## Dependencies & Prerequisites

### External Dependencies
- OS keychain libraries for secure credential storage
- HTTP client libraries with proper SSL/TLS support
- JSON schema validation libraries

### Development Prerequisites
- Go 1.22+ with required modules
- Node.js 18+ with required packages
- Understanding of OAuth2 flows and API authentication
- Familiarity with React hooks and state management

## Risk Assessment & Mitigation

### High Risk Areas
1. **Credential Security**: Implement defense-in-depth security measures
2. **External API Reliability**: Design robust retry and fallback mechanisms
3. **Schema Migration**: Test thoroughly with various registry configurations

### Mitigation Strategies
- Comprehensive testing with real external services
- Incremental rollout with feature flags
- Detailed logging and monitoring for troubleshooting
- Fallback mechanisms for external service unavailability

## Conclusion

This implementation plan provides a comprehensive roadmap for adding external MCP server support to MCP Manager. The phased approach ensures steady progress while maintaining system stability and user experience quality.

The plan is designed for agentic implementation with clear deliverables, well-defined interfaces, and comprehensive testing coverage. Each phase builds upon the previous one, allowing for incremental delivery and testing.

Total estimated effort: **14-20 development days** across 6 phases, with potential for parallel development in some areas.