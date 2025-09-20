package install

import (
	"context"
	"fmt"
	"time"
)

// ConcreteGitInstaller implements the Installer interface for Git-based installations
type ConcreteGitInstaller struct {
	gitInstaller *GitInstaller
	options      GitInstallOptions
}

// NewConcreteGitInstaller creates a new concrete git installer
func NewConcreteGitInstaller(options GitInstallOptions) *ConcreteGitInstaller {
	return &ConcreteGitInstaller{
		options: options,
	}
}

// Install implements the Installer interface for Git installations
func (cgi *ConcreteGitInstaller) Install(ctx context.Context, job *InstallationJob) (*InstallationResult, error) {
	logger := NewInstallationJobLogger(job, LogLevelInfo, StageValidation)
	cgi.gitInstaller = NewGitInstaller(ExecRunner{}, logger)
	
	job.UpdateStage(StageValidation, 0)
	logger.SetStage(StageValidation)
	
	// Validation stage
	job.Logf(LogLevelInfo, StageValidation, "Validating git repository: %s", cgi.options.URI)
	job.UpdateStage(StageValidation, 50)
	
	// The GitInstaller will handle the actual installation
	result, err := cgi.gitInstaller.Install(ctx, job.Slug, cgi.options)
	if err != nil {
		return nil, fmt.Errorf("git installation failed: %w", err)
	}
	
	job.UpdateStage(StageCompleted, 100)
	
	// Convert to InstallationResult
	installResult := &InstallationResult{
		Success:          result.Success,
		InstallPath:      result.InstallPath,
		RuntimePath:      result.RuntimePath,
		BinPath:          result.BinPath,
		EntryCommand:     result.EntryCommand,
		EntryArgs:        result.EntryArgs,
		Environment:      result.Environment,
		Runtime:          result.DetectedRuntime,
		PackageManager:   result.DetectedManager,
		InstalledVersion: "git-latest", // Git repos don't have traditional versions
		Metadata: map[string]interface{}{
			"installTime": time.Now(),
			"gitURI":      cgi.options.URI,
			"branch":      cgi.options.Branch,
			"commit":      cgi.options.Commit,
			"hasVenv":     false, // Git installations typically don't use venv directly
		},
	}
	
	return installResult, nil
}

// ConcreteNPMInstaller implements the Installer interface for NPM-based installations
type ConcreteNPMInstaller struct {
	npmInstaller *NPMInstaller
	options      NPMInstallOptions
}

// NewConcreteNPMInstaller creates a new concrete npm installer
func NewConcreteNPMInstaller(options NPMInstallOptions) *ConcreteNPMInstaller {
	return &ConcreteNPMInstaller{
		options: options,
	}
}

// Install implements the Installer interface for NPM installations
func (cni *ConcreteNPMInstaller) Install(ctx context.Context, job *InstallationJob) (*InstallationResult, error) {
	logger := NewInstallationJobLogger(job, LogLevelInfo, StageValidation)
	cni.npmInstaller = NewNPMInstaller(ExecRunner{}, logger)
	
	job.UpdateStage(StageValidation, 0)
	logger.SetStage(StageValidation)
	
	// Validation stage
	job.Logf(LogLevelInfo, StageValidation, "Validating npm package: %s", cni.options.Package)
	job.UpdateStage(StageValidation, 30)
	
	job.UpdateStage(StageDownloading, 0)
	logger.SetStage(StageDownloading)
	job.Logf(LogLevelInfo, StageDownloading, "Downloading package: %s", cni.options.Package)
	job.UpdateStage(StageDownloading, 50)
	
	job.UpdateStage(StageInstalling, 0)
	logger.SetStage(StageInstalling)
	
	// The NPMInstaller will handle the actual installation
	result, err := cni.npmInstaller.Install(ctx, job.Slug, cni.options)
	if err != nil {
		return nil, fmt.Errorf("npm installation failed: %w", err)
	}
	
	job.UpdateStage(StageConfiguring, 0)
	logger.SetStage(StageConfiguring)
	job.Logf(LogLevelInfo, StageConfiguring, "Configuring npm package")
	job.UpdateStage(StageConfiguring, 100)
	
	job.UpdateStage(StageCompleted, 100)
	
	// Convert to InstallationResult
	installResult := &InstallationResult{
		Success:          result.Success,
		InstallPath:      result.InstallPath,
		RuntimePath:      result.RuntimePath,
		BinPath:          result.BinPath,
		EntryCommand:     result.EntryCommand,
		EntryArgs:        result.EntryArgs,
		Environment:      result.Environment,
		Runtime:          "node",
		PackageManager:   result.PackageManager,
		InstalledVersion: result.InstalledVersion,
		Metadata: map[string]interface{}{
			"installTime":    time.Now(),
			"npmPackage":     cni.options.Package,
			"packageManager": result.PackageManager,
			"hasVenv":        false,
			"packageInfo":    result.PackageInfo,
		},
	}
	
	return installResult, nil
}

// ConcretePipInstaller implements the Installer interface for Pip-based installations
type ConcretePipInstaller struct {
	pipInstaller *PipInstaller
	options      PipInstallOptions
}

// NewConcretePipInstaller creates a new concrete pip installer
func NewConcretePipInstaller(options PipInstallOptions) *ConcretePipInstaller {
	return &ConcretePipInstaller{
		options: options,
	}
}

// Install implements the Installer interface for Pip installations
func (cpi *ConcretePipInstaller) Install(ctx context.Context, job *InstallationJob) (*InstallationResult, error) {
	logger := NewInstallationJobLogger(job, LogLevelInfo, StageValidation)
	cpi.pipInstaller = NewPipInstaller(ExecRunner{}, logger)
	
	job.UpdateStage(StageValidation, 0)
	logger.SetStage(StageValidation)
	
	// Validation stage
	job.Logf(LogLevelInfo, StageValidation, "Validating pip package: %s", cpi.options.Package)
	job.UpdateStage(StageValidation, 30)
	
	job.UpdateStage(StageDownloading, 0)
	logger.SetStage(StageDownloading)
	job.Logf(LogLevelInfo, StageDownloading, "Setting up virtual environment")
	job.UpdateStage(StageDownloading, 50)
	
	job.UpdateStage(StageInstalling, 0)
	logger.SetStage(StageInstalling)
	job.Logf(LogLevelInfo, StageInstalling, "Installing package: %s", cpi.options.Package)
	
	// The PipInstaller will handle the actual installation
	result, err := cpi.pipInstaller.Install(ctx, job.Slug, cpi.options)
	if err != nil {
		return nil, fmt.Errorf("pip installation failed: %w", err)
	}
	
	job.UpdateStage(StageConfiguring, 0)
	logger.SetStage(StageConfiguring)
	job.Logf(LogLevelInfo, StageConfiguring, "Configuring python environment")
	job.UpdateStage(StageConfiguring, 100)
	
	job.UpdateStage(StageCompleted, 100)
	
	// Convert to InstallationResult
	installResult := &InstallationResult{
		Success:          result.Success,
		InstallPath:      result.InstallPath,
		RuntimePath:      result.RuntimePath,
		BinPath:          result.BinPath,
		EntryCommand:     result.EntryCommand,
		EntryArgs:        result.EntryArgs,
		Environment:      result.Environment,
		Runtime:          "python",
		PackageManager:   "pip",
		InstalledVersion: result.InstalledVersion,
		Metadata: map[string]interface{}{
			"installTime":    time.Now(),
			"pipPackage":     cpi.options.Package,
			"pythonPath":     result.PythonPath,
			"venvPath":       result.VenvPath,
			"hasVenv":        result.VenvPath != "",
			"packageInfo":    result.PackageInfo,
		},
	}
	
	return installResult, nil
}

// AdvancedInstallationService provides a high-level interface for managing installations
type AdvancedInstallationService struct {
	jobManager         *JobManager
	registryIntegrator *RegistryIntegrator
}

// NewAdvancedInstallationService creates a new advanced installation service
func NewAdvancedInstallationService(maxConcurrentJobs int) (*AdvancedInstallationService, error) {
	registryIntegrator, err := NewRegistryIntegrator()
	if err != nil {
		return nil, fmt.Errorf("failed to create registry integrator: %w", err)
	}
	
	return &AdvancedInstallationService{
		jobManager:         NewJobManager(maxConcurrentJobs),
		registryIntegrator: registryIntegrator,
	}, nil
}

// InstallFromGit starts a git-based installation
func (ais *AdvancedInstallationService) InstallFromGit(ctx context.Context, slug, uri string, options GitInstallOptions) (string, error) {
	options.URI = uri
	installer := NewConcreteGitInstaller(options)
	job := ais.jobManager.CreateJob(slug, SrcGit, uri, installer)
	
	if err := ais.jobManager.StartJob(job.ID); err != nil {
		return "", fmt.Errorf("failed to start git installation job: %w", err)
	}
	
	return job.ID, nil
}

// InstallFromNPM starts an npm-based installation
func (ais *AdvancedInstallationService) InstallFromNPM(ctx context.Context, slug, packageName string, options NPMInstallOptions) (string, error) {
	options.Package = packageName
	installer := NewConcreteNPMInstaller(options)
	job := ais.jobManager.CreateJob(slug, SrcNpm, packageName, installer)
	
	if err := ais.jobManager.StartJob(job.ID); err != nil {
		return "", fmt.Errorf("failed to start npm installation job: %w", err)
	}
	
	return job.ID, nil
}

// InstallFromPip starts a pip-based installation
func (ais *AdvancedInstallationService) InstallFromPip(ctx context.Context, slug, packageName string, options PipInstallOptions) (string, error) {
	options.Package = packageName
	installer := NewConcretePipInstaller(options)
	job := ais.jobManager.CreateJob(slug, SrcPip, packageName, installer)
	
	if err := ais.jobManager.StartJob(job.ID); err != nil {
		return "", fmt.Errorf("failed to start pip installation job: %w", err)
	}
	
	return job.ID, nil
}

// GetJobStatus returns the current status of an installation job
func (ais *AdvancedInstallationService) GetJobStatus(jobID string) (*InstallationJob, error) {
	job, exists := ais.jobManager.GetJob(jobID)
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	
	return job.GetSnapshot(), nil
}

// CancelJob cancels a running installation job
func (ais *AdvancedInstallationService) CancelJob(jobID string) error {
	return ais.jobManager.CancelJob(jobID)
}

// ListJobs returns all jobs, optionally filtered by status
func (ais *AdvancedInstallationService) ListJobs(statusFilter ...JobStatus) []*InstallationJob {
	jobs := ais.jobManager.ListJobs(statusFilter...)
	snapshots := make([]*InstallationJob, len(jobs))
	for i, job := range jobs {
		snapshots[i] = job.GetSnapshot()
	}
	return snapshots
}

// FinalizeInstallation completes the installation by registering the server
func (ais *AdvancedInstallationService) FinalizeInstallation(ctx context.Context, jobID string) error {
	job, exists := ais.jobManager.GetJob(jobID)
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}
	
	if !job.IsCompleted() || job.Result == nil {
		return fmt.Errorf("job %s is not completed or has no result", jobID)
	}
	
	if !job.Result.Success {
		return fmt.Errorf("job %s failed, cannot finalize", jobID)
	}
	
	job.UpdateStage(StageRegistering, 0)
	job.Logf(LogLevelInfo, StageRegistering, "Registering server in registry")
	
	// Validate the installation
	if err := ais.registryIntegrator.ValidateServerInstallation(ctx, job.Slug, job.Result); err != nil {
		job.Logf(LogLevelError, StageRegistering, "Installation validation failed: %v", err)
		return fmt.Errorf("installation validation failed: %w", err)
	}
	
	// Create server manifest
	if err := ais.registryIntegrator.CreateManifest(job.Slug, job.Result, job.Type, job.URI); err != nil {
		job.Logf(LogLevelWarning, StageRegistering, "Failed to create manifest: %v", err)
		// Don't fail the installation for manifest creation failure
	}
	
	// Register server in registry
	serverEntry, err := ais.registryIntegrator.RegisterServer(ctx, job.Slug, job.Result, job.Type, job.URI)
	if err != nil {
		job.Logf(LogLevelError, StageRegistering, "Failed to register server: %v", err)
		return fmt.Errorf("failed to register server: %w", err)
	}
	
	// Update job result with server entry
	job.mu.Lock()
	job.Result.ServerEntry = serverEntry
	job.mu.Unlock()
	
	job.UpdateStage(StageRegistering, 100)
	job.Logf(LogLevelInfo, StageRegistering, "Server successfully registered")
	
	return nil
}

// RemoveServer removes a server installation and unregisters it
func (ais *AdvancedInstallationService) RemoveServer(ctx context.Context, slug string) error {
	// TODO: Implement server removal logic
	// This would involve:
	// 1. Stopping the server if it's running
	// 2. Removing installation files
	// 3. Unregistering from registry
	// 4. Cleaning up client configurations
	
	return fmt.Errorf("server removal not implemented yet")
}

// GetRegistry returns the current server registry
func (ais *AdvancedInstallationService) GetRegistry() (*RegistryIntegrator, error) {
	return ais.registryIntegrator, nil
}