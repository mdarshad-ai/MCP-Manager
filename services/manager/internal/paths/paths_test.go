package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureAllDirectories(t *testing.T) {
	// This test will create directories in the actual user home directory
	// So we need to be careful and clean up if needed
	
	// Test that EnsureAllDirectories doesn't fail
	if err := EnsureAllDirectories(); err != nil {
		t.Fatalf("EnsureAllDirectories failed: %v", err)
	}

	// Test that all directories exist after calling EnsureAllDirectories
	dirs := []func() (string, error){
		HomeMCP,
		LogsDir,
		ServersDir,
		CacheDir,
		SecretsDir,
	}

	for _, dirFunc := range dirs {
		dir, err := dirFunc()
		if err != nil {
			t.Errorf("Failed to get directory path: %v", err)
			continue
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s does not exist after EnsureAllDirectories", dir)
		}
	}
}

func TestDirectoryPermissions(t *testing.T) {
	// Test that secrets directory has more restrictive permissions
	secretsDir, err := SecretsDir()
	if err != nil {
		t.Fatalf("Failed to get secrets directory: %v", err)
	}

	stat, err := os.Stat(secretsDir)
	if err != nil {
		t.Fatalf("Failed to stat secrets directory: %v", err)
	}

	// Check that secrets directory has 0700 permissions (rwx------)
	if stat.Mode().Perm() != 0o700 {
		t.Errorf("Secrets directory has wrong permissions: got %o, expected 0700", stat.Mode().Perm())
	}

	// Test that other directories have standard permissions (0755)
	dirs := []func() (string, error){
		LogsDir,
		ServersDir,
		CacheDir,
	}

	for _, dirFunc := range dirs {
		dir, err := dirFunc()
		if err != nil {
			t.Errorf("Failed to get directory: %v", err)
			continue
		}

		stat, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Failed to stat directory %s: %v", dir, err)
			continue
		}

		if stat.Mode().Perm() != 0o755 {
			t.Errorf("Directory %s has wrong permissions: got %o, expected 0755", 
				filepath.Base(dir), stat.Mode().Perm())
		}
	}
}

func TestDirectoryStructure(t *testing.T) {
	// Test that all directories are created under ~/.mcp
	baseMCP, err := HomeMCP()
	if err != nil {
		t.Fatalf("Failed to get base MCP directory: %v", err)
	}

	expectedDirs := []string{"logs", "servers", "cache", "secrets"}
	dirFuncs := []func() (string, error){LogsDir, ServersDir, CacheDir, SecretsDir}

	for i, dirFunc := range dirFuncs {
		dir, err := dirFunc()
		if err != nil {
			t.Errorf("Failed to get %s directory: %v", expectedDirs[i], err)
			continue
		}

		expectedPath := filepath.Join(baseMCP, expectedDirs[i])
		if dir != expectedPath {
			t.Errorf("Directory %s has wrong path: got %s, expected %s", 
				expectedDirs[i], dir, expectedPath)
		}
	}
}