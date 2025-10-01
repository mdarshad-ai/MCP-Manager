package install

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mcp/manager/internal/paths"
)

// NPMInstaller handles npm-based MCP server installations
type NPMInstaller struct {
	runner Runner
	logger Logger
}

// NewNPMInstaller creates a new npm installer instance
func NewNPMInstaller(runner Runner, logger Logger) *NPMInstaller {
	if runner == nil {
		runner = ExecRunner{}
	}
	return &NPMInstaller{
		runner: runner,
		logger: logger,
	}
}

// NPMInstallOptions contains configuration for npm-based installations
type NPMInstallOptions struct {
	Package       string            `json:"package"`                 // npm package name (e.g., "@anthropic/mcp-cli")
	Version       string            `json:"version,omitempty"`       // specific version (e.g., "1.0.0", "^1.0.0", "latest")
	Registry      string            `json:"registry,omitempty"`      // custom npm registry URL
	Global        bool              `json:"global,omitempty"`        // install globally
	Token         string            `json:"token,omitempty"`         // npm auth token
	Username      string            `json:"username,omitempty"`      // npm username for auth
	Password      string            `json:"password,omitempty"`      // npm password for auth
	Email         string            `json:"email,omitempty"`         // npm email for auth
	Scope         string            `json:"scope,omitempty"`         // npm scope for scoped packages
	PreferManager string            `json:"preferManager,omitempty"` // preferred package manager (npm, yarn, pnpm)
	Development   bool              `json:"development,omitempty"`   // install dev dependencies
	Production    bool              `json:"production,omitempty"`    // install only production dependencies
	Environment   map[string]string `json:"environment,omitempty"`   // environment variables
	NodeVersion   string            `json:"nodeVersion,omitempty"`   // required Node.js version
	PostInstall   []string          `json:"postInstall,omitempty"`   // commands to run after install
	MCPConfig     *NPMMCPConfig     `json:"mcpConfig,omitempty"`     // MCP-specific configuration
}

// NPMMCPConfig contains MCP-specific npm configuration
type NPMMCPConfig struct {
	EntryScript  string            `json:"entryScript,omitempty"`  // main script for the MCP server
	EntryCommand string            `json:"entryCommand,omitempty"` // command to run the server
	Args         []string          `json:"args,omitempty"`         // default arguments
	Environment  map[string]string `json:"environment,omitempty"`  // environment variables specific to MCP
	Transport    string            `json:"transport,omitempty"`    // MCP transport type (stdio, ws, sse)
	Capabilities []string          `json:"capabilities,omitempty"` // MCP server capabilities
}

// NPMInstallResult contains the result of an npm installation
type NPMInstallResult struct {
	Success          bool              `json:"success"`
	InstallPath      string            `json:"installPath"`
	RuntimePath      string            `json:"runtimePath"`
	BinPath          string            `json:"binPath"`
	PackageManager   string            `json:"packageManager"`
	InstalledVersion string            `json:"installedVersion"`
	EntryCommand     string            `json:"entryCommand"`
	EntryArgs        []string          `json:"entryArgs"`
	Environment      map[string]string `json:"environment"`
	BinExecutables   []string          `json:"binExecutables"`
	PackageInfo      *NPMPackageInfo   `json:"packageInfo,omitempty"`
	Logs             []string          `json:"logs"`
	Error            string            `json:"error,omitempty"`
}

// NPMPackageInfo contains information about the installed npm package
type NPMPackageInfo struct {
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description"`
	Main            string                 `json:"main,omitempty"`
	Bin             map[string]string      `json:"bin,omitempty"`
	Scripts         map[string]string      `json:"scripts,omitempty"`
	Dependencies    map[string]string      `json:"dependencies,omitempty"`
	DevDependencies map[string]string      `json:"devDependencies,omitempty"`
	Keywords        []string               `json:"keywords,omitempty"`
	Author          interface{}            `json:"author,omitempty"`
	License         string                 `json:"license,omitempty"`
	Homepage        string                 `json:"homepage,omitempty"`
	Repository      interface{}            `json:"repository,omitempty"`
	MCPMetadata     map[string]interface{} `json:"mcp,omitempty"`
}

// Install performs an npm-based installation of an MCP server
func (n *NPMInstaller) Install(ctx context.Context, slug string, options NPMInstallOptions) (*NPMInstallResult, error) {
	result := &NPMInstallResult{
		Environment: make(map[string]string),
	}

	// Create installation directories
	baseServers, err := paths.ServersDir()
	if err != nil {
		if n.logger != nil {
			logf(n.logger, "Failed to get servers directory: %v", err)
		}
		return result, fmt.Errorf("failed to get servers directory: %w", err)
	}
	if n.logger != nil {
		logf(n.logger, "Using servers directory: %s", baseServers)
	}

	serverDir := filepath.Join(baseServers, slug)
	installDir := filepath.Join(serverDir, "install")
	runtimeDir := filepath.Join(serverDir, "runtime")
	binDir := filepath.Join(serverDir, "bin")

	result.InstallPath = installDir
	result.RuntimePath = runtimeDir
	result.BinPath = binDir

	// Ensure directories exist
	for _, dir := range []string{serverDir, installDir, runtimeDir, binDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return result, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if n.logger != nil {
		if n.logger != nil {
			if n.logger != nil {
				logf(n.logger, "Starting npm installation for %s", slug)
				logf(n.logger, "Package: %s", options.Package)
			}
		}
	}

	// Detect and validate package manager
	packageManager, err := n.detectPackageManager(ctx, options.PreferManager)
	if err != nil {
		return result, fmt.Errorf("package manager detection failed: %w", err)
	}
	result.PackageManager = packageManager
	logf(n.logger, "Using package manager: %s", packageManager)

	// Validate Node.js version if specified
	if options.NodeVersion != "" {
		if err := n.validateNodeVersion(ctx, options.NodeVersion); err != nil {
			return result, fmt.Errorf("node.js version validation failed: %w", err)
		}
	}

	// Set up authentication if provided
	if err := n.setupAuthentication(ctx, options, runtimeDir); err != nil {
		return result, fmt.Errorf("authentication setup failed: %w", err)
	}

	// Validate package exists before installation
	if err := n.validatePackage(ctx, options, packageManager); err != nil {
		return result, fmt.Errorf("package validation failed: %w", err)
	}

	// Install package
	if err := n.installPackage(ctx, options, runtimeDir, packageManager); err != nil {
		return result, fmt.Errorf("package installation failed: %w", err)
	}

	// Get installed package information
	packageInfo, installedVersion, err := n.getPackageInfo(ctx, options.Package, runtimeDir, packageManager)
	if err != nil {
		logf(n.logger, "Warning: Failed to get package info: %v", err)
	} else {
		result.PackageInfo = packageInfo
		result.InstalledVersion = installedVersion
	}

	// Detect executables and entry points
	binExecutables, err := n.detectBinExecutables(runtimeDir, packageInfo)
	if err != nil {
		logf(n.logger, "Warning: Failed to detect executables: %v", err)
	} else {
		result.BinExecutables = binExecutables
	}

	// Determine entry point
	entryCmd, entryArgs, env, err := n.determineEntryPoint(options, packageInfo, runtimeDir)
	if err != nil {
		logf(n.logger, "Warning: Failed to determine entry point: %v", err)
	} else {
		result.EntryCommand = entryCmd
		result.EntryArgs = entryArgs
		for k, v := range env {
			result.Environment[k] = v
		}
	}

	// Run post-install commands if specified
	if len(options.PostInstall) > 0 {
		if err := n.runPostInstallCommands(ctx, runtimeDir, options); err != nil {
			return result, fmt.Errorf("post-install commands failed: %w", err)
		}
	}

	// Create executable script in bin directory
	if err := n.createBinScript(binDir, slug, result.EntryCommand, result.EntryArgs, result.Environment); err != nil {
		return result, fmt.Errorf("failed to create bin script: %w", err)
	}

	// Copy important files to install directory for reference
	if err := n.copyPackageFiles(runtimeDir, installDir, options.Package); err != nil {
		logf(n.logger, "Warning: Failed to copy package files: %v", err)
	}

	result.Success = true
	logf(n.logger, "NPM installation completed successfully for %s", slug)
	return result, nil
}

// detectPackageManager detects and validates the package manager to use
func (n *NPMInstaller) detectPackageManager(ctx context.Context, preferred string) (string, error) {
	managers := []string{"npm", "yarn", "pnpm"}

	if preferred != "" {
		if n.isPackageManagerAvailable(ctx, preferred) {
			return preferred, nil
		}
		logf(n.logger, "Preferred package manager %s not available, trying alternatives", preferred)
	}

	for _, manager := range managers {
		if n.isPackageManagerAvailable(ctx, manager) {
			return manager, nil
		}
	}

	return "", fmt.Errorf("no supported package manager found (tried: %s)", strings.Join(managers, ", "))
}

func (n *NPMInstaller) isPackageManagerAvailable(ctx context.Context, manager string) bool {
	_, _, err := n.runner.Run(ctx, manager, "--version")
	return err == nil
}

// validateNodeVersion validates the Node.js version meets requirements
func (n *NPMInstaller) validateNodeVersion(ctx context.Context, required string) error {
	stdout, _, err := n.runner.Run(ctx, "node", "--version")
	if err != nil {
		return fmt.Errorf("Node.js not found: %w", err)
	}

	currentVersion := strings.TrimSpace(strings.TrimPrefix(stdout, "v"))
	logf(n.logger, "Node.js version: %s (required: %s)", currentVersion, required)

	// For now, just log the versions. A full semver comparison would be needed for strict validation
	// This could be implemented using a semver library if needed

	return nil
}

// setupAuthentication configures npm authentication
func (n *NPMInstaller) setupAuthentication(ctx context.Context, options NPMInstallOptions, runtimeDir string) error {
	if options.Token == "" && options.Username == "" {
		return nil // No authentication needed
	}

	logf(n.logger, "Setting up npm authentication...")

	// Create .npmrc file in runtime directory
	npmrcPath := filepath.Join(runtimeDir, ".npmrc")

	var npmrcContent strings.Builder

	if options.Registry != "" {
		npmrcContent.WriteString(fmt.Sprintf("registry=%s\n", options.Registry))
	}

	if options.Token != "" {
		registry := options.Registry
		if registry == "" {
			registry = "//registry.npmjs.org/"
		}
		// Remove protocol from registry for token auth
		if strings.HasPrefix(registry, "https:") {
			registry = strings.TrimPrefix(registry, "https:")
		} else if strings.HasPrefix(registry, "http:") {
			registry = strings.TrimPrefix(registry, "http:")
		}
		npmrcContent.WriteString(fmt.Sprintf("%s:_authToken=%s\n", registry, options.Token))
	}

	if options.Username != "" && options.Password != "" && options.Email != "" {
		registry := options.Registry
		if registry == "" {
			registry = "//registry.npmjs.org/"
		}
		// Basic auth setup would require base64 encoding, which is more complex
		// For now, prefer token-based auth
		logf(n.logger, "Warning: Basic auth setup is complex, consider using token auth instead")
	}

	if npmrcContent.Len() > 0 {
		if err := os.WriteFile(npmrcPath, []byte(npmrcContent.String()), 0o600); err != nil {
			return fmt.Errorf("failed to write .npmrc: %w", err)
		}
	}

	return nil
}

// validatePackage checks if the package exists and is accessible
func (n *NPMInstaller) validatePackage(ctx context.Context, options NPMInstallOptions, packageManager string) error {
	logf(n.logger, "Validating package availability...")

	packageSpec := options.Package
	if options.Version != "" {
		packageSpec = fmt.Sprintf("%s@%s", options.Package, options.Version)
	}

	var args []string
	switch packageManager {
	case "npm":
		args = []string{"view", packageSpec, "version"}
	case "yarn":
		args = []string{"info", packageSpec, "version"}
	case "pnpm":
		args = []string{"view", packageSpec, "version"}
	default:
		return fmt.Errorf("unsupported package manager: %s", packageManager)
	}

	if options.Registry != "" {
		args = append(args, "--registry", options.Registry)
	}

	_, _, err := n.runner.Run(ctx, packageManager, args...)
	if err != nil {
		return fmt.Errorf("package not found or not accessible: %w", err)
	}

	logf(n.logger, "Package validated successfully")
	return nil
}

// installPackage performs the actual package installation
func (n *NPMInstaller) installPackage(ctx context.Context, options NPMInstallOptions, runtimeDir, packageManager string) error {
	logf(n.logger, "Installing package with %s...", packageManager)

	packageSpec := options.Package
	if options.Version != "" {
		packageSpec = fmt.Sprintf("%s@%s", options.Package, options.Version)
	}
	if options.Version != "" {
		packageSpec = fmt.Sprintf("%s@%s", options.Package, options.Version)
	}

	// Create package.json in runtime directory if it doesn't exist
	packageJSONPath := filepath.Join(runtimeDir, "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		initialPackageJSON := map[string]interface{}{
			"name":        fmt.Sprintf("mcp-server-%s", packageSpec),
			"version":     "1.0.0",
			"private":     true,
			"description": fmt.Sprintf("MCP Server installation for %s", packageSpec),
		}

		jsonData, _ := json.MarshalIndent(initialPackageJSON, "", "  ")
		if err := os.WriteFile(packageJSONPath, jsonData, 0o644); err != nil {
			return fmt.Errorf("failed to create package.json: %w", err)
		}
	}

	var args []string
	var cmd *exec.Cmd

	switch packageManager {
	case "npm":
		args = []string{"install", packageSpec}
		if options.Global {
			args = []string{"install", "-g", packageSpec}
		}
		if options.Production {
			args = append(args, "--production")
		}
		if options.Development {
			args = append(args, "--include=dev")
		}
		if options.Registry != "" {
			args = append(args, "--registry", options.Registry)
		}
		cmd = exec.CommandContext(ctx, "npm", args...)

	case "yarn":
		if options.Global {
			args = []string{"global", "add", packageSpec}
		} else {
			args = []string{"add", packageSpec}
		}
		if options.Development {
			args = append(args, "--dev")
		}
		if options.Registry != "" {
			args = append(args, "--registry", options.Registry)
		}
		cmd = exec.CommandContext(ctx, "yarn", args...)

	case "pnpm":
		args = []string{"add", packageSpec}
		if options.Global {
			args = []string{"add", "-g", packageSpec}
		}
		if options.Development {
			args = append(args, "--save-dev")
		}
		if options.Production {
			args = append(args, "--prod")
		}
		if options.Registry != "" {
			args = append(args, "--registry", options.Registry)
		}
		cmd = exec.CommandContext(ctx, "pnpm", args...)

	default:
		return fmt.Errorf("unsupported package manager: %s", packageManager)
	}

	// Set working directory and environment
	if !options.Global {
		cmd.Dir = runtimeDir
	}

	env := os.Environ()
	for k, v := range options.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	// Execute installation
	stdout, stderr, err := n.runner.Run(ctx, cmd.Path, cmd.Args[1:]...)
	if err != nil {
		return fmt.Errorf("installation failed: %w, stdout: %s, stderr: %s", err, stdout, stderr)
	}

	logf(n.logger, "Package installed successfully")
	return nil
}

// getPackageInfo retrieves information about the installed package
func (n *NPMInstaller) getPackageInfo(ctx context.Context, packageName, runtimeDir, packageManager string) (*NPMPackageInfo, string, error) {
	// Try to read package.json from node_modules
	packageJSONPath := filepath.Join(runtimeDir, "node_modules", packageName, "package.json")
	if data, err := os.ReadFile(packageJSONPath); err == nil {
		var info NPMPackageInfo
		if err := json.Unmarshal(data, &info); err == nil {
			return &info, info.Version, nil
		}
	}

	// Fallback: use package manager to get info
	var args []string
	switch packageManager {
	case "npm":
		args = []string{"list", packageName, "--json", "--depth=0"}
	case "yarn":
		args = []string{"list", "--pattern", packageName, "--json"}
	case "pnpm":
		args = []string{"list", packageName, "--json", "--depth=0"}
	default:
		return nil, "", fmt.Errorf("unsupported package manager: %s", packageManager)
	}

	cmd := exec.CommandContext(ctx, packageManager, args...)
	cmd.Dir = runtimeDir

	stdout, _, err := n.runner.Run(ctx, cmd.Path, cmd.Args[1:]...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get package info: %w", err)
	}

	// Parse package manager output (simplified - would need proper parsing for each manager)
	var info NPMPackageInfo
	if err := json.Unmarshal([]byte(stdout), &info); err != nil {
		return nil, "", fmt.Errorf("failed to parse package info: %w", err)
	}

	return &info, info.Version, nil
}

// detectBinExecutables finds executable binaries provided by the package
func (n *NPMInstaller) detectBinExecutables(runtimeDir string, packageInfo *NPMPackageInfo) ([]string, error) {
	var executables []string

	if packageInfo != nil && len(packageInfo.Bin) > 0 {
		for name, path := range packageInfo.Bin {
			execPath := filepath.Join(runtimeDir, "node_modules", ".bin", name)
			if _, err := os.Stat(execPath); err == nil {
				executables = append(executables, name)
			}
			_ = path // path in package.json might be relative
		}
	}

	// Also check node_modules/.bin directory
	binDir := filepath.Join(runtimeDir, "node_modules", ".bin")
	if entries, err := os.ReadDir(binDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				executables = append(executables, entry.Name())
			}
		}
	}

	return executables, nil
}

// determineEntryPoint determines the command and arguments to run the MCP server
func (n *NPMInstaller) determineEntryPoint(options NPMInstallOptions, packageInfo *NPMPackageInfo, runtimeDir string) (command string, args []string, env map[string]string, err error) {
	env = make(map[string]string)

	// Add node_modules/.bin to PATH
	binPath := filepath.Join(runtimeDir, "node_modules", ".bin")
	env["PATH"] = fmt.Sprintf("%s:%s", binPath, os.Getenv("PATH"))

	// MCP-specific configuration takes priority
	if options.MCPConfig != nil {
		if options.MCPConfig.EntryCommand != "" {
			return options.MCPConfig.EntryCommand, options.MCPConfig.Args, env, nil
		}
		if options.MCPConfig.EntryScript != "" {
			return "node", []string{filepath.Join(runtimeDir, options.MCPConfig.EntryScript)}, env, nil
		}
	}

	// Check if package has a binary entry
	if packageInfo != nil && len(packageInfo.Bin) > 0 {
		// Use the first binary entry
		for name, _ := range packageInfo.Bin {
			binPath := filepath.Join(runtimeDir, "node_modules", ".bin", name)
			if _, err := os.Stat(binPath); err == nil {
				return binPath, []string{}, env, nil
			}
		}
	}

	// Check package.json main field
	if packageInfo != nil && packageInfo.Main != "" {
		mainPath := filepath.Join(runtimeDir, "node_modules", options.Package, packageInfo.Main)
		if _, err := os.Stat(mainPath); err == nil {
			return "node", []string{mainPath}, env, nil
		}
	}

	// Check for common entry points
	possiblePaths := []string{
		filepath.Join(runtimeDir, "node_modules", options.Package, "index.js"),
		filepath.Join(runtimeDir, "node_modules", options.Package, "main.js"),
		filepath.Join(runtimeDir, "node_modules", options.Package, "server.js"),
		filepath.Join(runtimeDir, "node_modules", options.Package, "src", "index.js"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return "node", []string{path}, env, nil
		}
	}

	return "", nil, env, fmt.Errorf("no entry point found for package %s", options.Package)
}

// runPostInstallCommands executes user-defined post-install commands
func (n *NPMInstaller) runPostInstallCommands(ctx context.Context, runtimeDir string, options NPMInstallOptions) error {
	logf(n.logger, "Running post-install commands...")

	for i, cmdStr := range options.PostInstall {
		logf(n.logger, "Running post-install command %d: %s", i+1, cmdStr)

		// Parse command and arguments
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			continue
		}

		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		cmd.Dir = runtimeDir

		// Add environment variables
		env := os.Environ()
		for k, v := range options.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		// Add node_modules/.bin to PATH
		binPath := filepath.Join(runtimeDir, "node_modules", ".bin")
		env = append(env, fmt.Sprintf("PATH=%s:%s", binPath, os.Getenv("PATH")))
		cmd.Env = env

		stdout, stderr, err := n.runner.Run(ctx, cmd.Path, cmd.Args[1:]...)
		if err != nil {
			return fmt.Errorf("post-install command failed: %s: %w, stdout: %s, stderr: %s", cmdStr, err, stdout, stderr)
		}
	}

	return nil
}

// createBinScript creates an executable script in the bin directory
func (n *NPMInstaller) createBinScript(binDir, slug, command string, args []string, env map[string]string) error {
	scriptPath := filepath.Join(binDir, slug)

	// Create a shell script that executes the MCP server
	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("# Generated MCP server launcher\n\n")

	// Add environment variables
	for k, v := range env {
		script.WriteString(fmt.Sprintf("export %s=\"%s\"\n", k, v))
	}

	// Add the command
	if command != "" {
		script.WriteString(fmt.Sprintf("exec \"%s\"", command))
		for _, arg := range args {
			script.WriteString(fmt.Sprintf(" \"%s\"", arg))
		}
		script.WriteString(" \"$@\"\n")
	} else {
		script.WriteString("echo 'No entry point configured for this MCP server'\n")
		script.WriteString("exit 1\n")
	}

	if err := os.WriteFile(scriptPath, []byte(script.String()), 0o755); err != nil {
		return fmt.Errorf("failed to write script: %w", err)
	}

	return nil
}

// copyPackageFiles copies important package files to the install directory for reference
func (n *NPMInstaller) copyPackageFiles(runtimeDir, installDir, packageName string) error {
	packageDir := filepath.Join(runtimeDir, "node_modules", packageName)

	// Files to copy for reference
	filesToCopy := []string{"package.json", "README.md", "LICENSE", "CHANGELOG.md"}

	for _, file := range filesToCopy {
		srcPath := filepath.Join(packageDir, file)
		dstPath := filepath.Join(installDir, file)

		if _, err := os.Stat(srcPath); err == nil {
			if err := copyFile(srcPath, dstPath); err != nil {
				// Don't fail on copy errors, just log
				continue
			}
		}
	}

	return nil
}

// copyFile is a helper function to copy files
func copyFile(src, dst string) error {
	sourceData, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, sourceData, 0o644)
}
