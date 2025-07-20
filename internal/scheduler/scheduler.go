package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"web-scraper-api/internal/logger"
	"web-scraper-api/internal/scraper"

	"github.com/robfig/cron/v3"
)

type JobStatus string

const (
	JobStatusActive   JobStatus = "active"
	JobStatusPaused   JobStatus = "paused"
	JobStatusRunning  JobStatus = "running"
	JobStatusError    JobStatus = "error"
	JobStatusComplete JobStatus = "complete"
)

type ScheduledJob struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Schedule    string                   `json:"schedule"` // Cron expression
	URL         string                   `json:"url"`
	Options     *scraper.CrawlingOptions `json:"options"`
	Status      JobStatus                `json:"status"`
	LastRun     *time.Time               `json:"last_run"`
	NextRun     *time.Time               `json:"next_run"`
	RunCount    int                      `json:"run_count"`
	ErrorCount  int                      `json:"error_count"`
	LastError   string                   `json:"last_error"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
	Results     []*scraper.ScrapedData   `json:"results,omitempty"`
	LastResult  *scraper.ScrapedData     `json:"last_result,omitempty"`
}

type JobResult struct {
	JobID     string               `json:"job_id"`
	JobName   string               `json:"job_name"`
	Status    JobStatus            `json:"status"`
	Data      *scraper.ScrapedData `json:"data,omitempty"`
	Error     string               `json:"error,omitempty"`
	StartedAt time.Time            `json:"started_at"`
	EndedAt   time.Time            `json:"ended_at"`
	Duration  time.Duration        `json:"duration"`
}

type Scheduler struct {
	cron       *cron.Cron
	jobs       map[string]*ScheduledJob
	jobEntries map[string]cron.EntryID
	mutex      sync.RWMutex
	logger     *logger.Logger
	scraper    *scraper.Service
	// Callbacks for external integrations
	onJobStart    func(*JobResult)
	onJobComplete func(*JobResult)
	onJobError    func(*JobResult)
}

func NewScheduler(logger *logger.Logger, scraper *scraper.Service) *Scheduler {
	return &Scheduler{
		cron:       cron.New(cron.WithSeconds()),
		jobs:       make(map[string]*ScheduledJob),
		jobEntries: make(map[string]cron.EntryID),
		logger:     logger,
		scraper:    scraper,
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
	s.logger.Info("Scheduler started")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("Scheduler stopped")
}

func (s *Scheduler) AddJob(job *ScheduledJob) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Validate cron expression
	_, err := cron.ParseStandard(job.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Generate ID if not provided
	if job.ID == "" {
		job.ID = generateJobID()
	}

	// Set timestamps
	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now

	// Set initial status
	if job.Status == "" {
		job.Status = JobStatusActive
	}

	// Add to cron scheduler
	entryID, err := s.cron.AddFunc(job.Schedule, s.createJobFunction(job))
	if err != nil {
		return fmt.Errorf("failed to add job to cron: %w", err)
	}

	// Store job
	s.jobs[job.ID] = job
	s.jobEntries[job.ID] = entryID

	// Calculate next run
	s.updateNextRun(job)

	s.logger.Infof("Scheduled job added: %s (%s) - Next run: %s", job.Name, job.ID, job.NextRun.Format(time.RFC3339))

	return nil
}

func (s *Scheduler) RemoveJob(jobID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Remove from cron scheduler
	if entryID, exists := s.jobEntries[jobID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobEntries, jobID)
	}

	// Remove from jobs map
	delete(s.jobs, jobID)

	s.logger.Infof("Scheduled job removed: %s (%s)", job.Name, jobID)

	return nil
}

func (s *Scheduler) PauseJob(jobID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status == JobStatusPaused {
		return fmt.Errorf("job is already paused")
	}

	// Remove from cron scheduler
	if entryID, exists := s.jobEntries[jobID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobEntries, jobID)
	}

	job.Status = JobStatusPaused
	job.UpdatedAt = time.Now()
	job.NextRun = nil

	s.logger.Infof("Scheduled job paused: %s (%s)", job.Name, jobID)

	return nil
}

func (s *Scheduler) ResumeJob(jobID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status != JobStatusPaused {
		return fmt.Errorf("job is not paused")
	}

	// Add back to cron scheduler
	entryID, err := s.cron.AddFunc(job.Schedule, s.createJobFunction(job))
	if err != nil {
		return fmt.Errorf("failed to resume job: %w", err)
	}

	job.Status = JobStatusActive
	job.UpdatedAt = time.Now()
	s.jobEntries[jobID] = entryID

	// Calculate next run
	s.updateNextRun(job)

	s.logger.Infof("Scheduled job resumed: %s (%s) - Next run: %s", job.Name, jobID, job.NextRun.Format(time.RFC3339))

	return nil
}

func (s *Scheduler) GetJob(jobID string) (*ScheduledJob, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

func (s *Scheduler) GetAllJobs() []*ScheduledJob {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

func (s *Scheduler) RunJobNow(jobID string) error {
	job, err := s.GetJob(jobID)
	if err != nil {
		return err
	}

	// Run job in goroutine to avoid blocking
	go s.executeJob(job)

	return nil
}

func (s *Scheduler) SetCallbacks(onJobStart, onJobComplete, onJobError func(*JobResult)) {
	s.onJobStart = onJobStart
	s.onJobComplete = onJobComplete
	s.onJobError = onJobError
}

func (s *Scheduler) createJobFunction(job *ScheduledJob) func() {
	return func() {
		s.executeJob(job)
	}
}

func (s *Scheduler) executeJob(job *ScheduledJob) {
	s.mutex.Lock()
	job.Status = JobStatusRunning
	job.UpdatedAt = time.Now()
	s.mutex.Unlock()

	startTime := time.Now()
	result := &JobResult{
		JobID:     job.ID,
		JobName:   job.Name,
		Status:    JobStatusRunning,
		StartedAt: startTime,
	}

	// Notify job start
	if s.onJobStart != nil {
		s.onJobStart(result)
	}

	s.logger.Infof("Executing scheduled job: %s (%s)", job.Name, job.ID)

	// Execute scraping
	ctx, cancel := context.WithTimeout(context.Background(), job.Options.Timeout)
	defer cancel()

	data, err := s.scraper.ScrapeWebsiteWithOptions(ctx, job.URL, job.Options)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update job statistics
	job.LastRun = &endTime
	job.RunCount++
	job.UpdatedAt = endTime

	if err != nil {
		job.Status = JobStatusError
		job.ErrorCount++
		job.LastError = err.Error()
		result.Status = JobStatusError
		result.Error = err.Error()

		s.logger.Errorf("Scheduled job failed: %s (%s) - Error: %v", job.Name, job.ID, err)

		// Notify job error
		if s.onJobError != nil {
			result.EndedAt = endTime
			result.Duration = duration
			s.onJobError(result)
		}
	} else {
		job.Status = JobStatusComplete
		job.LastError = ""
		job.LastResult = data
		job.Results = append(job.Results, data)

		// Keep only last 10 results
		if len(job.Results) > 10 {
			job.Results = job.Results[len(job.Results)-10:]
		}

		result.Status = JobStatusComplete
		result.Data = data

		s.logger.Infof("Scheduled job completed: %s (%s) - Duration: %v", job.Name, job.ID, duration)

		// Notify job completion
		if s.onJobComplete != nil {
			result.EndedAt = endTime
			result.Duration = duration
			s.onJobComplete(result)
		}
	}

	// Calculate next run
	s.updateNextRun(job)
}

func (s *Scheduler) updateNextRun(job *ScheduledJob) {
	// Get next run time from cron
	entries := s.cron.Entries()
	for _, entry := range entries {
		if entry.ID == s.jobEntries[job.ID] {
			job.NextRun = &entry.Next
			break
		}
	}
}

func (s *Scheduler) GetJobStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_jobs":   len(s.jobs),
		"active_jobs":  0,
		"paused_jobs":  0,
		"running_jobs": 0,
		"error_jobs":   0,
		"total_runs":   0,
		"total_errors": 0,
	}

	for _, job := range s.jobs {
		switch job.Status {
		case JobStatusActive:
			stats["active_jobs"] = stats["active_jobs"].(int) + 1
		case JobStatusPaused:
			stats["paused_jobs"] = stats["paused_jobs"].(int) + 1
		case JobStatusRunning:
			stats["running_jobs"] = stats["running_jobs"].(int) + 1
		case JobStatusError:
			stats["error_jobs"] = stats["error_jobs"].(int) + 1
		}

		stats["total_runs"] = stats["total_runs"].(int) + job.RunCount
		stats["total_errors"] = stats["total_errors"].(int) + job.ErrorCount
	}

	return stats
}

func (s *Scheduler) ExportJobs() ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return json.MarshalIndent(s.jobs, "", "  ")
}

func (s *Scheduler) ImportJobs(data []byte) error {
	var jobs map[string]*ScheduledJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return fmt.Errorf("failed to parse jobs data: %w", err)
	}

	// Stop scheduler temporarily
	s.cron.Stop()
	defer s.cron.Start()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Clear existing jobs
	s.jobs = make(map[string]*ScheduledJob)
	s.jobEntries = make(map[string]cron.EntryID)

	// Import new jobs
	for _, job := range jobs {
		if err := s.addJobInternal(job); err != nil {
			s.logger.Errorf("Failed to import job %s: %v", job.ID, err)
			continue
		}
	}

	s.logger.Infof("Imported %d scheduled jobs", len(jobs))
	return nil
}

func (s *Scheduler) addJobInternal(job *ScheduledJob) error {
	// Validate cron expression
	_, err := cron.ParseStandard(job.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Add to cron scheduler
	entryID, err := s.cron.AddFunc(job.Schedule, s.createJobFunction(job))
	if err != nil {
		return fmt.Errorf("failed to add job to cron: %w", err)
	}

	// Store job
	s.jobs[job.ID] = job
	s.jobEntries[job.ID] = entryID

	// Calculate next run
	s.updateNextRun(job)

	return nil
}

func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
