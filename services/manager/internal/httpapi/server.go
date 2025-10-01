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
	"sync"
	"time"

	"mcp/manager/internal/clients"
	"mcp/manager/internal/health"
	"mcp/manager/internal/install"
	"mcp/manager/internal/logs"
	"mcp/manager/internal/registry"
)

type Server struct {
	reg               *registry.Registry
	sup               Supervisor
	healthMonitor     HealthMonitor
	logStreamer       LogStreamer
	jobs              map[string]*job
	jobsMu            sync.Mutex
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
	return &Server{reg: reg, jobs: map[string]*job{}}
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
	mux.HandleFunc("/v1/install/validate", s.handleInstallValidate)
	mux.HandleFunc("/v1/install/perform", s.handleInstallPerform)
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

func (s *Server) handleInstallValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var in install.Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	res, err := install.Validate(r.Context(), in, install.ExecRunner{})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "result": res})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleInstallPerform(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var in install.PerformInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	res, err := install.Perform(r.Context(), in, install.ExecRunner{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, res)
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
		registryPath = filepath.Join(os.Getenv("HOME"), ".mcp", "registry.json")
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
