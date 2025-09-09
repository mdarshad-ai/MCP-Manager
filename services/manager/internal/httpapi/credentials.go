package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"mcp/manager/internal/health"
	"mcp/manager/internal/providers"
	"mcp/manager/internal/vault"
)

// CredentialManager handles secure credential operations
type CredentialManager struct {
	vault         *vault.KeychainVault
	healthChecker *health.ExternalHealthChecker
	rateLimiter   *rateLimiter
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager() (*CredentialManager, error) {
	keychainVault, err := vault.NewKeychainVault("mcp-manager")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keychain vault: %w", err)
	}

	return &CredentialManager{
		vault:         keychainVault,
		healthChecker: health.NewExternalHealthChecker(),
		rateLimiter:   newRateLimiter(),
	}, nil
}

// Rate limiter for credential validation attempts
type rateLimiter struct {
	attempts map[string][]time.Time
	mu       sync.RWMutex
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		attempts: make(map[string][]time.Time),
	}
}

// isRateLimited checks if validation attempts are rate limited (max 5 per minute)
func (rl *rateLimiter) isRateLimited(provider string) bool {
	rl.mu.RLock()
	attempts := rl.attempts[provider]
	rl.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)

	// Count attempts in the last minute
	validAttempts := 0
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validAttempts++
		}
	}

	return validAttempts >= 5
}

// recordAttempt records a validation attempt
func (rl *rateLimiter) recordAttempt(provider string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	attempts := rl.attempts[provider]
	
	// Clean old attempts
	cutoff := now.Add(-1 * time.Minute)
	var validAttempts []time.Time
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}
	
	// Add new attempt
	validAttempts = append(validAttempts, now)
	rl.attempts[provider] = validAttempts
}

// API request/response structures

type StoreCredentialsRequest struct {
	Provider    string            `json:"provider"`
	Credentials map[string]string `json:"credentials"`
}

type StoreCredentialsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GetCredentialRequirementsResponse struct {
	Provider     string                  `json:"provider"`
	Credentials  []providers.Credential  `json:"credentials"`
	AuthType     providers.AuthType      `json:"authType"`
	Description  string                  `json:"description"`
}

type UpdateCredentialsRequest struct {
	Credentials map[string]string `json:"credentials"`
}

type UpdateCredentialsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type DeleteCredentialsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ValidateCredentialsRequest struct {
	Provider    string            `json:"provider"`
	Credentials map[string]string `json:"credentials"`
}

type ValidateCredentialsResponse struct {
    Valid       bool   `json:"valid"`
    Status      string `json:"status"`
    Message     string `json:"message"`
    HealthCheck *health.ExternalHealth `json:"healthCheck,omitempty"`
}

type CredentialStatusResponse struct {
    Provider string `json:"provider"`
    Exists   bool   `json:"exists"`
}

// HTTP Handlers

// handleCredentialsStore handles POST /v1/credentials
func (s *Server) handleCredentialsStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req StoreCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding store credentials request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, StoreCredentialsResponse{
			Success: false,
			Message: "Invalid request format",
		})
		return
	}

	if req.Provider == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, StoreCredentialsResponse{
			Success: false,
			Message: "Provider name is required",
		})
		return
	}

	if len(req.Credentials) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, StoreCredentialsResponse{
			Success: false,
			Message: "Credentials are required",
		})
		return
	}

	// Validate provider exists
	if !providers.IsProviderSupported(req.Provider) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, StoreCredentialsResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported provider: %s", req.Provider),
		})
		return
	}

	// Validate credential format
	if err := providers.ValidateProviderConfig(req.Provider, req.Credentials); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, StoreCredentialsResponse{
			Success: false,
			Message: fmt.Sprintf("Credential validation failed: %v", err),
		})
		return
	}

	// Initialize credential manager if not already done
	if s.credentialManager == nil {
		cm, err := NewCredentialManager()
		if err != nil {
			log.Printf("Error initializing credential manager: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, StoreCredentialsResponse{
				Success: false,
				Message: "Internal server error",
			})
			return
		}
		s.credentialManager = cm
	}

	// Store credentials securely
	if err := s.credentialManager.vault.Store(req.Provider, req.Credentials); err != nil {
		log.Printf("Error storing credentials for provider %s: %v", req.Provider, err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, StoreCredentialsResponse{
			Success: false,
			Message: "Failed to store credentials securely",
		})
		return
	}

	log.Printf("[AUDIT] Credentials stored successfully for provider: %s", req.Provider)
	writeJSON(w, StoreCredentialsResponse{
		Success: true,
		Message: "Credentials stored successfully",
	})
}

// handleCredentialsGet handles GET /v1/credentials/{provider}
func (s *Server) handleCredentialsGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	provider := parts[3]
	if provider == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "Provider name is required"})
		return
	}

	// Get credential requirements for the provider
	credentials, err := providers.GetCredentialRequirements(provider)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Provider not found: %s", provider)})
		return
	}

	providerInfo, err := providers.GetProvider(provider)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": fmt.Sprintf("Provider not found: %s", provider)})
		return
	}

	response := GetCredentialRequirementsResponse{
		Provider:    provider,
		Credentials: credentials,
		AuthType:    providerInfo.AuthType,
		Description: providerInfo.Description,
	}

	writeJSON(w, response)
}

// handleCredentialsUpdate handles PUT /v1/credentials/{provider}
func (s *Server) handleCredentialsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	provider := parts[3]
	if provider == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, UpdateCredentialsResponse{
			Success: false,
			Message: "Provider name is required",
		})
		return
	}

	var req UpdateCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding update credentials request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, UpdateCredentialsResponse{
			Success: false,
			Message: "Invalid request format",
		})
		return
	}

	if len(req.Credentials) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, UpdateCredentialsResponse{
			Success: false,
			Message: "Credentials are required",
		})
		return
	}

	// Validate provider exists
	if !providers.IsProviderSupported(provider) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, UpdateCredentialsResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported provider: %s", provider),
		})
		return
	}

	// Initialize credential manager if not already done
	if s.credentialManager == nil {
		cm, err := NewCredentialManager()
		if err != nil {
			log.Printf("Error initializing credential manager: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, UpdateCredentialsResponse{
				Success: false,
				Message: "Internal server error",
			})
			return
		}
		s.credentialManager = cm
	}

	// Update credentials
	if err := s.credentialManager.vault.Update(provider, req.Credentials); err != nil {
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, UpdateCredentialsResponse{
				Success: false,
				Message: fmt.Sprintf("No credentials found for provider: %s", provider),
			})
			return
		}

		log.Printf("Error updating credentials for provider %s: %v", provider, err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, UpdateCredentialsResponse{
			Success: false,
			Message: "Failed to update credentials",
		})
		return
	}

	log.Printf("[AUDIT] Credentials updated successfully for provider: %s", provider)
	writeJSON(w, UpdateCredentialsResponse{
		Success: true,
		Message: "Credentials updated successfully",
	})
}

// handleCredentialsDelete handles DELETE /v1/credentials/{provider}
func (s *Server) handleCredentialsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	provider := parts[3]
	if provider == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, DeleteCredentialsResponse{
			Success: false,
			Message: "Provider name is required",
		})
		return
	}

	// Initialize credential manager if not already done
	if s.credentialManager == nil {
		cm, err := NewCredentialManager()
		if err != nil {
			log.Printf("Error initializing credential manager: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, DeleteCredentialsResponse{
				Success: false,
				Message: "Internal server error",
			})
			return
		}
		s.credentialManager = cm
	}

	// Delete credentials
	if err := s.credentialManager.vault.Delete(provider); err != nil {
		log.Printf("Error deleting credentials for provider %s: %v", provider, err)
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, DeleteCredentialsResponse{
			Success: false,
			Message: "Failed to delete credentials",
		})
		return
	}

	log.Printf("[AUDIT] Credentials deleted successfully for provider: %s", provider)
	writeJSON(w, DeleteCredentialsResponse{
		Success: true,
		Message: "Credentials deleted successfully",
	})
}

// handleCredentialsValidate handles POST /v1/credentials/validate
func (s *Server) handleCredentialsValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req ValidateCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding validate credentials request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ValidateCredentialsResponse{
			Valid:   false,
			Status:  "error",
			Message: "Invalid request format",
		})
		return
	}

	if req.Provider == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ValidateCredentialsResponse{
			Valid:   false,
			Status:  "error",
			Message: "Provider name is required",
		})
		return
	}

	// Initialize credential manager if not already done
	if s.credentialManager == nil {
		cm, err := NewCredentialManager()
		if err != nil {
			log.Printf("Error initializing credential manager: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, ValidateCredentialsResponse{
				Valid:   false,
				Status:  "error",
				Message: "Internal server error",
			})
			return
		}
		s.credentialManager = cm
	}

	// Check rate limiting
	if s.credentialManager.rateLimiter.isRateLimited(req.Provider) {
		w.WriteHeader(http.StatusTooManyRequests)
		writeJSON(w, ValidateCredentialsResponse{
			Valid:   false,
			Status:  "rate_limited",
			Message: "Too many validation attempts. Please wait before trying again.",
		})
		return
	}

	// Record this attempt
	s.credentialManager.rateLimiter.recordAttempt(req.Provider)

	// Validate provider exists
	if !providers.IsProviderSupported(req.Provider) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ValidateCredentialsResponse{
			Valid:   false,
			Status:  "error",
			Message: fmt.Sprintf("Unsupported provider: %s", req.Provider),
		})
		return
	}

	// Validate credential format first
	if err := providers.ValidateProviderConfig(req.Provider, req.Credentials); err != nil {
		writeJSON(w, ValidateCredentialsResponse{
			Valid:   false,
			Status:  "invalid_format",
			Message: fmt.Sprintf("Credential format validation failed: %v", err),
		})
		return
	}

	// Get provider information for health check
	providerInfo, err := providers.GetProvider(req.Provider)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ValidateCredentialsResponse{
			Valid:   false,
			Status:  "error",
			Message: "Failed to get provider information",
		})
		return
	}

	// Perform health check with credentials
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Construct API key for health check (simplified - would need provider-specific logic)
	apiKey := ""
	if providerInfo.AuthType == providers.AuthAPIKey {
		if key, exists := req.Credentials["api_key"]; exists {
			apiKey = key
		} else if key, exists := req.Credentials["personal_access_token"]; exists {
			apiKey = key
		}
	}

	healthCheck, err := s.credentialManager.healthChecker.CheckHealth(ctx, providerInfo.HealthEndpoint, apiKey)
	if err != nil {
		log.Printf("Error during health check for provider %s: %v", req.Provider, err)
		writeJSON(w, ValidateCredentialsResponse{
			Valid:       false,
			Status:      "health_check_failed",
			Message:     "Failed to perform health check",
			HealthCheck: healthCheck,
		})
		return
	}

	// Determine validation result based on health check
	valid := healthCheck.Status == "healthy"
	status := "valid"
	message := "Credentials are valid and working"

	if !valid {
		if healthCheck.StatusCode == 401 || healthCheck.StatusCode == 403 {
			status = "invalid_credentials"
			message = "Credentials are invalid or have insufficient permissions"
		} else {
			status = "service_unavailable"
			message = "Service is currently unavailable, but credentials format is correct"
		}
	}

	log.Printf("[AUDIT] Credential validation performed for provider: %s, result: %s", req.Provider, status)
	
	writeJSON(w, ValidateCredentialsResponse{
		Valid:       valid,
		Status:      status,
		Message:     message,
		HealthCheck: healthCheck,
	})
}

// handleCredentialsStatus handles GET /v1/credentials/status[?provider=x]
func (s *Server) handleCredentialsStatus(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    // Ensure manager
    if s.credentialManager == nil {
        cm, err := NewCredentialManager()
        if err != nil { w.WriteHeader(http.StatusInternalServerError); return }
        s.credentialManager = cm
    }

    q := r.URL.Query().Get("provider")
    if q != "" {
        exists := s.credentialManager.vault.Exists(q)
        writeJSON(w, CredentialStatusResponse{ Provider: q, Exists: exists })
        return
    }

    // List for all known providers
    list := providers.GetAllProviders()
    out := make([]CredentialStatusResponse, 0, len(list))
    for _, p := range list {
        out = append(out, CredentialStatusResponse{ Provider: p.Name, Exists: s.credentialManager.vault.Exists(p.Name) })
    }
    writeJSON(w, out)
}

// handleCredentialsValidateStored handles POST /v1/credentials/validate-stored
// Body: { "provider": "name" }
func (s *Server) handleCredentialsValidateStored(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }

    var body struct{ Provider string `json:"provider"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Provider == "" {
        w.WriteHeader(http.StatusBadRequest)
        writeJSON(w, map[string]string{"error":"provider required"})
        return
    }

    if s.credentialManager == nil {
        cm, err := NewCredentialManager(); if err != nil { w.WriteHeader(http.StatusInternalServerError); return }
        s.credentialManager = cm
    }

    creds, err := s.credentialManager.vault.Retrieve(body.Provider)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        writeJSON(w, map[string]string{"error":"stored credentials not found"})
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
    defer cancel()

    hc, err := s.credentialManager.healthChecker.CheckProviderHealth(ctx, body.Provider, creds)
    if err != nil {
        writeJSON(w, ValidateCredentialsResponse{ Valid: false, Status: "health_check_failed", Message: "Failed during provider health check", HealthCheck: hc })
        return
    }

    valid := hc.Status == "healthy"
    status := "valid"; msg := "Credentials are valid and working"
    if !valid {
        if hc.StatusCode == 401 || hc.StatusCode == 403 { status = "invalid_credentials"; msg = "Stored credentials appear invalid" } else { status = "service_unavailable"; msg = "Service unavailable or insufficient scopes" }
    }
    writeJSON(w, ValidateCredentialsResponse{ Valid: valid, Status: status, Message: msg, HealthCheck: hc })
}

// Add credential manager to Server struct (this would go in server.go)
// credentialManager *CredentialManager

// Add credential routes to Router method (this would be added in server.go Router method)
func (s *Server) addCredentialRoutes(mux *http.ServeMux) {
	// Credential management endpoints
	mux.HandleFunc("/v1/credentials", s.handleCredentialsStore)
	mux.HandleFunc("/v1/credentials/", func(w http.ResponseWriter, r *http.Request) {
		// Route based on path segments
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) == 4 {
			// /v1/credentials/{provider}
			switch r.Method {
			case http.MethodGet:
				s.handleCredentialsGet(w, r)
			case http.MethodPut:
				s.handleCredentialsUpdate(w, r)
			case http.MethodDelete:
				s.handleCredentialsDelete(w, r)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		} else if len(parts) == 5 && parts[4] == "validate" {
			// /v1/credentials/{provider}/validate - but we'll use the main validate endpoint
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/v1/credentials/validate", s.handleCredentialsValidate)
}
