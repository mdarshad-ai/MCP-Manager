package clients

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Store struct {
    Known map[string]string `json:"known"` // client -> config path
}

func (s *Store) Load(path string) error {
    b, err := os.ReadFile(path)
    if err != nil { return err }
    return json.Unmarshal(b, s)
}

func (s *Store) Save(path string) error {
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { return err }
    b, _ := json.MarshalIndent(s, "", "  ")
    return os.WriteFile(path, b, 0o644)
}

