package providers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// AuthType represents the authentication method used by a provider
type AuthType string

const (
	AuthAPIKey AuthType = "api_key"
	AuthOAuth2 AuthType = "oauth2"
	AuthBasic  AuthType = "basic"
)

// Credential represents a required credential with validation
type Credential struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`     // Should be stored securely
	Validation  string `json:"validation"` // Regex pattern for validation
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

// Provider represents a template for external MCP service providers
type Provider struct {
	Name           string                 `json:"name"`
	DisplayName    string                 `json:"displayName"`
	Description    string                 `json:"description"`
	AuthType       AuthType               `json:"authType"`
	HealthEndpoint string                 `json:"healthEndpoint"`
	BaseURL        string                 `json:"baseUrl,omitempty"`
	Credentials    []Credential           `json:"credentials"`
	ConfigSchema   map[string]interface{} `json:"configSchema,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
}

// ValidationError represents a credential validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// Registry holds all available provider templates
var providerRegistry = map[string]Provider{
	"notion": {
		Name:           "notion",
		DisplayName:    "Notion",
		Description:    "Access and manage Notion databases, pages, and blocks through the Notion API",
		AuthType:       AuthAPIKey,
		HealthEndpoint: "https://api.notion.com/v1/users/me",
		BaseURL:        "https://api.notion.com",
		Credentials: []Credential{
			{
				Key:         "api_key",
				DisplayName: "API Key",
				Required:    true,
				Secret:      true,
				Validation:  `^secret_[a-zA-Z0-9]{40,}$`,
				Description: "Notion integration API key from your workspace settings",
				Example:     "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
		},
		ConfigSchema: map[string]interface{}{
			"version": "2022-06-28",
		},
		Tags: []string{"productivity", "documents", "databases"},
	},
	"slack": {
		Name:           "slack",
		DisplayName:    "Slack",
		Description:    "Send messages, manage channels, and interact with Slack workspaces",
		AuthType:       AuthOAuth2,
		HealthEndpoint: "https://slack.com/api/auth.test",
		BaseURL:        "https://slack.com/api",
		Credentials: []Credential{
			{
				Key:         "bot_token",
				DisplayName: "Bot User OAuth Token",
				Required:    true,
				Secret:      true,
				Validation:  `^xoxb-[0-9]+-[0-9]+-[a-zA-Z0-9]+$`,
				Description: "Bot token starting with xoxb- from your Slack app settings",
				Example:     "xoxb-<bot-token-example>",
			},
			{
				Key:         "user_token",
				DisplayName: "User OAuth Token",
				Required:    false,
				Secret:      true,
				Validation:  `^xoxp-[0-9]+-[0-9]+-[0-9]+-[a-zA-Z0-9]+$`,
				Description: "User token starting with xoxp- for additional permissions",
				Example:     "xoxp-<user-token-example>",
			},
		},
		Tags: []string{"communication", "collaboration", "messaging"},
	},
	"github": {
		Name:           "github",
		DisplayName:    "GitHub",
		Description:    "Manage repositories, issues, pull requests, and other GitHub resources",
		AuthType:       AuthAPIKey,
		HealthEndpoint: "https://api.github.com/user",
		BaseURL:        "https://api.github.com",
		Credentials: []Credential{
			{
				Key:         "personal_access_token",
				DisplayName: "Personal Access Token",
				Required:    true,
				Secret:      true,
				Validation:  `^gh[a-z]_[a-zA-Z0-9]{36,255}$`,
				Description: "GitHub personal access token with appropriate scopes",
				Example:     "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
		},
		ConfigSchema: map[string]interface{}{
			"accept": "application/vnd.github+json",
		},
		Tags: []string{"development", "version-control", "code"},
	},
	"google": {
		Name:           "google",
		DisplayName:    "Google Workspace",
		Description:    "Access Google Workspace services including Drive, Sheets, Calendar, and Gmail",
		AuthType:       AuthOAuth2,
		HealthEndpoint: "https://www.googleapis.com/oauth2/v1/tokeninfo",
		BaseURL:        "https://www.googleapis.com",
		Credentials: []Credential{
			{
				Key:         "access_token",
				DisplayName: "Access Token",
				Required:    true,
				Secret:      true,
				Validation:  `^ya29\.[a-zA-Z0-9_-]+$`,
				Description: "Google OAuth2 access token",
				Example:     "ya29.a0AfH6SMC...",
			},
			{
				Key:         "refresh_token",
				DisplayName: "Refresh Token",
				Required:    false,
				Secret:      true,
				Validation:  `^1//[a-zA-Z0-9_-]+$`,
				Description: "Google OAuth2 refresh token for token renewal",
				Example:     "1//04XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
			},
			{
				Key:         "client_id",
				DisplayName: "Client ID",
				Required:    true,
				Secret:      false,
				Validation:  `^[0-9]+-[a-zA-Z0-9]+\.apps\.googleusercontent\.com$`,
				Description: "OAuth2 client ID from Google Cloud Console",
				Example:     "123456789012-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com",
			},
			{
				Key:         "client_secret",
				DisplayName: "Client Secret",
				Required:    true,
				Secret:      true,
				Validation:  `^GOCSPX-[a-zA-Z0-9_-]+$`,
				Description: "OAuth2 client secret from Google Cloud Console",
				Example:     "GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxx",
			},
		},
		Tags: []string{"productivity", "google", "workspace", "cloud"},
	},
	"microsoft": {
		Name:           "microsoft",
		DisplayName:    "Microsoft 365",
		Description:    "Access Microsoft 365 services including Outlook, OneDrive, Teams, and SharePoint",
		AuthType:       AuthOAuth2,
		HealthEndpoint: "https://graph.microsoft.com/v1.0/me",
		BaseURL:        "https://graph.microsoft.com",
		Credentials: []Credential{
			{
				Key:         "access_token",
				DisplayName: "Access Token",
				Required:    true,
				Secret:      true,
				Validation:  `^[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*$`,
				Description: "Microsoft Graph API access token",
				Example:     "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6...",
			},
			{
				Key:         "refresh_token",
				DisplayName: "Refresh Token",
				Required:    false,
				Secret:      true,
				Validation:  `^[A-Za-z0-9-_]+$`,
				Description: "Microsoft OAuth2 refresh token for token renewal",
				Example:     "M.C123_BAY.CZBJGTVy1aq...",
			},
			{
				Key:         "client_id",
				DisplayName: "Application (Client) ID",
				Required:    true,
				Secret:      false,
				Validation:  `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
				Description: "Application ID from Azure App Registration",
				Example:     "12345678-1234-1234-1234-123456789012",
			},
			{
				Key:         "client_secret",
				DisplayName: "Client Secret",
				Required:    true,
				Secret:      true,
				Validation:  `^[a-zA-Z0-9~._-]+$`,
				Description: "Client secret from Azure App Registration",
				Example:     "abcdefghijklmnopqrstuvwxyz123456789",
			},
		},
		Tags: []string{"productivity", "microsoft", "office365", "cloud"},
	},
	"openai": {
		Name:           "openai",
		DisplayName:    "OpenAI",
		Description:    "Access OpenAI API for GPT models, completions, embeddings, and other AI services",
		AuthType:       AuthAPIKey,
		HealthEndpoint: "https://api.openai.com/v1/models",
		BaseURL:        "https://api.openai.com",
		Credentials: []Credential{
			{
				Key:         "api_key",
				DisplayName: "API Key",
				Required:    true,
				Secret:      true,
				Validation:  `^sk-[a-zA-Z0-9]{20,}$`,
				Description: "OpenAI API key starting with sk- from your OpenAI account",
				Example:     "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
			{
				Key:         "organization_id",
				DisplayName: "Organization ID",
				Required:    false,
				Secret:      false,
				Validation:  `^org-[a-zA-Z0-9]+$`,
				Description: "Optional organization ID for API usage tracking",
				Example:     "org-xxxxxxxxxxxxxxxxxxxxxxxx",
			},
		},
		ConfigSchema: map[string]interface{}{
			"model":       "gpt-4",
			"temperature": 0.7,
		},
		Tags: []string{"ai", "gpt", "language-model", "openai"},
	},
}

// GetAllProviders returns all available provider templates
func GetAllProviders() map[string]Provider {
	// Return a copy to prevent external modification
	result := make(map[string]Provider)
	for k, v := range providerRegistry {
		result[k] = v
	}
	return result
}

// GetProvider returns a specific provider template by name
func GetProvider(name string) (Provider, error) {
	name = strings.ToLower(name)
	provider, exists := providerRegistry[name]
	if !exists {
		return Provider{}, fmt.Errorf("provider '%s' not found", name)
	}
	return provider, nil
}

// GetProviderNames returns a list of all available provider names
func GetProviderNames() []string {
	names := make([]string, 0, len(providerRegistry))
	for name := range providerRegistry {
		names = append(names, name)
	}
	return names
}

// ValidateProviderConfig validates a provider configuration against its template
func ValidateProviderConfig(providerName string, credentials map[string]string) error {
	provider, err := GetProvider(providerName)
	if err != nil {
		return err
	}

	// Check required credentials
	for _, cred := range provider.Credentials {
		value, exists := credentials[cred.Key]

		if cred.Required && (!exists || value == "") {
			return ValidationError{
				Field:   cred.Key,
				Message: fmt.Sprintf("required credential '%s' is missing. Please provide a valid %s.", cred.DisplayName, cred.DisplayName),
			}
		}

		// Skip validation if credential is not provided and not required
		if !exists || value == "" {
			continue
		}

		// Validate format if validation pattern is provided
		if cred.Validation != "" {
			matched, err := regexp.MatchString(cred.Validation, value)
			if err != nil {
				return ValidationError{
					Field:   cred.Key,
					Message: fmt.Sprintf("invalid validation pattern for '%s'", cred.DisplayName),
				}
			}
			if !matched {
				return ValidationError{
					Field:   cred.Key,
					Message: fmt.Sprintf("credential '%s' does not match required format. Example: %s", cred.DisplayName, cred.Example),
				}
			}
		}
	}

	// Check for unexpected credentials
	for key := range credentials {
		found := false
		for _, cred := range provider.Credentials {
			if cred.Key == key {
				found = true
				break
			}
		}
		if !found {
			return ValidationError{
				Field:   key,
				Message: fmt.Sprintf("unexpected credential '%s' for provider '%s'", key, providerName),
			}
		}
	}

	return nil
}

// GetCredentialRequirements returns the credential requirements for a provider
func GetCredentialRequirements(providerName string) ([]Credential, error) {
	provider, err := GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Return a copy to prevent external modification
	credentials := make([]Credential, len(provider.Credentials))
	copy(credentials, provider.Credentials)
	return credentials, nil
}

// IsProviderSupported checks if a provider name is supported
func IsProviderSupported(name string) bool {
	_, exists := providerRegistry[strings.ToLower(name)]
	return exists
}

// GetProvidersByTag returns providers that have the specified tag
func GetProvidersByTag(tag string) map[string]Provider {
	result := make(map[string]Provider)
	for name, provider := range providerRegistry {
		for _, providerTag := range provider.Tags {
			if strings.EqualFold(providerTag, tag) {
				result[name] = provider
				break
			}
		}
	}
	return result
}

// AddProvider adds a new provider to the registry (for extensibility)
func AddProvider(provider Provider) error {
	if provider.Name == "" {
		return errors.New("provider name cannot be empty")
	}

	name := strings.ToLower(provider.Name)
	if _, exists := providerRegistry[name]; exists {
		return fmt.Errorf("provider '%s' already exists", name)
	}

	// Basic validation
	if provider.DisplayName == "" {
		return errors.New("provider display name cannot be empty")
	}
	if provider.HealthEndpoint == "" {
		return errors.New("provider health endpoint cannot be empty")
	}
	if len(provider.Credentials) == 0 {
		return errors.New("provider must have at least one credential")
	}

	// Validate credentials
	for i, cred := range provider.Credentials {
		if cred.Key == "" {
			return fmt.Errorf("credential %d: key cannot be empty", i)
		}
		if cred.DisplayName == "" {
			return fmt.Errorf("credential %d: display name cannot be empty", i)
		}
		if cred.Validation != "" {
			if _, err := regexp.Compile(cred.Validation); err != nil {
				return fmt.Errorf("credential %d: invalid validation pattern: %w", i, err)
			}
		}
	}

	providerRegistry[name] = provider
	return nil
}
