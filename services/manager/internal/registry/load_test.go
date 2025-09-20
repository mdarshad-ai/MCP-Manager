package registry

import (
    "os"
    "path/filepath"
    "testing"
)

func writeTemp(t *testing.T, data string) string {
    t.Helper()
    dir := t.TempDir()
    p := filepath.Join(dir, "registry.json")
    if err := os.WriteFile(p, []byte(data), 0o644); err != nil { t.Fatal(err) }
    return p
}

func TestLoad_Valid(t *testing.T) {
    p := writeTemp(t, `{
        "version":"1.0",
        "servers":[{
            "name":"fs","slug":"filesystem",
            "source":{"type":"git","uri":"https://x"},
            "runtime":{"kind":"node"},
            "entry":{"transport":"stdio","command":"node","args":[]},
            "health":{"probe":"mcp","method":"ping","intervalSec":20,"timeoutSec":5,"restartPolicy":"always","maxRestarts":5},
            "clients": {"claudeDesktop":{"enabled":true}}
        }]
    }`)
    r, err := Load(p)
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if len(r.Servers) != 1 { t.Fatalf("expected 1 server") }
}

func TestLoad_InvalidSlug(t *testing.T) {
    p := writeTemp(t, `{"version":"1.0","servers":[{"name":"x","slug":"Bad!","source":{"type":"git","uri":"u"},"runtime":{"kind":"node"},"entry":{"transport":"stdio","command":"node"},"health":{"probe":"mcp","method":"ping","intervalSec":20,"timeoutSec":5,"restartPolicy":"always","maxRestarts":5},"clients":{}}]}`)
    if _, err := Load(p); err == nil { t.Fatal("expected error") }
}

