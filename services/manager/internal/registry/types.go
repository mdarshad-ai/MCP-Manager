package registry

import (
    "fmt"
    "time"
)

type Registry struct {
    Version string   `json:"version"`
    Servers []Server `json:"servers"`
}

type Server struct {
    Name     string        `json:"name"`
    Slug     string        `json:"slug"`
    Source   Source        `json:"source"`
    Runtime  Runtime       `json:"runtime"`
    Entry    Entry         `json:"entry"`
    Perms    *Perms        `json:"permissions,omitempty"`
    Auto     *Autostart    `json:"autostart,omitempty"`
    Health   Health        `json:"health"`
    Clients  Clients       `json:"clients"`
    External *ExternalInfo `json:"external,omitempty"`
}

type Source struct {
    Type string `json:"type"`
    URI  string `json:"uri"`
}

type Runtime struct {
    Kind   string       `json:"kind"` // "node", "python", "binary", "external"
    Node   *NodeRuntime `json:"node,omitempty"`
    Python *PyRuntime   `json:"python,omitempty"`
}

type NodeRuntime struct { PackageManager string `json:"packageManager"` }
type PyRuntime struct {
    Manager string `json:"manager"`
    Venv    bool   `json:"venv"`
}

type Entry struct {
    Transport string            `json:"transport"`
    Command   string            `json:"command"`
    Args      []string          `json:"args,omitempty"`
    Env       map[string]string `json:"env,omitempty"`
}

type Perms struct {
    FS  []string `json:"fs,omitempty"`
    Net []string `json:"net,omitempty"`
}

type Autostart struct {
    Enabled bool   `json:"enabled"`
    Scope   string `json:"scope"`
}

type Health struct {
    Probe         string `json:"probe"`
    Method        string `json:"method"`
    IntervalSec   int    `json:"intervalSec"`
    TimeoutSec    int    `json:"timeoutSec"`
    RestartPolicy string `json:"restartPolicy"`
    MaxRestarts   int    `json:"maxRestarts"`
}

type Clients struct {
    ClaudeDesktop *ClientFlag `json:"claudeDesktop,omitempty"`
    CursorGlobal  *ClientFlag `json:"cursorGlobal,omitempty"`
    Continue      *ClientFlag `json:"continue,omitempty"`
}

type ClientFlag struct { Enabled bool `json:"enabled"` }

// ExternalInfo contains information for external/cloud MCP services
type ExternalInfo struct {
    Provider      string                 `json:"provider"`      // e.g., "notion", "slack", "github"
    DisplayName   string                 `json:"displayName"`   // User-friendly name for UI display
    APIEndpoint   string                 `json:"apiEndpoint"`   // API URL for health checks
    AuthType      string                 `json:"authType"`      // "api_key", "oauth2", "basic"
    CredentialRef string                 `json:"credentialRef"` // Reference to stored credentials
    Config        map[string]interface{} `json:"config,omitempty"`        // Provider-specific configuration
    LastSync      *time.Time             `json:"lastSync,omitempty"`      // Last successful synchronization
    Status        ExternalStatus         `json:"status"`        // Detailed status information
    
    // Legacy fields for backward compatibility
    APIKey      string            `json:"apiKey,omitempty"`      // Deprecated: use CredentialRef
    Credentials map[string]string `json:"credentials,omitempty"` // Deprecated: use CredentialRef
    WebhookURL  string            `json:"webhookUrl,omitempty"`  // For services that use webhooks
}

// ExternalStatus provides detailed status tracking for external servers
type ExternalStatus struct {
    State        string     `json:"state"`        // "active", "inactive", "error", "connecting", "syncing"
    Message      string     `json:"message"`      // Human-readable status message
    LastChecked  *time.Time `json:"lastChecked"`  // Last health check timestamp
    ResponseTime *int64     `json:"responseTime"` // Response time in milliseconds
}

// CredentialRequirement defines what credentials a provider needs
type CredentialRequirement struct {
    Key         string `json:"key"`         // Credential key name
    DisplayName string `json:"displayName"` // User-friendly name
    Type        string `json:"type"`        // "string", "secret", "url", "select"
    Required    bool   `json:"required"`    // Whether this credential is required
    Description string `json:"description"` // Help text for users
    Options     []string `json:"options,omitempty"` // For "select" type
    Default     string   `json:"default,omitempty"` // Default value
    Validation  string   `json:"validation,omitempty"` // Regex pattern for validation
}

// ProviderTemplate defines configuration templates for external providers
type ProviderTemplate struct {
    ID                   string                   `json:"id"`                   // Unique provider identifier
    Name                 string                   `json:"name"`                 // Display name
    Description          string                   `json:"description"`          // Provider description
    Category             string                   `json:"category"`             // e.g., "productivity", "development", "communication"
    AuthType             string                   `json:"authType"`             // "api_key", "oauth2", "basic"
    APIEndpoint          string                   `json:"apiEndpoint"`          // Base API URL
    CredentialRequirements []CredentialRequirement `json:"credentialRequirements"` // Required credentials
    ConfigSchema         map[string]interface{}   `json:"configSchema"`         // JSON schema for configuration
    DefaultConfig        map[string]interface{}   `json:"defaultConfig"`        // Default configuration values
    HealthCheckPath      string                   `json:"healthCheckPath"`      // Path for health checks
    Documentation        string                   `json:"documentation"`        // URL to documentation
    IconURL              string                   `json:"iconUrl,omitempty"`    // Provider icon URL
    Tags                 []string                 `json:"tags,omitempty"`       // Search tags
}

// Methods for Server struct

// IsExternal returns true if this server is an external/cloud MCP service
func (s *Server) IsExternal() bool {
    return s.External != nil
}

// GetExternalConfig returns the external server configuration
func (s *Server) GetExternalConfig() *ExternalInfo {
    return s.External
}

// ValidateExternalSetup validates that the external server has required configuration
func (s *Server) ValidateExternalSetup() error {
    if !s.IsExternal() {
        return nil // Not an external server, no validation needed
    }

    ext := s.External
    if ext.Provider == "" {
        return fmt.Errorf("external server missing provider")
    }
    if ext.AuthType == "" {
        return fmt.Errorf("external server missing auth type")
    }
    if ext.APIEndpoint == "" {
        return fmt.Errorf("external server missing API endpoint")
    }
    
    // Check for credential reference or legacy credentials
    if ext.CredentialRef == "" && ext.APIKey == "" && len(ext.Credentials) == 0 {
        return fmt.Errorf("external server missing credentials")
    }

    return nil
}

// Methods for ExternalInfo struct

// IsActive returns true if the external server is in an active state
func (e *ExternalInfo) IsActive() bool {
    return e.Status.State == "active"
}

// NeedsSync returns true if the server hasn't been synced recently
func (e *ExternalInfo) NeedsSync(maxAge time.Duration) bool {
    if e.LastSync == nil {
        return true
    }
    return time.Since(*e.LastSync) > maxAge
}

// IsHealthy returns true if the last health check was successful and recent
func (e *ExternalInfo) IsHealthy(maxAge time.Duration) bool {
    if e.Status.LastChecked == nil {
        return false
    }
    if time.Since(*e.Status.LastChecked) > maxAge {
        return false // Too old
    }
    return e.Status.State == "active" || e.Status.State == "syncing"
}

// GetDisplayName returns the display name or falls back to provider name
func (e *ExternalInfo) GetDisplayName() string {
    if e.DisplayName != "" {
        return e.DisplayName
    }
    return e.Provider
}

// UpdateStatus updates the external status with new information
func (e *ExternalInfo) UpdateStatus(state, message string, responseTime *int64) {
    now := time.Now()
    e.Status.State = state
    e.Status.Message = message
    e.Status.LastChecked = &now
    e.Status.ResponseTime = responseTime
    
    if state == "active" {
        e.LastSync = &now
    }
}

// Methods for ProviderTemplate struct

// ValidateCredentialRequirements validates that all required credentials are provided
func (p *ProviderTemplate) ValidateCredentialRequirements(credentials map[string]string) []string {
    var missing []string
    
    for _, req := range p.CredentialRequirements {
        if req.Required {
            if value, exists := credentials[req.Key]; !exists || value == "" {
                missing = append(missing, req.Key)
            }
        }
    }
    
    return missing
}

// GetCredentialRequirement returns a specific credential requirement by key
func (p *ProviderTemplate) GetCredentialRequirement(key string) *CredentialRequirement {
    for i, req := range p.CredentialRequirements {
        if req.Key == key {
            return &p.CredentialRequirements[i]
        }
    }
    return nil
}

// CreateExternalInfo creates an ExternalInfo struct from this template
func (p *ProviderTemplate) CreateExternalInfo(displayName, credentialRef string, config map[string]interface{}) *ExternalInfo {
    return &ExternalInfo{
        Provider:      p.ID,
        DisplayName:   displayName,
        APIEndpoint:   p.APIEndpoint,
        AuthType:      p.AuthType,
        CredentialRef: credentialRef,
        Config:        config,
        Status: ExternalStatus{
            State:   "inactive",
            Message: "Not configured",
        },
    }
}

