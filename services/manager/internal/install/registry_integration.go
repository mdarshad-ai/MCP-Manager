package install

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mcp/manager/internal/paths"
	"mcp/manager/internal/registry"
)

// RegistryIntegrator handles the integration between installation results and the server registry
type RegistryIntegrator struct {
	registryPath string
}

// NewRegistryIntegrator creates a new registry integrator
func NewRegistryIntegrator() (*RegistryIntegrator, error) {
	baseDir, err := paths.HomeMCP()
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP home directory: %w", err)
	}
	
	registryPath := filepath.Join(baseDir, "registry.json")
	return &RegistryIntegrator{
		registryPath: registryPath,
	}, nil
}

// RegisterServer adds a server to the registry based on installation results
func (ri *RegistryIntegrator) RegisterServer(ctx context.Context, slug string, installResult *InstallationResult, sourceType SourceType, sourceURI string) (*registry.Server, error) {
	// Load existing registry
	reg, err := ri.loadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}
	
	// Check if server already exists
	for i, server := range reg.Servers {
		if server.Slug == slug {
			// Update existing server
			updatedServer := ri.createServerEntry(slug, installResult, sourceType, sourceURI)
			reg.Servers[i] = *updatedServer
			
			if err := ri.saveRegistry(reg); err != nil {
				return nil, fmt.Errorf("failed to save registry: %w", err)
			}
			
			return updatedServer, nil
		}
	}
	
	// Add new server
	newServer := ri.createServerEntry(slug, installResult, sourceType, sourceURI)
	reg.Servers = append(reg.Servers, *newServer)
	
	if err := ri.saveRegistry(reg); err != nil {
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}
	
	return newServer, nil
}

// UnregisterServer removes a server from the registry
func (ri *RegistryIntegrator) UnregisterServer(ctx context.Context, slug string) error {
	reg, err := ri.loadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	
	// Find and remove the server
	for i, server := range reg.Servers {
		if server.Slug == slug {
			reg.Servers = append(reg.Servers[:i], reg.Servers[i+1:]...)
			break
		}
	}
	
	return ri.saveRegistry(reg)
}

// GetServerEntry retrieves a server entry from the registry
func (ri *RegistryIntegrator) GetServerEntry(slug string) (*registry.Server, error) {
	reg, err := ri.loadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}
	
	for _, server := range reg.Servers {
		if server.Slug == slug {
			return &server, nil
		}
	}
	
	return nil, fmt.Errorf("server %s not found in registry", slug)
}

// ValidateServerInstallation validates that a server installation is complete and functional
func (ri *RegistryIntegrator) ValidateServerInstallation(ctx context.Context, slug string, installResult *InstallationResult) error {
	// Check if all required paths exist
	if installResult.InstallPath != "" {
		if _, err := os.Stat(installResult.InstallPath); os.IsNotExist(err) {
			return fmt.Errorf("install path does not exist: %s", installResult.InstallPath)
		}
	}
	
	if installResult.RuntimePath != "" {
		if _, err := os.Stat(installResult.RuntimePath); os.IsNotExist(err) {
			return fmt.Errorf("runtime path does not exist: %s", installResult.RuntimePath)
		}
	}
	
	if installResult.BinPath != "" {
		if _, err := os.Stat(installResult.BinPath); os.IsNotExist(err) {
			return fmt.Errorf("bin path does not exist: %s", installResult.BinPath)
		}
	}
	
	// Check if entry command is executable
	if installResult.EntryCommand != "" {
		if _, err := os.Stat(installResult.EntryCommand); os.IsNotExist(err) {
			return fmt.Errorf("entry command does not exist: %s", installResult.EntryCommand)
		}
		
		// Check if it's executable
		if info, err := os.Stat(installResult.EntryCommand); err == nil {
			if info.Mode()&0111 == 0 {
				return fmt.Errorf("entry command is not executable: %s", installResult.EntryCommand)
			}
		}
	}
	
	return nil
}

// CreateManifest creates a server manifest file with installation details
func (ri *RegistryIntegrator) CreateManifest(slug string, installResult *InstallationResult, sourceType SourceType, sourceURI string) error {
	baseServers, err := paths.ServersDir()
	if err != nil {
		return fmt.Errorf("failed to get servers directory: %w", err)
	}
	
	serverDir := filepath.Join(baseServers, slug)
	manifestPath := filepath.Join(serverDir, "manifest.json")
	
	manifest := ServerManifest{
		Version: "1.0",
		Slug:    slug,
		Source: ManifestSource{
			Type: string(sourceType),
			URI:  sourceURI,
		},
		Installation: ManifestInstallation{
			Timestamp:       installResult.Metadata["installTime"],
			Runtime:         installResult.Runtime,
			PackageManager:  installResult.PackageManager,
			InstalledVersion: installResult.InstalledVersion,
			InstallPath:     installResult.InstallPath,
			RuntimePath:     installResult.RuntimePath,
			BinPath:         installResult.BinPath,
		},
		Entry: ManifestEntry{
			Command:     installResult.EntryCommand,
			Args:        installResult.EntryArgs,
			Environment: installResult.Environment,
		},
		Metadata: installResult.Metadata,
	}
	
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	
	return nil
}

// loadRegistry loads the server registry from disk
func (ri *RegistryIntegrator) loadRegistry() (*registry.Registry, error) {
	if _, err := os.Stat(ri.registryPath); os.IsNotExist(err) {
		// Create a new empty registry
		return &registry.Registry{
			Version: "1.0",
			Servers: make([]registry.Server, 0),
		}, nil
	}
	
	data, err := os.ReadFile(ri.registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}
	
	var reg registry.Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal registry: %w", err)
	}
	
	return &reg, nil
}

// saveRegistry saves the server registry to disk
func (ri *RegistryIntegrator) saveRegistry(reg *registry.Registry) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}
	
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(ri.registryPath), 0o755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}
	
	// Write to a temporary file first, then rename for atomic operation
	tempPath := ri.registryPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temporary registry file: %w", err)
	}
	
	if err := os.Rename(tempPath, ri.registryPath); err != nil {
		return fmt.Errorf("failed to rename registry file: %w", err)
	}
	
	return nil
}

// createServerEntry creates a registry server entry from installation results
func (ri *RegistryIntegrator) createServerEntry(slug string, installResult *InstallationResult, sourceType SourceType, sourceURI string) *registry.Server {
	server := &registry.Server{
		Name: slug,
		Slug: slug,
		Source: registry.Source{
			Type: string(sourceType),
			URI:  sourceURI,
		},
		Runtime: ri.createRuntimeEntry(installResult),
		Entry: registry.Entry{
			Transport: "stdio", // Default transport
			Command:   installResult.EntryCommand,
			Args:      installResult.EntryArgs,
			Env:       installResult.Environment,
		},
		Health: registry.Health{
			Probe:         "command",
			Method:        "status",
			IntervalSec:   30,
			TimeoutSec:    10,
			RestartPolicy: "on-failure",
			MaxRestarts:   3,
		},
		Clients: registry.Clients{
			// Initially disabled - user can enable as needed
			ClaudeDesktop: &registry.ClientFlag{Enabled: false},
			CursorGlobal:  &registry.ClientFlag{Enabled: false},
			Continue:      &registry.ClientFlag{Enabled: false},
		},
	}
	
	return server
}

// createRuntimeEntry creates a runtime entry based on the detected runtime
func (ri *RegistryIntegrator) createRuntimeEntry(installResult *InstallationResult) registry.Runtime {
	runtime := registry.Runtime{
		Kind: installResult.Runtime,
	}
	
	switch installResult.Runtime {
	case "node":
		runtime.Node = &registry.NodeRuntime{
			PackageManager: installResult.PackageManager,
		}
	case "python":
		runtime.Python = &registry.PyRuntime{
			Manager: installResult.PackageManager,
			Venv:    installResult.Metadata["hasVenv"] == true,
		}
	}
	
	return runtime
}

// ServerManifest represents the structure of a server manifest file
type ServerManifest struct {
	Version      string               `json:"version"`
	Slug         string               `json:"slug"`
	Source       ManifestSource       `json:"source"`
	Installation ManifestInstallation `json:"installation"`
	Entry        ManifestEntry        `json:"entry"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ManifestSource represents the source information in the manifest
type ManifestSource struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

// ManifestInstallation represents installation details in the manifest
type ManifestInstallation struct {
	Timestamp        interface{}       `json:"timestamp"`
	Runtime          string            `json:"runtime"`
	PackageManager   string            `json:"packageManager,omitempty"`
	InstalledVersion string            `json:"installedVersion,omitempty"`
	InstallPath      string            `json:"installPath"`
	RuntimePath      string            `json:"runtimePath"`
	BinPath          string            `json:"binPath"`
}

// ManifestEntry represents the entry point information in the manifest
type ManifestEntry struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}