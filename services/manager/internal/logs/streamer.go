package logs

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "os"
    "strings"
    "sync"
    "time"
)

// LogEntry represents a single log entry
type LogEntry struct {
    Timestamp  time.Time `json:"timestamp"`
    Process    string    `json:"process"`
    Level      string    `json:"level,omitempty"`
    Message    string    `json:"message"`
    Line       int64     `json:"line"`
}

// StreamClient represents a client listening to log streams
type StreamClient struct {
    ID       string
    Process  string
    Ch       chan LogEntry
    Cancel   context.CancelFunc
    LastSeen int64 // Last line number seen
    ctx      context.Context
}

// LogStreamer provides real-time log streaming capabilities
type LogStreamer struct {
    mu           sync.RWMutex
    clients      map[string]*StreamClient
    watchers     map[string]*LogWatcher // process -> watcher
    logsDir      string
    
    ctx          context.Context
    cancel       context.CancelFunc
    wg           sync.WaitGroup
}

// LogWatcher watches a single log file for changes
type LogWatcher struct {
    process     string
    filePath    string
    file        *os.File
    scanner     *bufio.Scanner
    position    int64
    lineCount   int64
    lastCheck   time.Time
    
    clients     map[string]*StreamClient
    mu          sync.RWMutex
    
    ctx         context.Context
    cancel      context.CancelFunc
}

// NewLogStreamer creates a new log streamer
func NewLogStreamer(logsDir string) *LogStreamer {
    ctx, cancel := context.WithCancel(context.Background())
    
    return &LogStreamer{
        clients:  make(map[string]*StreamClient),
        watchers: make(map[string]*LogWatcher),
        logsDir:  logsDir,
        ctx:      ctx,
        cancel:   cancel,
    }
}

// Start begins the log streaming service
func (ls *LogStreamer) Start() {
    // Start cleanup goroutine for disconnected clients
    ls.wg.Add(1)
    go ls.cleanupLoop()
}

// Stop stops the log streaming service
func (ls *LogStreamer) Stop() {
    ls.cancel()
    
    // Stop all watchers
    ls.mu.Lock()
    for _, watcher := range ls.watchers {
        watcher.Stop()
    }
    ls.mu.Unlock()
    
    // Close all client channels
    ls.mu.Lock()
    for _, client := range ls.clients {
        client.Cancel()
        close(client.Ch)
    }
    ls.clients = make(map[string]*StreamClient)
    ls.mu.Unlock()
    
    ls.wg.Wait()
}

// StreamLogs creates a new streaming client for a process
func (ls *LogStreamer) StreamLogs(clientID, process string, fromLine int64) (*StreamClient, error) {
    ls.mu.Lock()
    defer ls.mu.Unlock()
    
    // Cancel existing client with same ID if exists
    if existing, exists := ls.clients[clientID]; exists {
        existing.Cancel()
        close(existing.Ch)
        delete(ls.clients, clientID)
    }
    
    // Create client context
    ctx, cancel := context.WithCancel(ls.ctx)
    
    // Create client
    client := &StreamClient{
        ID:       clientID,
        Process:  process,
        Ch:       make(chan LogEntry, 100), // Buffered channel
        Cancel:   cancel,
        LastSeen: fromLine,
        ctx:      ctx,
    }
    
    ls.clients[clientID] = client
    
    // Ensure watcher exists for this process
    watcher, exists := ls.watchers[process]
    if !exists {
        var err error
        watcher, err = ls.createWatcher(process)
        if err != nil {
            delete(ls.clients, clientID)
            cancel()
            close(client.Ch)
            return nil, fmt.Errorf("failed to create log watcher: %w", err)
        }
        ls.watchers[process] = watcher
    }
    
    // Register client with watcher
    watcher.AddClient(client)
    
    // Send historical logs if requested
    if fromLine >= 0 {
        go ls.sendHistoricalLogs(client, fromLine)
    }
    
    return client, nil
}

// StopStream stops streaming for a specific client
func (ls *LogStreamer) StopStream(clientID string) {
    ls.mu.Lock()
    defer ls.mu.Unlock()
    
    if client, exists := ls.clients[clientID]; exists {
        // Remove client from watcher
        if watcher, watcherExists := ls.watchers[client.Process]; watcherExists {
            watcher.RemoveClient(clientID)
        }
        
        client.Cancel()
        close(client.Ch)
        delete(ls.clients, clientID)
    }
}

// createWatcher creates a new log watcher for a process
func (ls *LogStreamer) createWatcher(process string) (*LogWatcher, error) {
    filePath := fmt.Sprintf("%s/%s.log", ls.logsDir, process)
    
    ctx, cancel := context.WithCancel(ls.ctx)
    
    watcher := &LogWatcher{
        process:  process,
        filePath: filePath,
        clients:  make(map[string]*StreamClient),
        ctx:      ctx,
        cancel:   cancel,
    }
    
    // Start the watcher
    if err := watcher.Start(); err != nil {
        cancel()
        return nil, err
    }
    
    return watcher, nil
}

// sendHistoricalLogs sends historical log entries to a client
func (ls *LogStreamer) sendHistoricalLogs(client *StreamClient, fromLine int64) {
    filePath := fmt.Sprintf("%s/%s.log", ls.logsDir, client.Process)
    
    file, err := os.Open(filePath)
    if err != nil {
        // File might not exist yet, which is OK
        return
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    lineNum := int64(0)
    
    for scanner.Scan() {
        lineNum++
        
        if lineNum <= fromLine {
            continue
        }
        
        entry := LogEntry{
            Timestamp: time.Now(), // TODO: Parse timestamp from log line
            Process:   client.Process,
            Message:   scanner.Text(),
            Line:      lineNum,
        }
        
        select {
        case client.Ch <- entry:
            client.LastSeen = lineNum
        case <-client.ctx.Done():
            return
        default:
            // Channel is full, skip this entry
        }
    }
}

// cleanupLoop periodically cleans up disconnected clients
func (ls *LogStreamer) cleanupLoop() {
    defer ls.wg.Done()
    
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ls.ctx.Done():
            return
        case <-ticker.C:
            ls.cleanupDisconnectedClients()
        }
    }
}

// cleanupDisconnectedClients removes clients whose context has been cancelled
func (ls *LogStreamer) cleanupDisconnectedClients() {
    ls.mu.Lock()
    defer ls.mu.Unlock()
    
    for clientID, client := range ls.clients {
        select {
        case <-client.ctx.Done():
            // Client is disconnected
            if watcher, exists := ls.watchers[client.Process]; exists {
                watcher.RemoveClient(clientID)
            }
            close(client.Ch)
            delete(ls.clients, clientID)
        default:
            // Client is still active
        }
    }
    
    // Remove watchers with no clients
    for process, watcher := range ls.watchers {
        if watcher.ClientCount() == 0 {
            watcher.Stop()
            delete(ls.watchers, process)
        }
    }
}

// Start starts watching the log file
func (lw *LogWatcher) Start() error {
    // Try to open the file, create if it doesn't exist
    var err error
    lw.file, err = os.OpenFile(lw.filePath, os.O_RDONLY|os.O_CREATE, 0644)
    if err != nil {
        return fmt.Errorf("failed to open log file %s: %w", lw.filePath, err)
    }
    
    // Seek to end of file to start tailing
    stat, err := lw.file.Stat()
    if err != nil {
        lw.file.Close()
        return fmt.Errorf("failed to stat log file: %w", err)
    }
    
    lw.position = stat.Size()
    lw.file.Seek(lw.position, io.SeekStart)
    lw.scanner = bufio.NewScanner(lw.file)
    
    // Count existing lines
    if tempFile, err := os.Open(lw.filePath); err == nil {
        tempScanner := bufio.NewScanner(tempFile)
        for tempScanner.Scan() {
            lw.lineCount++
        }
        tempFile.Close()
    }
    
    // Start watching goroutine
    go lw.watch()
    
    return nil
}

// Stop stops the log watcher
func (lw *LogWatcher) Stop() {
    lw.cancel()
    if lw.file != nil {
        lw.file.Close()
    }
}

// AddClient adds a client to this watcher
func (lw *LogWatcher) AddClient(client *StreamClient) {
    lw.mu.Lock()
    defer lw.mu.Unlock()
    lw.clients[client.ID] = client
}

// RemoveClient removes a client from this watcher
func (lw *LogWatcher) RemoveClient(clientID string) {
    lw.mu.Lock()
    defer lw.mu.Unlock()
    delete(lw.clients, clientID)
}

// ClientCount returns the number of active clients
func (lw *LogWatcher) ClientCount() int {
    lw.mu.RLock()
    defer lw.mu.RUnlock()
    return len(lw.clients)
}

// watch continuously monitors the log file for changes
func (lw *LogWatcher) watch() {
    ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
    defer ticker.Stop()
    
    for {
        select {
        case <-lw.ctx.Done():
            return
        case <-ticker.C:
            lw.checkForNewLines()
        }
    }
}

// checkForNewLines checks for new lines in the log file
func (lw *LogWatcher) checkForNewLines() {
    // Check if file has grown
    stat, err := lw.file.Stat()
    if err != nil {
        return
    }
    
    if stat.Size() <= lw.position {
        return // No new content
    }
    
    // Read new content
    lw.file.Seek(lw.position, io.SeekStart)
    scanner := bufio.NewScanner(lw.file)
    
    var newEntries []LogEntry
    
    for scanner.Scan() {
        lw.lineCount++
        line := scanner.Text()
        
        entry := LogEntry{
            Timestamp: time.Now(), // TODO: Parse timestamp from log line
            Process:   lw.process,
            Message:   line,
            Line:      lw.lineCount,
            Level:     parseLogLevel(line), // TODO: Implement log level parsing
        }
        
        newEntries = append(newEntries, entry)
    }
    
    // Update position
    lw.position = stat.Size()
    lw.lastCheck = time.Now()
    
    // Send entries to all clients
    if len(newEntries) > 0 {
        lw.broadcastEntries(newEntries)
    }
}

// broadcastEntries sends log entries to all clients
func (lw *LogWatcher) broadcastEntries(entries []LogEntry) {
    lw.mu.RLock()
    clients := make(map[string]*StreamClient)
    for id, client := range lw.clients {
        clients[id] = client
    }
    lw.mu.RUnlock()
    
    for _, client := range clients {
        for _, entry := range entries {
            // Only send entries newer than what client has seen
            if entry.Line > client.LastSeen {
                select {
                case client.Ch <- entry:
                    client.LastSeen = entry.Line
                case <-client.ctx.Done():
                    // Client disconnected, will be cleaned up later
                    continue
                default:
                    // Channel is full, skip this entry
                    // TODO: Could implement overflow handling
                }
            }
        }
    }
}

// parseLogLevel attempts to parse log level from log line
func parseLogLevel(line string) string {
    // Simple log level detection
    line = strings.ToLower(line)
    
    if strings.Contains(line, "error") || strings.Contains(line, "err") {
        return "error"
    }
    if strings.Contains(line, "warn") {
        return "warning"
    }
    if strings.Contains(line, "info") {
        return "info"
    }
    if strings.Contains(line, "debug") {
        return "debug"
    }
    
    return ""
}

// GetActiveStreams returns information about active streams
func (ls *LogStreamer) GetActiveStreams() map[string]interface{} {
    ls.mu.RLock()
    defer ls.mu.RUnlock()
    
    result := map[string]interface{}{
        "totalClients": len(ls.clients),
        "totalWatchers": len(ls.watchers),
        "clients": make([]map[string]interface{}, 0, len(ls.clients)),
        "watchers": make([]map[string]interface{}, 0, len(ls.watchers)),
    }
    
    for _, client := range ls.clients {
        clientInfo := map[string]interface{}{
            "id":       client.ID,
            "process":  client.Process,
            "lastSeen": client.LastSeen,
        }
        result["clients"] = append(result["clients"].([]map[string]interface{}), clientInfo)
    }
    
    for _, watcher := range ls.watchers {
        watcherInfo := map[string]interface{}{
            "process":     watcher.process,
            "filePath":    watcher.filePath,
            "lineCount":   watcher.lineCount,
            "position":    watcher.position,
            "clientCount": watcher.ClientCount(),
            "lastCheck":   watcher.lastCheck,
        }
        result["watchers"] = append(result["watchers"].([]map[string]interface{}), watcherInfo)
    }
    
    return result
}