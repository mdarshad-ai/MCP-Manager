package supervisor

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "os/signal"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"
    "syscall"
    "time"

    "mcp/manager/internal/health"
    "mcp/manager/internal/paths"
    "mcp/manager/internal/registry"
)

// ProcessState represents the possible states of a managed process
type ProcessState int

const (
    ProcessStopped ProcessState = iota
    ProcessStarting
    ProcessRunning
    ProcessStopping
    ProcessFailed
    ProcessRestarting
)

func (s ProcessState) String() string {
    switch s {
    case ProcessStopped:
        return "stopped"
    case ProcessStarting:
        return "starting"
    case ProcessRunning:
        return "running"
    case ProcessStopping:
        return "stopping"
    case ProcessFailed:
        return "failed"
    case ProcessRestarting:
        return "restarting"
    default:
        return "unknown"
    }
}

// RestartPolicy defines how processes should be restarted
type RestartPolicy struct {
    Policy      string        // "always", "on-failure", "never"
    MaxRestarts int           // Maximum number of restarts (-1 for unlimited)
    BackoffMin  time.Duration // Minimum backoff time
    BackoffMax  time.Duration // Maximum backoff time
    BackoffMult float64       // Backoff multiplier
}

// DefaultRestartPolicy returns a sensible default restart policy
func DefaultRestartPolicy() RestartPolicy {
    return RestartPolicy{
        Policy:      "on-failure",
        MaxRestarts: -1, // unlimited
        BackoffMin:  1 * time.Second,
        BackoffMax:  30 * time.Second,
        BackoffMult: 2.0,
    }
}

type ProcState struct {
    mu             sync.RWMutex
    Slug           string
    Name           string
    Cmd            *exec.Cmd
    Process        *os.Process
    PID            int
    State          ProcessState
    Status         health.Status
    Restarts       int
    StartedAt      time.Time
    StoppedAt      time.Time
    LastPingMs     int
    MissedPings    int
    Uptime         time.Duration
    Stopping       int32 // atomic flag
    CPUPercent     float64
    RSSBytes       int64
    LogPath        string
    LogFile        *os.File
    LastLogAt      time.Time
    Transport      string
    HTTPURL        string
    RestartsAt     []time.Time
    HandshakeReady bool
    RestartPolicy  RestartPolicy
    
    // Control channels
    stopCh      chan struct{}
    stoppedCh   chan struct{}
    healthStopCh chan struct{}
    metricsStopCh chan struct{}
    
    // Context for process lifetime
    ctx    context.Context
    cancel context.CancelFunc
}

type Supervisor struct {
    mu         sync.RWMutex
    reg        *registry.Registry
    procs      map[string]*ProcState
    perCap     int64
    globCap    int64
    
    // Global control
    ctx        context.Context
    cancel     context.CancelFunc
    shutdownCh chan struct{}
    
    // Metrics and monitoring
    totalRestarts  int64
    totalStarts    int64
    totalStops     int64
    
    // Background tasks
    wg sync.WaitGroup
}

func New(reg *registry.Registry, perFileCap, globalCap int64) *Supervisor {
    ctx, cancel := context.WithCancel(context.Background())
    s := &Supervisor{
        reg:        reg,
        procs:      make(map[string]*ProcState),
        perCap:     perFileCap,
        globCap:    globalCap,
        ctx:        ctx,
        cancel:     cancel,
        shutdownCh: make(chan struct{}),
    }
    
    // Start the global supervisor goroutines
    s.wg.Add(1)
    go s.signalHandler()
    
    return s
}

// signalHandler handles system signals for graceful shutdown
func (s *Supervisor) signalHandler() {
    defer s.wg.Done()
    
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    select {
    case <-sigCh:
        s.Shutdown(30 * time.Second)
    case <-s.ctx.Done():
        return
    }
}

// Shutdown gracefully stops all processes and shuts down the supervisor
func (s *Supervisor) Shutdown(timeout time.Duration) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Signal shutdown
    select {
    case <-s.shutdownCh:
        return nil // already shutting down
    default:
        close(s.shutdownCh)
    }
    
    // Stop all processes
    var wg sync.WaitGroup
    for slug := range s.procs {
        wg.Add(1)
        go func(slug string) {
            defer wg.Done()
            _ = s.stopProcess(slug, timeout/time.Duration(len(s.procs)))
        }(slug)
    }
    
    // Wait for all processes to stop with timeout
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        // All processes stopped gracefully
    case <-time.After(timeout):
        // Force kill remaining processes
        for _, ps := range s.procs {
            if ps.Process != nil {
                _ = ps.Process.Kill()
            }
        }
    }
    
    // Cancel context and wait for background tasks
    s.cancel()
    s.wg.Wait()
    
    return nil
}

func (s *Supervisor) Summary() []map[string]any {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    out := make([]map[string]any, 0, len(s.reg.Servers))
    for _, sv := range s.reg.Servers {
        ps := s.procs[sv.Slug]
        status := health.Down
        state := ProcessStopped
        uptime := time.Duration(0)
        restarts := 0
        lastPing := 0
        pid := 0
        
        if ps != nil {
            ps.mu.RLock()
            status = ps.Status
            state = ps.State
            if !ps.StartedAt.IsZero() {
                if ps.State == ProcessRunning {
                    uptime = time.Since(ps.StartedAt)
                } else if !ps.StoppedAt.IsZero() {
                    uptime = ps.StoppedAt.Sub(ps.StartedAt)
                }
            }
            restarts = ps.Restarts
            lastPing = ps.LastPingMs
            pid = ps.PID
            cpuPercent := ps.CPUPercent
            rssBytes := ps.RSSBytes
            ps.mu.RUnlock()
            
            out = append(out, map[string]any{
                "name":       sv.Name,
                "slug":       sv.Slug,
                "status":     string(status),
                "state":      state.String(),
                "uptime":     int(uptime.Seconds()),
                "restarts":   restarts,
                "lastPingMs": lastPing,
                "pid":        pid,
                "cpu":        cpuPercent,
                "ramMB":      rssBytes / 1024 / 1024,
            })
        } else {
            out = append(out, map[string]any{
                "name":       sv.Name,
                "slug":       sv.Slug,
                "status":     string(status),
                "state":      state.String(),
                "uptime":     0,
                "restarts":   0,
                "lastPingMs": 0,
                "pid":        0,
                "cpu":        0.0,
                "ramMB":      int64(0),
            })
        }
    }
    return out
}

// GetProcessState returns the current state of a process
func (s *Supervisor) GetProcessState(slug string) (ProcessState, bool) {
    s.mu.RLock()
    ps := s.procs[slug]
    s.mu.RUnlock()
    
    if ps == nil {
        return ProcessStopped, false
    }
    
    ps.mu.RLock()
    state := ps.State
    ps.mu.RUnlock()
    
    return state, true
}

// GetProcessInfo returns detailed information about a process
func (s *Supervisor) GetProcessInfo(slug string) map[string]interface{} {
    s.mu.RLock()
    ps := s.procs[slug]
    s.mu.RUnlock()
    
    if ps == nil {
        return map[string]interface{}{
            "exists": false,
        }
    }
    
    ps.mu.RLock()
    defer ps.mu.RUnlock()
    
    info := map[string]interface{}{
        "exists":         true,
        "slug":           ps.Slug,
        "name":           ps.Name,
        "state":          ps.State.String(),
        "status":         string(ps.Status),
        "pid":            ps.PID,
        "restarts":       ps.Restarts,
        "startedAt":      ps.StartedAt,
        "stoppedAt":      ps.StoppedAt,
        "lastPingMs":     ps.LastPingMs,
        "missedPings":    ps.MissedPings,
        "cpuPercent":     ps.CPUPercent,
        "rssBytes":       ps.RSSBytes,
        "logPath":        ps.LogPath,
        "transport":      ps.Transport,
        "httpURL":        ps.HTTPURL,
        "handshakeReady": ps.HandshakeReady,
        "restartPolicy":  ps.RestartPolicy,
    }
    
    if !ps.StartedAt.IsZero() {
        if ps.State == ProcessRunning {
            info["uptime"] = time.Since(ps.StartedAt).Seconds()
        } else if !ps.StoppedAt.IsZero() {
            info["uptime"] = ps.StoppedAt.Sub(ps.StartedAt).Seconds()
        }
    }
    
    return info
}

// Stats returns global supervisor statistics
func (s *Supervisor) Stats() map[string]interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    running := 0
    stopped := 0
    failed := 0
    
    for _, ps := range s.procs {
        ps.mu.RLock()
        switch ps.State {
        case ProcessRunning:
            running++
        case ProcessStopped:
            stopped++
        case ProcessFailed:
            failed++
        }
        ps.mu.RUnlock()
    }
    
    return map[string]interface{}{
        "totalProcesses": len(s.procs),
        "running":        running,
        "stopped":        stopped,
        "failed":         failed,
        "totalStarts":    atomic.LoadInt64(&s.totalStarts),
        "totalStops":     atomic.LoadInt64(&s.totalStops),
        "totalRestarts":  atomic.LoadInt64(&s.totalRestarts),
    }
}

func (s *Supervisor) Start(slug string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Check if supervisor is shutting down
    select {
    case <-s.shutdownCh:
        return fmt.Errorf("supervisor is shutting down")
    default:
    }
    
    // Find server configuration
    var sv *registry.Server
    for i := range s.reg.Servers {
        if s.reg.Servers[i].Slug == slug {
            sv = &s.reg.Servers[i]
            break
        }
    }
    if sv == nil {
        return fmt.Errorf("unknown slug: %s", slug)
    }
    
    // Check if process already exists and is running
    if ps, exists := s.procs[slug]; exists {
        ps.mu.RLock()
        state := ps.State
        ps.mu.RUnlock()
        
        if state == ProcessRunning || state == ProcessStarting {
            return nil // Already running or starting
        }
        
        // If process exists but is stopped, we can restart it
        if state == ProcessStopped || state == ProcessFailed {
            return s.startProcess(ps, sv)
        }
    }
    
    // Create new process state
    return s.createAndStartProcess(slug, sv)
}

// createAndStartProcess creates a new process state and starts the process
func (s *Supervisor) createAndStartProcess(slug string, sv *registry.Server) error {
    logsDir, err := paths.LogsDir()
    if err != nil {
        return fmt.Errorf("failed to get logs directory: %w", err)
    }
    
    logPath := filepath.Join(logsDir, slug+".log")
    
    // Create or open log file
    logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
    if err != nil {
        return fmt.Errorf("failed to open log file: %w", err)
    }
    
    // Create process context
    ctx, cancel := context.WithCancel(s.ctx)
    
    // Determine restart policy from server configuration
    restartPolicy := DefaultRestartPolicy()
    if sv.Health.RestartPolicy != "" {
        restartPolicy.Policy = sv.Health.RestartPolicy
    }
    if sv.Health.MaxRestarts > 0 {
        restartPolicy.MaxRestarts = sv.Health.MaxRestarts
    }
    
    // Create process state
    ps := &ProcState{
        Slug:          slug,
        Name:          sv.Name,
        State:         ProcessStopped,
        Status:        health.Down,
        LogPath:       logPath,
        LogFile:       logFile,
        Transport:     sv.Entry.Transport,
        RestartPolicy: restartPolicy,
        ctx:           ctx,
        cancel:        cancel,
        stopCh:        make(chan struct{}),
        stoppedCh:     make(chan struct{}),
        healthStopCh:  make(chan struct{}),
        metricsStopCh: make(chan struct{}),
    }
    
    if ps.Transport == "http" {
        ps.HTTPURL = deriveHTTPURL(sv.Entry.Args, sv.Entry.Env)
    }
    
    s.procs[slug] = ps
    
    return s.startProcess(ps, sv)
}

// startProcess starts or restarts a process
func (s *Supervisor) startProcess(ps *ProcState, sv *registry.Server) error {
    ps.mu.Lock()
    defer ps.mu.Unlock()
    
    // Check if already starting or running
    if ps.State == ProcessStarting || ps.State == ProcessRunning {
        return nil
    }
    
    // Set state to starting
    ps.State = ProcessStarting
    ps.Status = health.Down
    atomic.AddInt64(&s.totalStarts, 1)
    
    // Start the process management goroutine
    s.wg.Add(1)
    go s.runProcess(ps, sv)
    
    return nil
}

// runProcess manages the lifecycle of a single process with proper state management
func (s *Supervisor) runProcess(ps *ProcState, sv *registry.Server) {
    defer s.wg.Done()
    defer func() {
        ps.mu.Lock()
        if ps.LogFile != nil {
            ps.LogFile.Close()
            ps.LogFile = nil
        }
        ps.mu.Unlock()
    }()
    
    for {
        select {
        case <-ps.ctx.Done():
            return
        case <-ps.stopCh:
            return
        default:
        }
        
        // Check restart policy
        ps.mu.Lock()
        if ps.RestartPolicy.MaxRestarts >= 0 && ps.Restarts >= ps.RestartPolicy.MaxRestarts {
            ps.State = ProcessFailed
            ps.Status = health.Down
            ps.mu.Unlock()
            return
        }
        ps.mu.Unlock()
        
        // Calculate backoff delay for restarts
        if ps.Restarts > 0 {
            delay := s.calculateBackoff(ps.Restarts, ps.RestartPolicy)
            ps.mu.Lock()
            ps.State = ProcessRestarting
            ps.mu.Unlock()
            
            select {
            case <-time.After(delay):
            case <-ps.ctx.Done():
                return
            case <-ps.stopCh:
                return
            }
            
            atomic.AddInt64(&s.totalRestarts, 1)
        }
        
        // Attempt to start the process
        if err := s.attemptProcessStart(ps, sv); err != nil {
            ps.mu.Lock()
            ps.State = ProcessFailed
            ps.Status = health.Down
            ps.Restarts++
            ps.RestartsAt = append(ps.RestartsAt, time.Now())
            ps.mu.Unlock()
            
            // Log the error
            if ps.LogFile != nil {
                fmt.Fprintf(ps.LogFile, "[%s] Failed to start process: %v\n", 
                    time.Now().Format(time.RFC3339), err)
            }
            
            continue // Retry with backoff
        }
        
        // Process started successfully, update state
        ps.mu.Lock()
        ps.State = ProcessRunning
        ps.StartedAt = time.Now()
        ps.PID = ps.Process.Pid
        ps.mu.Unlock()
        
        // Start monitoring goroutines
        s.startMonitoring(ps)
        
        // Wait for process to exit
        err := ps.Cmd.Wait()
        
        // Stop monitoring
        s.stopMonitoring(ps)
        
        // Update state based on exit
        ps.mu.Lock()
        ps.StoppedAt = time.Now()
        ps.Process = nil
        ps.PID = 0
        
        if atomic.LoadInt32(&ps.Stopping) == 1 {
            // Process was intentionally stopped
            ps.State = ProcessStopped
            ps.Status = health.Down
            ps.mu.Unlock()
            
            // Signal that we've stopped
            select {
            case <-ps.stoppedCh:
            default:
                close(ps.stoppedCh)
            }
            return
        }
        
        // Process exited unexpectedly
        ps.State = ProcessFailed
        ps.Status = health.Down
        ps.Restarts++
        ps.RestartsAt = append(ps.RestartsAt, time.Now())
        
        if ps.LogFile != nil {
            if err != nil {
                fmt.Fprintf(ps.LogFile, "[%s] Process exited with error: %v\n", 
                    time.Now().Format(time.RFC3339), err)
            } else {
                fmt.Fprintf(ps.LogFile, "[%s] Process exited normally\n", 
                    time.Now().Format(time.RFC3339))
            }
        }
        ps.mu.Unlock()
        
        // Check if we should restart based on policy
        if ps.RestartPolicy.Policy == "never" || 
           (ps.RestartPolicy.Policy == "on-failure" && err == nil) {
            ps.mu.Lock()
            ps.State = ProcessStopped
            ps.mu.Unlock()
            return
        }
        
        // Continue loop to restart
    }
}

// attemptProcessStart tries to start the actual process
func (s *Supervisor) attemptProcessStart(ps *ProcState, sv *registry.Server) error {
    ps.mu.Lock()
    defer ps.mu.Unlock()
    
    // Create command
    cmd := exec.CommandContext(ps.ctx, sv.Entry.Command, sv.Entry.Args...)
    
    // Set working directory if specified
    if srvDir, _ := paths.ServersDir(); srvDir != "" {
        cmd.Dir = filepath.Join(srvDir, ps.Slug)
    }
    
    // Set environment variables
    if len(sv.Entry.Env) > 0 {
        env := os.Environ()
        for key, value := range sv.Entry.Env {
            env = append(env, fmt.Sprintf("%s=%s", key, value))
        }
        cmd.Env = env
    }
    
    // Set up logging
    if ps.LogFile != nil {
        cmd.Stdout = ps.LogFile
        cmd.Stderr = ps.LogFile
        
        // Log the start attempt
        fmt.Fprintf(ps.LogFile, "[%s] Starting process: %s %v\n", 
            time.Now().Format(time.RFC3339), sv.Entry.Command, sv.Entry.Args)
    }
    
    // Start the process
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start command: %w", err)
    }
    
    ps.Cmd = cmd
    ps.Process = cmd.Process
    
    return nil
}

// calculateBackoff calculates the backoff delay for restarts
func (s *Supervisor) calculateBackoff(restarts int, policy RestartPolicy) time.Duration {
    if restarts <= 0 {
        return 0
    }
    
    // Calculate exponential backoff
    delay := policy.BackoffMin
    for i := 1; i < restarts && delay < policy.BackoffMax; i++ {
        delay = time.Duration(float64(delay) * policy.BackoffMult)
    }
    
    if delay > policy.BackoffMax {
        delay = policy.BackoffMax
    }
    
    return delay
}

// startMonitoring starts health and metrics monitoring for a process
func (s *Supervisor) startMonitoring(ps *ProcState) {
    // Start health monitoring
    s.wg.Add(1)
    go s.healthMonitor(ps)
    
    // Start metrics monitoring
    s.wg.Add(1)
    go s.metricsMonitor(ps)
}

// stopMonitoring stops monitoring for a process
func (s *Supervisor) stopMonitoring(ps *ProcState) {
    // Signal health monitor to stop
    select {
    case <-ps.healthStopCh:
    default:
        close(ps.healthStopCh)
    }
    
    // Signal metrics monitor to stop
    select {
    case <-ps.metricsStopCh:
    default:
        close(ps.metricsStopCh)
    }
}

// healthMonitor continuously monitors the health of a process
func (s *Supervisor) healthMonitor(ps *ProcState) {
    defer s.wg.Done()
    
    // Determine health check interval (default 20 seconds)
    interval := 20 * time.Second
    if sv := s.findServer(ps.Slug); sv != nil && sv.Health.IntervalSec > 0 {
        interval = time.Duration(sv.Health.IntervalSec) * time.Second
    }
    
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ps.healthStopCh:
            return
        case <-ps.ctx.Done():
            return
        case <-ticker.C:
            s.performHealthCheck(ps)
        }
    }
}

// performHealthCheck performs a single health check on a process
func (s *Supervisor) performHealthCheck(ps *ProcState) {
    ps.mu.Lock()
    if ps.State != ProcessRunning {
        ps.mu.Unlock()
        return
    }
    
    transport := ps.Transport
    httpURL := ps.HTTPURL
    logPath := ps.LogPath
    handshakeReady := ps.HandshakeReady
    ps.mu.Unlock()
    
    // Check for handshake readiness if not already ready
    if !handshakeReady {
        if ok, _ := logContains(logPath, []string{"notifications/initialized", "initialized"}); ok {
            ps.mu.Lock()
            ps.HandshakeReady = true
            handshakeReady = true
            ps.mu.Unlock()
        }
    }
    
    var pingError error
    var pingTime time.Duration
    
    if transport == "http" && httpURL != "" {
        // HTTP health check
        start := time.Now()
        client := &http.Client{Timeout: 3 * time.Second}
        resp, err := client.Get(httpURL)
        pingTime = time.Since(start)
        pingError = err
        
        if resp != nil {
            resp.Body.Close()
        }
    } else {
        // For stdio transport, use log activity as health indicator
        if info, err := os.Stat(logPath); err == nil {
            ps.mu.Lock()
            if info.ModTime().After(ps.LastLogAt) {
                ps.LastLogAt = info.ModTime()
                ps.MissedPings = 0
                ps.LastPingMs = 0
            } else {
                ps.MissedPings++
                if ps.LastLogAt.IsZero() {
                    ps.LastLogAt = time.Now()
                }
                ps.LastPingMs = int(time.Since(ps.LastLogAt).Milliseconds())
            }
            ps.mu.Unlock()
        }
        return // Skip the rest for stdio transport
    }
    
    // Update ping results
    ps.mu.Lock()
    if pingError == nil {
        ps.LastPingMs = int(pingTime.Milliseconds())
        ps.MissedPings = 0
    } else {
        ps.MissedPings++
    }
    
    // Evaluate health status
    in := health.ProbeInput{
        ProcessRunning:  ps.State == ProcessRunning,
        MissedPings:     ps.MissedPings,
        LastPingMs:      ps.LastPingMs,
        RestartsLast10m: len(ps.RestartsAt), // Simplified for now
    }
    
    status := health.Evaluate(in)
    
    // Gate ready status until handshake is complete
    if !ps.HandshakeReady && status == health.Ready {
        status = health.Degraded
    }
    
    ps.Status = status
    ps.mu.Unlock()
}

// metricsMonitor continuously collects metrics for a process
func (s *Supervisor) metricsMonitor(ps *ProcState) {
    defer s.wg.Done()
    
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ps.metricsStopCh:
            return
        case <-ps.ctx.Done():
            return
        case <-ticker.C:
            s.sampleProcessStats(ps)
        }
    }
}

// findServer finds a server configuration by slug
func (s *Supervisor) findServer(slug string) *registry.Server {
    for i := range s.reg.Servers {
        if s.reg.Servers[i].Slug == slug {
            return &s.reg.Servers[i]
        }
    }
    return nil
}

// sampleProcessStats updates CPUPercent and RSSBytes using `ps` if available.
func (s *Supervisor) sampleProcessStats(ps *ProcState) {
    ps.mu.RLock()
    process := ps.Process
    ps.mu.RUnlock()
    
    if process == nil {
        return
    }
    
    pid := process.Pid
    
    // ps -o pid=,pcpu=,rss= -p <pid>
    out, err := exec.Command("ps", "-o", "pid=,pcpu=,rss=", "-p", fmt.Sprint(pid)).Output()
    if err != nil {
        // Process might have exited
        return
    }
    
    // Expected: " 12345  1.2  54321\n"
    fields := strings.Fields(string(out))
    if len(fields) < 3 {
        return
    }
    
    cpu, _ := strconv.ParseFloat(fields[1], 64)
    rssKB, _ := strconv.ParseInt(fields[2], 10, 64)
    
    ps.mu.Lock()
    ps.CPUPercent = cpu
    ps.RSSBytes = rssKB * 1024
    ps.mu.Unlock()
}

func (s *Supervisor) restartsInLast(ps *ProcState, win time.Duration) int {
    ps.mu.Lock()
    defer ps.mu.Unlock()
    
    cutoff := time.Now().Add(-win)
    n := 0
    // also trim old entries to avoid growth
    kept := ps.RestartsAt[:0]
    for _, t := range ps.RestartsAt {
        if t.After(cutoff) {
            n++
            kept = append(kept, t)
        }
    }
    ps.RestartsAt = kept
    return n
}

// deriveHTTPURL tries to construct a local HTTP URL from args or env.
// Priority: env[HEALTH_HTTP_URL], then --port=NNNN or -p NNNN in args → http://127.0.0.1:NNNN
func deriveHTTPURL(args []string, env map[string]string) string {
    if env != nil {
        if u, ok := env["HEALTH_HTTP_URL"]; ok && u != "" { return u }
    }
    var port string
    for i := 0; i < len(args); i++ {
        a := args[i]
        if strings.HasPrefix(a, "--port=") {
            port = strings.TrimPrefix(a, "--port=")
            break
        }
        if a == "-p" && i+1 < len(args) { port = args[i+1]; break }
    }
    if port != "" { return "http://127.0.0.1:" + port }
    return ""
}

// logContains returns true if the end of the file contains any of the substrings.
func logContains(path string, subs []string) (bool, error) {
    f, err := os.Open(path)
    if err != nil { return false, err }
    defer f.Close()
    const maxRead = 64 * 1024
    fi, err := f.Stat(); if err != nil { return false, err }
    size := fi.Size()
    off := size - maxRead
    if off < 0 { off = 0 }
    buf := make([]byte, size-off)
    if _, err := f.ReadAt(buf, off); err != nil && err != io.EOF { return false, err }
    s := string(buf)
    for _, sub := range subs {
        if strings.Contains(s, sub) { return true, nil }
    }
    return false, nil
}

func (s *Supervisor) Stop(slug string, graceful time.Duration) error {
    return s.stopProcess(slug, graceful)
}

// stopProcess stops a process with the given timeout for graceful shutdown
func (s *Supervisor) stopProcess(slug string, graceful time.Duration) error {
    s.mu.RLock()
    ps := s.procs[slug]
    s.mu.RUnlock()
    
    if ps == nil {
        return nil // Process doesn't exist, consider it stopped
    }
    
    ps.mu.Lock()
    if ps.State == ProcessStopped || ps.State == ProcessStopping {
        ps.mu.Unlock()
        return nil // Already stopped or stopping
    }
    
    // Set stopping flag
    atomic.StoreInt32(&ps.Stopping, 1)
    ps.State = ProcessStopping
    ps.Status = health.Down
    
    process := ps.Process
    ps.mu.Unlock()
    
    atomic.AddInt64(&s.totalStops, 1)
    
    // Signal the process to stop
    select {
    case <-ps.stopCh:
        // Already signaled
    default:
        close(ps.stopCh)
    }
    
    if process == nil {
        // No actual process to stop
        ps.mu.Lock()
        ps.State = ProcessStopped
        ps.mu.Unlock()
        return nil
    }
    
    // Send SIGTERM for graceful shutdown
    if err := process.Signal(syscall.SIGTERM); err != nil {
        // Process might have already exited
        ps.mu.Lock()
        ps.State = ProcessStopped
        ps.mu.Unlock()
        return nil
    }
    
    // Wait for graceful shutdown with timeout
    select {
    case <-ps.stoppedCh:
        // Process stopped gracefully
        return nil
    case <-time.After(graceful):
        // Timeout exceeded, force kill
        if err := process.Kill(); err != nil {
            // Process might have already exited
        }
        
        // Wait a bit more for the kill to take effect
        select {
        case <-ps.stoppedCh:
        case <-time.After(5 * time.Second):
        }
        
        ps.mu.Lock()
        ps.State = ProcessStopped
        ps.mu.Unlock()
        
        return nil
    }
}

func (s *Supervisor) Restart(slug string) error {
    // Stop the process with a reasonable timeout
    if err := s.Stop(slug, 10*time.Second); err != nil {
        return fmt.Errorf("failed to stop process %s: %w", slug, err)
    }
    
    // Wait a moment for cleanup
    time.Sleep(100 * time.Millisecond)
    
    // Start the process again
    if err := s.Start(slug); err != nil {
        return fmt.Errorf("failed to start process %s after restart: %w", slug, err)
    }
    
    return nil
}

// UpdateRegistry updates the supervisor's registry reference
// This is needed when servers are added/removed after startup
func (s *Supervisor) UpdateRegistry(newReg *registry.Registry) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.reg = newReg
}
