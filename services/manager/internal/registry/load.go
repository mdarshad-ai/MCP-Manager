package registry

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
)

var slugRE = regexp.MustCompile(`^[a-z0-9-]+$`)

// Load reads and parses a registry from the specified path.
// Returns an error if the file doesn't exist or contains invalid data.
func Load(path string) (*Registry, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read registry file: %w", err)
    }
    var r Registry
    if err := json.Unmarshal(b, &r); err != nil {
        return nil, fmt.Errorf("failed to parse registry JSON: %w", err)
    }
    if err := validate(&r); err != nil {
        return nil, fmt.Errorf("registry validation failed: %w", err)
    }
    return &r, nil
}

func DefaultPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil { return "", err }
    return filepath.Join(home, ".mcp", "registry.json"), nil
}

func validate(r *Registry) error {
    if r.Version == "" {
        return errors.New("version required")
    }
    seen := map[string]bool{}
    for i := range r.Servers {
        s := &r.Servers[i]
        if s.Slug == "" || !slugRE.MatchString(s.Slug) {
            return fmt.Errorf("invalid slug: %q", s.Slug)
        }
        if seen[s.Slug] {
            return fmt.Errorf("duplicate slug: %q", s.Slug)
        }
        seen[s.Slug] = true
        if s.Entry.Transport != "stdio" && s.Entry.Transport != "http" {
            return fmt.Errorf("invalid transport for %s", s.Slug)
        }
        // External servers don't need a command since they're accessed via HTTP APIs
        if s.Entry.Command == "" && !s.IsExternal() {
            return fmt.Errorf("command required for %s", s.Slug)
        }
        if s.Health.IntervalSec <= 0 || s.Health.TimeoutSec <= 0 {
            return fmt.Errorf("invalid health timing for %s", s.Slug)
        }
    }
    return nil
}

// LoadDefault loads the registry from the default location (~/.mcp/registry.json).
// If the file doesn't exist, returns a new empty registry with default version.
func LoadDefault() (*Registry, error) {
    path, err := DefaultPath()
    if err != nil {
        return nil, fmt.Errorf("failed to get default path: %w", err)
    }
    return LoadOrDefault(path)
}

// LoadOrDefault loads a registry from the specified path.
// If the file doesn't exist, returns a new empty registry with default version.
// Returns an error only for parsing or validation failures.
func LoadOrDefault(path string) (*Registry, error) {
    r, err := Load(path)
    if err != nil {
        // If file doesn't exist, return default registry
        if os.IsNotExist(err) {
            return NewDefault(), nil
        }
        return nil, err
    }
    return r, nil
}

// NewDefault creates a new empty registry with default settings.
func NewDefault() *Registry {
    return &Registry{
        Version: "1.0",
        Servers: []Server{},
    }
}

