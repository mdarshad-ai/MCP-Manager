package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mcp/manager/internal/health"
	"mcp/manager/internal/providers"
	"mcp/manager/internal/registry"
)

func TestCredentialManager(t *testing.T) {
	// Create a credential manager for testing
	cm, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	provider := "test-provider"
	testCredentials := map[string]string{
		"api_key": "test-key-12345",
	}

	// Test storing credentials
	t.Run("Store", func(t *testing.T) {
		err := cm.vault.Store(provider, testCredentials)
		if err != nil {
			t.Errorf("Failed to store credentials: %v", err)
		}
	})

	// Test retrieving credentials
	t.Run("Retrieve", func(t *testing.T) {
		retrieved, err := cm.vault.Retrieve(provider)
		if err != nil {
			t.Errorf("Failed to retrieve credentials: %v", err)
			return
		}

		if retrieved["api_key"] != "test-key-12345" {
			t.Errorf("Expected api_key=test-key-12345, got %s", retrieved["api_key"])
		}
	})

	// Cleanup
	_ = cm.vault.Delete(provider)
}

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter()
	provider := "test-provider"

	// Should not be rate limited initially
	if rl.isRateLimited(provider) {
		t.Error("Should not be rate limited initially")
	}

	// Add 5 attempts (the limit)
	for i := 0; i < 5; i++ {
		rl.recordAttempt(provider)
	}

	// Should now be rate limited
	if !rl.isRateLimited(provider) {
		t.Error("Should be rate limited after 5 attempts")
	}

	// Test that different providers are independent
	if rl.isRateLimited("other-provider") {
		t.Error("Different provider should not be rate limited")
	}
}

func TestCredentialsStoreAPI(t *testing.T) {
	// Create a test server
	server := &Server{
		reg: &registry.Registry{},
	}

	// Test valid request
	t.Run("ValidRequest", func(t *testing.T) {
		requestBody := StoreCredentialsRequest{
			Provider: "notion",
			Credentials: map[string]string{
				"api_key": "secret_1234567890123456789012345678901234567890",
			},
		}

		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/credentials", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		server.handleCredentialsStore(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response StoreCredentialsResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got false. Message: %s", response.Message)
		}
	})

	// Test invalid provider
	t.Run("InvalidProvider", func(t *testing.T) {
		requestBody := StoreCredentialsRequest{
			Provider: "invalid-provider",
			Credentials: map[string]string{
				"api_key": "test-key",
			},
		}

		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/credentials", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		server.handleCredentialsStore(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	// Test missing credentials
	t.Run("MissingCredentials", func(t *testing.T) {
		requestBody := StoreCredentialsRequest{
			Provider:    "notion",
			Credentials: map[string]string{},
		}

		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/credentials", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		server.handleCredentialsStore(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	// Test invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/credentials", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		server.handleCredentialsStore(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})
}

func TestCredentialsGetAPI(t *testing.T) {
	server := &Server{
		reg: &registry.Registry{},
	}

	// Test valid provider
	t.Run("ValidProvider", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/credentials/notion", nil)
		
		rr := httptest.NewRecorder()
		server.handleCredentialsGet(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response GetCredentialRequirementsResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.Provider != "notion" {
			t.Errorf("Expected provider=notion, got %s", response.Provider)
		}

		if len(response.Credentials) == 0 {
			t.Error("Expected credentials to be returned")
		}
	})

	// Test invalid provider
	t.Run("InvalidProvider", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/credentials/invalid", nil)
		
		rr := httptest.NewRecorder()
		server.handleCredentialsGet(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}
	})

	// Test malformed URL
	t.Run("MalformedURL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/credentials/", nil)
		
		rr := httptest.NewRecorder()
		server.handleCredentialsGet(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})
}

func TestCredentialsValidateAPI(t *testing.T) {
	server := &Server{
		reg: &registry.Registry{},
	}

	// Test valid format (but might fail health check)
	t.Run("ValidFormat", func(t *testing.T) {
		requestBody := ValidateCredentialsRequest{
			Provider: "notion",
			Credentials: map[string]string{
				"api_key": "secret_1234567890123456789012345678901234567890",
			},
		}

		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/credentials/validate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		server.handleCredentialsValidate(rr, req)

		// Should get a response (might be invalid due to fake credentials, but format is correct)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response ValidateCredentialsResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		// The validation should complete, though credentials might be invalid
		if response.Status == "" {
			t.Error("Expected status to be set")
		}
	})

	// Test invalid format
	t.Run("InvalidFormat", func(t *testing.T) {
		requestBody := ValidateCredentialsRequest{
			Provider: "notion",
			Credentials: map[string]string{
				"api_key": "invalid-format",
			},
		}

		bodyBytes, _ := json.Marshal(requestBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/credentials/validate", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		server.handleCredentialsValidate(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response ValidateCredentialsResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.Valid {
			t.Error("Expected valid=false for invalid format")
		}

		if response.Status != "invalid_format" {
			t.Errorf("Expected status=invalid_format, got %s", response.Status)
		}
	})

	// Test rate limiting
	t.Run("RateLimiting", func(t *testing.T) {
		// Initialize credential manager
		cm, _ := NewCredentialManager()
		server.credentialManager = cm

		requestBody := ValidateCredentialsRequest{
			Provider: "github",
			Credentials: map[string]string{
				"personal_access_token": "ghp_1234567890123456789012345678901234567890",
			},
		}

		bodyBytes, _ := json.Marshal(requestBody)

		// Make 6 requests (rate limit is 5)
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest(http.MethodPost, "/v1/credentials/validate", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			server.handleCredentialsValidate(rr, req)

			if i < 5 {
				// First 5 should succeed (or at least not be rate limited)
				if rr.Code == http.StatusTooManyRequests {
					t.Errorf("Request %d should not be rate limited", i+1)
				}
			} else {
				// 6th should be rate limited
				if rr.Code != http.StatusTooManyRequests {
					t.Errorf("Request %d should be rate limited, got status %d", i+1, rr.Code)
				}
			}
		}
	})
}

func TestCredentialsDeleteAPI(t *testing.T) {
	server := &Server{
		reg: &registry.Registry{},
	}

	// Initialize credential manager and store a test credential
	cm, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}
	server.credentialManager = cm

	provider := "test-delete-provider"
	testCredentials := map[string]string{
		"api_key": "test-key-for-deletion",
	}

	// Store credentials first
	err = cm.vault.Store(provider, testCredentials)
	if err != nil {
		t.Fatalf("Failed to store test credentials: %v", err)
	}

	// Test deletion
	t.Run("ValidDelete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/credentials/"+provider, nil)
		
		rr := httptest.NewRecorder()
		server.handleCredentialsDelete(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response DeleteCredentialsResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got false. Message: %s", response.Message)
		}

		// Verify credentials are actually deleted
		if cm.vault.HasCredentials(provider) {
			t.Error("Credentials should be deleted")
		}
	})
}

// Mock external health checker for testing
type mockExternalHealthChecker struct {
	shouldFail bool
}

func (m *mockExternalHealthChecker) CheckHealth(ctx context.Context, endpoint string, apiKey string) (*health.ExternalHealth, error) {
	if m.shouldFail {
		return &health.ExternalHealth{
			Status:     "error",
			StatusCode: 401,
			Error:      "Unauthorized",
			Timestamp:  time.Now(),
		}, nil
	}

	return &health.ExternalHealth{
		Status:       "healthy",
		StatusCode:   200,
		ResponseTime: 100,
		Timestamp:    time.Now(),
	}, nil
}

func TestProviderValidation(t *testing.T) {
	// Test that the provider validation works with the existing providers
	testCases := []struct {
		provider    string
		credentials map[string]string
		shouldPass  bool
	}{
		{
			provider: "notion",
			credentials: map[string]string{
				"api_key": "secret_1234567890123456789012345678901234567890",
			},
			shouldPass: true,
		},
		{
			provider: "notion",
			credentials: map[string]string{
				"api_key": "invalid-format",
			},
			shouldPass: false,
		},
		{
			provider: "github",
			credentials: map[string]string{
				"personal_access_token": "ghp_1234567890123456789012345678901234567890",
			},
			shouldPass: true,
		},
		{
			provider: "openai",
			credentials: map[string]string{
				"api_key": "sk-test123456789012345678901234567890",
			},
			shouldPass: true,
		},
		{
			provider: "openai",
			credentials: map[string]string{
				"api_key": "invalid-key-format",
			},
			shouldPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.provider, func(t *testing.T) {
			err := providers.ValidateProviderConfig(tc.provider, tc.credentials)
			if tc.shouldPass && err != nil {
				t.Errorf("Expected validation to pass, but got error: %v", err)
			}
			if !tc.shouldPass && err == nil {
				t.Error("Expected validation to fail, but it passed")
			}
		})
	}
}