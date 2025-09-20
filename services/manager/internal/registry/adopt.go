package registry

import (
    "fmt"
    "strings"
    "mcp/manager/internal/clients"
)

// AdoptExistingMCP converts a detected MCP server into a registry Server entry
func AdoptExistingMCP(mcp clients.MCPServer) Server {
    // Generate a slug from the name
    slug := strings.ToLower(strings.ReplaceAll(mcp.Name, " ", "-"))
    slug = strings.ReplaceAll(slug, "_", "-")
    
    // Determine runtime based on command
    runtime := determineRuntime(mcp.Command, mcp.Args)
    
    // Determine transport (stdio is most common)
    transport := "stdio"
    if mcp.Env != nil {
        if t, ok := mcp.Env["TRANSPORT"]; ok {
            transport = t
        }
    }
    
    server := Server{
        Name: mcp.Name,
        Slug: slug,
        Source: Source{
            Type: "existing",
            URI:  mcp.Command, // Store the original command path
        },
        Runtime: runtime,
        Entry: Entry{
            Transport: transport,
            Command:   mcp.Command,
            Args:      mcp.Args,
            Env:       mcp.Env,
        },
        Health: Health{
            Probe:         "startup",
            Method:        "startup",
            IntervalSec:   30,
            TimeoutSec:    5,
            RestartPolicy: "on-failure",
            MaxRestarts:   3,
        },
        Clients: Clients{
            // Enable for the client it came from
            ClaudeDesktop: &ClientFlag{Enabled: strings.Contains(mcp.Source, "Claude")},
            CursorGlobal:  &ClientFlag{Enabled: strings.Contains(mcp.Source, "Cursor")},
            Continue:      &ClientFlag{Enabled: strings.Contains(mcp.Source, "Continue")},
        },
    }
    
    return server
}

func determineRuntime(command string, args []string) Runtime {
    // Check if it's a Node.js runtime
    if strings.Contains(command, "node") || strings.Contains(command, "npx") || strings.Contains(command, "npm") {
        return Runtime{
            Kind: "node",
            Node: &NodeRuntime{
                PackageManager: "npm",
            },
        }
    }
    
    // Check for Python runtime
    if strings.Contains(command, "python") || strings.Contains(command, "pip") {
        return Runtime{
            Kind: "python",
            Python: &PyRuntime{
                Manager: "pip",
                Venv:    strings.Contains(command, "venv") || strings.Contains(command, ".env"),
            },
        }
    }
    
    // Check for UV (Python package manager)
    if strings.Contains(command, "uv") || strings.Contains(command, "uvx") {
        return Runtime{
            Kind: "python",
            Python: &PyRuntime{
                Manager: "uv",
                Venv:    false,
            },
        }
    }
    
    // Check args for runtime hints
    if len(args) > 0 {
        firstArg := args[0]
        if strings.HasSuffix(firstArg, ".js") || strings.HasSuffix(firstArg, ".mjs") {
            return Runtime{
                Kind: "node",
                Node: &NodeRuntime{
                    PackageManager: "npm",
                },
            }
        }
        if strings.HasSuffix(firstArg, ".py") {
            return Runtime{
                Kind: "python",
                Python: &PyRuntime{
                    Manager: "pip",
                    Venv:    false,
                },
            }
        }
    }
    
    // Default to binary/executable
    return Runtime{
        Kind: "binary",
    }
}

// AdoptMultipleMCPs adopts multiple detected MCP servers into the registry
func (r *Registry) AdoptMCPs(mcps []clients.MCPServer) []Server {
    adoptedServers := make([]Server, 0, len(mcps))
    
    for _, mcp := range mcps {
        // Check if server already exists
        exists := false
        for _, existing := range r.Servers {
            if existing.Name == mcp.Name || existing.Entry.Command == mcp.Command {
                exists = true
                break
            }
        }
        
        if !exists {
            server := AdoptExistingMCP(mcp)
            r.Servers = append(r.Servers, server)
            adoptedServers = append(adoptedServers, server)
        }
    }
    
    return adoptedServers
}

// DetectAndAdoptMCPs scans client configurations and adopts any found MCPs
func (r *Registry) DetectAndAdoptMCPs(paths clients.Paths) ([]Server, error) {
    detections := clients.DetectKnown(paths)
    
    allMCPs := []clients.MCPServer{}
    for _, detection := range detections {
        if detection.Detected && len(detection.ExistingMCPs) > 0 {
            allMCPs = append(allMCPs, detection.ExistingMCPs...)
        }
    }
    
    if len(allMCPs) == 0 {
        return nil, fmt.Errorf("no existing MCPs found in client configurations")
    }
    
    adopted := r.AdoptMCPs(allMCPs)
    return adopted, nil
}