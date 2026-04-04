package distributed

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// JobState represents the lifecycle state of a crawl job.
type JobState string

// JobState values.
const (
	JobCreated   JobState = "created"
	JobRunning   JobState = "running"
	JobPaused    JobState = "paused"
	JobCompleted JobState = "completed"
	JobFailed    JobState = "failed"
	JobCancelled JobState = "cancelled"
)

// CrawlJob represents a distributed crawl job with its configuration
// and current state.
type CrawlJob struct {
	// ID uniquely identifies this crawl job.
	ID string `json:"id"`

	// Name is a human-readable name for the job.
	Name string `json:"name"`

	// SeedURLs are the starting URLs for the crawl.
	SeedURLs []string `json:"seed_urls"`

	// MaxDepth limits crawl depth from seed URLs.
	MaxDepth int `json:"max_depth"`

	// State is the current lifecycle state.
	State JobState `json:"state"`

	// CreatedAt is when the job was created.
	CreatedAt time.Time `json:"created_at"`

	// StartedAt is when the job began running.
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt is when the job finished.
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// validTransitions defines allowed state transitions.
var validTransitions = map[JobState][]JobState{
	JobCreated:   {JobRunning, JobCancelled},
	JobRunning:   {JobPaused, JobCompleted, JobFailed, JobCancelled},
	JobPaused:    {JobRunning, JobCancelled},
	JobCompleted: {},
	JobFailed:    {JobRunning}, // Allow restart from failed.
	JobCancelled: {},
}

// canTransitionTo checks if a state transition is valid.
func (j *CrawlJob) canTransitionTo(target JobState) bool {
	for _, allowed := range validTransitions[j.State] {
		if allowed == target {
			return true
		}
	}
	return false
}

// JobManager orchestrates distributed crawl jobs by managing their lifecycle
// and sending control messages to worker nodes via Kafka.
type JobManager struct {
	mu           sync.RWMutex
	jobs         map[string]*CrawlJob
	producer     *Producer
	controlTopic string
	brokers      []string
	logger       *slog.Logger
}

// JobManagerConfig configures the job manager.
type JobManagerConfig struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string

	// TaskTopic is the Kafka topic for crawl tasks.
	TaskTopic string

	// ControlTopic is the Kafka topic for control messages.
	// Default: "crawl-control".
	ControlTopic string

	// Logger for job manager operations.
	Logger *slog.Logger
}

func (c *JobManagerConfig) defaults() {
	if c.TaskTopic == "" {
		c.TaskTopic = "crawl-tasks"
	}
	if c.ControlTopic == "" {
		c.ControlTopic = "crawl-control"
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
}

// NewJobManager creates a job manager for distributed crawl orchestration.
func NewJobManager(cfg JobManagerConfig) *JobManager {
	cfg.defaults()

	return &JobManager{
		jobs: make(map[string]*CrawlJob),
		producer: NewProducer(ProducerConfig{
			Brokers: cfg.Brokers,
			Topic:   cfg.TaskTopic,
		}),
		controlTopic: cfg.ControlTopic,
		brokers:      cfg.Brokers,
		logger:       cfg.Logger,
	}
}

// CreateJob registers a new crawl job in the Created state.
func (m *JobManager) CreateJob(job *CrawlJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[job.ID]; exists {
		return fmt.Errorf("distributed: job %s already exists", job.ID)
	}

	job.State = JobCreated
	job.CreatedAt = time.Now()
	m.jobs[job.ID] = job

	m.logger.Info("job created", "job_id", job.ID, "name", job.Name)
	return nil
}

// StartJob transitions a job to Running and publishes seed URLs.
func (m *JobManager) StartJob(ctx context.Context, jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return fmt.Errorf("distributed: job %s not found", jobID)
	}

	if !job.canTransitionTo(JobRunning) {
		return fmt.Errorf("distributed: cannot start job in state %s", job.State)
	}

	// Publish seed URLs as crawl tasks.
	tasks := make([]*CrawlTask, 0, len(job.SeedURLs))
	for _, u := range job.SeedURLs {
		tasks = append(tasks, &CrawlTask{
			JobID:    jobID,
			URL:      u,
			Depth:    0,
			MaxDepth: job.MaxDepth,
			Priority: 10, // Seed URLs get high priority.
		})
	}

	if err := m.producer.PublishBatch(ctx, tasks); err != nil {
		return fmt.Errorf("distributed: publish seed URLs: %w", err)
	}

	now := time.Now()
	job.State = JobRunning
	job.StartedAt = &now

	m.logger.Info("job started", "job_id", jobID, "seeds", len(job.SeedURLs))
	return nil
}

// PauseJob transitions a job to Paused and sends a pause control message.
func (m *JobManager) PauseJob(ctx context.Context, jobID string) error {
	return m.controlTransition(ctx, jobID, JobPaused, ControlPause)
}

// ResumeJob transitions a job from Paused to Running and sends a resume control.
func (m *JobManager) ResumeJob(ctx context.Context, jobID string) error {
	return m.controlTransition(ctx, jobID, JobRunning, ControlResume)
}

// StopJob transitions a job to Cancelled and sends a stop control message.
func (m *JobManager) StopJob(ctx context.Context, jobID string) error {
	return m.controlTransition(ctx, jobID, JobCancelled, ControlStop)
}

// GetJob returns the current state of a job.
func (m *JobManager) GetJob(jobID string) (*CrawlJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("distributed: job %s not found", jobID)
	}
	return job, nil
}

// ListJobs returns all registered jobs.
func (m *JobManager) ListJobs() []*CrawlJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*CrawlJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		result = append(result, job)
	}
	return result
}

// Close shuts down the job manager and its producer.
func (m *JobManager) Close() error {
	return m.producer.Close()
}

// controlTransition validates a state transition, updates the job, and
// sends a control message to workers.
func (m *JobManager) controlTransition(ctx context.Context, jobID string, target JobState, ctrl ControlType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return fmt.Errorf("distributed: job %s not found", jobID)
	}

	if !job.canTransitionTo(target) {
		return fmt.Errorf("distributed: cannot transition job from %s to %s", job.State, target)
	}

	// Send control message.
	msg := &ControlMessage{
		Type:      ctrl,
		JobID:     jobID,
		Timestamp: time.Now(),
	}

	data, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("distributed: encode control: %w", err)
	}

	w := &kafka.Writer{
		Addr:  kafka.TCP(m.brokers...),
		Topic: m.controlTopic,
	}
	defer func() { _ = w.Close() }()

	if err := w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(jobID),
		Value: data,
	}); err != nil {
		return fmt.Errorf("distributed: publish control: %w", err)
	}

	job.State = target
	if target == JobCancelled || target == JobCompleted || target == JobFailed {
		now := time.Now()
		job.CompletedAt = &now
	}

	m.logger.Info("job state changed",
		"job_id", jobID,
		"state", target,
		"control", ctrl,
	)
	return nil
}
