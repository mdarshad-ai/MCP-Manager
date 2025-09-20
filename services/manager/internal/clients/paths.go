package clients

import (
    "os"
    "path/filepath"
)

type Paths struct {
    ClaudeDesktop string
    CursorGlobal  string
    Store         string // ~/.mcp/clients/config.json
}

func DefaultPaths() (Paths, error) {
    home, err := os.UserHomeDir()
    if err != nil { return Paths{}, err }
    return Paths{
        ClaudeDesktop: filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
        CursorGlobal:  filepath.Join(home, ".cursor", "mcp.json"),
        Store:         filepath.Join(home, ".mcp", "clients", "config.json"),
    }, nil
}

