package install

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"mcp/manager/internal/paths"
)

// GitInstaller handles git-based MCP server installations
type GitInstaller struct {
	runner Runner
	logger Logger
}

// NewGitInstaller creates a new git installer instance
func NewGitInstaller(runner Runner, logger Logger) *GitInstaller {
	if runner == nil {
		runner = ExecRunner{}
	}
	return &GitInstaller{
		runner: runner,
		logger: logger,
	}
}

// GitInstallOptions contains configuration for git-based installations
type GitInstallOptions struct {
	URI           string            `json:"uri"`
	Branch        string            `json:"branch,omitempty"`        // specific branch to clone
	Tag           string            `json:"tag,omitempty"`           // specific tag to clone
	Commit        string            `json:"commit,omitempty"`        // specific commit to checkout
	Depth         int               `json:"depth,omitempty"`         // clone depth, default 1 for shallow clone
	Recursive     bool              `json:"recursive,omitempty"`     // include submodules
	SSHKey        string            `json:"sshKey,omitempty"`        // path to SSH private key
	Token         string            `json:"token,omitempty"`         // GitHub/GitLab token for auth
	Username      string            `json:"username,omitempty"`      // username for basic auth
	Password      string            `json:"password,omitempty"`      // password for basic auth
	PostInstall   []string          `json:"postInstall,omitempty"`   // commands to run after clone
	Environment   map[string]string `json:"environment,omitempty"`   // environment variables for commands
	SkipDepsCheck bool              `json:"skipDepsCheck,omitempty"` // skip dependency detection and installation
}

// GitInstallResult contains the result of a git installation
type GitInstallResult struct {
	Success       bool              `json:"success"`
	InstallPath   string            `json:"installPath"`
	RuntimePath   string            `json:"runtimePath"`
	BinPath       string            `json:"binPath"`
	DetectedRuntime string          `json:"detectedRuntime"`
	DetectedManager string          `json:"detectedManager"`
	EntryCommand  string            `json:"entryCommand"`
	EntryArgs     []string          `json:"entryArgs"`
	Environment   map[string]string `json:"environment"`
	Logs          []string          `json:"logs"`
	Error         string            `json:"error,omitempty"`
}

// Install performs a git-based installation of an MCP server
func (g *GitInstaller) Install(ctx context.Context, slug string, options GitInstallOptions) (*GitInstallResult, error) {
	result := &GitInstallResult{
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

	logf(g.logger, "Starting git installation for %s", slug)
	logf(g.logger, "Repository: %s", options.URI)

	// Validate git repository accessibility
	if err := g.validateRepository(ctx, options); err != nil {
		result.Error = fmt.Sprintf("Repository validation failed: %v", err)
		logf(g.logger, result.Error)
		return result, nil
	}

	// Clone repository
	if err := g.cloneRepository(ctx, options, installDir); err != nil {
		result.Error = fmt.Sprintf("Repository clone failed: %v", err)
		logf(g.logger, result.Error)
		return result, nil
	}

	// Detect runtime and dependencies
	if !options.SkipDepsCheck {
		runtime, manager, err := g.detectRuntime(installDir)
		if err != nil {
			logf(g.logger, "Warning: Runtime detection failed: %v", err)
		} else {
			result.DetectedRuntime = runtime
			result.DetectedManager = manager
			logf(g.logger, "Detected runtime: %s with manager: %s", runtime, manager)
		}

		// Install dependencies based on detected runtime
		if err := g.installDependencies(ctx, installDir, runtimeDir, runtime, manager, options); err != nil {
			result.Error = fmt.Sprintf("Dependency installation failed: %v", err)
			logf(g.logger, result.Error)
			return result, nil
		}
	}

	// Run post-install commands if specified
	if len(options.PostInstall) > 0 {
		if err := g.runPostInstallCommands(ctx, installDir, options); err != nil {
			result.Error = fmt.Sprintf("Post-install commands failed: %v", err)
			logf(g.logger, result.Error)
			return result, nil
		}
	}

	// Detect entry point
	entryCmd, entryArgs, env, err := g.detectEntryPoint(installDir, result.DetectedRuntime)
	if err != nil {
		logf(g.logger, "Warning: Entry point detection failed: %v", err)
	} else {
		result.EntryCommand = entryCmd
		result.EntryArgs = entryArgs
		for k, v := range env {
			result.Environment[k] = v
		}
	}

	// Create executable script in bin directory
	if err := g.createBinScript(binDir, slug, result.EntryCommand, result.EntryArgs, result.Environment); err != nil {
		result.Error = fmt.Sprintf("Failed to create bin script: %v", err)
		logf(g.logger, result.Error)
		return result, nil
	}

	result.Success = true
	logf(g.logger, "Git installation completed successfully for %s", slug)
	return result, nil
}

// validateRepository checks if the git repository is accessible
func (g *GitInstaller) validateRepository(ctx context.Context, options GitInstallOptions) error {
	logf(g.logger, "Validating repository access...")
	
	args := []string{"ls-remote", "--heads"}
	
	// Add authentication if provided
	uri := options.URI
	if options.Token != "" {
		uri = g.addTokenToURI(uri, options.Token)
	} else if options.Username != "" && options.Password != "" {
		uri = g.addBasicAuthToURI(uri, options.Username, options.Password)
	}
	
	args = append(args, uri)
	
	// Set up environment for SSH key if provided
	env := os.Environ()
	if options.SSHKey != "" {
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no", options.SSHKey))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = env
	
	if _, _, err := g.runCommand(ctx, cmd); err != nil {
		return fmt.Errorf("repository not accessible: %w", err)
	}
	
	return nil
}

// cloneRepository clones the git repository to the install directory
func (g *GitInstaller) cloneRepository(ctx context.Context, options GitInstallOptions, installDir string) error {
	logf(g.logger, "Cloning repository...")
	
	args := []string{"clone"}
	
	// Add depth for shallow clone
	depth := options.Depth
	if depth == 0 {
		depth = 1 // Default shallow clone
	}
	args = append(args, "--depth", fmt.Sprintf("%d", depth))
	
	// Add recursive flag for submodules
	if options.Recursive {
		args = append(args, "--recursive")
	}
	
	// Add branch or tag specification
	if options.Branch != "" {
		args = append(args, "--branch", options.Branch)
	} else if options.Tag != "" {
		args = append(args, "--branch", options.Tag)
	}
	
	// Prepare URI with authentication
	uri := options.URI
	if options.Token != "" {
		uri = g.addTokenToURI(uri, options.Token)
	} else if options.Username != "" && options.Password != "" {
		uri = g.addBasicAuthToURI(uri, options.Username, options.Password)
	}
	
	args = append(args, uri, installDir)
	
	// Set up environment
	env := os.Environ()
	if options.SSHKey != "" {
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no", options.SSHKey))
	}
	
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = env
	
	if _, _, err := g.runCommand(ctx, cmd); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	
	// If specific commit is requested, checkout that commit
	if options.Commit != "" {
		logf(g.logger, "Checking out specific commit: %s", options.Commit)
		cmd := exec.CommandContext(ctx, "git", "-C", installDir, "checkout", options.Commit)
		if _, _, err := g.runCommand(ctx, cmd); err != nil {
			return fmt.Errorf("git checkout failed: %w", err)
		}
	}
	
	return nil
}

// detectRuntime analyzes the repository to determine the runtime environment
func (g *GitInstaller) detectRuntime(installDir string) (runtime, manager string, err error) {
	// Check for Node.js
	if _, err := os.Stat(filepath.Join(installDir, "package.json")); err == nil {
		packageManager := "npm" // default
		
		// Check for preferred package manager
		if _, err := os.Stat(filepath.Join(installDir, "package-lock.json")); err == nil {
			packageManager = "npm"
		} else if _, err := os.Stat(filepath.Join(installDir, "yarn.lock")); err == nil {
			packageManager = "yarn"
		} else if _, err := os.Stat(filepath.Join(installDir, "pnpm-lock.yaml")); err == nil {
			packageManager = "pnpm"
		}
		
		return "node", packageManager, nil
	}
	
	// Check for Python
	pyFiles := []string{"requirements.txt", "setup.py", "pyproject.toml", "Pipfile"}
	for _, file := range pyFiles {
		if _, err := os.Stat(filepath.Join(installDir, file)); err == nil {
			manager := "pip" // default
			
			// Check for specific Python package managers
			if file == "Pipfile" {
				manager = "pipenv"
			} else if file == "pyproject.toml" {
				// Check if it's a poetry project
				if content, err := os.ReadFile(filepath.Join(installDir, file)); err == nil {
					if strings.Contains(string(content), "[tool.poetry]") {
						manager = "poetry"
					}
				}
			}
			
			return "python", manager, nil
		}
	}
	
	// Check for Go
	if _, err := os.Stat(filepath.Join(installDir, "go.mod")); err == nil {
		return "go", "go", nil
	}
	
	// Check for Rust
	if _, err := os.Stat(filepath.Join(installDir, "Cargo.toml")); err == nil {
		return "rust", "cargo", nil
	}
	
	// Check for Docker
	if _, err := os.Stat(filepath.Join(installDir, "Dockerfile")); err == nil {
		return "docker", "docker", nil
	}
	
	return "binary", "", nil
}

// installDependencies installs dependencies based on the detected runtime
func (g *GitInstaller) installDependencies(ctx context.Context, installDir, runtimeDir, runtime, manager string, options GitInstallOptions) error {
	switch runtime {
	case "node":
		return g.installNodeDependencies(ctx, installDir, runtimeDir, manager, options)
	case "python":
		return g.installPythonDependencies(ctx, installDir, runtimeDir, manager, options)
	case "go":
		return g.installGoDependencies(ctx, installDir, options)
	case "rust":
		return g.installRustDependencies(ctx, installDir, options)
	default:
		logf(g.logger, "No dependency installation needed for runtime: %s", runtime)
		return nil
	}
}

// installNodeDependencies installs Node.js dependencies
func (g *GitInstaller) installNodeDependencies(ctx context.Context, installDir, runtimeDir, manager string, options GitInstallOptions) error {
	logf(g.logger, "Installing Node.js dependencies with %s...", manager)
	
	var cmd *exec.Cmd
	switch manager {
	case "npm":
		cmd = exec.CommandContext(ctx, "npm", "install", "--prefix", runtimeDir)
		cmd.Dir = installDir
	case "yarn":
		// Copy package.json to runtime dir for yarn
		if err := g.copyFile(filepath.Join(installDir, "package.json"), filepath.Join(runtimeDir, "package.json")); err != nil {
			return fmt.Errorf("failed to copy package.json: %w", err)
		}
		cmd = exec.CommandContext(ctx, "yarn", "install")
		cmd.Dir = runtimeDir
	case "pnpm":
		cmd = exec.CommandContext(ctx, "pnpm", "install", "--prefix", runtimeDir)
		cmd.Dir = installDir
	default:
		return fmt.Errorf("unsupported Node.js package manager: %s", manager)
	}
	
	// Add environment variables
	env := os.Environ()
	for k, v := range options.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
	
	if _, _, err := g.runCommand(ctx, cmd); err != nil {
		return fmt.Errorf("failed to install Node.js dependencies: %w", err)
	}
	
	return nil
}

// installPythonDependencies installs Python dependencies
func (g *GitInstaller) installPythonDependencies(ctx context.Context, installDir, runtimeDir, manager string, options GitInstallOptions) error {
	logf(g.logger, "Installing Python dependencies with %s...", manager)
	
	// Create virtual environment
	venvDir := filepath.Join(runtimeDir, "venv")
	cmd := exec.CommandContext(ctx, "python3", "-m", "venv", venvDir)
	if _, _, err := g.runCommand(ctx, cmd); err != nil {
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}
	
	// Determine pip executable path
	pipExec := filepath.Join(venvDir, "bin", "pip")
	if _, err := os.Stat(pipExec); err != nil {
		pipExec = filepath.Join(venvDir, "Scripts", "pip.exe") // Windows
	}
	
	var installCmd *exec.Cmd
	switch manager {
	case "pip":
		// Install from requirements.txt if it exists
		reqFile := filepath.Join(installDir, "requirements.txt")
		if _, err := os.Stat(reqFile); err == nil {
			installCmd = exec.CommandContext(ctx, pipExec, "install", "-r", reqFile)
		} else {
			// Install the package itself
			installCmd = exec.CommandContext(ctx, pipExec, "install", ".")
			installCmd.Dir = installDir
		}
	case "pipenv":
		// Install pipenv and then use it
		cmd := exec.CommandContext(ctx, pipExec, "install", "pipenv")
		if _, _, err := g.runCommand(ctx, cmd); err != nil {
			return fmt.Errorf("failed to install pipenv: %w", err)
		}
		installCmd = exec.CommandContext(ctx, filepath.Join(venvDir, "bin", "pipenv"), "install")
		installCmd.Dir = installDir
	case "poetry":
		// Install poetry and then use it
		cmd := exec.CommandContext(ctx, pipExec, "install", "poetry")
		if _, _, err := g.runCommand(ctx, cmd); err != nil {
			return fmt.Errorf("failed to install poetry: %w", err)
		}
		installCmd = exec.CommandContext(ctx, filepath.Join(venvDir, "bin", "poetry"), "install")
		installCmd.Dir = installDir
	default:
		return fmt.Errorf("unsupported Python package manager: %s", manager)
	}
	
	// Add environment variables
	env := os.Environ()
	for k, v := range options.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	installCmd.Env = env
	
	if _, _, err := g.runCommand(ctx, installCmd); err != nil {
		return fmt.Errorf("failed to install Python dependencies: %w", err)
	}
	
	return nil
}

// installGoDependencies installs Go dependencies
func (g *GitInstaller) installGoDependencies(ctx context.Context, installDir string, options GitInstallOptions) error {
	logf(g.logger, "Installing Go dependencies...")
	
	cmd := exec.CommandContext(ctx, "go", "mod", "download")
	cmd.Dir = installDir
	
	// Add environment variables
	env := os.Environ()
	for k, v := range options.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
	
	if _, _, err := g.runCommand(ctx, cmd); err != nil {
		return fmt.Errorf("failed to download Go dependencies: %w", err)
	}
	
	// Build the Go project
	cmd = exec.CommandContext(ctx, "go", "build", "-o", "server")
	cmd.Dir = installDir
	cmd.Env = env
	
	if _, _, err := g.runCommand(ctx, cmd); err != nil {
		return fmt.Errorf("failed to build Go project: %w", err)
	}
	
	return nil
}

// installRustDependencies installs Rust dependencies
func (g *GitInstaller) installRustDependencies(ctx context.Context, installDir string, options GitInstallOptions) error {
	logf(g.logger, "Installing Rust dependencies...")
	
	cmd := exec.CommandContext(ctx, "cargo", "build", "--release")
	cmd.Dir = installDir
	
	// Add environment variables
	env := os.Environ()
	for k, v := range options.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
	
	if _, _, err := g.runCommand(ctx, cmd); err != nil {
		return fmt.Errorf("failed to build Rust project: %w", err)
	}
	
	return nil
}

// runPostInstallCommands executes user-defined post-install commands
func (g *GitInstaller) runPostInstallCommands(ctx context.Context, installDir string, options GitInstallOptions) error {
	logf(g.logger, "Running post-install commands...")
	
	for i, cmdStr := range options.PostInstall {
		logf(g.logger, "Running post-install command %d: %s", i+1, cmdStr)
		
		// Parse command and arguments
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			continue
		}
		
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		cmd.Dir = installDir
		
		// Add environment variables
		env := os.Environ()
		for k, v := range options.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
		
		if _, _, err := g.runCommand(ctx, cmd); err != nil {
			return fmt.Errorf("post-install command failed: %s: %w", cmdStr, err)
		}
	}
	
	return nil
}

// detectEntryPoint tries to detect the main entry point for the MCP server
func (g *GitInstaller) detectEntryPoint(installDir, runtime string) (command string, args []string, env map[string]string, err error) {
	env = make(map[string]string)
	
	switch runtime {
	case "node":
		return g.detectNodeEntryPoint(installDir, env)
	case "python":
		return g.detectPythonEntryPoint(installDir, env)
	case "go":
		return g.detectGoEntryPoint(installDir, env)
	case "rust":
		return g.detectRustEntryPoint(installDir, env)
	case "binary":
		return g.detectBinaryEntryPoint(installDir, env)
	default:
		return "", nil, env, fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

// detectNodeEntryPoint detects Node.js entry point
func (g *GitInstaller) detectNodeEntryPoint(installDir string, env map[string]string) (string, []string, map[string]string, error) {
	// Check package.json for entry point
	packageFile := filepath.Join(installDir, "package.json")
	if content, err := os.ReadFile(packageFile); err == nil {
		var pkg map[string]interface{}
		if err := json.Unmarshal(content, &pkg); err == nil {
			// Check for main field
			if main, ok := pkg["main"].(string); ok {
				return "node", []string{filepath.Join(installDir, main)}, env, nil
			}
			// Check for bin field
			if bin, ok := pkg["bin"].(map[string]interface{}); ok {
				for name, path := range bin {
					if pathStr, ok := path.(string); ok {
						return "node", []string{filepath.Join(installDir, pathStr)}, env, nil
					}
					_ = name // Use first bin entry
					break
				}
			}
		}
	}
	
	// Fallback to common entry points
	commonEntries := []string{"index.js", "main.js", "server.js", "src/index.js", "src/main.js"}
	for _, entry := range commonEntries {
		if _, err := os.Stat(filepath.Join(installDir, entry)); err == nil {
			return "node", []string{filepath.Join(installDir, entry)}, env, nil
		}
	}
	
	return "", nil, env, fmt.Errorf("no Node.js entry point found")
}

// detectPythonEntryPoint detects Python entry point
func (g *GitInstaller) detectPythonEntryPoint(installDir string, env map[string]string) (string, []string, map[string]string, error) {
	// Check setup.py or pyproject.toml for entry points
	setupFile := filepath.Join(installDir, "setup.py")
	if content, err := os.ReadFile(setupFile); err == nil {
		// Simple regex to find entry points (this could be more sophisticated)
		re := regexp.MustCompile(`entry_points\s*=\s*{[^}]*['"]console_scripts['"][^}]*}`)
		if re.Match(content) {
			// For now, assume main.py or __main__.py
			commonEntries := []string{"main.py", "__main__.py", "src/main.py"}
			for _, entry := range commonEntries {
				if _, err := os.Stat(filepath.Join(installDir, entry)); err == nil {
					return "python3", []string{filepath.Join(installDir, entry)}, env, nil
				}
			}
		}
	}
	
	// Check for common Python entry points
	commonEntries := []string{"main.py", "__main__.py", "server.py", "app.py", "src/main.py"}
	for _, entry := range commonEntries {
		if _, err := os.Stat(filepath.Join(installDir, entry)); err == nil {
			return "python3", []string{filepath.Join(installDir, entry)}, env, nil
		}
	}
	
	// Check if it's a package with __main__.py
	if _, err := os.Stat(filepath.Join(installDir, "__main__.py")); err == nil {
		return "python3", []string{"-m", filepath.Base(installDir)}, env, nil
	}
	
	return "", nil, env, fmt.Errorf("no Python entry point found")
}

// detectGoEntryPoint detects Go entry point
func (g *GitInstaller) detectGoEntryPoint(installDir string, env map[string]string) (string, []string, map[string]string, error) {
	// Look for built binary
	binaryPath := filepath.Join(installDir, "server")
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, []string{}, env, nil
	}
	
	// Look for main.go and try to infer binary name
	if _, err := os.Stat(filepath.Join(installDir, "main.go")); err == nil {
		// Try to find built binary with different names
		commonNames := []string{"main", "server", "app", filepath.Base(installDir)}
		for _, name := range commonNames {
			binaryPath := filepath.Join(installDir, name)
			if _, err := os.Stat(binaryPath); err == nil {
				return binaryPath, []string{}, env, nil
			}
		}
		
		// If no binary found, we might need to build it
		return "", nil, env, fmt.Errorf("Go binary not found, may need to build")
	}
	
	return "", nil, env, fmt.Errorf("no Go entry point found")
}

// detectRustEntryPoint detects Rust entry point
func (g *GitInstaller) detectRustEntryPoint(installDir string, env map[string]string) (string, []string, map[string]string, error) {
	// Look in target/release directory
	targetDir := filepath.Join(installDir, "target", "release")
	if entries, err := os.ReadDir(targetDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && isExecutable(filepath.Join(targetDir, entry.Name())) {
				return filepath.Join(targetDir, entry.Name()), []string{}, env, nil
			}
		}
	}
	
	return "", nil, env, fmt.Errorf("no Rust binary found")
}

// detectBinaryEntryPoint detects binary entry point
func (g *GitInstaller) detectBinaryEntryPoint(installDir string, env map[string]string) (string, []string, map[string]string, error) {
	// Look for executable files in the install directory
	entries, err := os.ReadDir(installDir)
	if err != nil {
		return "", nil, env, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			fullPath := filepath.Join(installDir, entry.Name())
			if isExecutable(fullPath) {
				return fullPath, []string{}, env, nil
			}
		}
	}
	
	return "", nil, env, fmt.Errorf("no executable binary found")
}

// createBinScript creates an executable script in the bin directory
func (g *GitInstaller) createBinScript(binDir, slug, command string, args []string, env map[string]string) error {
	scriptPath := filepath.Join(binDir, slug)
	
	// Create a shell script that executes the MCP server
	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("# Generated MCP server launcher\n\n")
	
	// Add environment variables
	for k, v := range env {
		script.WriteString(fmt.Sprintf("export %s=%s\n", k, v))
	}
	
	// Add the command
	if command != "" {
		script.WriteString(fmt.Sprintf("exec %s", command))
		for _, arg := range args {
			script.WriteString(fmt.Sprintf(" %s", arg))
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

// Helper functions

// addTokenToURI adds a GitHub/GitLab token to the repository URI
func (g *GitInstaller) addTokenToURI(uri, token string) string {
	if strings.HasPrefix(uri, "https://github.com/") {
		return strings.Replace(uri, "https://", fmt.Sprintf("https://%s@", token), 1)
	}
	if strings.HasPrefix(uri, "https://gitlab.com/") {
		return strings.Replace(uri, "https://", fmt.Sprintf("https://oauth2:%s@", token), 1)
	}
	return uri
}

// addBasicAuthToURI adds basic authentication to the repository URI
func (g *GitInstaller) addBasicAuthToURI(uri, username, password string) string {
	if strings.HasPrefix(uri, "https://") {
		return strings.Replace(uri, "https://", fmt.Sprintf("https://%s:%s@", username, password), 1)
	}
	return uri
}

// copyFile copies a file from src to dst
func (g *GitInstaller) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	return err
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// runCommand executes a command and logs output
func (g *GitInstaller) runCommand(ctx context.Context, cmd *exec.Cmd) (stdout, stderr string, err error) {
	if g.runner != nil {
		return g.runner.Run(ctx, cmd.Path, cmd.Args[1:]...)
	}
	
	// Fallback to direct execution
	return ExecRunner{}.Run(ctx, cmd.Path, cmd.Args[1:]...)
}