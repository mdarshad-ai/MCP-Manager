package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSave(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create a test registry
	reg := &Registry{
		Version: "1.0",
		Servers: []Server{
			{
				Name: "test-server",
				Slug: "test-server",
				Source: Source{
					Type: "npm",
					URI:  "test-package",
				},
				Runtime: Runtime{
					Kind: "node",
					Node: &NodeRuntime{PackageManager: "npm"},
				},
				Entry: Entry{
					Transport: "stdio",
					Command:   "node",
					Args:      []string{"index.js"},
				},
				Health: Health{
					Probe:         "simple",
					Method:        "connect",
					IntervalSec:   30,
					TimeoutSec:    10,
					RestartPolicy: "always",
					MaxRestarts:   5,
				},
			},
		},
	}

	// Test saving
	if err := Save(reg, registryPath); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Fatal("registry file was not created")
	}

	// Test loading the saved registry
	loaded, err := Load(registryPath)
	if err != nil {
		t.Fatalf("failed to load saved registry: %v", err)
	}

	// Verify loaded registry matches original
	if loaded.Version != reg.Version {
		t.Errorf("version mismatch: expected %s, got %s", reg.Version, loaded.Version)
	}

	if len(loaded.Servers) != len(reg.Servers) {
		t.Errorf("server count mismatch: expected %d, got %d", len(reg.Servers), len(loaded.Servers))
	}

	if loaded.Servers[0].Name != reg.Servers[0].Name {
		t.Errorf("server name mismatch: expected %s, got %s", reg.Servers[0].Name, loaded.Servers[0].Name)
	}
}

func TestSaveDefault(t *testing.T) {
	// Test saving to default location (will use user's home directory)
	// We'll create a minimal registry for this test
	reg := NewDefault()
	
	// We can't easily test SaveDefault without affecting the user's actual registry
	// So we'll just test that it doesn't error with a valid registry
	defaultPath, err := DefaultPath()
	if err != nil {
		t.Fatalf("failed to get default path: %v", err)
	}

	// Create a backup of existing registry if it exists
	var backup []byte
	if data, err := os.ReadFile(defaultPath); err == nil {
		backup = data
	}

	// Test SaveDefault
	if err := SaveDefault(reg); err != nil {
		t.Fatalf("failed to save default registry: %v", err)
	}

	// Verify the file was created/updated
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Error("default registry file was not created")
	}

	// Restore backup if it existed
	if backup != nil {
		_ = os.WriteFile(defaultPath, backup, 0o644)
	}
}

func TestSaveAtomicOperations(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create a test registry
	reg := NewDefault()

	// Save registry
	if err := Save(reg, registryPath); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Verify no temp files are left behind
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	tempFiles := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".tmp" {
			tempFiles++
		}
	}

	if tempFiles > 0 {
		t.Errorf("found %d temp files, atomic save should clean up", tempFiles)
	}

	// Verify the content is valid JSON
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("failed to read registry file: %v", err)
	}

	var parsed Registry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("saved registry is not valid JSON: %v", err)
	}
}

func TestSaveNilRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Test saving nil registry should fail
	if err := Save(nil, registryPath); err == nil {
		t.Error("Save should fail with nil registry")
	}
}

func TestSaveInvalidRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create an invalid registry (no version)
	invalidReg := &Registry{
		Version: "", // Invalid: version is required
		Servers: []Server{},
	}

	// Test saving invalid registry should fail
	if err := Save(invalidReg, registryPath); err == nil {
		t.Error("Save should fail with invalid registry")
	}
}