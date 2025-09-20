package paths

import (
    "os"
    "path/filepath"
)

// HomeMCP returns the base ~/.mcp directory and ensures it exists.
func HomeMCP() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil { return "", err }
    base := filepath.Join(home, ".mcp")
    if err := os.MkdirAll(base, 0o755); err != nil { return "", err }
    return base, nil
}

func LogsDir() (string, error) {
    base, err := HomeMCP()
    if err != nil { return "", err }
    p := filepath.Join(base, "logs")
    if err := os.MkdirAll(p, 0o755); err != nil { return "", err }
    return p, nil
}

func ServersDir() (string, error) {
    base, err := HomeMCP()
    if err != nil { return "", err }
    p := filepath.Join(base, "servers")
    if err := os.MkdirAll(p, 0o755); err != nil { return "", err }
    return p, nil
}

// CacheDir returns the cache directory (~/.mcp/cache) and ensures it exists.
func CacheDir() (string, error) {
    base, err := HomeMCP()
    if err != nil { return "", err }
    p := filepath.Join(base, "cache")
    if err := os.MkdirAll(p, 0o755); err != nil { return "", err }
    return p, nil
}

// SecretsDir returns the secrets directory (~/.mcp/secrets) and ensures it exists.
func SecretsDir() (string, error) {
    base, err := HomeMCP()
    if err != nil { return "", err }
    p := filepath.Join(base, "secrets")
    if err := os.MkdirAll(p, 0o700); err != nil { return "", err } // More restrictive permissions for secrets
    return p, nil
}

// EnsureAllDirectories creates all required MCP directories if they don't exist.
// This is useful for initialization to ensure the complete directory structure exists.
func EnsureAllDirectories() error {
    dirs := []func() (string, error){
        HomeMCP,
        LogsDir,
        ServersDir,
        CacheDir,
        SecretsDir,
    }
    
    for _, dirFunc := range dirs {
        if _, err := dirFunc(); err != nil {
            return err
        }
    }
    
    return nil
}

