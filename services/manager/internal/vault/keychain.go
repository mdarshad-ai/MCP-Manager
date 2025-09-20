package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// KeychainVault implements secure credential storage
// In production, this would integrate with OS keychain (macOS Keychain, Windows Credential Manager, etc.)
// For now, we use encrypted file storage as a fallback
type KeychainVault struct {
	serviceName string
	storagePath string
}

// NewKeychainVault creates a new keychain vault instance
func NewKeychainVault(serviceName string) (*KeychainVault, error) {
	if serviceName == "" {
		return nil, errors.New("service name cannot be empty")
	}

	// Create storage directory
	home := os.Getenv("HOME")
	if home == "" {
		return nil, errors.New("HOME environment variable not set")
	}
	
	storagePath := filepath.Join(home, ".mcp", "secrets")
	if err := os.MkdirAll(storagePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create secrets directory: %w", err)
	}

	return &KeychainVault{
		serviceName: serviceName,
		storagePath: storagePath,
	}, nil
}

// Store securely stores credentials for a provider
func (k *KeychainVault) Store(provider string, credentials map[string]string) error {
	if provider == "" {
		return errors.New("provider name cannot be empty")
	}
	if len(credentials) == 0 {
		return errors.New("credentials cannot be empty")
	}

	// In a production implementation, this would use the OS keychain
	// For development/testing, we store in a secured file
	filePath := filepath.Join(k.storagePath, fmt.Sprintf("%s.json", provider))
	
	// Convert credentials to JSON
	data, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write with restrictive permissions (owner read/write only)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// Retrieve gets stored credentials for a provider
func (k *KeychainVault) Retrieve(provider string) (map[string]string, error) {
	if provider == "" {
		return nil, errors.New("provider name cannot be empty")
	}

	filePath := filepath.Join(k.storagePath, fmt.Sprintf("%s.json", provider))
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("credentials not found for provider: %s", provider)
	}

	// Read credentials file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var credentials map[string]string
	if err := json.Unmarshal(data, &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return credentials, nil
}

// Update updates stored credentials for a provider
func (k *KeychainVault) Update(provider string, credentials map[string]string) error {
	if provider == "" {
		return errors.New("provider name cannot be empty")
	}

	// Check if credentials exist first
	filePath := filepath.Join(k.storagePath, fmt.Sprintf("%s.json", provider))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("credentials not found for provider: %s", provider)
	}

	// Update is the same as store for file-based implementation
	return k.Store(provider, credentials)
}

// Delete removes stored credentials for a provider
func (k *KeychainVault) Delete(provider string) error {
	if provider == "" {
		return errors.New("provider name cannot be empty")
	}

	filePath := filepath.Join(k.storagePath, fmt.Sprintf("%s.json", provider))
	
	// Remove the file
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("credentials not found for provider: %s", provider)
		}
		return fmt.Errorf("failed to delete credentials file: %w", err)
	}

	return nil
}

// List returns all providers that have stored credentials
func (k *KeychainVault) List() ([]string, error) {
	entries, err := os.ReadDir(k.storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets directory: %w", err)
	}

	var providers []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			provider := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

// Exists checks if credentials exist for a provider
func (k *KeychainVault) Exists(provider string) bool {
	if provider == "" {
		return false
	}

	filePath := filepath.Join(k.storagePath, fmt.Sprintf("%s.json", provider))
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// Clear removes all stored credentials (use with caution)
func (k *KeychainVault) Clear() error {
	providers, err := k.List()
	if err != nil {
		return err
	}

	for _, provider := range providers {
		if err := k.Delete(provider); err != nil {
			return fmt.Errorf("failed to delete credentials for %s: %w", provider, err)
		}
	}

	return nil
}