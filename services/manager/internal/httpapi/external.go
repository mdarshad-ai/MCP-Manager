package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mcp/manager/internal/providers"
	"mcp/manager/internal/registry"
)

// ExternalServerRequest represents the request payload for creating/updating external servers
type ExternalServerRequest struct {
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Provider    string                 `json:"provider"`
	DisplayName string                 `json:"displayName,omitempty"`
	Credentials map[string]string      `json:"credentials"`
	Config      map[string]interface{} `json:"config,omitempty"`
	AutoStart   bool                   `json:"autoStart,omitempty"`
}

// ExternalServerResponse represents the response for external server operations
type ExternalServerResponse struct {
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Provider    string                 `json:"provider"`
	DisplayName string                 `json:"displayName"`
	Status      registry.ExternalStatus `json:"status"`
	Config      map[string]interface{} `json:"config,omitempty"`
	AutoStart   bool                   `json:"autoStart"`
	LastSync    *time.Time             `json:"lastSync,omitempty"`
	APIEndpoint string                 `json:"apiEndpoint"`
	AuthType    string                 `json:"authType"`
}

// ExternalServerTestResponse represents the response for connection testing
type ExternalServerTestResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ResponseTime *int64 `json:"responseTime,omitempty"`
}

// ExternalProviderResponse represents provider template information
type ExternalProviderResponse struct {
	Name           string                    `json:"name"`
	DisplayName    string                    `json:"displayName"`
	Description    string                    `json:"description"`
	AuthType       string                    `json:"authType"`
	BaseURL        string                    `json:"baseUrl"`
	HealthEndpoint string                    `json:"healthEndpoint"`
	Credentials    []providers.Credential    `json:"credentials"`
	ConfigSchema   map[string]interface{}    `json:"configSchema,omitempty"`
	Tags           []string                  `json:"tags,omitempty"`
}

// handleExternalMCPs handles requests to /v1/external/servers
func (s *Server) handleExternalMCPs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListExternalServers(w, r)
	case http.MethodPost:
		s.handleCreateExternalServer(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleExternalMCPActions handles requests to /v1/external/servers/{slug} and /v1/external/servers/{slug}/test
func (s *Server) handleExternalMCPActions(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/external/servers/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slug := parts[0]

	// Handle /v1/external/servers/{slug}/test
	if len(parts) == 2 && parts[1] == "test" {
		if r.Method == http.MethodPost {
			s.handleTestExternalServer(w, r, slug)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle /v1/external/servers/{slug}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleGetExternalServer(w, r, slug)
		case http.MethodPut:
			s.handleUpdateExternalServer(w, r, slug)
		case http.MethodDelete:
			s.handleDeleteExternalServer(w, r, slug)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

// handleListExternalServers handles GET /v1/external/servers
func (s *Server) handleListExternalServers(w http.ResponseWriter, r *http.Request) {
	var externalServers []ExternalServerResponse

	for _, server := range s.reg.Servers {
		if server.IsExternal() {
			ext := server.GetExternalConfig()
			response := ExternalServerResponse{
				Name:        server.Name,
				Slug:        server.Slug,
				Provider:    ext.Provider,
				DisplayName: ext.GetDisplayName(),
				Status:      ext.Status,
				Config:      ext.Config,
				AutoStart:   server.Auto != nil && server.Auto.Enabled,
				LastSync:    ext.LastSync,
				APIEndpoint: ext.APIEndpoint,
				AuthType:    ext.AuthType,
			}
			externalServers = append(externalServers, response)
		}
	}

	writeJSON(w, externalServers)
}

// handleGetExternalServer handles GET /v1/external/servers/{slug}
func (s *Server) handleGetExternalServer(w http.ResponseWriter, r *http.Request, slug string) {
	server := s.findServer(slug)
	if server == nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "Server not found"})
		return
	}

	if !server.IsExternal() {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "Server is not an external server"})
		return
	}

	ext := server.GetExternalConfig()
	response := ExternalServerResponse{
		Name:        server.Name,
		Slug:        server.Slug,
		Provider:    ext.Provider,
		DisplayName: ext.GetDisplayName(),
		Status:      ext.Status,
		Config:      ext.Config,
		AutoStart:   server.Auto != nil && server.Auto.Enabled,
		LastSync:    ext.LastSync,
		APIEndpoint: ext.APIEndpoint,
		AuthType:    ext.AuthType,
	}

	writeJSON(w, response)
}

// handleCreateExternalServer handles POST /v1/external/servers
func (s *Server) ensureCredentialManager() error {
	if s.credentialManager != nil { return nil }
	cm, err := NewCredentialManager()
	if err != nil { return err }
	s.credentialManager = cm
	return nil
}

// handleCreateExternalServer handles POST /v1/external/servers
func (s *Server) handleCreateExternalServer(w http.ResponseWriter, r *http.Request) {
	var req ExternalServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.Name == "" || req.Slug == "" || req.Provider == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "Name, slug, and provider are required"})
		return
	}

	// Check if slug already exists
	if s.findServer(req.Slug) != nil {
		w.WriteHeader(http.StatusConflict)
		writeJSON(w, map[string]string{"error": "Server with this slug already exists"})
		return
	}

	// Validate provider exists
	provider, err := providers.GetProvider(req.Provider)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Unknown provider: %s", req.Provider)})
		return
	}

	// Validate credentials
	if err := providers.ValidateProviderConfig(req.Provider, req.Credentials); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Invalid credentials: %v", err)})
		return
	}

	// Create external info
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Name
	}

	externalInfo := &registry.ExternalInfo{
		Provider:      req.Provider,
		DisplayName:   displayName,
		APIEndpoint:   provider.BaseURL,
		AuthType:      string(provider.AuthType),
		CredentialRef: "",
		Config:        req.Config,
		Status: registry.ExternalStatus{
			State:   "inactive",
			Message: "Created but not tested",
		},
	}

	// Persist credentials securely and set credential reference
	if len(req.Credentials) > 0 {
		if err := s.ensureCredentialManager(); err == nil {
			credRef := fmt.Sprintf("ext:%s:%s", req.Provider, req.Slug)
			_ = s.credentialManager.vault.Store(credRef, req.Credentials)
			externalInfo.CredentialRef = credRef
		}
		// do not persist raw credentials in registry (legacy fields)
		externalInfo.Credentials = nil
		externalInfo.APIKey = ""
	}

	// Create the server entry
	server := registry.Server{
		Name: req.Name,
		Slug: req.Slug,
		Source: registry.Source{
			Type: "external",
			URI:  req.Provider,
		},
		Runtime: registry.Runtime{
			Kind: "external",
		},
		Entry: registry.Entry{
			Transport: "http",
			Command:   "", // Not applicable for external servers
		},
		Health: registry.Health{
			Probe:         "http",
			Method:        "GET",
			IntervalSec:   30,
			TimeoutSec:    10,
			RestartPolicy: "never",
			MaxRestarts:   0,
		},
		External: externalInfo,
	}

	// Add autostart configuration if requested
	if req.AutoStart {
		server.Auto = &registry.Autostart{
			Enabled: true,
			Scope:   "user",
		}
	}

	// Validate the complete server setup
	if err := server.ValidateExternalSetup(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Server validation failed: %v", err)})
		return
	}

	// Add to registry
	s.reg.Servers = append(s.reg.Servers, server)

	// Save registry
	if err := s.saveRegistry(); err != nil {
		// Remove the server we just added
		s.reg.Servers = s.reg.Servers[:len(s.reg.Servers)-1]
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Failed to save registry: %v", err)})
		return
	}

	// Update supervisor with new registry if available
	if s.sup != nil {
		s.sup.UpdateRegistry(s.reg)
	}

	// Add to health monitoring if available
	if s.healthMonitor != nil {
		s.healthMonitor.AddProcess(req.Slug, "http", provider.HealthEndpoint, "")
	}

	// Return the created server
	response := ExternalServerResponse{
		Name:        server.Name,
		Slug:        server.Slug,
		Provider:    externalInfo.Provider,
		DisplayName: externalInfo.GetDisplayName(),
		Status:      externalInfo.Status,
		Config:      externalInfo.Config,
		AutoStart:   req.AutoStart,
		LastSync:    externalInfo.LastSync,
		APIEndpoint: externalInfo.APIEndpoint,
		AuthType:    externalInfo.AuthType,
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, response)
}

// handleUpdateExternalServer handles PUT /v1/external/servers/{slug}
func (s *Server) handleUpdateExternalServer(w http.ResponseWriter, r *http.Request, slug string) {
	server := s.findServer(slug)
	if server == nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "Server not found"})
		return
	}

	if !server.IsExternal() {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "Server is not an external server"})
		return
	}

	var req ExternalServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate provider if changed
	if req.Provider != "" && req.Provider != server.External.Provider {
		provider, err := providers.GetProvider(req.Provider)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"error": fmt.Sprintf("Unknown provider: %s", req.Provider)})
			return
		}
		
		// Validate credentials for new provider
		if err := providers.ValidateProviderConfig(req.Provider, req.Credentials); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"error": fmt.Sprintf("Invalid credentials: %v", err)})
			return
		}

		// Update provider-specific fields
		server.External.Provider = req.Provider
		server.External.APIEndpoint = provider.BaseURL
		server.External.AuthType = string(provider.AuthType)
	} else if len(req.Credentials) > 0 {
		// Validate credentials with existing provider
		if err := providers.ValidateProviderConfig(server.External.Provider, req.Credentials); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"error": fmt.Sprintf("Invalid credentials: %v", err)})
			return
		}
	}

	// Update fields
	if req.Name != "" {
		server.Name = req.Name
	}
	if req.DisplayName != "" {
		server.External.DisplayName = req.DisplayName
	}
	if len(req.Credentials) > 0 {
		// Store updated credentials in vault and reference them
		credRef := server.External.CredentialRef
		if credRef == "" {
			credRef = fmt.Sprintf("ext:%s:%s", server.External.Provider, server.Slug)
		}
		if err := s.ensureCredentialManager(); err == nil {
			_ = s.credentialManager.vault.Store(credRef, req.Credentials)
			server.External.CredentialRef = credRef
		}
		// Do not persist raw credentials in registry
		server.External.Credentials = nil
		server.External.APIKey = ""
		server.External.Credentials = req.Credentials
		// Reset status since credentials changed
		server.External.Status = registry.ExternalStatus{
			State:   "inactive",
			Message: "Credentials updated, needs testing",
		}
	}
	if req.Config != nil {
		server.External.Config = req.Config
	}

	// Update autostart configuration
	if req.AutoStart && server.Auto == nil {
		server.Auto = &registry.Autostart{
			Enabled: true,
			Scope:   "user",
		}
	} else if server.Auto != nil {
		server.Auto.Enabled = req.AutoStart
	}

	// Validate the updated server
	if err := server.ValidateExternalSetup(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Server validation failed: %v", err)})
		return
	}

	// Save registry
	if err := s.saveRegistry(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Failed to save registry: %v", err)})
		return
	}

	// Update supervisor with new registry if available
	if s.sup != nil {
		s.sup.UpdateRegistry(s.reg)
	}

	// Return the updated server
	response := ExternalServerResponse{
		Name:        server.Name,
		Slug:        server.Slug,
		Provider:    server.External.Provider,
		DisplayName: server.External.GetDisplayName(),
		Status:      server.External.Status,
		Config:      server.External.Config,
		AutoStart:   server.Auto != nil && server.Auto.Enabled,
		LastSync:    server.External.LastSync,
		APIEndpoint: server.External.APIEndpoint,
		AuthType:    server.External.AuthType,
	}

	writeJSON(w, response)
}

// handleDeleteExternalServer handles DELETE /v1/external/servers/{slug}
func (s *Server) handleDeleteExternalServer(w http.ResponseWriter, r *http.Request, slug string) {
	// Find server index
	serverIndex := -1
	for i, server := range s.reg.Servers {
		if server.Slug == slug {
			if !server.IsExternal() {
				w.WriteHeader(http.StatusBadRequest)
				writeJSON(w, map[string]string{"error": "Server is not an external server"})
				return
			}
			serverIndex = i
			break
		}
	}

	if serverIndex == -1 {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "Server not found"})
		return
	}

	// Remove from health monitoring first
	if s.healthMonitor != nil {
		s.healthMonitor.RemoveProcess(slug)
	}

	// Delete stored credentials if any (best effort)
	if s.reg.Servers[serverIndex].External != nil {
		credRef := s.reg.Servers[serverIndex].External.CredentialRef
		if credRef != "" {
			_ = s.ensureCredentialManager()
			if s.credentialManager != nil { _ = s.credentialManager.vault.Delete(credRef) }
		}
	}

	// Remove server from registry
	s.reg.Servers = append(s.reg.Servers[:serverIndex], s.reg.Servers[serverIndex+1:]...)

	// Save registry
	if err := s.saveRegistry(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Failed to save registry: %v", err)})
		return
	}

	// Update supervisor with new registry if available
	if s.sup != nil {
		s.sup.UpdateRegistry(s.reg)
	}

	writeJSON(w, map[string]string{"status": "deleted"})
}

// handleTestExternalServer handles POST /v1/external/servers/{slug}/test
func (s *Server) handleTestExternalServer(w http.ResponseWriter, r *http.Request, slug string) {
	server := s.findServer(slug)
	if server == nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "Server not found"})
		return
	}

	if !server.IsExternal() {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "Server is not an external server"})
		return
	}

	ext := server.External
	provider, err := providers.GetProvider(ext.Provider)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ExternalServerTestResponse{
			Success: false,
			Message: fmt.Sprintf("Provider configuration error: %v", err),
		})
		return
	}

	// Perform health check
	start := time.Now()
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request to health endpoint
	req, err := http.NewRequest("GET", provider.HealthEndpoint, nil)
	if err != nil {
		responseTime := time.Since(start).Milliseconds()
		writeJSON(w, ExternalServerTestResponse{
			Success:      false,
			Message:      fmt.Sprintf("Failed to create request: %v", err),
			ResponseTime: &responseTime,
		})
		return
	}

	// Add authentication headers based on provider type
	if ext.Credentials != nil {
		switch provider.AuthType {
		case providers.AuthAPIKey:
			if apiKey, ok := ext.Credentials["api_key"]; ok {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			}
		case providers.AuthOAuth2:
			if accessToken, ok := ext.Credentials["access_token"]; ok {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
			}
		case providers.AuthBasic:
			// Basic auth would be handled differently
		}
	}

	// Perform the request
	resp, err := client.Do(req)
	responseTime := time.Since(start).Milliseconds()

	if err != nil {
		// Update server status
		ext.UpdateStatus("error", fmt.Sprintf("Connection failed: %v", err), nil)
		s.saveRegistry() // Best effort save

		writeJSON(w, ExternalServerTestResponse{
			Success:      false,
			Message:      fmt.Sprintf("Connection failed: %v", err),
			ResponseTime: &responseTime,
		})
		return
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	var message string
	var status string

	if success {
		message = fmt.Sprintf("Connection successful (HTTP %d)", resp.StatusCode)
		status = "active"
	} else {
		message = fmt.Sprintf("Connection failed with HTTP %d", resp.StatusCode)
		status = "error"
	}

	// Update server status
	ext.UpdateStatus(status, message, &responseTime)
	s.saveRegistry() // Best effort save

	// Update health monitoring if available
	if s.healthMonitor != nil {
		if success {
			s.healthMonitor.AddProcess(slug, "http", provider.HealthEndpoint, "")
		} else {
			s.healthMonitor.RemoveProcess(slug)
		}
	}

	writeJSON(w, ExternalServerTestResponse{
		Success:      success,
		Message:      message,
		ResponseTime: &responseTime,
	})
}

// handleListProviders handles GET /v1/external/providers
func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	allProviders := providers.GetAllProviders()
	var providerList []ExternalProviderResponse

	for _, provider := range allProviders {
		response := ExternalProviderResponse{
			Name:           provider.Name,
			DisplayName:    provider.DisplayName,
			Description:    provider.Description,
			AuthType:       string(provider.AuthType),
			BaseURL:        provider.BaseURL,
			HealthEndpoint: provider.HealthEndpoint,
			Credentials:    provider.Credentials,
			ConfigSchema:   provider.ConfigSchema,
			Tags:           provider.Tags,
		}
		providerList = append(providerList, response)
	}

	writeJSON(w, providerList)
}

// handleGetProvider handles GET /v1/external/providers/{name}
func (s *Server) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	providerName := parts[4]
	provider, err := providers.GetProvider(providerName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Provider not found: %s", providerName)})
		return
	}

	response := ExternalProviderResponse{
		Name:           provider.Name,
		DisplayName:    provider.DisplayName,
		Description:    provider.Description,
		AuthType:       string(provider.AuthType),
		BaseURL:        provider.BaseURL,
		HealthEndpoint: provider.HealthEndpoint,
		Credentials:    provider.Credentials,
		ConfigSchema:   provider.ConfigSchema,
		Tags:           provider.Tags,
	}

	writeJSON(w, response)
}

// saveRegistry is a helper function to save the registry to disk
func (s *Server) saveRegistry() error {
	registryPath := os.Getenv("MCP_REGISTRY_PATH")
	if registryPath == "" {
		registryPath = filepath.Join(os.Getenv("HOME"), ".mcp", "registry.json")
	}
	return s.reg.Save(registryPath)
}
