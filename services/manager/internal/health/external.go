package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ExternalHealthChecker monitors external/cloud MCP services
type ExternalHealthChecker struct {
	client *http.Client
}

// NewExternalHealthChecker creates a new external health checker
func NewExternalHealthChecker() *ExternalHealthChecker {
	return &ExternalHealthChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckHealth performs a health check on an external MCP service
func (e *ExternalHealthChecker) CheckHealth(ctx context.Context, endpoint string, apiKey string) (*ExternalHealth, error) {
	return e.CheckHealthWithCredentials(ctx, endpoint, map[string]string{"api_key": apiKey})
}

// CheckHealthWithCredentials performs a health check with flexible credential support
func (e *ExternalHealthChecker) CheckHealthWithCredentials(ctx context.Context, endpoint string, credentials map[string]string) (*ExternalHealth, error) {
	start := time.Now()
	
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return &ExternalHealth{
			Status:    "error",
			Error:     fmt.Sprintf("failed to create request: %v", err),
			Timestamp: time.Now(),
		}, nil
	}
	
	// Add credentials based on provider requirements
	if apiKey, exists := credentials["api_key"]; exists && apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	
	if token, exists := credentials["oauth_token"]; exists && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	
	if username, exists := credentials["username"]; exists {
		if password, exists := credentials["password"]; exists {
			req.SetBasicAuth(username, password)
		}
	}
	
	resp, err := e.client.Do(req)
	if err != nil {
		return &ExternalHealth{
			Status:       "error",
			Error:        fmt.Sprintf("request failed: %v", err),
			Timestamp:    time.Now(),
			ResponseTime: time.Since(start).Milliseconds(),
		}, nil
	}
	defer resp.Body.Close()
	
	health := &ExternalHealth{
		StatusCode:   resp.StatusCode,
		ResponseTime: time.Since(start).Milliseconds(),
		Timestamp:    time.Now(),
	}
	
	// Enhanced status determination with rate limiting detection
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		health.Status = "healthy"
	case resp.StatusCode == 401:
		health.Status = "error"
		health.Error = "unauthorized - credential may be expired or invalid"
	case resp.StatusCode == 403:
		health.Status = "error"
		health.Error = "forbidden - insufficient permissions or quota exceeded"
	case resp.StatusCode == 429:
		health.Status = "warning"
		health.Error = "rate limited"
		// Extract rate limit information if available
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			health.Error += fmt.Sprintf(" (retry after %s)", retryAfter)
		}
		if resetTime := resp.Header.Get("X-RateLimit-Reset"); resetTime != "" {
			health.Error += fmt.Sprintf(" (reset at %s)", resetTime)
		}
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		health.Status = "error"
		health.Error = fmt.Sprintf("client error: %d", resp.StatusCode)
	case resp.StatusCode >= 500:
		health.Status = "unhealthy"
		health.Error = fmt.Sprintf("server error: %d", resp.StatusCode)
	default:
		health.Status = "warning"
		health.Error = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
	}
	
	return health, nil
}

// ExternalHealth represents the health status of an external MCP service
type ExternalHealth struct {
	Status       string    `json:"status"` // "healthy", "unhealthy", "warning", "error"
	StatusCode   int       `json:"statusCode,omitempty"`
	ResponseTime int64     `json:"responseTime"` // milliseconds
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ProviderHealthEndpoints defines health check endpoints for known providers
var ProviderHealthEndpoints = map[string]string{
	"notion":     "https://api.notion.com/v1/users/me",
	"slack":      "https://slack.com/api/auth.test",
	"github":     "https://api.github.com/user",
	"google":     "https://www.googleapis.com/oauth2/v1/tokeninfo",
	"microsoft":  "https://graph.microsoft.com/v1.0/me",
	"openai":     "https://api.openai.com/v1/models",
	"anthropic":  "https://api.anthropic.com/v1/messages",
}

// GetProviderEndpoint returns the health check endpoint for a known provider
func GetProviderEndpoint(provider string) (string, bool) {
	endpoint, ok := ProviderHealthEndpoints[provider]
	return endpoint, ok
}

// CheckProviderHealth performs provider-specific health checks with enhanced error detection
func (e *ExternalHealthChecker) CheckProviderHealth(ctx context.Context, provider string, credentials map[string]string) (*ExternalHealth, error) {
	endpoint, exists := GetProviderEndpoint(provider)
	if !exists {
		return &ExternalHealth{
			Status:    "error",
			Error:     fmt.Sprintf("unknown provider: %s", provider),
			Timestamp: time.Now(),
		}, nil
	}
	
	// Provider-specific credential handling
	providerCredentials := e.normalizeCredentials(provider, credentials)
	
	health, err := e.CheckHealthWithCredentials(ctx, endpoint, providerCredentials)
	if err != nil {
		return health, err
	}
	
	// Provider-specific response interpretation
	health = e.enhanceHealthWithProviderSpecifics(provider, health)
	
	return health, nil
}

// normalizeCredentials converts credentials to the format expected by each provider
func (e *ExternalHealthChecker) normalizeCredentials(provider string, credentials map[string]string) map[string]string {
	normalized := make(map[string]string)
	
	switch provider {
	case "notion":
		// Notion uses internal integration tokens
		if token, exists := credentials["token"]; exists {
			normalized["api_key"] = token
		}
	case "slack":
		// Slack uses OAuth tokens or bot tokens
		if botToken, exists := credentials["bot_token"]; exists {
			normalized["api_key"] = botToken
		} else if oauthToken, exists := credentials["oauth_token"]; exists {
			normalized["oauth_token"] = oauthToken
		}
	case "github":
		// GitHub uses personal access tokens or OAuth
		if pat, exists := credentials["personal_access_token"]; exists {
			normalized["api_key"] = pat
		} else if token, exists := credentials["token"]; exists {
			normalized["api_key"] = token
		}
	case "google":
		// Google uses OAuth 2.0 tokens
		if accessToken, exists := credentials["access_token"]; exists {
			normalized["oauth_token"] = accessToken
		}
	case "microsoft":
		// Microsoft Graph uses OAuth 2.0 tokens
		if accessToken, exists := credentials["access_token"]; exists {
			normalized["oauth_token"] = accessToken
		}
	case "openai":
		// OpenAI uses API keys
		if apiKey, exists := credentials["api_key"]; exists {
			normalized["api_key"] = apiKey
		}
	case "anthropic":
		// Anthropic uses API keys
		if apiKey, exists := credentials["api_key"]; exists {
			normalized["api_key"] = apiKey
		}
	default:
		// Default: copy all credentials as-is
		for k, v := range credentials {
			normalized[k] = v
		}
	}
	
	return normalized
}

// enhanceHealthWithProviderSpecifics adds provider-specific health interpretation
func (e *ExternalHealthChecker) enhanceHealthWithProviderSpecifics(provider string, health *ExternalHealth) *ExternalHealth {
	switch provider {
	case "notion":
		if health.StatusCode == 401 {
			health.Error = "Invalid Notion integration token - please check your integration settings"
		} else if health.StatusCode == 403 {
			health.Error = "Notion integration lacks required permissions"
		}
	case "slack":
		if health.StatusCode == 401 {
			health.Error = "Invalid Slack token - token may have been revoked"
		} else if health.StatusCode == 403 {
			health.Error = "Slack app lacks required scopes or permissions"
		}
	case "github":
		if health.StatusCode == 401 {
			health.Error = "Invalid GitHub token - token may have expired"
		} else if health.StatusCode == 403 {
			health.Error = "GitHub token lacks required permissions or rate limit exceeded"
		}
	case "google":
		if health.StatusCode == 401 {
			health.Error = "Google OAuth token expired - please re-authenticate"
		} else if health.StatusCode == 403 {
			health.Error = "Google API quota exceeded or insufficient permissions"
		}
	case "microsoft":
		if health.StatusCode == 401 {
			health.Error = "Microsoft Graph token expired - please re-authenticate"
		} else if health.StatusCode == 403 {
			health.Error = "Microsoft Graph API quota exceeded or insufficient permissions"
		}
	case "openai":
		if health.StatusCode == 401 {
			health.Error = "Invalid OpenAI API key"
		} else if health.StatusCode == 429 {
			health.Error = "OpenAI API rate limit exceeded - consider upgrading your plan"
		}
	case "anthropic":
		if health.StatusCode == 401 {
			health.Error = "Invalid Anthropic API key"
		} else if health.StatusCode == 429 {
			health.Error = "Anthropic API rate limit exceeded"
		}
	}
	
	return health
}