package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var saveMutex sync.Mutex

// Save writes the registry to the specified path using atomic file operations.
// It creates a temporary file first, writes the data, then renames it to the final path
// to ensure atomic updates and prevent corruption from partial writes.
func Save(r *Registry, path string) error {
	if r == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	// Validate the registry before saving
	if err := validate(r); err != nil {
		return fmt.Errorf("registry validation failed: %w", err)
	}

	// Ensure thread-safe writes
	saveMutex.Lock()
	defer saveMutex.Unlock()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file in the same directory as the target
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".registry-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on any error
	defer func() {
		tempFile.Close()
		os.Remove(tempPath)
	}()

	// Marshal registry to pretty-printed JSON
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Ensure data is flushed to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomically move temp file to final location
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Successfully saved - don't remove temp file in defer
	tempFile = nil
	return nil
}

// SaveDefault saves the registry to the default location (~/.mcp/registry.json).
func SaveDefault(r *Registry) error {
	path, err := DefaultPath()
	if err != nil {
		return fmt.Errorf("failed to get default path: %w", err)
	}
	return Save(r, path)
}

// Save method for Registry to save itself
func (r *Registry) Save(path string) error {
	return Save(r, path)
}