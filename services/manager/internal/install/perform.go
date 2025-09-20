package install

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "mcp/manager/internal/paths"
)

type PerformInput struct {
    Type    SourceType `json:"type"`
    URI     string     `json:"uri"`
    Slug    string     `json:"slug"`
    Runtime string     `json:"runtime"`
    Manager string     `json:"manager"`
}

type PerformResult struct {
    OK      bool     `json:"ok"`
    Logs    []string `json:"logs"`
    Message string   `json:"message"`
}

type Logger interface { Log(line string) }

type sliceLogger struct{ lines *[]string }
func (l sliceLogger) Log(line string) { *l.lines = append(*l.lines, line) }

func logf(dst Logger, format string, a ...any) {
    dst.Log(time.Now().Format("15:04:05 ")+fmt.Sprintf(format, a...))
}

// Perform executes installation based on input and normalizes the layout under ~/.mcp/servers/<slug>/.
func Perform(ctx context.Context, in PerformInput, r Runner) (PerformResult, error) {
    var logs []string
    res, err := PerformStream(ctx, in, r, sliceLogger{lines: &logs})
    if err != nil { return PerformResult{OK: false, Logs: logs, Message: err.Error()}, nil }
    res.Logs = logs
    return res, nil
}

// PerformStream is like Perform but streams logs to a logger.
func PerformStream(ctx context.Context, in PerformInput, r Runner, lg Logger) (PerformResult, error) {
    baseServers, err := paths.ServersDir()
    if err != nil { return PerformResult{OK: false}, err }
    dest := filepath.Join(baseServers, in.Slug)
    if err := os.MkdirAll(dest, 0o755); err != nil { return PerformResult{OK: false}, err }
    installDir := filepath.Join(dest, "install")
    runtimeDir := filepath.Join(dest, "runtime")
    binDir := filepath.Join(dest, "bin")
    if err := os.MkdirAll(installDir, 0o755); err != nil { return PerformResult{OK: false}, err }
    if err := os.MkdirAll(runtimeDir, 0o755); err != nil { return PerformResult{OK: false}, err }
    if err := os.MkdirAll(binDir, 0o755); err != nil { return PerformResult{OK: false}, err }

    if r == nil { r = ExecRunner{} }
    switch in.Type {
    case SrcGit:
        logf(lg, "git clone %s", in.URI)
        if _, _, err := r.Run(ctx, "git", "clone", "--depth", "1", in.URI, installDir); err != nil {
            logf(lg, "git clone failed: %v", err); return PerformResult{OK: false}, nil
        }
    case SrcNpm:
        logf(lg, "npm init runtime and install %s", in.URI)
        if _, _, err := r.Run(ctx, "npm", "init", "-y"); err != nil {
            // ignore init failure; continue
        }
        if _, _, err := r.Run(ctx, "npm", "--prefix", runtimeDir, "install", in.URI); err != nil {
            logf(lg, "npm install failed: %v", err); return PerformResult{OK: false}, nil
        }
    case SrcPip:
        logf(lg, "pip install %s", in.URI)
        if _, _, err := r.Run(ctx, "python3", "-m", "venv", filepath.Join(runtimeDir, "venv")); err == nil {
            // ok
        }
        // Note: using pip within venv would require activation; left for runtime.
        if _, _, err := r.Run(ctx, "pip", "install", "--target", runtimeDir, in.URI); err != nil {
            logf(lg, "pip install failed: %v", err); return PerformResult{OK: false}, nil
        }
    case SrcDocker:
        logf(lg, "docker pull %s", in.URI)
        if _, _, err := r.Run(ctx, "docker", "pull", in.URI); err != nil {
            logf(lg, "docker pull failed: %v", err); return PerformResult{OK: false}, nil
        }
    case SrcCompose:
        logf(lg, "docker compose config")
        if _, _, err := r.Run(ctx, "docker", "compose", "config"); err != nil {
            logf(lg, "compose config failed: %v", err); return PerformResult{OK: false}, nil
        }
    default:
        return PerformResult{OK: false}, fmt.Errorf("unsupported type: %s", in.Type)
    }

    // Write a minimal manifest.json
    manifest := map[string]any{
        "source": map[string]string{"type": string(in.Type), "uri": in.URI},
        "runtime": in.Runtime,
        "manager": in.Manager,
    }
    b, _ := json.MarshalIndent(manifest, "", "  ")
    if err := os.WriteFile(filepath.Join(dest, "manifest.json"), b, 0o644); err != nil {
        return PerformResult{OK: false, Message: err.Error()}, nil
    }

    // Create a bin entrypoint placeholder; real command will be set via registry entry
    entry := filepath.Join(binDir, in.Slug)
    _ = os.WriteFile(entry, []byte("#!/bin/sh\necho 'MCP server placeholder; configure entry.command in registry.'\n"), 0o755)

    logf(lg, "installed %s", in.Slug)
    return PerformResult{OK: true, Message: "installed"}, nil
}
