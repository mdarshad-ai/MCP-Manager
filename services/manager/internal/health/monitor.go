package health

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"
    
    "mcp/manager/internal/registry"
)

// MCPInitializeRequest represents an MCP initialize request
type MCPInitializeRequest struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int         `json:"id"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params"`
}

// MCPInitializeParams represents the parameters for MCP initialize
type MCPInitializeParams struct {
    ProtocolVersion string                 `json:"protocolVersion"`
    Capabilities    map[string]interface{} `json:"capabilities"`
    ClientInfo      ClientInfo             `json:"clientInfo"`
}

// ClientInfo represents client information in MCP
type ClientInfo struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

// MCPResponse represents a generic MCP response
type MCPResponse struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int         `json:"id"`
    Result  interface{} `json:"result,omitempty"`
    Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// HealthMonitor provides advanced health monitoring capabilities
type HealthMonitor struct {
    mu                    sync.RWMutex
    processes             map[string]*ProcessHealth
    externalProcesses     map[string]*ExternalProcessHealth
    checkInterval         time.Duration
    externalCheckInterval time.Duration
    httpTimeout           time.Duration
    mcpTimeout            time.Duration
    retryAttempts         int
    retryBackoff          time.Duration
    
    // External health checking
    externalChecker *ExternalHealthChecker
    
    // Registry integration
    registryUpdater func(slug string, status registry.ExternalStatus)
    
    // Callbacks
    onHealthChange func(processName string, oldStatus, newStatus Status)
    onFailure      func(processName string, reason string)
    
    // Control
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

// ProcessHealth tracks health information for a single process
type ProcessHealth struct {
    Name           string
    Transport      string
    HTTPURL        string
    LogPath        string
    
    // Current state
    Status         Status
    LastCheck      time.Time
    LastSuccess    time.Time
    ConsecutiveFails int
    TotalChecks    int64
    TotalFailures  int64
    
    // Timing metrics
    MinResponseTime time.Duration
    MaxResponseTime time.Duration
    AvgResponseTime time.Duration
    
    // MCP specific
    MCPHandshakeComplete bool
    MCPProtocolVersion   string
    MCPCapabilities      map[string]interface{}
    
    // History
    CheckHistory   []HealthCheck
    maxHistorySize int
}

// HealthCheck represents a single health check result
type HealthCheck struct {
    Timestamp    time.Time
    Status       Status
    ResponseTime time.Duration
    Error        string
    CheckType    string // "http", "mcp", "log"
}

// ExternalProcessHealth tracks health information for an external server
type ExternalProcessHealth struct {
    Name         string
    Provider     string
    APIEndpoint  string
    AuthType     string
    
    // Current state
    Status         Status
    LastCheck      time.Time
    LastSuccess    time.Time
    ConsecutiveFails int
    TotalChecks    int64
    TotalFailures  int64
    
    // Timing metrics
    MinResponseTime time.Duration
    MaxResponseTime time.Duration
    AvgResponseTime time.Duration
    
    // External specific
    CredentialExpiry   *time.Time
    CredentialWarning  bool
    RateLimited        bool
    RateLimitReset     *time.Time
    LastErrorCode      int
    
    // History
    CheckHistory   []HealthCheck
    maxHistorySize int
    
    // Provider-specific metrics
    ServiceMetrics map[string]interface{}
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(checkInterval time.Duration) *HealthMonitor {
    ctx, cancel := context.WithCancel(context.Background())
    
    return &HealthMonitor{
        processes:             make(map[string]*ProcessHealth),
        externalProcesses:     make(map[string]*ExternalProcessHealth),
        checkInterval:         checkInterval,
        externalCheckInterval: 2 * time.Minute, // Less frequent for external servers
        httpTimeout:           5 * time.Second,
        mcpTimeout:            10 * time.Second,
        retryAttempts:         3,
        retryBackoff:          time.Second,
        externalChecker:       NewExternalHealthChecker(),
        ctx:                   ctx,
        cancel:                cancel,
    }
}

// SetCallbacks sets callback functions for health events
func (h *HealthMonitor) SetCallbacks(
    onHealthChange func(string, Status, Status),
    onFailure func(string, string),
) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    h.onHealthChange = onHealthChange
    h.onFailure = onFailure
}

// SetRegistryUpdater sets the callback function for updating external server status in registry
func (h *HealthMonitor) SetRegistryUpdater(updater func(string, registry.ExternalStatus)) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    h.registryUpdater = updater
}

// AddProcess adds a process to be monitored
func (h *HealthMonitor) AddProcess(name, transport, httpURL, logPath string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    h.processes[name] = &ProcessHealth{
        Name:           name,
        Transport:      transport,
        HTTPURL:        httpURL,
        LogPath:        logPath,
        Status:         Down,
        CheckHistory:   make([]HealthCheck, 0),
        maxHistorySize: 100,
        MinResponseTime: time.Hour, // Initialize to a large value
    }
}

// AddExternalProcess adds an external server to be monitored
func (h *HealthMonitor) AddExternalProcess(name, provider, apiEndpoint, authType string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    h.externalProcesses[name] = &ExternalProcessHealth{
        Name:            name,
        Provider:        provider,
        APIEndpoint:     apiEndpoint,
        AuthType:        authType,
        Status:          Down,
        CheckHistory:    make([]HealthCheck, 0),
        maxHistorySize:  100,
        MinResponseTime: time.Hour, // Initialize to a large value
        ServiceMetrics:  make(map[string]interface{}),
    }
}

// RemoveProcess removes a process from monitoring
func (h *HealthMonitor) RemoveProcess(name string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    delete(h.processes, name)
}

// RemoveExternalProcess removes an external server from monitoring
func (h *HealthMonitor) RemoveExternalProcess(name string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    delete(h.externalProcesses, name)
}

// Start begins health monitoring
func (h *HealthMonitor) Start() {
    h.wg.Add(2)
    go h.monitorLoop()
    go h.externalMonitorLoop()
}

// Stop stops health monitoring
func (h *HealthMonitor) Stop() {
    h.cancel()
    h.wg.Wait()
}

// GetProcessHealth returns the health information for a process
func (h *HealthMonitor) GetProcessHealth(name string) (*ProcessHealth, bool) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    ph, exists := h.processes[name]
    if !exists {
        return nil, false
    }
    
    // Return a copy to avoid race conditions
    copy := *ph
    return &copy, true
}

// GetExternalProcessHealth returns the health information for an external server
func (h *HealthMonitor) GetExternalProcessHealth(name string) (*ExternalProcessHealth, bool) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    ph, exists := h.externalProcesses[name]
    if !exists {
        return nil, false
    }
    
    // Return a copy to avoid race conditions
    copy := *ph
    return &copy, true
}

// GetAllHealth returns health information for all monitored processes
func (h *HealthMonitor) GetAllHealth() map[string]*ProcessHealth {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    result := make(map[string]*ProcessHealth)
    for name, ph := range h.processes {
        copy := *ph
        result[name] = &copy
    }
    
    return result
}

// GetAllExternalHealth returns health information for all external servers
func (h *HealthMonitor) GetAllExternalHealth() map[string]*ExternalProcessHealth {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    result := make(map[string]*ExternalProcessHealth)
    for name, ph := range h.externalProcesses {
        copy := *ph
        result[name] = &copy
    }
    
    return result
}

// monitorLoop is the main monitoring loop
func (h *HealthMonitor) monitorLoop() {
    defer h.wg.Done()
    
    ticker := time.NewTicker(h.checkInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-h.ctx.Done():
            return
        case <-ticker.C:
            h.performAllHealthChecks()
        }
    }
}

// performAllHealthChecks performs health checks on all monitored processes
func (h *HealthMonitor) performAllHealthChecks() {
    h.mu.RLock()
    processes := make([]*ProcessHealth, 0, len(h.processes))
    for _, ph := range h.processes {
        processes = append(processes, ph)
    }
    h.mu.RUnlock()
    
    // Perform checks concurrently
    var wg sync.WaitGroup
    for _, ph := range processes {
        wg.Add(1)
        go func(ph *ProcessHealth) {
            defer wg.Done()
            h.performHealthCheck(ph)
        }(ph)
    }
    
    wg.Wait()
}

// performHealthCheck performs a health check on a single process
func (h *HealthMonitor) performHealthCheck(ph *ProcessHealth) {
    checkStart := time.Now()
    
    var status Status
    var err error
    var responseTime time.Duration
    var checkType string
    
    // Determine check type based on transport
    switch ph.Transport {
    case "http":
        if ph.HTTPURL != "" {
            status, responseTime, err = h.performHTTPCheck(ph)
            checkType = "http"
        } else {
            status = Down
            err = fmt.Errorf("HTTP transport but no URL configured")
            checkType = "config"
        }
    case "stdio":
        // For stdio transport, check MCP handshake and log activity
        if !ph.MCPHandshakeComplete {
            status, err = h.checkMCPHandshake(ph)
            checkType = "mcp-handshake"
        } else {
            status, err = h.checkLogActivity(ph)
            checkType = "log"
        }
        responseTime = time.Since(checkStart)
    default:
        status = Down
        err = fmt.Errorf("unsupported transport: %s", ph.Transport)
        checkType = "unsupported"
    }
    
    // Update process health
    h.updateProcessHealth(ph, status, responseTime, err, checkType)
}

// performHTTPCheck performs an HTTP health check
func (h *HealthMonitor) performHTTPCheck(ph *ProcessHealth) (Status, time.Duration, error) {
    client := &http.Client{Timeout: h.httpTimeout}
    
    for attempt := 0; attempt < h.retryAttempts; attempt++ {
        if attempt > 0 {
            time.Sleep(h.retryBackoff)
        }
        
        start := time.Now()
        resp, err := client.Get(ph.HTTPURL)
        responseTime := time.Since(start)
        
        if err != nil {
            if attempt == h.retryAttempts-1 {
                return Down, responseTime, fmt.Errorf("HTTP check failed after %d attempts: %w", h.retryAttempts, err)
            }
            continue
        }
        
        resp.Body.Close()
        
        // Consider 2xx and 3xx as healthy
        if resp.StatusCode >= 200 && resp.StatusCode < 400 {
            return Ready, responseTime, nil
        }
        
        if attempt == h.retryAttempts-1 {
            return Degraded, responseTime, fmt.Errorf("HTTP check returned status %d", resp.StatusCode)
        }
    }
    
    return Down, 0, fmt.Errorf("HTTP check exhausted retries")
}

// checkMCPHandshake checks if MCP handshake is complete by looking for initialization messages in logs
func (h *HealthMonitor) checkMCPHandshake(ph *ProcessHealth) (Status, error) {
    if ph.LogPath == "" {
        return Down, fmt.Errorf("no log path configured for MCP handshake check")
    }
    
    // Look for MCP initialization messages in the log file
    initPatterns := []string{
        "notifications/initialized",
        "initialized",
        "mcp/initialize",
        "protocol initialized",
        "handshake complete",
    }
    
    found, err := h.logContainsAny(ph.LogPath, initPatterns)
    if err != nil {
        return Down, fmt.Errorf("failed to check MCP handshake in logs: %w", err)
    }
    
    if found {
        ph.MCPHandshakeComplete = true
        return Ready, nil
    }
    
    return Degraded, fmt.Errorf("MCP handshake not yet complete")
}

// checkLogActivity checks recent log activity for stdio transport
func (h *HealthMonitor) checkLogActivity(ph *ProcessHealth) (Status, error) {
    if ph.LogPath == "" {
        return Down, fmt.Errorf("no log path configured for log activity check")
    }
    
    info, err := os.Stat(ph.LogPath)
    if err != nil {
        return Down, fmt.Errorf("failed to stat log file: %w", err)
    }
    
    // Consider recent activity (within last 60 seconds) as healthy
    if time.Since(info.ModTime()) < 60*time.Second {
        return Ready, nil
    }
    
    // Check if there's any activity within last 5 minutes
    if time.Since(info.ModTime()) < 5*time.Minute {
        return Degraded, nil
    }
    
    return Down, fmt.Errorf("no recent log activity (last modified: %s)", info.ModTime().Format(time.RFC3339))
}

// logContainsAny checks if log file contains any of the specified patterns
func (h *HealthMonitor) logContainsAny(logPath string, patterns []string) (bool, error) {
    file, err := os.Open(logPath)
    if err != nil {
        return false, err
    }
    defer file.Close()
    
    // Read the last 64KB of the file for efficiency
    const maxRead = 64 * 1024
    info, err := file.Stat()
    if err != nil {
        return false, err
    }
    
    size := info.Size()
    offset := size - maxRead
    if offset < 0 {
        offset = 0
    }
    
    content := make([]byte, size-offset)
    _, err = file.ReadAt(content, offset)
    if err != nil && err != io.EOF {
        return false, err
    }
    
    contentStr := strings.ToLower(string(content))
    for _, pattern := range patterns {
        if strings.Contains(contentStr, strings.ToLower(pattern)) {
            return true, nil
        }
    }
    
    return false, nil
}

// updateProcessHealth updates the health status of a process
func (h *HealthMonitor) updateProcessHealth(ph *ProcessHealth, status Status, responseTime time.Duration, err error, checkType string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    oldStatus := ph.Status
    ph.LastCheck = time.Now()
    ph.TotalChecks++
    
    // Create health check record
    check := HealthCheck{
        Timestamp:    ph.LastCheck,
        Status:       status,
        ResponseTime: responseTime,
        CheckType:    checkType,
    }
    
    if err != nil {
        check.Error = err.Error()
        ph.TotalFailures++
        ph.ConsecutiveFails++
    } else {
        ph.LastSuccess = ph.LastCheck
        ph.ConsecutiveFails = 0
        
        // Update response time metrics
        if responseTime > 0 {
            if responseTime < ph.MinResponseTime {
                ph.MinResponseTime = responseTime
            }
            if responseTime > ph.MaxResponseTime {
                ph.MaxResponseTime = responseTime
            }
            
            // Simple moving average for now
            if ph.AvgResponseTime == 0 {
                ph.AvgResponseTime = responseTime
            } else {
                ph.AvgResponseTime = (ph.AvgResponseTime + responseTime) / 2
            }
        }
    }
    
    // Add to history
    ph.CheckHistory = append(ph.CheckHistory, check)
    if len(ph.CheckHistory) > ph.maxHistorySize {
        ph.CheckHistory = ph.CheckHistory[1:] // Remove oldest entry
    }
    
    ph.Status = status
    
    // Trigger callbacks if status changed
    if oldStatus != status {
        if h.onHealthChange != nil {
            go h.onHealthChange(ph.Name, oldStatus, status)
        }
    }
    
    // Trigger failure callback for consecutive failures
    if ph.ConsecutiveFails >= 3 && h.onFailure != nil {
        errorMsg := "unknown error"
        if err != nil {
            errorMsg = err.Error()
        }
        go h.onFailure(ph.Name, fmt.Sprintf("consecutive failures: %d, last error: %s", ph.ConsecutiveFails, errorMsg))
    }
}

// GetHealthSummary returns a summary of health status for all processes
func (h *HealthMonitor) GetHealthSummary() map[string]interface{} {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    totalProcesses := len(h.processes) + len(h.externalProcesses)
    summary := map[string]interface{}{
        "totalProcesses": totalProcesses,
        "healthy":        0,
        "degraded":       0,
        "down":           0,
        "processes":      make([]map[string]interface{}, 0, totalProcesses),
        "external": map[string]interface{}{
            "totalExternal": len(h.externalProcesses),
            "processes":     make([]map[string]interface{}, 0, len(h.externalProcesses)),
        },
    }
    
    // Count local processes
    for _, ph := range h.processes {
        switch ph.Status {
        case Ready:
            summary["healthy"] = summary["healthy"].(int) + 1
        case Degraded:
            summary["degraded"] = summary["degraded"].(int) + 1
        case Down:
            summary["down"] = summary["down"].(int) + 1
        }
        
        processInfo := map[string]interface{}{
            "name":              ph.Name,
            "status":            string(ph.Status),
            "transport":         ph.Transport,
            "lastCheck":         ph.LastCheck,
            "lastSuccess":       ph.LastSuccess,
            "consecutiveFails":  ph.ConsecutiveFails,
            "totalChecks":       ph.TotalChecks,
            "totalFailures":     ph.TotalFailures,
            "avgResponseTime":   ph.AvgResponseTime.Milliseconds(),
            "mcpHandshakeComplete": ph.MCPHandshakeComplete,
        }
        
        summary["processes"] = append(summary["processes"].([]map[string]interface{}), processInfo)
    }
    
    // Count external processes
    for _, ph := range h.externalProcesses {
        switch ph.Status {
        case Ready:
            summary["healthy"] = summary["healthy"].(int) + 1
        case Degraded:
            summary["degraded"] = summary["degraded"].(int) + 1
        case Down:
            summary["down"] = summary["down"].(int) + 1
        }
        
        processInfo := map[string]interface{}{
            "name":               ph.Name,
            "status":             string(ph.Status),
            "provider":           ph.Provider,
            "transport":          "external",
            "lastCheck":          ph.LastCheck,
            "lastSuccess":        ph.LastSuccess,
            "consecutiveFails":   ph.ConsecutiveFails,
            "totalChecks":        ph.TotalChecks,
            "totalFailures":      ph.TotalFailures,
            "avgResponseTime":    ph.AvgResponseTime.Milliseconds(),
            "credentialWarning":  ph.CredentialWarning,
            "rateLimited":        ph.RateLimited,
            "lastErrorCode":      ph.LastErrorCode,
        }
        
        summary["external"].(map[string]interface{})["processes"] = append(
            summary["external"].(map[string]interface{})["processes"].([]map[string]interface{}), 
            processInfo,
        )
    }
    
    return summary
}

// externalMonitorLoop is the monitoring loop for external servers
func (h *HealthMonitor) externalMonitorLoop() {
    defer h.wg.Done()
    
    ticker := time.NewTicker(h.externalCheckInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-h.ctx.Done():
            return
        case <-ticker.C:
            h.performAllExternalHealthChecks()
        }
    }
}

// performAllExternalHealthChecks performs health checks on all external servers
func (h *HealthMonitor) performAllExternalHealthChecks() {
    h.mu.RLock()
    processes := make([]*ExternalProcessHealth, 0, len(h.externalProcesses))
    for _, ph := range h.externalProcesses {
        processes = append(processes, ph)
    }
    h.mu.RUnlock()
    
    // Perform checks concurrently
    var wg sync.WaitGroup
    for _, ph := range processes {
        wg.Add(1)
        go func(ph *ExternalProcessHealth) {
            defer wg.Done()
            h.performExternalHealthCheck(ph)
        }(ph)
    }
    
    wg.Wait()
}

// performExternalHealthCheck performs a health check on a single external server
func (h *HealthMonitor) performExternalHealthCheck(ph *ExternalProcessHealth) {
    checkStart := time.Now()
    
    ctx, cancel := context.WithTimeout(h.ctx, h.httpTimeout)
    defer cancel()
    
    // Get API key from provider endpoint mapping or use configured endpoint
    endpoint := ph.APIEndpoint
    if endpoint == "" {
        if providerEndpoint, exists := GetProviderEndpoint(ph.Provider); exists {
            endpoint = providerEndpoint
        } else {
            h.updateExternalProcessHealth(ph, Down, 0, fmt.Errorf("no health endpoint configured for provider %s", ph.Provider), "config")
            return
        }
    }
    
    // Perform health check using the external checker
    // For now, we'll use empty API key - this should be retrieved from credential store
    health, err := h.externalChecker.CheckHealth(ctx, endpoint, "")
    
    var status Status
    var responseTime time.Duration = time.Since(checkStart)
    var checkErr error
    
    if err != nil {
        status = Down
        checkErr = err
    } else {
        // Convert external health status to our internal status
        switch health.Status {
        case "healthy":
            status = Ready
        case "warning":
            status = Degraded
        case "unhealthy", "error":
            status = Down
        default:
            status = Down
            checkErr = fmt.Errorf("unknown health status: %s", health.Status)
        }
        
        responseTime = time.Duration(health.ResponseTime) * time.Millisecond
        
        // Update external-specific metrics
        ph.LastErrorCode = health.StatusCode
        ph.RateLimited = health.StatusCode == 429
        if ph.RateLimited && health.StatusCode == 429 {
            // Set rate limit reset time (estimate 1 hour from now)
            resetTime := time.Now().Add(time.Hour)
            ph.RateLimitReset = &resetTime
        }
        
        if health.Error != "" {
            checkErr = fmt.Errorf(health.Error)
        }
    }
    
    // Update process health
    h.updateExternalProcessHealth(ph, status, responseTime, checkErr, "external")
}

// updateExternalProcessHealth updates the health status of an external server
func (h *HealthMonitor) updateExternalProcessHealth(ph *ExternalProcessHealth, status Status, responseTime time.Duration, err error, checkType string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    oldStatus := ph.Status
    ph.LastCheck = time.Now()
    ph.TotalChecks++
    
    // Create health check record
    check := HealthCheck{
        Timestamp:    ph.LastCheck,
        Status:       status,
        ResponseTime: responseTime,
        CheckType:    checkType,
    }
    
    if err != nil {
        check.Error = err.Error()
        ph.TotalFailures++
        ph.ConsecutiveFails++
    } else {
        ph.LastSuccess = ph.LastCheck
        ph.ConsecutiveFails = 0
        
        // Update response time metrics
        if responseTime > 0 {
            if responseTime < ph.MinResponseTime {
                ph.MinResponseTime = responseTime
            }
            if responseTime > ph.MaxResponseTime {
                ph.MaxResponseTime = responseTime
            }
            
            // Simple moving average for now
            if ph.AvgResponseTime == 0 {
                ph.AvgResponseTime = responseTime
            } else {
                ph.AvgResponseTime = (ph.AvgResponseTime + responseTime) / 2
            }
        }
    }
    
    // Add to history
    ph.CheckHistory = append(ph.CheckHistory, check)
    if len(ph.CheckHistory) > ph.maxHistorySize {
        ph.CheckHistory = ph.CheckHistory[1:] // Remove oldest entry
    }
    
    ph.Status = status
    
    // Check for credential expiry warnings
    if ph.CredentialExpiry != nil && time.Until(*ph.CredentialExpiry) < 7*24*time.Hour {
        ph.CredentialWarning = true
    }
    
    // Trigger callbacks if status changed
    if oldStatus != status {
        if h.onHealthChange != nil {
            go h.onHealthChange(ph.Name, oldStatus, status)
        }
    }
    
    // Trigger failure callback for consecutive failures
    if ph.ConsecutiveFails >= 3 && h.onFailure != nil {
        errorMsg := "unknown error"
        if err != nil {
            errorMsg = err.Error()
        }
        go h.onFailure(ph.Name, fmt.Sprintf("consecutive failures: %d, last error: %s", ph.ConsecutiveFails, errorMsg))
    }
    
    // Update registry with external server status
    if h.registryUpdater != nil {
        registryStatus := registry.ExternalStatus{
            State:        h.convertStatusToRegistryState(status),
            Message:      h.createStatusMessage(ph, err),
            LastChecked:  &ph.LastCheck,
            ResponseTime: func() *int64 { rt := responseTime.Milliseconds(); return &rt }(),
        }
        go h.registryUpdater(ph.Name, registryStatus)
    }
}

// convertStatusToRegistryState converts health Status to registry state
func (h *HealthMonitor) convertStatusToRegistryState(status Status) string {
    switch status {
    case Ready:
        return "active"
    case Degraded:
        return "connecting"
    case Down:
        return "error"
    default:
        return "inactive"
    }
}

// createStatusMessage creates a human-readable status message
func (h *HealthMonitor) createStatusMessage(ph *ExternalProcessHealth, err error) string {
    if err != nil {
        if ph.RateLimited {
            return fmt.Sprintf("Rate limited (last error: %s)", err.Error())
        }
        if ph.CredentialWarning {
            return fmt.Sprintf("Credential warning (last error: %s)", err.Error())
        }
        return err.Error()
    }
    
    if ph.RateLimited {
        resetMsg := ""
        if ph.RateLimitReset != nil {
            resetMsg = fmt.Sprintf(" (resets at %s)", ph.RateLimitReset.Format(time.RFC3339))
        }
        return fmt.Sprintf("Rate limited%s", resetMsg)
    }
    
    if ph.CredentialWarning {
        return "Credential expiring soon"
    }
    
    switch ph.Status {
    case Ready:
        return "Service is healthy"
    case Degraded:
        return "Service is responding but may have issues"
    case Down:
        return "Service is not responding"
    default:
        return "Status unknown"
    }
}