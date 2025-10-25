package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mcp/manager/internal/clients"
	"mcp/manager/internal/health"
	"mcp/manager/internal/install"
	"mcp/manager/internal/logs"
	"mcp/manager/internal/paths"
	"mcp/manager/internal/registry"
)

type Server struct {
	reg               *registry.Registry
	sup               Supervisor
	healthMonitor     HealthMonitor
	logStreamer       LogStreamer
	installService    *install.AdvancedInstallationService
	credentialManager *CredentialManager
}

type Supervisor interface {
	Summary() []map[string]any
	Start(slug string) error
	Stop(slug string, graceful time.Duration) error
	Restart(slug string) error
	GetProcessInfo(slug string) map[string]interface{}
	Stats() map[string]interface{}
	Shutdown(timeout time.Duration) error
	UpdateRegistry(newReg *registry.Registry)
}

type HealthMonitor interface {
	AddProcess(name, transport, httpURL, logPath string)
	RemoveProcess(name string)
	AddExternalProcess(name, provider, apiEndpoint, authType string)
	RemoveExternalProcess(name string)
	GetProcessHealth(name string) (*health.ProcessHealth, bool)
	GetExternalProcessHealth(name string) (*health.ExternalProcessHealth, bool)
	GetAllHealth() map[string]*health.ProcessHealth
	GetAllExternalHealth() map[string]*health.ExternalProcessHealth
	GetHealthSummary() map[string]interface{}
	Start()
	Stop()
}

type LogStreamer interface {
	StreamLogs(clientID, process string, fromLine int64) (*logs.StreamClient, error)
	StopStream(clientID string)
	GetActiveStreams() map[string]interface{}
	Start()
	Stop()
}

func NewServer(reg *registry.Registry) *Server {
	return &Server{reg: reg}
}

func (s *Server) WithSupervisor(sup Supervisor) *Server {
	s.sup = sup
	return s
}

func (s *Server) WithHealthMonitor(hm HealthMonitor) *Server {
	s.healthMonitor = hm
	return s
}

func (s *Server) WithLogStreamer(ls LogStreamer) *Server {
	s.logStreamer = ls
	return s
}

func (s *Server) WithCredentialManager(cm *CredentialManager) *Server {
	s.credentialManager = cm
	return s
}

// Router returns the HTTP handler.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	// Core server management
	mux.HandleFunc("/v1/servers", s.handleServers)
	mux.HandleFunc("/v1/servers/", s.handleServerActions) // /v1/servers/{slug}/actions or /v1/servers/{slug}/info or /v1/servers/{slug}/env

	// Enhanced monitoring endpoints
	mux.HandleFunc("/v1/health", s.handleHealth)
	mux.HandleFunc("/v1/health/", s.handleHealthDetail) // /v1/health/{slug}
	mux.HandleFunc("/v1/health/external", s.handleExternalHealthSummary)
	mux.HandleFunc("/v1/health/external/", s.handleExternalHealthDetail) // /v1/health/external/{slug}
	mux.HandleFunc("/v1/stats", s.handleStats)

	// Log streaming endpoints
	mux.HandleFunc("/v1/logs/stream/", s.handleLogStream) // /v1/logs/stream/{slug}
	mux.HandleFunc("/v1/logs/", s.handleLogs)             // /v1/logs/{slug}

	// Installation endpoints
	mux.HandleFunc("/v1/install/start", s.handleInstallStart)
	mux.HandleFunc("/v1/install/logs", s.handleInstallLogs)
	mux.HandleFunc("/v1/install/cancel", s.handleInstallCancel)
	mux.HandleFunc("/v1/install/finalize", s.handleInstallFinalize)
	mux.HandleFunc("/v1/install/list", s.handleInstallList)

	// Client configuration endpoints
	mux.HandleFunc("/v1/clients/detect", s.handleClientsDetect)
	mux.HandleFunc("/v1/clients/apply", s.handleClientsApply)
	mux.HandleFunc("/v1/clients/preview", s.handleClientsPreview)
	mux.HandleFunc("/v1/clients/current", s.handleClientsCurrent)
	mux.HandleFunc("/v1/clients/paths", s.handleClientsPaths)
	mux.HandleFunc("/v1/clients/adopt", s.handleClientsAdopt)

	// External server management endpoints
	mux.HandleFunc("/v1/external/servers", s.handleExternalMCPs)
	mux.HandleFunc("/v1/external/servers/", s.handleExternalMCPActions) // /v1/external/servers/{slug} or /v1/external/servers/{slug}/test
	mux.HandleFunc("/v1/external/providers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleListProviders(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/v1/external/providers/", func(w http.ResponseWriter, r *http.Request) {
		// Handle /v1/external/providers/{name}
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) == 5 && r.Method == http.MethodGet {
			s.handleGetProvider(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// Settings endpoints
	mux.HandleFunc("/v1/settings/autostart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleAutostartGet(w, r)
			return
		}
		if r.Method == http.MethodPost {
			s.handleAutostartSet(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/v1/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleSettingsGet(w, r)
			return
		}
		if r.Method == http.MethodPut || r.Method == http.MethodPost {
			s.handleSettingsUpdate(w, r)
			return
		}
		if r.Method == http.MethodPatch {
			s.handleSettingsPartial(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/v1/settings/reset", s.handleSettingsReset)

	// Storage management endpoints
	mux.HandleFunc("/v1/storage/clear", s.handleStorageClear)

	// System endpoints
	mux.HandleFunc("/v1/system/open", s.handleSystemOpen)
	mux.HandleFunc("/v1/system/macos/autostart", s.handleMacOSAutostart)

	// Credential management endpoints
	mux.HandleFunc("/v1/credentials", s.handleCredentialsStore)
	mux.HandleFunc("/v1/credentials/", func(w http.ResponseWriter, r *http.Request) {
		// Route based on path segments
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) == 4 {
			// /v1/credentials/{provider}
			switch r.Method {
			case http.MethodGet:
				s.handleCredentialsGet(w, r)
			case http.MethodPut:
				s.handleCredentialsUpdate(w, r)
			case http.MethodDelete:
				s.handleCredentialsDelete(w, r)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/v1/credentials/validate", s.handleCredentialsValidate)
	mux.HandleFunc("/v1/credentials/status", s.handleCredentialsStatus)
	mux.HandleFunc("/v1/credentials/validate-stored", s.handleCredentialsValidateStored)

	// Admin panel endpoints
	mux.HandleFunc("/v1/admin/servers", s.handleAdminListServers)
	mux.HandleFunc("/v1/admin/servers/", s.handleAdminServerActions) // /v1/admin/servers/{slug}
	mux.HandleFunc("/v1/admin/marketplace", s.handleAdminMarketplace)
	mux.HandleFunc("/v1/admin/marketplace/", s.handleAdminMarketplaceActions) // /v1/admin/marketplace/{slug} or /v1/admin/marketplace/{slug}/resolve-attention

	return withCORS(logRequests(mux))
}

func (s *Server) handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if s.sup != nil {
			writeJSON(w, s.sup.Summary())
			return
		}
		type outServer struct{ Name, Slug, Status string }
		var out []outServer
		for _, v := range s.reg.Servers {
			// Only include servers that are not external
			if !v.IsExternal() {
				out = append(out, outServer{Name: v.Name, Slug: v.Slug, Status: "down"})
			}
		}
		writeJSON(w, out)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleServerActions(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slug := parts[3]
	action := parts[4]

	switch action {
	case "actions":
		s.handleServerActionsPost(w, r, slug)
	case "info":
		s.handleServerInfo(w, r, slug)
	case "env":
		s.handleServerEnv(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleServerActionsPost handles POST requests to /v1/servers/{slug}/actions
func (s *Server) handleServerActionsPost(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if s.sup == nil {
		log.Printf("action %s for %s ignored: no supervisor", body.Action, slug)
		writeJSON(w, map[string]string{"status": "noop", "message": "supervisor not available"})
		return
	}

	switch body.Action {
	case "start":
		if err := s.sup.Start(slug); err != nil {
			writeJSON(w, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		// Add to health monitoring if available
		if s.healthMonitor != nil {
			if sv := s.findServer(slug); sv != nil {
				httpURL := ""
				if sv.Entry.Transport == "http" {
					httpURL = deriveHTTPURL(sv.Entry.Args, sv.Entry.Env)
				}
				logPath := fmt.Sprintf("/var/log/mcp/%s.log", slug) // TODO: Use proper logs dir
				s.healthMonitor.AddProcess(slug, sv.Entry.Transport, httpURL, logPath)
			}
		}
	case "restart":
		if err := s.sup.Restart(slug); err != nil {
			writeJSON(w, map[string]string{"status": "error", "message": err.Error()})
			return
		}
	case "stop":
		if err := s.sup.Stop(slug, 10*time.Second); err != nil {
			writeJSON(w, map[string]string{"status": "error", "message": err.Error()})
			return
		}
		// Remove from health monitoring
		if s.healthMonitor != nil {
			s.healthMonitor.RemoveProcess(slug)
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

// handleServerInfo handles GET requests to /v1/servers/{slug}/info
func (s *Server) handleServerInfo(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.sup == nil {
		writeJSON(w, map[string]string{"error": "supervisor not available"})
		return
	}

	info := s.sup.GetProcessInfo(slug)
	writeJSON(w, info)
}

// findServer finds a server by slug
func (s *Server) findServer(slug string) *registry.Server {
	for i := range s.reg.Servers {
		if s.reg.Servers[i].Slug == slug {
			return &s.reg.Servers[i]
		}
	}
	return nil
}

// handleHealth handles GET requests to /v1/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.healthMonitor == nil {
		writeJSON(w, map[string]string{"error": "health monitoring not available"})
		return
	}

	summary := s.healthMonitor.GetHealthSummary()
	writeJSON(w, summary)
}

// handleHealthDetail handles GET requests to /v1/health/{slug}
func (s *Server) handleHealthDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slug := parts[3]

	if s.healthMonitor == nil {
		writeJSON(w, map[string]string{"error": "health monitoring not available"})
		return
	}

	health, exists := s.healthMonitor.GetProcessHealth(slug)
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeJSON(w, health)
}

// handleExternalHealthSummary handles GET requests to /v1/health/external
func (s *Server) handleExternalHealthSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.healthMonitor == nil {
		writeJSON(w, map[string]string{"error": "health monitoring not available"})
		return
	}

	externalHealth := s.healthMonitor.GetAllExternalHealth()

	summary := map[string]interface{}{
		"totalExternal": len(externalHealth),
		"healthy":       0,
		"degraded":      0,
		"down":          0,
		"servers":       make([]map[string]interface{}, 0, len(externalHealth)),
	}

	for _, ph := range externalHealth {
		switch ph.Status {
		case health.Ready:
			summary["healthy"] = summary["healthy"].(int) + 1
		case health.Degraded:
			summary["degraded"] = summary["degraded"].(int) + 1
		case health.Down:
			summary["down"] = summary["down"].(int) + 1
		}

		serverInfo := map[string]interface{}{
			"name":              ph.Name,
			"provider":          ph.Provider,
			"status":            string(ph.Status),
			"lastCheck":         ph.LastCheck,
			"lastSuccess":       ph.LastSuccess,
			"consecutiveFails":  ph.ConsecutiveFails,
			"totalChecks":       ph.TotalChecks,
			"totalFailures":     ph.TotalFailures,
			"avgResponseTime":   ph.AvgResponseTime.Milliseconds(),
			"credentialWarning": ph.CredentialWarning,
			"rateLimited":       ph.RateLimited,
			"lastErrorCode":     ph.LastErrorCode,
		}

		summary["servers"] = append(summary["servers"].([]map[string]interface{}), serverInfo)
	}

	writeJSON(w, summary)
}

// handleExternalHealthDetail handles GET requests to /v1/health/external/{slug}
func (s *Server) handleExternalHealthDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slug := parts[4]

	if s.healthMonitor == nil {
		writeJSON(w, map[string]string{"error": "health monitoring not available"})
		return
	}

	health, exists := s.healthMonitor.GetExternalProcessHealth(slug)
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeJSON(w, health)
}

// handleStats handles GET requests to /v1/stats
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{}

	// Add supervisor stats
	if s.sup != nil {
		response["supervisor"] = s.sup.Stats()
	}

	// Add health monitoring stats (includes both local and external)
	if s.healthMonitor != nil {
		response["health"] = s.healthMonitor.GetHealthSummary()
	}

	// Add log streaming stats
	if s.logStreamer != nil {
		response["logs"] = s.logStreamer.GetActiveStreams()
	}

	writeJSON(w, response)
}

// handleLogStream handles WebSocket connections for log streaming
func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slug := parts[4]
	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = fmt.Sprintf("client-%d", time.Now().UnixNano())
	}

	// Parse optional fromLine parameter
	fromLine := int64(-1)
	if fromLineStr := r.URL.Query().Get("fromLine"); fromLineStr != "" {
		if parsed, err := strconv.ParseInt(fromLineStr, 10, 64); err == nil {
			fromLine = parsed
		}
	}

	if s.logStreamer == nil {
		http.Error(w, "log streaming not available", http.StatusServiceUnavailable)
		return
	}

	// For now, return JSON streaming instead of WebSocket
	// TODO: Implement proper WebSocket support
	client, err := s.logStreamer.StreamLogs(clientID, slug, fromLine)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to start log stream: %v", err), http.StatusInternalServerError)
		return
	}

	// Set up Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream log entries
	for {
		select {
		case entry, ok := <-client.Ch:
			if !ok {
				return // Channel closed
			}

			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			s.logStreamer.StopStream(clientID)
			return
		}
	}
}

func (s *Server) handleClientsDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	p, err := clients.DefaultPaths()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	out := clients.DetectKnown(p)
	writeJSON(w, out)
}

func (s *Server) handleClientsApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Client string         `json:"client"`
		Config map[string]any `json:"config"`
		Path   string         `json:"path,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p, _ := clients.DefaultPaths()
	switch body.Client {
	case "Claude Desktop":
		path := body.Path
		if path == "" {
			path = p.ClaudeDesktop
		}
		if err := clients.WriteClaudeDesktop(path, body.Config); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case "Cursor (Global)":
		path := body.Path
		if path == "" {
			path = p.CursorGlobal
		}
		if err := clients.WriteCursorGlobal(path, body.Config); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		// For CLI tools, we only provide snippet generation on the UI side in v0
		writeJSON(w, map[string]string{"status": "snippet-only"})
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleClientsPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	client := r.URL.Query().Get("client")
	cfg := buildClientConfig(s.reg)
	// For v0 we return the same structure for all supported writers.
	_ = client // placeholder for client-specific transforms later
	writeJSON(w, cfg)
}

func (s *Server) handleClientsCurrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	client := r.URL.Query().Get("client")
	p, _ := clients.DefaultPaths()
	var path string
	switch client {
	case "Claude Desktop":
		path = p.ClaudeDesktop
	case "Cursor (Global)":
		path = p.CursorGlobal
	default:
		writeJSON(w, map[string]any{})
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (s *Server) handleClientsPaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	p, err := clients.DefaultPaths()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "Failed to get client paths: " + err.Error()})
		return
	}

	writeJSON(w, map[string]string{
		"claudeDesktop": p.ClaudeDesktop,
		"cursorGlobal":  p.CursorGlobal,
		"store":         p.Store,
	})
}

func (s *Server) handleClientsAdopt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Get client paths
	p, err := clients.DefaultPaths()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "Failed to get client paths"})
		return
	}

	// Detect and adopt existing MCPs
	adopted, err := s.reg.DetectAndAdoptMCPs(p)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"adopted": 0,
		})
		return
	}

	// Save the updated registry
	registryPath := os.Getenv("MCP_REGISTRY_PATH")
	if registryPath == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE") // Windows fallback
		}
		registryPath = filepath.Join(homeDir, ".mcp", "registry.json")
	}

	if err := s.reg.Save(registryPath); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{"error": "Failed to save registry: " + err.Error()})
		return
	}

	// Update supervisor with new registry if available
	if s.sup != nil {
		s.sup.UpdateRegistry(s.reg)
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"adopted": len(adopted),
		"servers": adopted,
	})
}

// buildClientConfig produces a minimal MCP config projection from the registry.
func buildClientConfig(reg *registry.Registry) map[string]any {
	out := map[string]any{"servers": []map[string]any{}}
	if reg == nil {
		return out
	}
	arr := []map[string]any{}
	for _, s := range reg.Servers {
		arr = append(arr, map[string]any{
			"slug":      s.Slug,
			"name":      s.Name,
			"transport": s.Entry.Transport,
			"command":   s.Entry.Command,
			"args":      s.Entry.Args,
			"env":       s.Entry.Env,
		})
	}
	out["servers"] = arr
	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Client-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// deriveHTTPURL constructs HTTP URL from command args and environment
func deriveHTTPURL(args []string, env map[string]string) string {
	if env != nil {
		if u, ok := env["HEALTH_HTTP_URL"]; ok && u != "" {
			return u
		}
	}

	var port string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--port=") {
			port = strings.TrimPrefix(a, "--port=")
			break
		}
		if a == "-p" && i+1 < len(args) {
			port = args[i+1]
			break
		}
	}

	if port != "" {
		return "http://127.0.0.1:" + port
	}

	return ""
}

// Admin Panel Handlers

func (s *Server) handleAdminListServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Build list of installed servers from registry
	servers := []map[string]interface{}{}

	for _, server := range s.reg.Servers {
		// Skip external servers for installed servers list
		if server.IsExternal() {
			continue
		}

		// Get server status from supervisor if available
		status := "stopped"
		if s.sup != nil {
			summary := s.sup.Summary()
			for _, proc := range summary {
				if proc["slug"] == server.Slug {
					status = proc["status"].(string)
					break
				}
			}
		}

		serverInfo := map[string]interface{}{
			"slug":   server.Slug,
			"name":   server.Name,
			"status": status,
			"path":   server.Entry.Command, // Use command as path since InstalledPath doesn't exist
		}

		// Add source info
		if server.Source.Type != "" {
			serverInfo["installType"] = server.Source.Type
		}
		if server.Source.URI != "" {
			serverInfo["installUri"] = server.Source.URI
		}

		servers = append(servers, serverInfo)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func (s *Server) handleAdminServerActions(w http.ResponseWriter, r *http.Request) {
	// Extract slug from path: /v1/admin/servers/{slug}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	slug := parts[4]

	switch r.Method {
	case http.MethodDelete:
		s.handleAdminDeleteServer(w, r, slug)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminDeleteServer(w http.ResponseWriter, r *http.Request, slug string) {
	// Stop the server first if running
	if err := s.sup.Stop(slug, 10*time.Second); err != nil {
		log.Printf("Warning: failed to stop server %s before deletion: %v", slug, err)
	}

	// Get servers directory
	serversDir, err := paths.ServersDir()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get servers directory: %v", err), http.StatusInternalServerError)
		return
	}

	serverPath := filepath.Join(serversDir, slug)

	// Check if server exists
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		http.Error(w, "server not found", http.StatusNotFound)
		return
	}

	// Remove the server directory
	if err := os.RemoveAll(serverPath); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete server: %v", err), http.StatusInternalServerError)
		return
	}

	// Remove from registry
	for i, server := range s.reg.Servers {
		if server.Slug == slug {
			s.reg.Servers = append(s.reg.Servers[:i], s.reg.Servers[i+1:]...)
			break
		}
	}

	// Save registry
	if err := registry.SaveDefault(s.reg); err != nil {
		log.Printf("Warning: failed to save registry after server deletion: %v", err)
	}

	// Remove from health monitoring
	s.healthMonitor.RemoveProcess(slug)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminMarketplace(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAdminListMarketplace(w, r)
	case http.MethodPost:
		s.handleAdminAddMarketplaceItem(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminListMarketplace(w http.ResponseWriter, r *http.Request) {
	// Return ALL catalog items with caution labels for incomplete ones
	marketplaceItems := []map[string]interface{}{
		// Reasoning Category
		{
			"slug":          "sequential-thinking",
			"name":          "Sequential Thinking",
			"description":   "Step-by-step structured reasoning.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://www.npmjs.com/package/@modelcontextprotocol/server-sequential-thinking",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "@modelcontextprotocol/server-sequential-thinking"},
			"configExample": "{ \"mcpServers\": { \"sequential-thinking\": { \"command\": \"npx\", \"args\": [\"-y\",\"@modelcontextprotocol/server-sequential-thinking\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "crash",
			"name":          "CRASH (Cascaded Reasoning)",
			"description":   "Confidence tracking, branching, adaptive steps.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "confidence", "branching"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/crash-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "crash-mcp"},
			"configExample": "{ \"mcpServers\": { \"crash\": { \"command\": \"npx\", \"args\": [\"-y\",\"crash-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "reasoner",
			"name":          "MCP Reasoner (Beam/MCTS)",
			"description":   "Beam search & MCTS reasoning.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "beam-search", "mcts"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://glama.ai/mcp/Reasoner",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "mcp-reasoner"},
			"configExample": "{ \"mcpServers\": { \"reasoner\": { \"command\": \"npx\", \"args\": [\"-y\",\"mcp-reasoner\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "mcts",
			"name":          "MCTS MCP",
			"description":   "Monte Carlo Tree Search.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "mcts", "search"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/search?q=MCTS+MCP+server",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "mcts-mcp"},
			"configExample": "{ \"mcpServers\": { \"mcts\": { \"command\": \"npx\", \"args\": [\"-y\",\"mcts-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "got",
			"name":          "Graph-of-Thoughts",
			"description":   "Graph reasoning (Neo4j optional).",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "graph", "neo4j"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/saptadey/got-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "got-mcp"},
			"configExample": "{ \"mcpServers\": { \"got\": { \"command\": \"npx\", \"args\": [\"-y\",\"got-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "dre",
			"name":          "Deliberate Reasoning Engine (DRE)",
			"description":   "DAG / thought-graph reasoning.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "dag", "thought-graph"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://glama.ai/mcp/DRE",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "dre-mcp"},
			"configExample": "{ \"mcpServers\": { \"dre\": { \"command\": \"npx\", \"args\": [\"-y\",\"dre-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "branch-thinking",
			"name":          "Branch Thinking",
			"description":   "Manage multiple reasoning branches.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "branches", "planning"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://glama.ai/mcp/BranchThinking",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "branch-thinking-mcp"},
			"configExample": "{ \"mcpServers\": { \"branch-thinking\": { \"command\": \"npx\", \"args\": [\"-y\",\"branch-thinking-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "cot",
			"name":          "Chain-of-Thought (beverm2391)",
			"description":   "Exposes CoT tokens.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "cot", "tokens"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/beverm2391/cot-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "cot-mcp"},
			"configExample": "{ \"mcpServers\": { \"cot\": { \"command\": \"npx\", \"args\": [\"-y\",\"cot-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "cot-task",
			"name":          "CoT Task Manager",
			"description":   "Task decomposition with CoT.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "cot", "tasks"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/liorfranko/cot-task-manager",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "cot-task-mcp"},
			"configExample": "{ \"mcpServers\": { \"cot-task\": { \"command\": \"npx\", \"args\": [\"-y\",\"cot-task-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "planner",
			"name":          "Software Planner",
			"description":   "Software project planning.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "planning", "software"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/planner-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "planner-mcp"},
			"configExample": "{ \"mcpServers\": { \"planner\": { \"command\": \"npx\", \"args\": [\"-y\",\"planner-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "memory",
			"name":          "Memory MCP",
			"description":   "Persistent knowledge graph memory.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "memory", "knowledge-graph"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/memory-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "memory-mcp"},
			"configExample": "{ \"mcpServers\": { \"memory\": { \"command\": \"npx\", \"args\": [\"-y\",\"memory-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "mindmap",
			"name":          "Mindmap MCP",
			"description":   "Convert ideas into mindmaps.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "mindmap", "visualization"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/mindmap-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "mindmap-mcp"},
			"configExample": "{ \"mcpServers\": { \"mindmap\": { \"command\": \"npx\", \"args\": [\"-y\",\"mindmap-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "context-crystallizer",
			"name":          "Context Crystallizer",
			"description":   "Distill docs/repos into structured knowledge.",
			"category":      "Reasoning",
			"tags":          []string{"reasoning", "context", "knowledge"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/context-crystallizer-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "context-crystallizer-mcp"},
			"configExample": "{ \"mcpServers\": { \"context-crystallizer\": { \"command\": \"npx\", \"args\": [\"-y\",\"context-crystallizer-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// UI / Frontend
		{
			"slug":          "shadcn-ui",
			"name":          "Shadcn UI MCP",
			"description":   "Search/install shadcn/ui components.",
			"category":      "UI / Frontend",
			"tags":          []string{"ui", "frontend", "components"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/Jpisnice/shadcn-ui-mcp-server",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "shadcn-ui-mcp-server"},
			"configExample": "{ \"mcpServers\": { \"shadcn-ui\": { \"command\": \"npx\", \"args\": [\"-y\",\"shadcn-ui-mcp-server\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "assistant-ui-docs",
			"name":          "assistant-ui Docs MCP",
			"description":   "Assistant-ui docs/examples in IDE.",
			"category":      "UI / Frontend Docs",
			"tags":          []string{"ui", "frontend", "docs", "ide"},
			"version":       "1.0.0",
			"author":        "Assistant UI Team",
			"repository":    "https://github.com/assistant-ui/mcp-docs",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "@assistant-ui/mcp-docs-server"},
			"configExample": "{ \"mcpServers\": { \"assistant-ui-docs\": { \"command\": \"npx\", \"args\": [\"-y\",\"@assistant-ui/mcp-docs-server\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Hosting / Infra
		{
			"slug":          "render",
			"name":          "Render MCP",
			"description":   "Manage Render services/deploys.",
			"category":      "Hosting / Infra",
			"tags":          []string{"hosting", "infrastructure", "render"},
			"version":       "1.0.0",
			"author":        "Render",
			"repository":    "https://github.com/render-oss/render-mcp-server",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "render-mcp-server"},
			"remote":        map[string]interface{}{"apiEndpoint": "https://mcp.render.com/mcp", "provider": "Render", "authType": "api_key"},
			"configExample": "{ \"mcpServers\": { \"render\": { \"url\": \"https://mcp.render.com/mcp\", \"headers\": { \"Authorization\": \"Bearer <RENDER_API_KEY>\" } } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "flyio",
			"name":          "Fly.io MCP",
			"description":   "Manage Fly.io apps via flyctl.",
			"category":      "Hosting / Infra",
			"tags":          []string{"hosting", "infrastructure", "flyio"},
			"version":       "1.0.0",
			"author":        "Fly.io",
			"repository":    "https://github.com/superfly/flymcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "flymcp"},
			"configExample": "{ \"mcpServers\": { \"flyio\": { \"command\": \"fly\", \"args\": [\"mcp\",\"server\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Cloud
		{
			"slug":          "aws-ccapi",
			"name":          "AWS CCAPI MCP",
			"description":   "Natural language AWS resource management.",
			"category":      "Cloud",
			"tags":          []string{"aws", "cloud", "infrastructure"},
			"version":       "1.0.0",
			"author":        "AWS Labs",
			"repository":    "https://github.com/awslabs/mcp",
			"license":       "Apache-2.0",
			"install":       map[string]string{"type": "npm", "uri": "@awslabs/ccapi-mcp-server"},
			"configExample": "{ \"mcpServers\": { \"aws-ccapi\": { \"command\": \"npx\", \"args\": [\"-y\",\"@awslabs/ccapi-mcp-server\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "aws-serverless",
			"name":          "AWS Serverless MCP",
			"description":   "Lambda/serverless guidance.",
			"category":      "Cloud",
			"tags":          []string{"aws", "cloud", "serverless"},
			"version":       "1.0.0",
			"author":        "AWS Labs",
			"repository":    "https://github.com/awslabs/mcp-serverless",
			"license":       "Apache-2.0",
			"install":       map[string]string{"type": "npm", "uri": "aws-serverless-mcp"},
			"configExample": "{ \"mcpServers\": { \"aws-serverless\": { \"command\": \"npx\", \"args\": [\"-y\",\"aws-serverless-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "gcp",
			"name":          "GCP MCP",
			"description":   "Google Cloud Platform.",
			"category":      "Cloud",
			"tags":          []string{"gcp", "cloud", "google"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/devinschumacher/gcp-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "gcp-mcp"},
			"configExample": "{ \"mcpServers\": { \"gcp\": { \"command\": \"npx\", \"args\": [\"-y\",\"gcp-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "azure",
			"name":          "Azure MCP",
			"description":   "Manage Azure/DevOps.",
			"category":      "Cloud",
			"tags":          []string{"azure", "cloud", "microsoft"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/devinschumacher/azure-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "azure-mcp"},
			"configExample": "{ \"mcpServers\": { \"azure\": { \"command\": \"npx\", \"args\": [\"-y\",\"azure-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "supabase",
			"name":          "Supabase MCP",
			"description":   "Manage Supabase DB/projects.",
			"category":      "Cloud / DB",
			"tags":          []string{"supabase", "database", "cloud"},
			"version":       "1.0.0",
			"author":        "Supabase Community",
			"repository":    "https://github.com/supabase-community/supabase-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "supabase-mcp"},
			"configExample": "{ \"mcpServers\": { \"supabase\": { \"command\": \"npx\", \"args\": [\"-y\",\"supabase-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Dev
		{
			"slug":          "github",
			"name":          "GitHub MCP",
			"description":   "Manage issues, PRs, repos.",
			"category":      "Dev",
			"tags":          []string{"github", "git", "api"},
			"version":       "1.0.0",
			"author":        "GitHub",
			"repository":    "https://github.com/github/github-mcp-server",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "github-mcp"},
			"remote":        map[string]interface{}{"apiEndpoint": "https://api.githubcopilot.com/mcp/", "provider": "GitHub", "authType": "oauth2"},
			"configExample": "{ \"servers\": { \"github\": { \"type\": \"http\", \"url\": \"https://api.githubcopilot.com/mcp/\" } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "figma",
			"name":          "Figma MCP",
			"description":   "Extract assets, metadata.",
			"category":      "Design",
			"tags":          []string{"design", "figma", "assets"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/figma-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "figma-mcp"},
			"configExample": "{ \"mcpServers\": { \"figma\": { \"command\": \"npx\", \"args\": [\"-y\",\"figma-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Comms
		{
			"slug":          "slack",
			"name":          "Slack MCP",
			"description":   "Read, post, and search Slack messages and channels.",
			"category":      "Comms",
			"tags":          []string{"slack", "communication", "messaging"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/augmentcode/slack-mcp-server",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "slack-mcp-server"},
			"configExample": "{ \"mcpServers\": { \"slack\": { \"command\": \"npx\", \"args\": [\"-y\",\"slack-mcp-server\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Email
		{
			"slug":          "gmail",
			"name":          "Gmail MCP",
			"description":   "Read and write Gmail messages.",
			"category":      "Email",
			"tags":          []string{"gmail", "email", "google"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/GongRzhe/Gmail-MCP-Server",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "gmail-mcp-server"},
			"configExample": "{ \"mcpServers\": { \"gmail\": { \"command\": \"npx\", \"args\": [\"-y\",\"gmail-mcp-server\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Docs
		{
			"slug":            "gdocs",
			"name":            "Google Docs MCP",
			"description":     "Query/edit Docs.",
			"category":        "Docs",
			"tags":            []string{"docs", "google", "collaboration"},
			"version":         "1.0.0",
			"author":          "MCP Team",
			"repository":      "https://github.com/modelcontextprotocol/google-docs-mcp",
			"license":         "MIT",
			"install":         map[string]string{"type": "git", "uri": "https://github.com/modelcontextprotocol/google-docs-mcp"},
			"configExample":   "{ \"mcpServers\": { \"gdocs\": { \"command\": \"node\", \"args\": [\"server.js\"], \"env\":{ \"GOOGLE_APPLICATION_CREDENTIALS\":\"./credentials.json\" } } } }",
			"needsAttention":  true,
			"attentionReason": "Requires Google credentials setup",
			"createdAt":       "2024-01-01T00:00:00Z",
			"updatedAt":       "2024-01-01T00:00:00Z",
		},
		// Calendar
		{
			"slug":            "gcal",
			"name":            "Google Calendar MCP",
			"description":     "Manage events.",
			"category":        "Calendar",
			"tags":            []string{"calendar", "google", "events"},
			"version":         "1.0.0",
			"author":          "MCP Team",
			"repository":      "https://github.com/modelcontextprotocol/google-calendar-mcp",
			"license":         "MIT",
			"install":         map[string]string{"type": "git", "uri": "https://github.com/modelcontextprotocol/google-calendar-mcp"},
			"configExample":   "{ \"mcpServers\": { \"gcal\": { \"command\": \"node\", \"args\": [\"index.js\"], \"env\":{ \"GOOGLE_CLIENT_ID\":\"...\",\"GOOGLE_CLIENT_SECRET\":\"...\" } } } }",
			"needsAttention":  true,
			"attentionReason": "Requires Google OAuth setup",
			"createdAt":       "2024-01-01T00:00:00Z",
			"updatedAt":       "2024-01-01T00:00:00Z",
		},
		// Notes
		{
			"slug":          "notion",
			"name":          "Notion MCP",
			"description":   "Manage Notion databases and pages.",
			"category":      "Notes",
			"tags":          []string{"notion", "database", "notes"},
			"version":       "1.0.0",
			"author":        "Notion",
			"repository":    "https://github.com/makenotion/notion-mcp-server",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "@notionhq/notion-mcp-server"},
			"remote":        map[string]interface{}{"apiEndpoint": "https://mcp.notion.com/mcp", "provider": "Notion", "authType": "oauth2"},
			"configExample": "{ \"mcpServers\": { \"Notion\": { \"url\": \"https://mcp.notion.com/mcp\" } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Social
		{
			"slug":            "youtube",
			"name":            "YouTube MCP",
			"description":     "Query/fetch YouTube metadata.",
			"category":        "Social",
			"tags":            []string{"youtube", "video", "metadata"},
			"version":         "1.0.0",
			"author":          "MCP Team",
			"repository":      "https://github.com/modelcontextprotocol/youtube-mcp",
			"license":         "MIT",
			"install":         map[string]string{"type": "git", "uri": "https://github.com/modelcontextprotocol/youtube-mcp"},
			"configExample":   "{ \"mcpServers\": { \"youtube\": { \"command\": \"node\", \"args\": [\"server.js\"] } } }",
			"needsAttention":  true,
			"attentionReason": "Requires YouTube API key setup",
			"createdAt":       "2024-01-01T00:00:00Z",
			"updatedAt":       "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "twitter",
			"name":          "TweetBinder MCP",
			"description":   "Twitter analytics.",
			"category":      "Social",
			"tags":          []string{"twitter", "analytics", "social"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/tweetbinder-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "tweetbinder-mcp"},
			"configExample": "{ \"mcpServers\": { \"twitter\": { \"command\": \"npx\", \"args\": [\"-y\",\"tweetbinder-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Messaging
		{
			"slug":          "telegram",
			"name":          "Telegram MCP",
			"description":   "Messaging integration.",
			"category":      "Messaging",
			"tags":          []string{"telegram", "messaging", "chat"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/telegram-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "telegram-mcp"},
			"configExample": "{ \"mcpServers\": { \"telegram\": { \"command\": \"npx\", \"args\": [\"-y\",\"telegram-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Social / Newsletters
		{
			"slug":          "substack",
			"name":          "Substack MCP",
			"description":   "Manage Substack posts.",
			"category":      "Social / Newsletters",
			"tags":          []string{"substack", "newsletter", "publishing"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/substack-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "substack-mcp"},
			"configExample": "{ \"mcpServers\": { \"substack\": { \"command\": \"npx\", \"args\": [\"-y\",\"substack-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Containers
		{
			"slug":          "docker",
			"name":          "Docker MCP",
			"description":   "Manage Docker containers and images.",
			"category":      "Containers",
			"tags":          []string{"docker", "containers", "devops"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/docker-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "docker-mcp"},
			"configExample": "{ \"mcpServers\": { \"docker\": { \"command\": \"npx\", \"args\": [\"-y\",\"docker-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Infra as Code
		{
			"slug":          "terraform",
			"name":          "Terraform MCP",
			"description":   "Terraform plan/apply via LLM.",
			"category":      "Infra as Code",
			"tags":          []string{"terraform", "infrastructure", "iac"},
			"version":       "1.0.0",
			"author":        "MCP Community",
			"repository":    "https://github.com/devinschumacher/tfmcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "tfmcp"},
			"configExample": "{ \"mcpServers\": { \"terraform\": { \"command\": \"npx\", \"args\": [\"-y\",\"tfmcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Monitoring
		{
			"slug":          "prometheus",
			"name":          "Prometheus MCP",
			"description":   "Query Prometheus metrics.",
			"category":      "Monitoring",
			"tags":          []string{"prometheus", "monitoring", "metrics"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/prometheus-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "prometheus-mcp"},
			"configExample": "{ \"mcpServers\": { \"prometheus\": { \"command\": \"npx\", \"args\": [\"-y\",\"prometheus-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		// Security / Recon
		{
			"slug":          "shodan",
			"name":          "Shodan MCP",
			"description":   "Shodan scans & OSINT.",
			"category":      "Security / Recon",
			"tags":          []string{"shodan", "security", "osint"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/shodan-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "shodan-mcp"},
			"configExample": "{ \"mcpServers\": { \"shodan\": { \"command\": \"npx\", \"args\": [\"-y\",\"shodan-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
		{
			"slug":          "nmap",
			"name":          "Nmap MCP",
			"description":   "Network scanning.",
			"category":      "Security / Net",
			"tags":          []string{"nmap", "security", "network"},
			"version":       "1.0.0",
			"author":        "MCP Team",
			"repository":    "https://github.com/modelcontextprotocol/nmap-mcp",
			"license":       "MIT",
			"install":       map[string]string{"type": "npm", "uri": "nmap-mcp"},
			"configExample": "{ \"mcpServers\": { \"nmap\": { \"command\": \"npx\", \"args\": [\"-y\",\"nmap-mcp\"] } } }",
			"createdAt":     "2024-01-01T00:00:00Z",
			"updatedAt":     "2024-01-01T00:00:00Z",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(marketplaceItems)
}

func (s *Server) handleAdminAddMarketplaceItem(w http.ResponseWriter, r *http.Request) {
	var item map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Basic validation
	if item["slug"] == nil || item["name"] == nil || item["category"] == nil {
		http.Error(w, "missing required fields: slug, name, category", http.StatusBadRequest)
		return
	}

	// Add timestamps
	now := time.Now().Format(time.RFC3339)
	item["createdAt"] = now
	item["updatedAt"] = now

	// In a real implementation, this would save to a database
	// For now, we'll just return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (s *Server) handleAdminMarketplaceActions(w http.ResponseWriter, r *http.Request) {
	// Extract slug from path: /v1/admin/marketplace/{slug}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	slug := parts[4]

	switch r.Method {
	case http.MethodPut:
		// Check if this is a resolve-attention request
		if strings.HasSuffix(r.URL.Path, "/resolve-attention") {
			s.handleResolveAttentionItem(w, r, slug)
		} else {
			s.handleAdminUpdateMarketplaceItem(w, r, slug)
		}
	case http.MethodDelete:
		s.handleAdminDeleteMarketplaceItem(w, r, slug)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminUpdateMarketplaceItem(w http.ResponseWriter, r *http.Request, slug string) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Add update timestamp
	updates["updatedAt"] = time.Now().Format(time.RFC3339)

	// In a real implementation, this would update in database
	// For now, we'll just return the updates
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"slug":    slug,
		"updated": true,
		"updates": updates,
	})
}

func (s *Server) handleAdminDeleteMarketplaceItem(w http.ResponseWriter, r *http.Request, slug string) {
	// In a real implementation, this would delete from database
	// For now, we'll just return success
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleResolveAttentionItem(w http.ResponseWriter, r *http.Request, slug string) {
	// In a real implementation, this would update the database to remove attention flags
	// For now, we'll just return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"slug":      slug,
		"resolved":  true,
		"message":   "Attention flag removed successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
