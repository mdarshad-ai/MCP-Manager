package providers

import (
	"testing"
)

func TestGetProvider(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid provider", "notion", false},
		{"case insensitive", "NOTION", false},
		{"invalid provider", "nonexistent", true},
		{"empty name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetProvider(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAllProviders(t *testing.T) {
	providers := GetAllProviders()
	
	// Should contain expected providers
	expectedProviders := []string{"notion", "slack", "github", "google", "microsoft", "openai"}
	for _, expected := range expectedProviders {
		if _, exists := providers[expected]; !exists {
			t.Errorf("Expected provider '%s' not found", expected)
		}
	}
	
	// Should not be empty
	if len(providers) == 0 {
		t.Error("GetAllProviders() returned empty map")
	}
}

func TestValidateProviderConfig(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		credentials map[string]string
		wantErr     bool
	}{
		{
			"valid notion config",
			"notion",
			map[string]string{"api_key": "secret_abcdefghijklmnopqrstuvwxyz1234567890123456"},
			false,
		},
		{
			"missing required credential",
			"notion",
			map[string]string{},
			true,
		},
		{
			"invalid api key format",
			"notion",
			map[string]string{"api_key": "invalid_key_format"},
			true,
		},
		{
			"valid github config",
			"github",
			map[string]string{"personal_access_token": "ghp_abcdefghijklmnopqrstuvwxyz1234567890"},
			false,
		},
		{
			"unexpected credential",
			"notion",
			map[string]string{
				"api_key": "secret_abcdefghijklmnopqrstuvwxyz1234567890123",
				"extra":   "unexpected",
			},
			true,
		},
		{
			"invalid provider",
			"nonexistent",
			map[string]string{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderConfig(tt.provider, tt.credentials)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCredentialRequirements(t *testing.T) {
	creds, err := GetCredentialRequirements("notion")
	if err != nil {
		t.Errorf("GetCredentialRequirements() error = %v", err)
		return
	}

	if len(creds) == 0 {
		t.Error("Expected credentials for notion provider")
		return
	}

	// Check that api_key requirement exists
	found := false
	for _, cred := range creds {
		if cred.Key == "api_key" {
			found = true
			if !cred.Required {
				t.Error("Expected api_key to be required for notion")
			}
			if !cred.Secret {
				t.Error("Expected api_key to be secret for notion")
			}
			break
		}
	}
	if !found {
		t.Error("Expected api_key credential for notion provider")
	}
}

func TestIsProviderSupported(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"supported provider", "notion", true},
		{"case insensitive", "GITHUB", true},
		{"unsupported provider", "unsupported", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsProviderSupported(tt.provider); got != tt.want {
				t.Errorf("IsProviderSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProvidersByTag(t *testing.T) {
	providers := GetProvidersByTag("productivity")
	
	// Should include providers with productivity tag
	expectedProviders := []string{"notion", "google", "microsoft"}
	for _, expected := range expectedProviders {
		if _, exists := providers[expected]; !exists {
			t.Errorf("Expected provider '%s' with productivity tag not found", expected)
		}
	}
	
	// Should not include providers without the tag
	if _, exists := providers["openai"]; exists {
		t.Error("OpenAI should not be in productivity providers")
	}
}

func TestAddProvider(t *testing.T) {
	// Save original registry size
	originalSize := len(providerRegistry)
	
	// Test adding valid provider
	testProvider := Provider{
		Name:           "test-provider",
		DisplayName:    "Test Provider",
		Description:    "A test provider",
		AuthType:       AuthAPIKey,
		HealthEndpoint: "https://api.test.com/health",
		Credentials: []Credential{
			{
				Key:         "test_key",
				DisplayName: "Test Key",
				Required:    true,
				Secret:      true,
				Validation:  "^test_[a-zA-Z0-9]+$",
			},
		},
	}
	
	err := AddProvider(testProvider)
	if err != nil {
		t.Errorf("AddProvider() error = %v", err)
	}
	
	// Verify provider was added
	if len(providerRegistry) != originalSize+1 {
		t.Error("Provider registry size should have increased by 1")
	}
	
	// Verify we can retrieve the added provider
	retrieved, err := GetProvider("test-provider")
	if err != nil {
		t.Errorf("Failed to retrieve added provider: %v", err)
	}
	if retrieved.DisplayName != testProvider.DisplayName {
		t.Errorf("Retrieved provider display name = %v, want %v", retrieved.DisplayName, testProvider.DisplayName)
	}
	
	// Test adding provider with same name should fail
	err = AddProvider(testProvider)
	if err == nil {
		t.Error("Adding provider with duplicate name should fail")
	}
	
	// Cleanup - remove test provider
	delete(providerRegistry, "test-provider")
}

func TestAddProviderValidation(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		wantErr  bool
	}{
		{
			"empty name",
			Provider{DisplayName: "Test"},
			true,
		},
		{
			"empty display name",
			Provider{Name: "test"},
			true,
		},
		{
			"empty health endpoint",
			Provider{Name: "test", DisplayName: "Test"},
			true,
		},
		{
			"no credentials",
			Provider{Name: "test", DisplayName: "Test", HealthEndpoint: "https://test.com"},
			true,
		},
		{
			"invalid validation pattern",
			Provider{
				Name:           "test",
				DisplayName:    "Test",
				HealthEndpoint: "https://test.com",
				Credentials: []Credential{
					{Key: "test", DisplayName: "Test", Validation: "[invalid"},
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddProvider(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{Field: "api_key", Message: "invalid format"}
	expected := "validation error for api_key: invalid format"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expected)
	}
}