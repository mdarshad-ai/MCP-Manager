package clients

import (
    "encoding/json"
    "os"
    "path/filepath"
)

// WriteClaudeDesktop merges and writes configuration for Claude Desktop.
func WriteClaudeDesktop(path string, servers any) error {
    return writeJSON(path, servers)
}

// WriteCursorGlobal writes global Cursor MCP config.
func WriteCursorGlobal(path string, servers any) error { return writeJSON(path, servers) }

func writeJSON(path string, v any) error {
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
    b, _ := json.MarshalIndent(v, "", "  ")
    return os.WriteFile(path, b, 0o644)
}

