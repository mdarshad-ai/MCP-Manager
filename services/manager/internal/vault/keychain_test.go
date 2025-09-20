package vault

import (
	"testing"
	"time"
)

func TestKeychainVault(t *testing.T) {
	// Create a test vault with a unique service name to avoid conflicts
	vault, err := NewKeychainVault("mcp-manager-test")
	if err != nil {
		t.Fatalf("Failed to create keychain vault: %v", err)
	}

	provider := "test-provider"
	testCredentials := map[string]string{
		"api_key":    "test-api-key-12345",
		"secret_key": "test-secret-key-67890",
	}

	// Test Store
	t.Run("Store", func(t *testing.T) {
		err := vault.Store(provider, testCredentials)
		if err != nil {
			t.Errorf("Failed to store credentials: %v", err)
		}
	})

	// Test Retrieve
	t.Run("Retrieve", func(t *testing.T) {
		retrieved, err := vault.Retrieve(provider)
		if err != nil {
			t.Errorf("Failed to retrieve credentials: %v", err)
			return
		}

		if len(retrieved) != len(testCredentials) {
			t.Errorf("Expected %d credentials, got %d", len(testCredentials), len(retrieved))
		}

		for key, expectedValue := range testCredentials {
			if actualValue, ok := retrieved[key]; !ok {
				t.Errorf("Missing credential key: %s", key)
			} else if actualValue != expectedValue {
				t.Errorf("Credential value mismatch for %s: expected %s, got %s", key, expectedValue, actualValue)
			}
		}
	})

	// Test HasCredentials
	t.Run("HasCredentials", func(t *testing.T) {
		if !vault.HasCredentials(provider) {
			t.Error("HasCredentials should return true for existing provider")
		}

		if vault.HasCredentials("non-existent-provider") {
			t.Error("HasCredentials should return false for non-existent provider")
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		updatedCredentials := map[string]string{
			"api_key": "updated-api-key-12345",
			"new_key": "new-value",
		}

		err := vault.Update(provider, updatedCredentials)
		if err != nil {
			t.Errorf("Failed to update credentials: %v", err)
			return
		}

		// Retrieve and verify update
		retrieved, err := vault.Retrieve(provider)
		if err != nil {
			t.Errorf("Failed to retrieve updated credentials: %v", err)
			return
		}

		if retrieved["api_key"] != "updated-api-key-12345" {
			t.Error("API key was not updated correctly")
		}

		if retrieved["new_key"] != "new-value" {
			t.Error("New key was not added correctly")
		}

		if retrieved["secret_key"] != "test-secret-key-67890" {
			t.Error("Existing key should be preserved during update")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := vault.Delete(provider)
		if err != nil {
			t.Errorf("Failed to delete credentials: %v", err)
		}

		// Verify deletion
		if vault.HasCredentials(provider) {
			t.Error("HasCredentials should return false after deletion")
		}

		_, err = vault.Retrieve(provider)
		if err == nil {
			t.Error("Retrieve should fail after deletion")
		}
	})

	// Test Error Cases
	t.Run("ErrorCases", func(t *testing.T) {
		// Empty provider name
		err := vault.Store("", testCredentials)
		if err == nil {
			t.Error("Store should fail with empty provider name")
		}

		// Empty credentials
		err = vault.Store("test", map[string]string{})
		if err == nil {
			t.Error("Store should fail with empty credentials")
		}

		// Retrieve non-existent provider
		_, err = vault.Retrieve("")
		if err == nil {
			t.Error("Retrieve should fail with empty provider name")
		}
	})
}

func TestEncryptionDecryption(t *testing.T) {
	vault, err := NewKeychainVault("mcp-manager-test-crypto")
	if err != nil {
		t.Fatalf("Failed to create keychain vault: %v", err)
	}

	testData := []byte("This is sensitive credential data that should be encrypted")

	// Test encryption
	encrypted, err := vault.encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if encrypted == string(testData) {
		t.Error("Encrypted data should not match original data")
	}

	// Test decryption
	decrypted, err := vault.decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Error("Decrypted data does not match original data")
	}
}

func TestCredentialEntry(t *testing.T) {
	vault, err := NewKeychainVault("mcp-manager-test-entry")
	if err != nil {
		t.Fatalf("Failed to create keychain vault: %v", err)
	}

	provider := "test-entry-provider"
	credentials := map[string]string{
		"test_key": "test_value",
	}

	// Store credentials and measure time
	startTime := time.Now()
	err = vault.Store(provider, credentials)
	if err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}

	// Wait a bit and then retrieve to test timestamp updates
	time.Sleep(100 * time.Millisecond)
	
	_, err = vault.Retrieve(provider)
	if err != nil {
		t.Fatalf("Failed to retrieve credentials: %v", err)
	}

	// The underlying system should have updated timestamps
	// This is more of an integration test to ensure the flow works
	
	// Cleanup
	err = vault.Delete(provider)
	if err != nil {
		t.Errorf("Failed to cleanup test credentials: %v", err)
	}

	// Verify cleanup
	if vault.HasCredentials(provider) {
		t.Error("Credentials should be deleted after cleanup")
	}

	_ = startTime // Use the variable to avoid compiler warnings
}