package install

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mcp/manager/internal/paths"
)

// PipInstaller handles pip-based MCP server installations
type PipInstaller struct {
	runner Runner
	logger Logger
}

// NewPipInstaller creates a new pip installer instance
func NewPipInstaller(runner Runner, logger Logger) *PipInstaller {
	if runner == nil {
		runner = ExecRunner{}
	}
	return &PipInstaller{
		runner: runner,
		logger: logger,
	}
}

// PipInstallOptions contains configuration for pip-based installations
type PipInstallOptions struct {
	Package          string            `json:"package"`                    // pip package name (e.g., "anthropic-mcp")
	Version          string            `json:"version,omitempty"`          // specific version (e.g., "1.0.0", ">=1.0.0", "latest")
	ExtraIndexURL    string            `json:"extraIndexUrl,omitempty"`    // additional package index URL
	IndexURL         string            `json:"indexUrl,omitempty"`         // custom package index URL
	TrustedHost      string            `json:"trustedHost,omitempty"`      // trusted host for HTTPS
	PreRelease       bool              `json:"preRelease,omitempty"`       // allow pre-release versions
	ForceReinstall   bool              `json:"forceReinstall,omitempty"`   // force reinstall even if up to date
	NoDeps           bool              `json:"noDeps,omitempty"`           // don't install dependencies
	UseVenv          bool              `json:"useVenv,omitempty"`          // create and use virtual environment (default: true)
	UsePipx          bool              `json:"usePipx,omitempty"`          // use pipx for isolated installation
	PythonVersion    string            `json:"pythonVersion,omitempty"`    // required Python version
	RequirementsFile string            `json:"requirementsFile,omitempty"` // path to requirements.txt file
	Extras           []string          `json:"extras,omitempty"`           // package extras to install (e.g., ["dev", "test"])
	Environment      map[string]string `json:"environment,omitempty"`      // environment variables
	PostInstall      []string          `json:"postInstall,omitempty"`      // commands to run after install
	MCPConfig        *PipMCPConfig     `json:"mcpConfig,omitempty"`        // MCP-specific configuration
}

// PipMCPConfig contains MCP-specific pip configuration
type PipMCPConfig struct {
	EntryModule  string            `json:"entryModule,omitempty"`  // Python module to run (e.g., "mcp_server.main")
	EntryScript  string            `json:"entryScript,omitempty"`  // Python script to run
	EntryCommand string            `json:"entryCommand,omitempty"` // command to run the server
	Args         []string          `json:"args,omitempty"`         // default arguments
	Environment  map[string]string `json:"environment,omitempty"`  // environment variables specific to MCP
	Transport    string            `json:"transport,omitempty"`    // MCP transport type (stdio, ws, sse)
	Capabilities []string          `json:"capabilities,omitempty"` // MCP server capabilities
}

// PipInstallResult contains the result of a pip installation
type PipInstallResult struct {
	Success           bool              `json:"success"`
	InstallPath       string            `json:"installPath"`
	RuntimePath       string            `json:"runtimePath"`
	BinPath           string            `json:"binPath"`
	VenvPath          string            `json:"venvPath,omitempty"`
	PythonPath        string            `json:"pythonPath"`
	PipPath           string            `json:"pipPath"`
	InstalledVersion  string            `json:"installedVersion"`
	EntryCommand      string            `json:"entryCommand"`
	EntryArgs         []string          `json:"entryArgs"`
	Environment       map[string]string `json:"environment"`
	ConsoleScripts    []string          `json:"consoleScripts"`
	PackageInfo       *PipPackageInfo   `json:"packageInfo,omitempty"`
	InstalledPackages []string          `json:"installedPackages"`
	Logs              []string          `json:"logs"`
	Error             string            `json:"error,omitempty"`
}

// PipPackageInfo contains information about the installed pip package
type PipPackageInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Summary     string                 `json:"summary,omitempty"`
	Author      string                 `json:"author,omitempty"`
	AuthorEmail string                 `json:"author_email,omitempty"`
	License     string                 `json:"license,omitempty"`
	Homepage    string                 `json:"homepage,omitempty"`
	Location    string                 `json:"location,omitempty"`
	Requires    []string               `json:"requires,omitempty"`
	RequiredBy  []string               `json:"required_by,omitempty"`
	EntryPoints map[string][]string    `json:"entry_points,omitempty"`
	MCPMetadata map[string]interface{} `json:"mcp,omitempty"`
}

// Install performs a pip-based installation of an MCP server
func (p *PipInstaller) Install(ctx context.Context, slug string, options PipInstallOptions) (*PipInstallResult, error) {
	result := &PipInstallResult{
		Environment: make(map[string]string),
	}

	// Create installation directories
	baseServers, err := paths.ServersDir()
	if err != nil {
		return result, fmt.Errorf("failed to get servers directory: %w", err)
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

	logf(p.logger, "Starting pip installation for %s", slug)
	logf(p.logger, "Package: %s", options.Package)

	// Validate Python installation
	pythonExec, err := p.detectPythonExecutable(ctx, options.PythonVersion)
	if err != nil {
		result.Error = fmt.Sprintf("Python validation failed: %v", err)
		logf(p.logger, result.Error)
		return result, nil
	}
	result.PythonPath = pythonExec
	logf(p.logger, "Using Python: %s", pythonExec)

	// Handle pipx installation if requested
	if options.UsePipx {
		return p.installWithPipx(ctx, slug, options, result)
	}

	// Create virtual environment if requested (default behavior)
	if options.UseVenv {
		venvPath, err := p.createVirtualEnvironment(ctx, runtimeDir, pythonExec)
		if err != nil {
			result.Error = fmt.Sprintf("Virtual environment creation failed: %v", err)
			logf(p.logger, result.Error)
			return result, nil
		}
		result.VenvPath = venvPath

		// Update Python and pip paths for virtual environment
		if pythonPath, pipPath, err := p.getVenvExecutables(venvPath); err == nil {
			result.PythonPath = pythonPath
			result.PipPath = pipPath
			pythonExec = pythonPath
		}
	} else {
		// Use system pip
		if pipPath, err := p.detectPipExecutable(ctx, pythonExec); err == nil {
			result.PipPath = pipPath
		}
	}

	// Validate package exists before installation
	if err := p.validatePackage(ctx, options, result.PipPath); err != nil {
		result.Error = fmt.Sprintf("Package validation failed: %v", err)
		logf(p.logger, result.Error)
		return result, nil
	}

	// Install package
	if err := p.installPackage(ctx, options, result.PipPath, pythonExec); err != nil {
		result.Error = fmt.Sprintf("Package installation failed: %v", err)
		logf(p.logger, result.Error)
		return result, nil
	}

	// Get installed package information
	packageInfo, installedVersion, err := p.getPackageInfo(ctx, options.Package, result.PipPath, pythonExec)
	if err != nil {
		logf(p.logger, "Warning: Failed to get package info: %v", err)
	} else {
		result.PackageInfo = packageInfo
		result.InstalledVersion = installedVersion
	}

	// Detect console scripts and entry points
	consoleScripts, err := p.detectConsoleScripts(ctx, options.Package, result.VenvPath, pythonExec)
	if err != nil {
		logf(p.logger, "Warning: Failed to detect console scripts: %v", err)
	} else {
		result.ConsoleScripts = consoleScripts
	}

	// Determine entry point
	entryCmd, entryArgs, env, err := p.determineEntryPoint(ctx, options, packageInfo, result.VenvPath, pythonExec)
	if err != nil {
		logf(p.logger, "Warning: Failed to determine entry point: %v", err)
	} else {
		result.EntryCommand = entryCmd
		result.EntryArgs = entryArgs
		for k, v := range env {
			result.Environment[k] = v
		}
	}

	// Run post-install commands if specified
	if len(options.PostInstall) > 0 {
		if err := p.runPostInstallCommands(ctx, result.VenvPath, pythonExec, options); err != nil {
			result.Error = fmt.Sprintf("Post-install commands failed: %v", err)
			logf(p.logger, result.Error)
			return result, nil
		}
	}

	// Create executable script in bin directory
	if err := p.createBinScript(binDir, slug, result.EntryCommand, result.EntryArgs, result.Environment); err != nil {
		result.Error = fmt.Sprintf("Failed to create bin script: %v", err)
		logf(p.logger, result.Error)
		return result, nil
	}

	// Get list of all installed packages
	if installedPackages, err := p.listInstalledPackages(ctx, result.PipPath); err == nil {
		result.InstalledPackages = installedPackages
	}

	// Copy package metadata to install directory
	if err := p.copyPackageMetadata(options.Package, result.VenvPath, installDir, pythonExec); err != nil {
		logf(p.logger, "Warning: Failed to copy package metadata: %v", err)
	}

	result.Success = true
	logf(p.logger, "Pip installation completed successfully for %s", slug)
	return result, nil
}

// detectPythonExecutable finds and validates the Python executable
func (p *PipInstaller) detectPythonExecutable(ctx context.Context, requiredVersion string) (string, error) {
	pythonCandidates := []string{"python3", "python", "python3.12", "python3.11", "python3.10", "python3.9"}

	for _, candidate := range pythonCandidates {
		if stdout, _, err := p.runner.Run(ctx, candidate, "--version"); err == nil {
			version := strings.TrimSpace(stdout)
			logf(p.logger, "Found Python: %s - %s", candidate, version)

			if requiredVersion != "" {
				if !strings.Contains(version, requiredVersion) {
					logf(p.logger, "Python version %s does not meet requirement %s", version, requiredVersion)
					continue
				}
			}

			return candidate, nil
		}
	}

	return "", fmt.Errorf("no suitable Python executable found")
}

// detectPipExecutable finds the pip executable for the given Python
func (p *PipInstaller) detectPipExecutable(ctx context.Context, pythonExec string) (string, error) {
	// Try pip module first
	if _, _, err := p.runner.Run(ctx, pythonExec, "-m", "pip", "--version"); err == nil {
		return fmt.Sprintf("%s -m pip", pythonExec), nil
	}

	// Try standalone pip executables
	pipCandidates := []string{"pip3", "pip"}
	for _, candidate := range pipCandidates {
		if _, _, err := p.runner.Run(ctx, candidate, "--version"); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no suitable pip executable found")
}

// createVirtualEnvironment creates a Python virtual environment
func (p *PipInstaller) createVirtualEnvironment(ctx context.Context, runtimeDir, pythonExec string) (string, error) {
	venvPath := filepath.Join(runtimeDir, "venv")

	logf(p.logger, "Creating virtual environment at %s", venvPath)

	if _, err := os.Stat(venvPath); err == nil {
		if err := os.RemoveAll(venvPath); err != nil {
			return "", fmt.Errorf("failed to remove existing venv: %w", err)
		}
	}

	if _, _, err := p.runner.Run(ctx, pythonExec, "-m", "venv", venvPath); err != nil {
		return "", fmt.Errorf("failed to create virtual environment: %w", err)
	}

	pythonPath, _, err := p.getVenvExecutables(venvPath)
	if err != nil {
		return "", fmt.Errorf("failed to get venv executables: %w", err)
	}

	logf(p.logger, "Upgrading pip in virtual environment...")
	if _, _, err := p.runner.Run(ctx, pythonPath, "-m", "pip", "install", "--upgrade", "pip"); err != nil {
		logf(p.logger, "Warning: Failed to upgrade pip: %v", err)
	}

	return venvPath, nil
}

// getVenvExecutables returns the Python and pip executable paths for a virtual environment
func (p *PipInstaller) getVenvExecutables(venvPath string) (pythonPath, pipPath string, err error) {
	// Check for Unix-style venv structure
	pythonPath = filepath.Join(venvPath, "bin", "python")
	pipPath = filepath.Join(venvPath, "bin", "pip")

	if _, err := os.Stat(pythonPath); err == nil {
		return pythonPath, pipPath, nil
	}

	// Check for Windows-style venv structure
	pythonPath = filepath.Join(venvPath, "Scripts", "python.exe")
	pipPath = filepath.Join(venvPath, "Scripts", "pip.exe")

	if _, err := os.Stat(pythonPath); err == nil {
		return pythonPath, pipPath, nil
	}

	return "", "", fmt.Errorf("could not find Python executable in virtual environment")
}

// installWithPipx performs installation using pipx for isolation
func (p *PipInstaller) installWithPipx(ctx context.Context, slug string, options PipInstallOptions, result *PipInstallResult) (*PipInstallResult, error) {
	logf(p.logger, "Installing with pipx...")

	// Check if pipx is available
	if _, _, err := p.runner.Run(ctx, "pipx", "--version"); err != nil {
		result.Error = "pipx not found. Please install pipx first: pip install pipx"
		return result, nil
	}

	packageSpec := options.Package
	if options.Version != "" {
		packageSpec = fmt.Sprintf("%s==%s", options.Package, options.Version)
	}

	args := []string{"install", packageSpec}

	if options.ForceReinstall {
		args = append(args, "--force")
	}

	if options.PreRelease {
		args = append(args, "--pip-args", "--pre")
	}

	if _, _, err := p.runner.Run(ctx, "pipx", args...); err != nil {
		result.Error = fmt.Sprintf("pipx installation failed: %v", err)
		return result, nil
	}

	// Get pipx venv location
	stdout, _, err := p.runner.Run(ctx, "pipx", "list", "--short")
	if err == nil {
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			if strings.Contains(line, options.Package) {
				// Extract venv path - this is simplified
				result.VenvPath = fmt.Sprintf("%s/.local/share/pipx/venvs/%s", os.Getenv("HOME"), options.Package)
				break
			}
		}
	}

	// Find the installed command
	if stdout, _, err := p.runner.Run(ctx, "which", options.Package); err == nil {
		result.EntryCommand = strings.TrimSpace(stdout)
	}

	result.Success = true
	return result, nil
}

// validatePackage checks if the package exists and is accessible
func (p *PipInstaller) validatePackage(ctx context.Context, options PipInstallOptions, pipPath string) error {
	logf(p.logger, "Validating package availability...")

	packageSpec := options.Package
	if options.Version != "" {
		packageSpec = fmt.Sprintf("%s==%s", options.Package, options.Version)
	}

	args := []string{"install", "--dry-run", "--quiet", packageSpec}

	if options.IndexURL != "" {
		args = append(args, "--index-url", options.IndexURL)
	}
	if options.ExtraIndexURL != "" {
		args = append(args, "--extra-index-url", options.ExtraIndexURL)
	}
	if options.TrustedHost != "" {
		args = append(args, "--trusted-host", options.TrustedHost)
	}
	if options.PreRelease {
		args = append(args, "--pre")
	}

	// Execute pip install dry-run
	var cmd []string
	if strings.Contains(pipPath, "-m pip") {
		parts := strings.Fields(pipPath)
		cmd = append(parts, args...)
	} else {
		cmd = append([]string{pipPath}, args...)
	}

	_, _, err := p.runner.Run(ctx, cmd[0], cmd[1:]...)
	if err != nil {
		return fmt.Errorf("package validation failed: %w", err)
	}

	logf(p.logger, "Package validated successfully")
	return nil
}

// installPackage performs the actual package installation
func (p *PipInstaller) installPackage(ctx context.Context, options PipInstallOptions, pipPath, pythonExec string) error {
	logf(p.logger, "Installing package with pip...")

	var args []string

	if options.RequirementsFile != "" {
		args = []string{"install", "-r", options.RequirementsFile}
	} else {
		packageSpec := options.Package
		if options.Version != "" {
			packageSpec = fmt.Sprintf("%s==%s", options.Package, options.Version)
		}

		if len(options.Extras) > 0 {
			packageSpec = fmt.Sprintf("%s[%s]", packageSpec, strings.Join(options.Extras, ","))
		}

		args = []string{"install", packageSpec}
	}

	if options.ForceReinstall {
		args = append(args, "--force-reinstall")
	}
	if options.NoDeps {
		args = append(args, "--no-deps")
	}
	if options.PreRelease {
		args = append(args, "--pre")
	}
	if options.IndexURL != "" {
		args = append(args, "--index-url", options.IndexURL)
	}
	if options.ExtraIndexURL != "" {
		args = append(args, "--extra-index-url", options.ExtraIndexURL)
	}
	if options.TrustedHost != "" {
		args = append(args, "--trusted-host", options.TrustedHost)
	}

	var cmd *exec.Cmd
	if strings.Contains(pipPath, "-m pip") {
		parts := strings.Fields(pipPath)
		allArgs := append(parts[1:], args...)
		cmd = exec.CommandContext(ctx, parts[0], allArgs...)
	} else {
		cmd = exec.CommandContext(ctx, pipPath, args...)
	}

	env := os.Environ()
	for k, v := range options.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	stdout, stderr, err := p.runner.Run(ctx, cmd.Path, cmd.Args[1:]...)
	if err != nil {
		return fmt.Errorf("installation failed: %w, stdout: %s, stderr: %s", err, stdout, stderr)
	}

	logf(p.logger, "Package installed successfully")
	return nil
}

// getPackageInfo retrieves information about the installed package
func (p *PipInstaller) getPackageInfo(ctx context.Context, packageName, pipPath, pythonExec string) (*PipPackageInfo, string, error) {
	// Use pip show to get package information
	var cmd []string
	if strings.Contains(pipPath, "-m pip") {
		parts := strings.Fields(pipPath)
		cmd = append(parts, "show", packageName)
	} else {
		cmd = []string{pipPath, "show", packageName}
	}

	stdout, _, err := p.runner.Run(ctx, cmd[0], cmd[1:]...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get package info: %w", err)
	}

	// Parse pip show output
	info := &PipPackageInfo{}
	scanner := bufio.NewScanner(strings.NewReader(stdout))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Name":
			info.Name = value
		case "Version":
			info.Version = value
		case "Summary":
			info.Summary = value
		case "Author":
			info.Author = value
		case "Author-email":
			info.AuthorEmail = value
		case "License":
			info.License = value
		case "Home-page":
			info.Homepage = value
		case "Location":
			info.Location = value
		case "Requires":
			if value != "" {
				info.Requires = strings.Split(value, ", ")
			}
		case "Required-by":
			if value != "" {
				info.RequiredBy = strings.Split(value, ", ")
			}
		}
	}

	// Try to get entry points information
	if entryPoints, err := p.getEntryPoints(ctx, packageName, pythonExec); err == nil {
		info.EntryPoints = entryPoints
	}

	return info, info.Version, nil
}

// getEntryPoints retrieves entry points for the package
func (p *PipInstaller) getEntryPoints(ctx context.Context, packageName, pythonExec string) (map[string][]string, error) {
	// Use Python to get entry points
	script := fmt.Sprintf(`
import pkg_resources
import json
try:
    dist = pkg_resources.get_distribution("%s")
    entry_points = {}
    for ep in dist.get_entry_map():
        entry_points[ep] = []
        for name, ep_obj in dist.get_entry_map()[ep].items():
            entry_points[ep].append(f"{name} = {ep_obj}")
    print(json.dumps(entry_points))
except Exception as e:
    print("{}")
`, packageName)

	stdout, _, err := p.runner.Run(ctx, pythonExec, "-c", script)
	if err != nil {
		return nil, err
	}

	var entryPoints map[string][]string
	if err := json.Unmarshal([]byte(stdout), &entryPoints); err != nil {
		return nil, err
	}

	return entryPoints, nil
}

// detectConsoleScripts detects console scripts provided by the package
func (p *PipInstaller) detectConsoleScripts(ctx context.Context, packageName, venvPath, pythonExec string) ([]string, error) {
	var scripts []string

	// Get entry points and filter for console_scripts
	entryPoints, err := p.getEntryPoints(ctx, packageName, pythonExec)
	if err != nil {
		return scripts, err
	}

	if consoleScripts, exists := entryPoints["console_scripts"]; exists {
		for _, script := range consoleScripts {
			// Extract script name from "name = module:function" format
			parts := strings.Split(script, " = ")
			if len(parts) > 0 {
				scripts = append(scripts, strings.TrimSpace(parts[0]))
			}
		}
	}

	// Also check Scripts directory in venv
	if venvPath != "" {
		scriptsDir := filepath.Join(venvPath, "Scripts") // Windows
		if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
			scriptsDir = filepath.Join(venvPath, "bin") // Unix
		}

		if entries, err := os.ReadDir(scriptsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && !strings.HasSuffix(entry.Name(), ".exe") {
					// Check if it's a Python script by reading the shebang
					scriptPath := filepath.Join(scriptsDir, entry.Name())
					if p.isPythonScript(scriptPath) {
						scripts = append(scripts, entry.Name())
					}
				}
			}
		}
	}

	return scripts, nil
}

// isPythonScript checks if a file is a Python script
func (p *PipInstaller) isPythonScript(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		firstLine := scanner.Text()
		return strings.Contains(firstLine, "python")
	}

	return false
}

// determineEntryPoint determines the command and arguments to run the MCP server
func (p *PipInstaller) determineEntryPoint(ctx context.Context, options PipInstallOptions, packageInfo *PipPackageInfo, venvPath, pythonExec string) (command string, args []string, env map[string]string, err error) {
	env = make(map[string]string)

	// MCP-specific configuration takes priority
	if options.MCPConfig != nil {
		if options.MCPConfig.EntryCommand != "" {
			return options.MCPConfig.EntryCommand, options.MCPConfig.Args, env, nil
		}
		if options.MCPConfig.EntryModule != "" {
			return pythonExec, []string{"-m", options.MCPConfig.EntryModule}, env, nil
		}
		if options.MCPConfig.EntryScript != "" {
			scriptPath := filepath.Join(venvPath, options.MCPConfig.EntryScript)
			if _, err := os.Stat(scriptPath); err == nil {
				return pythonExec, []string{scriptPath}, env, nil
			}
		}
	}

	// Check for console scripts
	if packageInfo != nil && packageInfo.EntryPoints != nil {
		if consoleScripts, exists := packageInfo.EntryPoints["console_scripts"]; exists && len(consoleScripts) > 0 {
			// Use the first console script
			scriptSpec := consoleScripts[0]
			scriptName := strings.Split(scriptSpec, " = ")[0]

			// Check if the script exists in the venv
			if venvPath != "" {
				scriptPath := filepath.Join(venvPath, "bin", scriptName)
				if _, err := os.Stat(scriptPath); err == nil {
					return scriptPath, []string{}, env, nil
				}
				// Try Windows Scripts directory
				scriptPath = filepath.Join(venvPath, "Scripts", scriptName+".exe")
				if _, err := os.Stat(scriptPath); err == nil {
					return scriptPath, []string{}, env, nil
				}
			}

			// Try running as module
			parts := strings.Split(scriptSpec, " = ")
			if len(parts) > 1 {
				moduleParts := strings.Split(parts[1], ":")
				if len(moduleParts) > 0 {
					return pythonExec, []string{"-m", moduleParts[0]}, env, nil
				}
			}
		}
	}

	// Try common module names
	commonModules := []string{
		options.Package,
		strings.ReplaceAll(options.Package, "-", "_"),
		strings.ReplaceAll(options.Package, "-", "_") + ".main",
		strings.ReplaceAll(options.Package, "-", "_") + ".server",
	}

	for _, module := range commonModules {
		if p.canImportModule(ctx, pythonExec, module) {
			return pythonExec, []string{"-m", module}, env, nil
		}
	}

	return "", nil, env, fmt.Errorf("no entry point found for package %s", options.Package)
}

// canImportModule checks if a Python module can be imported
func (p *PipInstaller) canImportModule(ctx context.Context, pythonExec, module string) bool {
	script := fmt.Sprintf("import %s", module)
	_, _, err := p.runner.Run(ctx, pythonExec, "-c", script)
	return err == nil
}

// runPostInstallCommands executes user-defined post-install commands
func (p *PipInstaller) runPostInstallCommands(ctx context.Context, venvPath, pythonExec string, options PipInstallOptions) error {
	logf(p.logger, "Running post-install commands...")

	for i, cmdStr := range options.PostInstall {
		logf(p.logger, "Running post-install command %d: %s", i+1, cmdStr)

		// Parse command and arguments
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			continue
		}

		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

		// Add environment variables
		env := os.Environ()
		for k, v := range options.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}

		// Add venv to PATH if available
		if venvPath != "" {
			binPath := filepath.Join(venvPath, "bin")
			if _, err := os.Stat(binPath); os.IsNotExist(err) {
				binPath = filepath.Join(venvPath, "Scripts") // Windows
			}
			env = append(env, fmt.Sprintf("PATH=%s:%s", binPath, os.Getenv("PATH")))
			env = append(env, fmt.Sprintf("VIRTUAL_ENV=%s", venvPath))
		}

		cmd.Env = env

		stdout, stderr, err := p.runner.Run(ctx, cmd.Path, cmd.Args[1:]...)
		if err != nil {
			return fmt.Errorf("post-install command failed: %s: %w, stdout: %s, stderr: %s", cmdStr, err, stdout, stderr)
		}
	}

	return nil
}

// createBinScript creates an executable script in the bin directory
func (p *PipInstaller) createBinScript(binDir, slug, command string, args []string, env map[string]string) error {
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

// listInstalledPackages returns a list of all packages installed in the environment
func (p *PipInstaller) listInstalledPackages(ctx context.Context, pipPath string) ([]string, error) {
	var cmd []string
	if strings.Contains(pipPath, "-m pip") {
		parts := strings.Fields(pipPath)
		cmd = append(parts, "list", "--format=freeze")
	} else {
		cmd = []string{pipPath, "list", "--format=freeze"}
	}

	stdout, _, err := p.runner.Run(ctx, cmd[0], cmd[1:]...)
	if err != nil {
		return nil, err
	}

	var packages []string
	scanner := bufio.NewScanner(strings.NewReader(stdout))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			// Extract package name from "package==version" format
			parts := strings.Split(line, "==")
			if len(parts) > 0 {
				packages = append(packages, parts[0])
			}
		}
	}

	return packages, nil
}

// copyPackageMetadata copies package metadata to the install directory
func (p *PipInstaller) copyPackageMetadata(packageName, venvPath, installDir, pythonExec string) error {
	if venvPath == "" {
		return nil // Skip if no venv
	}

	// Find the package installation directory
	script := fmt.Sprintf(`
import pkg_resources
try:
    dist = pkg_resources.get_distribution("%s")
    print(dist.location)
except Exception as e:
    pass
`, packageName)

	stdout, _, err := p.runner.Run(context.Background(), pythonExec, "-c", script)
	if err != nil {
		return err
	}

	packageLocation := strings.TrimSpace(stdout)
	if packageLocation == "" {
		return fmt.Errorf("package location not found")
	}

	// Look for common metadata files
	metadataFiles := []string{"PKG-INFO", "METADATA", "README.md", "LICENSE", "CHANGELOG.md"}

	// Check in site-packages/package-info directory
	infoDir := filepath.Join(packageLocation, fmt.Sprintf("%s.dist-info", strings.ReplaceAll(packageName, "-", "_")))
	if _, err := os.Stat(infoDir); os.IsNotExist(err) {
		// Try alternative dist-info naming
		infoDir = filepath.Join(packageLocation, fmt.Sprintf("%s.dist-info", packageName))
	}
	for _, file := range metadataFiles {
		srcPath := filepath.Join(infoDir, file)
		if _, err := os.Stat(srcPath); err == nil {
			dstPath := filepath.Join(installDir, file)
			if err := copyFile(srcPath, dstPath); err == nil {
				// Successfully copied, continue with next file
				continue
			}
		}
	}

	return nil
}
