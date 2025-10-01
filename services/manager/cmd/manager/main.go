package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"mcp/manager/internal/health"
	api "mcp/manager/internal/httpapi"
	"mcp/manager/internal/logs"
	"mcp/manager/internal/paths"
	"mcp/manager/internal/registry"
	"mcp/manager/internal/supervisor"
)

func main() {
	log.SetPrefix("mcp-manager: ")
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run(ctx context.Context) error {
	log.Println("starting manager daemon")

	// Ensure all required directories exist
	if err := paths.EnsureAllDirectories(); err != nil {
		return fmt.Errorf("failed to create required directories: %w", err)
	}

	// Load registry from default location, creating new if doesn't exist
	reg, err := registry.LoadDefault()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Get logs directory for streaming and monitoring
	logsDir, err := paths.LogsDir()
	if err != nil {
		return fmt.Errorf("failed to get logs directory: %w", err)
	}

	// Initialize enhanced supervisor with caps (128MB per server, 1GB global)
	sup := supervisor.New(reg, 128*1024*1024, 1024*1024*1024)

	// Initialize health monitor
	healthMonitor := health.NewHealthMonitor(30 * time.Second)

	// Set up health monitor callbacks for automatic process management
	healthMonitor.SetCallbacks(
		func(processName string, oldStatus, newStatus health.Status) {
			log.Printf("Health status changed for %s: %s -> %s", processName, oldStatus, newStatus)
		},
		func(processName string, reason string) {
			log.Printf("Process %s failed: %s", processName, reason)
			// Attempt automatic restart for failed processes
			if err := sup.Restart(processName); err != nil {
				log.Printf("Failed to restart process %s: %v", processName, err)
			}
		},
	)

	// Set up registry updater for external server status persistence
	healthMonitor.SetRegistryUpdater(func(slug string, status registry.ExternalStatus) {
		log.Printf("Updating registry status for external server %s: %s - %s", slug, status.State, status.Message)

		// Find the server in the registry and update its status
		for i := range reg.Servers {
			if reg.Servers[i].Slug == slug && reg.Servers[i].IsExternal() {
				if reg.Servers[i].External != nil {
					reg.Servers[i].External.Status = status

					// Save updated registry periodically (every 10 updates or on significant status changes)
					// For now, we'll save immediately - could be optimized with batching
					if status.State == "error" || status.State == "active" {
						go func() {
							if saveErr := registry.SaveDefault(reg); saveErr != nil {
								log.Printf("Failed to save registry after status update: %v", saveErr)
							}
						}()
					}
				}
				break
			}
		}
	})

	// Initialize log streamer
	logStreamer := logs.NewLogStreamer(logsDir)

	// Create HTTP API server with all components
	cm, _ := api.NewCredentialManager()
	srv := api.NewServer(reg).WithSupervisor(sup).WithHealthMonitor(healthMonitor).WithLogStreamer(logStreamer).WithCredentialManager(cm)

	httpServer := &http.Server{
		Addr:         "127.0.0.1:7099",
		Handler:      srv.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	go func() {
		log.Println("HTTP server listening on 127.0.0.1:7099")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start health monitoring
	healthMonitor.Start()
	log.Println("Health monitoring started")

	// Start log streaming
	logStreamer.Start()
	log.Println("Log streaming started")

	// Start autostart servers and add them to health monitoring
	log.Println("Starting autostart servers...")
	for _, s := range reg.Servers {
		if s.Auto != nil && s.Auto.Enabled {
			if s.IsExternal() {
				// External servers don't need to be "started" by supervisor
				// but should be added to health monitoring
				log.Printf("Registering external autostart server for monitoring: %s", s.Name)
				ext := s.GetExternalConfig()
				healthMonitor.AddExternalProcess(s.Slug, ext.Provider, ext.APIEndpoint, ext.AuthType)
			} else {
				// Local servers need to be started and monitored
				log.Printf("Starting autostart server: %s", s.Name)
				if err := sup.Start(s.Slug); err != nil {
					log.Printf("Failed to start autostart server %s: %v", s.Name, err)
				} else {
					// Add to health monitoring
					httpURL := ""
					if s.Entry.Transport == "http" {
						httpURL = deriveHTTPURL(s.Entry.Args, s.Entry.Env)
					}
					logPath := fmt.Sprintf("%s/%s.log", logsDir, s.Slug)
					healthMonitor.AddProcess(s.Slug, s.Entry.Transport, httpURL, logPath)
				}
			}
		}
	}

	// Initialize all external servers in registry for health monitoring
	// even if they don't have autostart enabled
	log.Println("Registering external servers for health monitoring...")
	for _, s := range reg.Servers {
		if s.IsExternal() && (s.Auto == nil || !s.Auto.Enabled) {
			// Add external servers that aren't autostart enabled
			log.Printf("Registering external server for monitoring: %s", s.Name)
			ext := s.GetExternalConfig()
			healthMonitor.AddExternalProcess(s.Slug, ext.Provider, ext.APIEndpoint, ext.AuthType)
		}
	}

	// Start log rotation janitor
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		log.Println("Log rotation janitor started")

		for {
			select {
			case <-ctx.Done():
				log.Println("Log rotation janitor shutting down")
				return
			case <-ticker.C:
				files, sizes, err := logs.ListLogFiles(logsDir)
				if err != nil {
					log.Printf("Failed to list log files: %v", err)
					continue
				}

				trim := logs.PlanRotation(sizes, 128*1024*1024, 1024*1024*1024)
				if err := logs.ApplyRotation(files, trim); err != nil {
					log.Printf("Failed to apply log rotation: %v", err)
				}
			}
		}
	}()

	log.Println("Manager daemon fully started and ready")

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutdown signal received, beginning graceful shutdown...")

	// Create shutdown context with generous timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop health monitoring
	log.Println("Stopping health monitoring...")
	healthMonitor.Stop()

	// Stop log streaming
	log.Println("Stopping log streaming...")
	logStreamer.Stop()

	// Shutdown supervisor (stops all processes)
	log.Println("Shutting down supervisor and all processes...")
	if err := sup.Shutdown(20 * time.Second); err != nil {
		log.Printf("Warning: supervisor shutdown error: %v", err)
	}

	// Save registry before shutdown
	log.Println("Saving registry...")
	if err := registry.SaveDefault(reg); err != nil {
		log.Printf("Warning: failed to save registry on shutdown: %v", err)
	}

	// Shutdown HTTP server
	log.Println("Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Warning: HTTP server shutdown error: %v", err)
	}

	log.Println("Manager daemon shutdown complete")
	return nil
}

// deriveHTTPURL tries to construct a local HTTP URL from args or env.
// Priority: env[HEALTH_HTTP_URL], then --port=NNNN or -p NNNN in args → http://127.0.0.1:NNNN
func deriveHTTPURL(args []string, env map[string]string) string {
	if env != nil {
		if u, ok := env["HEALTH_HTTP_URL"]; ok && u != "" {
			return u
		}
	}

	var port string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 7 && a[:7] == "--port=" {
			port = a[7:]
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
