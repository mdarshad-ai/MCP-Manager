package httpapi

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"
    
    "mcp/manager/internal/paths"
    "mcp/manager/internal/registry"
)

// handleServerEnv handles environment variable updates for a specific server
func (s *Server) handleServerEnv(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut && r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    
    // Extract slug from path: /v1/servers/{slug}/env
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) < 5 {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    slug := parts[3]
    
    // Parse request body
    var body struct {
        EnvVars map[string]string `json:"envVars"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Find server in registry
    serverIndex := -1
    for i := range s.reg.Servers {
        if s.reg.Servers[i].Slug == slug {
            serverIndex = i
            break
        }
    }
    
    if serverIndex == -1 {
        http.Error(w, "server not found", http.StatusNotFound)
        return
    }
    
    // Update environment variables
    if s.reg.Servers[serverIndex].Entry.Env == nil {
        s.reg.Servers[serverIndex].Entry.Env = make(map[string]string)
    }
    
    for key, value := range body.EnvVars {
        if value == "" {
            // Remove if empty
            delete(s.reg.Servers[serverIndex].Entry.Env, key)
        } else {
            s.reg.Servers[serverIndex].Entry.Env[key] = value
        }
    }
    
    // Save registry
    if err := registry.SaveDefault(s.reg); err != nil {
        http.Error(w, "failed to save registry", http.StatusInternalServerError)
        return
    }
    
    // Update supervisor if available
    if s.sup != nil {
        s.sup.UpdateRegistry(s.reg)
    }
    
    writeJSON(w, map[string]string{"status": "ok"})
}

// handleStorageClear handles clearing of logs, cache, or all storage
func (s *Server) handleStorageClear(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    
    var body struct {
        Type string `json:"type"` // "logs", "cache", or "all"
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    
    var freedBytes int64
    
    switch body.Type {
    case "logs":
        // Clear logs directory
        logsDir, err := paths.LogsDir()
        if err != nil {
            http.Error(w, "failed to get logs directory", http.StatusInternalServerError)
            return
        }
        freed, err := clearDirectory(logsDir)
        if err != nil {
            http.Error(w, fmt.Sprintf("failed to clear logs: %v", err), http.StatusInternalServerError)
            return
        }
        freedBytes = freed
        
    case "cache":
        // Clear cache directory (if exists)
        cacheDir, err := paths.CacheDir()
        if err != nil {
            http.Error(w, "failed to get cache directory", http.StatusInternalServerError)
            return
        }
        freed, err := clearDirectory(cacheDir)
        if err != nil {
            http.Error(w, fmt.Sprintf("failed to clear cache: %v", err), http.StatusInternalServerError)
            return
        }
        freedBytes = freed
        
    case "all":
        // Clear both logs and cache
        logsDir, _ := paths.LogsDir()
        cacheDir, _ := paths.CacheDir()
        
        logsFreed, _ := clearDirectory(logsDir)
        cacheFreed, _ := clearDirectory(cacheDir)
        freedBytes = logsFreed + cacheFreed
        
    default:
        http.Error(w, "invalid type: must be 'logs', 'cache', or 'all'", http.StatusBadRequest)
        return
    }
    
    writeJSON(w, map[string]interface{}{
        "freed": freedBytes,
        "status": "ok",
    })
}

// handleSystemOpen handles opening files or URLs in the system's default application
func (s *Server) handleSystemOpen(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    
    var body struct {
        Path string `json:"path"`
        App  string `json:"app,omitempty"` // optional: specific app to use
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    
    if body.Path == "" {
        http.Error(w, "path is required", http.StatusBadRequest)
        return
    }
    
    // Use OS-specific command to open file/URL
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "darwin":
        if body.App != "" && body.App != "default" {
            cmd = exec.Command("open", "-a", body.App, body.Path)
        } else {
            cmd = exec.Command("open", body.Path)
        }
    case "linux":
        cmd = exec.Command("xdg-open", body.Path)
    case "windows":
        cmd = exec.Command("cmd", "/c", "start", body.Path)
    default:
        http.Error(w, "unsupported platform", http.StatusNotImplemented)
        return
    }
    
    if err := cmd.Start(); err != nil {
        http.Error(w, fmt.Sprintf("failed to open: %v", err), http.StatusInternalServerError)
        return
    }
    
    writeJSON(w, map[string]string{"status": "ok"})
}

// handleMacOSAutostart handles macOS-specific autostart configuration
func (s *Server) handleMacOSAutostart(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    
    if runtime.GOOS != "darwin" {
        http.Error(w, "only available on macOS", http.StatusBadRequest)
        return
    }
    
    var body struct {
        Enabled         bool   `json:"enabled"`
        AppPath         string `json:"appPath"`
        LaunchAgentPath string `json:"launchAgentPath"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    
    launchAgentPath := body.LaunchAgentPath
    if launchAgentPath == "" {
        homeDir, _ := os.UserHomeDir()
        launchAgentPath = filepath.Join(homeDir, "Library", "LaunchAgents", "com.mcp-manager.plist")
    }
    
    if body.Enabled {
        // Create LaunchAgent plist
        plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mcp-manager</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardErrorPath</key>
    <string>/tmp/mcp-manager.err</string>
    <key>StandardOutPath</key>
    <string>/tmp/mcp-manager.out</string>
</dict>
</plist>`, body.AppPath)
        
        // Ensure LaunchAgents directory exists
        launchAgentsDir := filepath.Dir(launchAgentPath)
        if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
            http.Error(w, fmt.Sprintf("failed to create LaunchAgents directory: %v", err), http.StatusInternalServerError)
            return
        }
        
        // Write plist file
        if err := os.WriteFile(launchAgentPath, []byte(plistContent), 0644); err != nil {
            http.Error(w, fmt.Sprintf("failed to write plist: %v", err), http.StatusInternalServerError)
            return
        }
        
        // Load the launch agent
        cmd := exec.Command("launchctl", "load", launchAgentPath)
        if err := cmd.Run(); err != nil {
            // Try to unload first in case it's already loaded
            exec.Command("launchctl", "unload", launchAgentPath).Run()
            if err := exec.Command("launchctl", "load", launchAgentPath).Run(); err != nil {
                http.Error(w, fmt.Sprintf("failed to load launch agent: %v", err), http.StatusInternalServerError)
                return
            }
        }
    } else {
        // Unload and remove launch agent
        exec.Command("launchctl", "unload", launchAgentPath).Run()
        os.Remove(launchAgentPath)
    }
    
    writeJSON(w, map[string]interface{}{
        "status": "ok",
        "enabled": body.Enabled,
        "path": launchAgentPath,
    })
}

// clearDirectory removes all files in a directory and returns the total bytes freed
func clearDirectory(dir string) (int64, error) {
    if dir == "" {
        return 0, fmt.Errorf("directory path is empty")
    }
    
    var totalFreed int64
    
    entries, err := os.ReadDir(dir)
    if err != nil {
        if os.IsNotExist(err) {
            return 0, nil // Directory doesn't exist, nothing to clear
        }
        return 0, err
    }
    
    for _, entry := range entries {
        path := filepath.Join(dir, entry.Name())
        info, err := entry.Info()
        if err != nil {
            continue
        }
        
        if entry.IsDir() {
            // Recursively clear subdirectory
            freed, _ := clearDirectory(path)
            totalFreed += freed
            os.Remove(path) // Try to remove empty directory
        } else {
            totalFreed += info.Size()
            os.Remove(path)
        }
    }
    
    return totalFreed, nil
}