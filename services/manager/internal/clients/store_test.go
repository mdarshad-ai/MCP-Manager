package clients

import (
    "os"
    "path/filepath"
    "testing"
)

func TestStore_LoadSave(t *testing.T) {
    dir := t.TempDir()
    p := filepath.Join(dir, "config.json")
    s := &Store{Known: map[string]string{"cursor": "/Users/you/.cursor/mcp.json"}}
    if err := s.Save(p); err != nil { t.Fatal(err) }
    if _, err := os.Stat(p); err != nil { t.Fatal(err) }
    var s2 Store
    if err := s2.Load(p); err != nil { t.Fatal(err) }
    if s2.Known["cursor"] == "" { t.Fatal("missing cursor path") }
}

