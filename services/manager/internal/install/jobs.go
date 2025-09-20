package install

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"mcp/manager/internal/registry"
)

// JobStatus represents the current status of an installation job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// JobStage represents different stages of the installation process
type JobStage string

const (
	StageValidation       JobStage = "validation"
	StageDownloading      JobStage = "downloading"
	StageExtracting       JobStage = "extracting"
	StageInstalling       JobStage = "installing"
	StageConfiguring      JobStage = "configuring"
	StagePostInstall      JobStage = "post_install"
	StageRegistering      JobStage = "registering"
	StageCompleted        JobStage = "completed"
	StageFailed           JobStage = "failed"
)

// InstallationJob represents a single installation job with detailed progress tracking
type InstallationJob struct {
	ID           string             `json:"id"`
	Slug         string             `json:"slug"`
	Type         SourceType         `json:"type"`
	URI          string             `json:"uri"`
	Status       JobStatus          `json:"status"`
	CurrentStage JobStage           `json:"currentStage"`
	Progress     float64            `json:"progress"` // 0-100
	StartTime    time.Time          `json:"startTime"`
	EndTime      *time.Time         `json:"endTime,omitempty"`
	Duration     time.Duration      `json:"duration"`
	Logs         []LogEntry         `json:"logs"`
	Result       *InstallationResult `json:"result,omitempty"`
	Error        string             `json:"error,omitempty"`
	
	// Internal fields
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	logChannel  chan LogEntry
	stageProgress map[JobStage]float64
	installer   Installer
}

// LogEntry represents a single log entry with metadata
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Stage     JobStage  `json:"stage"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
}

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

// InstallationResult contains comprehensive results of the installation
type InstallationResult struct {
	Success         bool                   `json:"success"`
	InstallPath     string                 `json:"installPath"`
	RuntimePath     string                 `json:"runtimePath"`
	BinPath         string                 `json:"binPath"`
	EntryCommand    string                 `json:"entryCommand"`
	EntryArgs       []string               `json:"entryArgs"`
	Environment     map[string]string      `json:"environment"`
	Runtime         string                 `json:"runtime"`
	PackageManager  string                 `json:"packageManager"`
	InstalledVersion string                `json:"installedVersion"`
	ServerEntry     *registry.Server       `json:"serverEntry,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Installer interface for different installation types
type Installer interface {
	Install(ctx context.Context, job *InstallationJob) (*InstallationResult, error)
}

// JobManager manages multiple installation jobs
type JobManager struct {
	jobs     map[string]*InstallationJob
	mu       sync.RWMutex
	maxJobs  int
	cleanupInterval time.Duration
}

// NewJobManager creates a new job manager
func NewJobManager(maxJobs int) *JobManager {
	if maxJobs <= 0 {
		maxJobs = 5 // Default maximum concurrent jobs
	}
	
	jm := &JobManager{
		jobs:     make(map[string]*InstallationJob),
		maxJobs:  maxJobs,
		cleanupInterval: 24 * time.Hour, // Clean up completed jobs after 24 hours
	}
	
	// Start cleanup goroutine
	go jm.cleanupLoop()
	
	return jm
}

// CreateJob creates a new installation job
func (jm *JobManager) CreateJob(slug string, sourceType SourceType, uri string, installer Installer) *InstallationJob {
	jobID := generateJobID()
	ctx, cancel := context.WithCancel(context.Background())
	
	job := &InstallationJob{
		ID:           jobID,
		Slug:         slug,
		Type:         sourceType,
		URI:          uri,
		Status:       JobStatusPending,
		CurrentStage: StageValidation,
		Progress:     0.0,
		StartTime:    time.Now(),
		Logs:         make([]LogEntry, 0),
		stageProgress: make(map[JobStage]float64),
		ctx:          ctx,
		cancel:       cancel,
		logChannel:   make(chan LogEntry, 100),
		installer:    installer,
	}
	
	// Start log collection goroutine
	go job.logCollector()
	
	jm.mu.Lock()
	jm.jobs[jobID] = job
	jm.mu.Unlock()
	
	return job
}

// GetJob retrieves a job by ID
func (jm *JobManager) GetJob(jobID string) (*InstallationJob, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	job, exists := jm.jobs[jobID]
	return job, exists
}

// ListJobs returns all jobs (optionally filtered by status)
func (jm *JobManager) ListJobs(statusFilter ...JobStatus) []*InstallationJob {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	
	var jobs []*InstallationJob
	for _, job := range jm.jobs {
		if len(statusFilter) == 0 {
			jobs = append(jobs, job)
			continue
		}
		
		for _, status := range statusFilter {
			if job.Status == status {
				jobs = append(jobs, job)
				break
			}
		}
	}
	
	return jobs
}

// StartJob starts the execution of a job
func (jm *JobManager) StartJob(jobID string) error {
	jm.mu.RLock()
	job, exists := jm.jobs[jobID]
	jm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}
	
	// Check if we have capacity for new jobs
	runningJobs := jm.ListJobs(JobStatusRunning)
	if len(runningJobs) >= jm.maxJobs {
		return fmt.Errorf("maximum number of concurrent jobs (%d) reached", jm.maxJobs)
	}
	
	// Start the job in a goroutine
	go jm.executeJob(job)
	
	return nil
}

// CancelJob cancels a running job
func (jm *JobManager) CancelJob(jobID string) error {
	jm.mu.RLock()
	job, exists := jm.jobs[jobID]
	jm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}
	
	job.mu.Lock()
	defer job.mu.Unlock()
	
	if job.Status == JobStatusRunning {
		job.cancel()
		job.Status = JobStatusCancelled
		job.updateEndTime()
		job.Log(LogLevelInfo, job.CurrentStage, "Job cancelled by user", "")
	}
	
	return nil
}

// executeJob executes an installation job
func (jm *JobManager) executeJob(job *InstallationJob) {
	job.mu.Lock()
	job.Status = JobStatusRunning
	job.StartTime = time.Now()
	job.mu.Unlock()
	
	job.Log(LogLevelInfo, StageValidation, fmt.Sprintf("Starting installation of %s from %s", job.Slug, job.URI), "")
	
	// Execute the installation
	result, err := job.installer.Install(job.ctx, job)
	
	job.mu.Lock()
	defer job.mu.Unlock()
	
	if err != nil {
		job.Status = JobStatusFailed
		job.CurrentStage = StageFailed
		job.Error = err.Error()
		job.Log(LogLevelError, StageFailed, "Installation failed", err.Error())
	} else {
		job.Status = JobStatusCompleted
		job.CurrentStage = StageCompleted
		job.Progress = 100.0
		job.Result = result
		job.Log(LogLevelInfo, StageCompleted, "Installation completed successfully", "")
	}
	
	job.updateEndTime()
}

// Log adds a log entry to the job
func (job *InstallationJob) Log(level LogLevel, stage JobStage, message, details string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Stage:     stage,
		Message:   message,
		Details:   details,
	}
	
	// Send to log channel (non-blocking)
	select {
	case job.logChannel <- entry:
	default:
		// Channel is full, skip this log entry
	}
}

// Logf adds a formatted log entry
func (job *InstallationJob) Logf(level LogLevel, stage JobStage, format string, args ...interface{}) {
	job.Log(level, stage, fmt.Sprintf(format, args...), "")
}

// UpdateStage updates the current stage and progress
func (job *InstallationJob) UpdateStage(stage JobStage, progress float64) {
	job.mu.Lock()
	defer job.mu.Unlock()
	
	job.CurrentStage = stage
	job.stageProgress[stage] = progress
	
	// Calculate overall progress based on stage completion
	job.calculateOverallProgress()
	
	job.Log(LogLevelInfo, stage, fmt.Sprintf("Stage %s: %.1f%% complete", stage, progress), "")
}

// UpdateProgress updates the progress of the current stage
func (job *InstallationJob) UpdateProgress(progress float64) {
	job.mu.Lock()
	defer job.mu.Unlock()
	
	job.stageProgress[job.CurrentStage] = progress
	job.calculateOverallProgress()
}

// calculateOverallProgress calculates the overall job progress based on stage completion
func (job *InstallationJob) calculateOverallProgress() {
	// Define stage weights (total should add up to 100)
	stageWeights := map[JobStage]float64{
		StageValidation:  5.0,
		StageDownloading: 20.0,
		StageExtracting:  10.0,
		StageInstalling:  40.0,
		StageConfiguring: 15.0,
		StagePostInstall: 5.0,
		StageRegistering: 5.0,
	}
	
	var totalProgress float64
	for stage, weight := range stageWeights {
		if progress, exists := job.stageProgress[stage]; exists {
			totalProgress += (progress / 100.0) * weight
		}
	}
	
	job.Progress = totalProgress
}

// GetSnapshot returns a read-only snapshot of the job state
func (job *InstallationJob) GetSnapshot() *InstallationJob {
	job.mu.RLock()
	defer job.mu.RUnlock()
	
	// Create a deep copy for thread-safe access
	snapshot := &InstallationJob{
		ID:           job.ID,
		Slug:         job.Slug,
		Type:         job.Type,
		URI:          job.URI,
		Status:       job.Status,
		CurrentStage: job.CurrentStage,
		Progress:     job.Progress,
		StartTime:    job.StartTime,
		EndTime:      job.EndTime,
		Duration:     job.Duration,
		Logs:         make([]LogEntry, len(job.Logs)),
		Result:       job.Result,
		Error:        job.Error,
	}
	
	copy(snapshot.Logs, job.Logs)
	
	return snapshot
}

// GetLogsSince returns logs since a specific timestamp
func (job *InstallationJob) GetLogsSince(since time.Time) []LogEntry {
	job.mu.RLock()
	defer job.mu.RUnlock()
	
	var logs []LogEntry
	for _, entry := range job.Logs {
		if entry.Timestamp.After(since) {
			logs = append(logs, entry)
		}
	}
	
	return logs
}

// IsRunning returns true if the job is currently running
func (job *InstallationJob) IsRunning() bool {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.Status == JobStatusRunning
}

// IsCompleted returns true if the job has completed (successfully or with error)
func (job *InstallationJob) IsCompleted() bool {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled
}

// updateEndTime updates the end time and calculates duration
func (job *InstallationJob) updateEndTime() {
	now := time.Now()
	job.EndTime = &now
	job.Duration = now.Sub(job.StartTime)
}

// logCollector collects log entries from the channel
func (job *InstallationJob) logCollector() {
	for entry := range job.logChannel {
		job.mu.Lock()
		job.Logs = append(job.Logs, entry)
		job.mu.Unlock()
	}
}

// cleanupLoop periodically cleans up old completed jobs
func (jm *JobManager) cleanupLoop() {
	ticker := time.NewTicker(jm.cleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		jm.cleanupOldJobs()
	}
}

// cleanupOldJobs removes jobs that have been completed for more than the cleanup interval
func (jm *JobManager) cleanupOldJobs() {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	
	cutoff := time.Now().Add(-jm.cleanupInterval)
	
	for jobID, job := range jm.jobs {
		if job.IsCompleted() && job.EndTime != nil && job.EndTime.Before(cutoff) {
			// Close the log channel
			close(job.logChannel)
			delete(jm.jobs, jobID)
		}
	}
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

// MarshalJSON implements custom JSON marshaling for job snapshots
func (job *InstallationJob) MarshalJSON() ([]byte, error) {
	job.mu.RLock()
	defer job.mu.RUnlock()
	
	// Create a struct with only the fields we want to marshal
	type jobJSON struct {
		ID           string              `json:"id"`
		Slug         string              `json:"slug"`
		Type         SourceType          `json:"type"`
		URI          string              `json:"uri"`
		Status       JobStatus           `json:"status"`
		CurrentStage JobStage            `json:"currentStage"`
		Progress     float64             `json:"progress"`
		StartTime    time.Time           `json:"startTime"`
		EndTime      *time.Time          `json:"endTime,omitempty"`
		Duration     time.Duration       `json:"duration"`
		Logs         []LogEntry          `json:"logs"`
		Result       *InstallationResult `json:"result,omitempty"`
		Error        string              `json:"error,omitempty"`
	}
	
	return json.Marshal(jobJSON{
		ID:           job.ID,
		Slug:         job.Slug,
		Type:         job.Type,
		URI:          job.URI,
		Status:       job.Status,
		CurrentStage: job.CurrentStage,
		Progress:     job.Progress,
		StartTime:    job.StartTime,
		EndTime:      job.EndTime,
		Duration:     job.Duration,
		Logs:         job.Logs,
		Result:       job.Result,
		Error:        job.Error,
	})
}

// InstallationJobLogger implements the Logger interface for installation jobs
type InstallationJobLogger struct {
	job   *InstallationJob
	level LogLevel
	stage JobStage
}

// NewInstallationJobLogger creates a logger that writes to an installation job
func NewInstallationJobLogger(job *InstallationJob, level LogLevel, stage JobStage) *InstallationJobLogger {
	return &InstallationJobLogger{
		job:   job,
		level: level,
		stage: stage,
	}
}

// Log implements the Logger interface
func (l *InstallationJobLogger) Log(line string) {
	l.job.Log(l.level, l.stage, line, "")
}

// SetStage updates the current stage for subsequent log entries
func (l *InstallationJobLogger) SetStage(stage JobStage) {
	l.stage = stage
}

// SetLevel updates the log level for subsequent log entries
func (l *InstallationJobLogger) SetLevel(level LogLevel) {
	l.level = level
}