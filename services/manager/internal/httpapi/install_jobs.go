package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"mcp/manager/internal/install"
	"mcp/manager/internal/registry"
)

// AdvancedInstallRequest represents a request for advanced installation
type AdvancedInstallRequest struct {
	Type    install.SourceType `json:"type"`
	URI     string             `json:"uri"`
	Slug    string             `json:"slug"`
	Options json.RawMessage    `json:"options,omitempty"`
}

// InstallJobResponse represents the response for installation operations
type InstallJobResponse struct {
	JobID   string `json:"jobId"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (s *Server) handleInstallStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Handle advanced installation request
	var advancedReq AdvancedInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&advancedReq); err != nil {
		writeJSON(w, InstallJobResponse{
			Status:  "error",
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	if advancedReq.Type == "" || advancedReq.Slug == "" {
		writeJSON(w, InstallJobResponse{
			Status:  "error",
			Message: "Missing required fields: type and slug are required",
		})
		return
	}

	s.handleAdvancedInstallStart(w, r, advancedReq)
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

func (s *Server) handleInstallLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
			"status":  "error",
			"message": "Installation service not available: " + err.Error(),
		})
		return
	}

	status, err := installService.GetJobStatus(id)
	if err != nil {
		writeJSON(w, map[string]string{
			"status":  "error",
			"message": "Job not found: " + err.Error(),
		})
		return
	}

	writeJSON(w, status)
}

func (s *Server) handleInstallCancel(w http.ResponseWriter, r *http.Request) {
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
			"status":  "error",
			"message": "Installation service not available: " + err.Error(),
		})
		return
	}

	if err := installService.CancelJob(id); err != nil {
		writeJSON(w, map[string]string{
			"status":  "error",
			"message": "Failed to cancel job: " + err.Error(),
		})
		return
	}

	writeJSON(w, map[string]string{"status": "cancelled"})
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
			"status":  "error",
			"message": "Installation service not available: " + err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := installService.FinalizeInstallation(ctx, id); err != nil {
		writeJSON(w, map[string]string{
			"status":  "error",
			"message": "Failed to finalize installation: " + err.Error(),
		})
		return
	}

	// After successful finalization, reload the registry to pick up newly registered servers
	if err := s.reloadRegistry(); err != nil {
		writeJSON(w, map[string]string{
			"status":  "warning",
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
