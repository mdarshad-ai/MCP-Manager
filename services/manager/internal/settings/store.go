package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Settings represents the application settings that persist across sessions.
type Settings struct {
	// Autostart configuration
	Autostart AutostartSettings `json:"autostart"`
	
	// UI theme preferences
	Theme ThemeSettings `json:"theme"`
	
	// Logging configuration
	Logs LogSettings `json:"logs"`
	
	// Manager daemon settings
	Manager ManagerSettings `json:"manager"`
	
	// Performance settings
	Performance PerformanceSettings `json:"performance"`
	
	// Storage information
	Storage StorageInfo `json:"storage"`
	
	// Logs cap in MB
	LogsCap int `json:"logsCap"`
}

// AutostartSettings controls daemon autostart behavior
type AutostartSettings struct {
	Enabled bool   `json:"enabled"`
	Scope   string `json:"scope"` // "user" or "system"
}

// ThemeSettings controls UI appearance
type ThemeSettings struct {
	Mode   string `json:"mode"`   // "light", "dark", "system"
	Accent string `json:"accent"` // accent color preference
}

// LogSettings controls logging behavior
type LogSettings struct {
	Level           string `json:"level"`           // "debug", "info", "warn", "error"
	MaxSizePerFile  int64  `json:"maxSizePerFile"`  // bytes
	MaxTotalSize    int64  `json:"maxTotalSize"`    // bytes
	RetentionDays   int    `json:"retentionDays"`   // days to keep logs
	RotationEnabled bool   `json:"rotationEnabled"` // enable automatic log rotation
}

// ManagerSettings controls daemon behavior
type ManagerSettings struct {
	Port            int    `json:"port"`            // HTTP API port
	MemoryLimitMB   int    `json:"memoryLimitMB"`   // per-server memory limit
	GlobalMemoryMB  int    `json:"globalMemoryMB"`  // global memory limit
	HealthCheckSec  int    `json:"healthCheckSec"`  // health check interval
	SaveIntervalSec int    `json:"saveIntervalSec"` // registry save interval
}

// PerformanceSettings contains performance-related settings
type PerformanceSettings struct {
	RefreshInterval int `json:"refreshInterval"` // in milliseconds
	MaxLogLines     int `json:"maxLogLines"`
}

// StorageInfo contains storage usage information
type StorageInfo struct {
	Used      int64 `json:"used"`      // bytes
	Available int64 `json:"available"` // bytes
}

var (
	settingsMutex sync.RWMutex
	cachedSettings *Settings
)

// DefaultPath returns the default settings file path (~/.mcp/settings.json).
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".mcp", "settings.json"), nil
}

// NewDefault creates a new Settings instance with default values.
func NewDefault() *Settings {
	return &Settings{
		Autostart: AutostartSettings{
			Enabled: false,
			Scope:   "user",
		},
		Theme: ThemeSettings{
			Mode:   "system",
			Accent: "blue",
		},
		Logs: LogSettings{
			Level:           "info",
			MaxSizePerFile:  10 * 1024 * 1024,  // 10MB per file
			MaxTotalSize:    100 * 1024 * 1024, // 100MB total
			RetentionDays:   30,                 // 30 days
			RotationEnabled: true,
		},
		Manager: ManagerSettings{
			Port:            38018,
			MemoryLimitMB:   128,  // 128MB per server
			GlobalMemoryMB:  1024, // 1GB global limit
			HealthCheckSec:  30,   // 30 second health checks
			SaveIntervalSec: 300,  // save registry every 5 minutes
		},
		Performance: PerformanceSettings{
			RefreshInterval: 5000, // 5 seconds
			MaxLogLines:     1000,
		},
		Storage: StorageInfo{
			Used:      0,
			Available: 0,
		},
		LogsCap: 100, // 100MB default logs cap
	}
}

// Load reads settings from the specified path.
func Load(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
	}

	if err := validate(&settings); err != nil {
		return nil, fmt.Errorf("settings validation failed: %w", err)
	}

	return &settings, nil
}

// LoadDefault loads settings from the default path (~/.mcp/settings.json).
// If the file doesn't exist, returns default settings.
func LoadDefault() (*Settings, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get default path: %w", err)
	}
	return LoadOrDefault(path)
}

// LoadOrDefault loads settings from the specified path.
// If the file doesn't exist, returns default settings.
func LoadOrDefault(path string) (*Settings, error) {
	settings, err := Load(path)
	if err != nil {
		// Use errors.Is to check for os.ErrNotExist in the error chain
		var pathError *os.PathError
		if errors.As(err, &pathError) && errors.Is(pathError.Err, os.ErrNotExist) {
			return NewDefault(), nil
		}
		return nil, err
	}
	return settings, nil
}

// Save writes settings to the specified path using atomic file operations.
func Save(settings *Settings, path string) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	if err := validate(settings); err != nil {
		return fmt.Errorf("settings validation failed: %w", err)
	}

	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on any error
	defer func() {
		tempFile.Close()
		os.Remove(tempPath)
	}()

	// Marshal settings to pretty-printed JSON
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
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

	// Update cache
	cachedSettings = settings

	// Successfully saved - don't remove temp file in defer
	tempFile = nil
	return nil
}

// SaveDefault saves settings to the default location.
func SaveDefault(settings *Settings) error {
	path, err := DefaultPath()
	if err != nil {
		return fmt.Errorf("failed to get default path: %w", err)
	}
	return Save(settings, path)
}

// GetCached returns the cached settings, loading from disk if not cached.
func GetCached() (*Settings, error) {
	settingsMutex.RLock()
	if cachedSettings != nil {
		defer settingsMutex.RUnlock()
		return cachedSettings, nil
	}
	settingsMutex.RUnlock()

	// Load from disk and cache
	settings, err := LoadDefault()
	if err != nil {
		return nil, err
	}

	settingsMutex.Lock()
	cachedSettings = settings
	settingsMutex.Unlock()

	return settings, nil
}

// UpdateCached updates the cached settings and saves to disk.
func UpdateCached(settings *Settings) error {
	if err := SaveDefault(settings); err != nil {
		return err
	}

	settingsMutex.Lock()
	cachedSettings = settings
	settingsMutex.Unlock()

	return nil
}

// validate checks if the settings are valid.
func validate(s *Settings) error {
	if s.Autostart.Scope != "user" && s.Autostart.Scope != "system" {
		return fmt.Errorf("invalid autostart scope: %s", s.Autostart.Scope)
	}

	if s.Theme.Mode != "light" && s.Theme.Mode != "dark" && s.Theme.Mode != "system" {
		return fmt.Errorf("invalid theme mode: %s", s.Theme.Mode)
	}

	logLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !logLevels[s.Logs.Level] {
		return fmt.Errorf("invalid log level: %s", s.Logs.Level)
	}

	if s.Logs.MaxSizePerFile <= 0 {
		return fmt.Errorf("maxSizePerFile must be positive")
	}

	if s.Logs.MaxTotalSize <= 0 {
		return fmt.Errorf("maxTotalSize must be positive")
	}

	if s.Logs.RetentionDays <= 0 {
		return fmt.Errorf("retentionDays must be positive")
	}

	if s.Manager.Port <= 0 || s.Manager.Port > 65535 {
		return fmt.Errorf("invalid port: %d", s.Manager.Port)
	}

	if s.Manager.MemoryLimitMB <= 0 {
		return fmt.Errorf("memoryLimitMB must be positive")
	}

	if s.Manager.GlobalMemoryMB <= 0 {
		return fmt.Errorf("globalMemoryMB must be positive")
	}

	if s.Manager.HealthCheckSec <= 0 {
		return fmt.Errorf("healthCheckSec must be positive")
	}

	if s.Manager.SaveIntervalSec <= 0 {
		return fmt.Errorf("saveIntervalSec must be positive")
	}

	return nil
}