package clients

import (
    "encoding/json"
    "os"
    "os/exec"
)

type Detection struct {
    Name           string       `json:"name"`
    Detected       bool         `json:"detected"`
    Path           string       `json:"path,omitempty"`
    ExistingMCPs   []MCPServer  `json:"existingMcps,omitempty"`
}

type MCPServer struct {
    Name      string            `json:"name"`
    Command   string            `json:"command"`
    Args      []string          `json:"args,omitempty"`
    Env       map[string]string `json:"env,omitempty"`
    Source    string            `json:"source"` // Which client config it came from
}

// DetectKnown checks a predefined set of clients by looking up commands and known config files.
func DetectKnown(p Paths) []Detection {
    list := []struct{ name, cmd, cfg string }{
        {"Claude Desktop", "", p.ClaudeDesktop},
        {"Cursor (Global)", "cursor", p.CursorGlobal},
        {"Continue", "continue", ""},
        {"Cursor CLI", "cursor", ""},
        {"Claude Code", "claude", ""},
        {"Codex CLI", "codex", ""},
        {"Gemini CLI", "gemini", ""},
    }
    out := make([]Detection, 0, len(list))
    for _, it := range list {
        det := Detection{Name: it.name}
        if it.cmd != "" {
            if path, err := exec.LookPath(it.cmd); err == nil { det.Detected, det.Path = true, path }
        }
        if it.cfg != "" {
            det.Detected = det.Detected || fileExists(it.cfg)
            if det.Path == "" { det.Path = it.cfg }
            // Scan for existing MCP servers in the config file
            if det.Detected {
                det.ExistingMCPs = scanMCPServers(it.cfg, it.name)
            }
        }
        out = append(out, det)
    }
    return out
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// scanMCPServers reads a client config file and extracts existing MCP server configurations
func scanMCPServers(configPath string, clientName string) []MCPServer {
    servers := []MCPServer{}
    
    if !fileExists(configPath) {
        return servers
    }
    
    data, err := os.ReadFile(configPath)
    if err != nil {
        return servers
    }
    
    var config map[string]interface{}
    if err := json.Unmarshal(data, &config); err != nil {
        return servers
    }
    
    // Handle different config formats
    // Claude Desktop format: mcpServers object
    if mcpServers, ok := config["mcpServers"].(map[string]interface{}); ok {
        for name, serverData := range mcpServers {
            if server, ok := serverData.(map[string]interface{}); ok {
                mcp := MCPServer{
                    Name:   name,
                    Source: clientName,
                }
                
                if cmd, ok := server["command"].(string); ok {
                    mcp.Command = cmd
                }
                
                if args, ok := server["args"].([]interface{}); ok {
                    mcp.Args = make([]string, len(args))
                    for i, arg := range args {
                        if s, ok := arg.(string); ok {
                            mcp.Args[i] = s
                        }
                    }
                }
                
                if env, ok := server["env"].(map[string]interface{}); ok {
                    mcp.Env = make(map[string]string)
                    for k, v := range env {
                        if s, ok := v.(string); ok {
                            mcp.Env[k] = s
                        }
                    }
                }
                
                servers = append(servers, mcp)
            }
        }
    }
    
    // Cursor/VS Code format: servers array
    if serversArray, ok := config["servers"].([]interface{}); ok {
        for _, serverData := range serversArray {
            if server, ok := serverData.(map[string]interface{}); ok {
                mcp := MCPServer{
                    Source: clientName,
                }
                
                if name, ok := server["name"].(string); ok {
                    mcp.Name = name
                }
                
                if cmd, ok := server["command"].(string); ok {
                    mcp.Command = cmd
                }
                
                if args, ok := server["args"].([]interface{}); ok {
                    mcp.Args = make([]string, len(args))
                    for i, arg := range args {
                        if s, ok := arg.(string); ok {
                            mcp.Args[i] = s
                        }
                    }
                }
                
                if env, ok := server["env"].(map[string]interface{}); ok {
                    mcp.Env = make(map[string]string)
                    for k, v := range env {
                        if s, ok := v.(string); ok {
                            mcp.Env[k] = s
                        }
                    }
                }
                
                if mcp.Name != "" {
                    servers = append(servers, mcp)
                }
            }
        }
    }
    
    return servers
}
