package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsLifecycle(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Test creating new default settings
	defaults := NewDefault()
	if defaults.Theme.Mode != "system" {
		t.Errorf("expected default theme mode 'system', got %s", defaults.Theme.Mode)
	}
	if defaults.Manager.Port != 38018 {
		t.Errorf("expected default port 38018, got %d", defaults.Manager.Port)
	}

	// Test saving settings
	if err := Save(defaults, settingsPath); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("settings file was not created")
	}

	// Test loading settings
	loaded, err := Load(settingsPath)
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	// Verify loaded settings match defaults
	if loaded.Theme.Mode != defaults.Theme.Mode {
		t.Errorf("loaded theme mode %s doesn't match default %s", loaded.Theme.Mode, defaults.Theme.Mode)
	}
	if loaded.Manager.Port != defaults.Manager.Port {
		t.Errorf("loaded port %d doesn't match default %d", loaded.Manager.Port, defaults.Manager.Port)
	}

	// Test LoadOrDefault with non-existent file
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.json")
	loadedOrDefault, err := LoadOrDefault(nonExistentPath)
	if err != nil {
		t.Fatalf("LoadOrDefault failed: %v", err)
	}
	if loadedOrDefault.Theme.Mode != "system" {
		t.Errorf("LoadOrDefault should return defaults for non-existent file")
	}

	// Test validation with invalid settings
	invalid := NewDefault()
	invalid.Theme.Mode = "invalid"
	if err := validate(invalid); err == nil {
		t.Error("validation should fail for invalid theme mode")
	}

	invalid2 := NewDefault()
	invalid2.Manager.Port = -1
	if err := validate(invalid2); err == nil {
		t.Error("validation should fail for invalid port")
	}
}

func TestAtomicSave(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Create initial settings
	settings := NewDefault()
	settings.Theme.Mode = "dark"

	// Save settings
	if err := Save(settings, settingsPath); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	// Verify atomic save doesn't leave temp files
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

	// Verify content is correct
	loaded, err := Load(settingsPath)
	if err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	if loaded.Theme.Mode != "dark" {
		t.Errorf("expected theme mode 'dark', got %s", loaded.Theme.Mode)
	}
}