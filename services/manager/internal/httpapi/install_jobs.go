package httpapi

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"

    "mcp/manager/internal/install"
    "mcp/manager/internal/registry"
)

// Legacy job structure for backward compatibility
type job struct {
    id   string
    logs []string
    ok   bool
    done bool
    msg  string
    mu   sync.Mutex
    cancel context.CancelFunc
}

// AdvancedInstallRequest represents a request for advanced installation
type AdvancedInstallRequest struct {
    Type    install.SourceType `json:"type"`
    URI     string              `json:"uri"`
    Slug    string              `json:"slug"`
    Options json.RawMessage     `json:"options,omitempty"`
}

// InstallJobResponse represents the response for installation operations
type InstallJobResponse struct {
    JobID   string `json:"jobId"`
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
}

func (s *Server) handleInstallStart(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
    
    // Check if it's an advanced installation request
    var advancedReq AdvancedInstallRequest
    if err := json.NewDecoder(r.Body).Decode(&advancedReq); err == nil && advancedReq.Type != "" && advancedReq.Slug != "" {
        s.handleAdvancedInstallStart(w, r, advancedReq)
        return
    }
    
    // Fall back to legacy installation for backward compatibility
    s.handleLegacyInstallStart(w, r)
}

func (s *Server) handleAdvancedInstallStart(w http.ResponseWriter, r *http.Request, req AdvancedInstallRequest) {
    ctx := context.Background()
    
    // Get or create the advanced installation service
    installService, err := s.getInstallationService()
    if err != nil {
        writeJSON(w, InstallJobResponse{
            Status:  "error",
            Message: "Failed to initialize installation service: " + err.Error(),
        })
        return
    }
    
    var jobID string
    
    switch req.Type {
    case install.SrcGit:
        var options install.GitInstallOptions
        if req.Options != nil {
            if err := json.Unmarshal(req.Options, &options); err != nil {
                writeJSON(w, InstallJobResponse{
                    Status:  "error",
                    Message: "Invalid git installation options: " + err.Error(),
                })
                return
            }
        }
        
        jobID, err = installService.InstallFromGit(ctx, req.Slug, req.URI, options)
        
    case install.SrcNpm:
        var options install.NPMInstallOptions
        if req.Options != nil {
            if err := json.Unmarshal(req.Options, &options); err != nil {
                writeJSON(w, InstallJobResponse{
                    Status:  "error",
                    Message: "Invalid npm installation options: " + err.Error(),
                })
                return
            }
        }
        
        jobID, err = installService.InstallFromNPM(ctx, req.Slug, req.URI, options)
        
    case install.SrcPip:
        var options install.PipInstallOptions
        if req.Options != nil {
            if err := json.Unmarshal(req.Options, &options); err != nil {
                writeJSON(w, InstallJobResponse{
                    Status:  "error",
                    Message: "Invalid pip installation options: " + err.Error(),
                })
                return
            }
        }
        
        jobID, err = installService.InstallFromPip(ctx, req.Slug, req.URI, options)
        
    default:
        writeJSON(w, InstallJobResponse{
            Status:  "error",
            Message: "Unsupported installation type: " + string(req.Type),
        })
        return
    }
    
    if err != nil {
        writeJSON(w, InstallJobResponse{
            Status:  "error",
            Message: "Failed to start installation: " + err.Error(),
        })
        return
    }
    
    writeJSON(w, InstallJobResponse{
        JobID:  jobID,
        Status: "started",
    })
}

func (s *Server) handleLegacyInstallStart(w http.ResponseWriter, r *http.Request) {
    var in install.PerformInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil { 
        w.WriteHeader(http.StatusBadRequest)
        return 
    }
    
    id := time.Now().Format("20060102T150405.000")
    ctx, cancel := context.WithCancel(context.Background())
    j := &job{id: id, cancel: cancel}
    s.jobsMu.Lock()
    s.jobs[id] = j
    s.jobsMu.Unlock()
    
    go func() {
        logger := jobLogger{j: j}
        res, _ := install.PerformStream(ctx, in, install.ExecRunner{}, logger)
        j.mu.Lock()
        j.ok = res.OK
        j.done = true
        j.msg = res.Message
        j.mu.Unlock()
    }()
    
    writeJSON(w, map[string]string{"id": id})
}

type jobLogger struct{ j *job }
func (l jobLogger) Log(line string) { 
    l.j.mu.Lock()
    l.j.logs = append(l.j.logs, line)
    l.j.mu.Unlock()
}

func (s *Server) handleInstallLogs(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    id := r.URL.Query().Get("id")
    
    // Try advanced installation service first
    installService, err := s.getInstallationService()
    if err == nil {
        if job, err := installService.GetJobStatus(id); err == nil {
            writeJSON(w, job)
            return
        }
    }
    
    // Fall back to legacy job system
    s.jobsMu.Lock()
    j := s.jobs[id]
    s.jobsMu.Unlock()
    
    if j == nil { 
        w.WriteHeader(http.StatusNotFound)
        return 
    }
    
    j.mu.Lock()
    defer j.mu.Unlock()
    writeJSON(w, map[string]any{
        "id": j.id, 
        "logs": j.logs, 
        "done": j.done, 
        "ok": j.ok, 
        "message": j.msg,
    })
}

func (s *Server) handleInstallCancel(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
    id := r.URL.Query().Get("id")
    
    // Try advanced installation service first
    installService, err := s.getInstallationService()
    if err == nil {
        if err := installService.CancelJob(id); err == nil {
            writeJSON(w, map[string]string{"status": "cancelled"})
            return
        }
    }
    
    // Fall back to legacy job system
    s.jobsMu.Lock()
    j := s.jobs[id]
    s.jobsMu.Unlock()
    
    if j == nil { 
        w.WriteHeader(http.StatusNotFound)
        return 
    }
    
    if j.cancel != nil { j.cancel() }
    writeJSON(w, map[string]string{"status": "cancelling"})
}

func (s *Server) handleInstallFinalize(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { 
        w.WriteHeader(http.StatusMethodNotAllowed)
        return 
    }
    
    id := r.URL.Query().Get("id")
    if id == "" {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    installService, err := s.getInstallationService()
    if err != nil {
        writeJSON(w, map[string]string{
            "status": "error",
            "message": "Installation service not available: " + err.Error(),
        })
        return
    }
    
    ctx := context.Background()
    if err := installService.FinalizeInstallation(ctx, id); err != nil {
        writeJSON(w, map[string]string{
            "status": "error",
            "message": "Failed to finalize installation: " + err.Error(),
        })
        return
    }
    
    // After successful finalization, reload the registry to pick up newly registered servers
    if err := s.reloadRegistry(); err != nil {
        writeJSON(w, map[string]string{
            "status": "warning", 
            "message": "Installation finalized but failed to reload registry: " + err.Error(),
        })
        return
    }
    
    writeJSON(w, map[string]string{"status": "finalized"})
}

func (s *Server) handleInstallList(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { 
        w.WriteHeader(http.StatusMethodNotAllowed)
        return 
    }
    
    installService, err := s.getInstallationService()
    if err != nil {
        writeJSON(w, map[string]string{
            "error": "Installation service not available: " + err.Error(),
        })
        return
    }
    
    jobs := installService.ListJobs()
    writeJSON(w, map[string]interface{}{
        "jobs": jobs,
    })
}

// getInstallationService returns the installation service, creating it if necessary
func (s *Server) getInstallationService() (*install.AdvancedInstallationService, error) {
    if s.installService == nil {
        var err error
        s.installService, err = install.NewAdvancedInstallationService(5) // Max 5 concurrent jobs
        if err != nil {
            return nil, err
        }
    }
    return s.installService, nil
}

// reloadRegistry reloads the registry from disk and updates the server's registry reference
func (s *Server) reloadRegistry() error {
    newReg, err := registry.LoadDefault()
    if err != nil {
        return fmt.Errorf("failed to reload registry: %w", err)
    }
    
    s.reg = newReg
    
    // Update the supervisor with the new registry so it can manage newly registered servers
    if s.sup != nil {
        s.sup.UpdateRegistry(newReg)
    }
    
    return nil
}

