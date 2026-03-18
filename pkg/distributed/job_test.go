package distributed

import (
	"testing"
)

func TestCrawlJob_StateTransitions(t *testing.T) {
	tests := []struct {
		name   string
		from   JobState
		to     JobState
		expect bool
	}{
		{"created to running", JobCreated, JobRunning, true},
		{"created to cancelled", JobCreated, JobCancelled, true},
		{"created to paused", JobCreated, JobPaused, false},
		{"running to paused", JobRunning, JobPaused, true},
		{"running to completed", JobRunning, JobCompleted, true},
		{"running to failed", JobRunning, JobFailed, true},
		{"running to cancelled", JobRunning, JobCancelled, true},
		{"paused to running", JobPaused, JobRunning, true},
		{"paused to cancelled", JobPaused, JobCancelled, true},
		{"paused to completed", JobPaused, JobCompleted, false},
		{"completed to running", JobCompleted, JobRunning, false},
		{"failed to running", JobFailed, JobRunning, true},
		{"cancelled to running", JobCancelled, JobRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &CrawlJob{State: tt.from}
			got := job.canTransitionTo(tt.to)
			if got != tt.expect {
				t.Errorf("canTransitionTo(%s → %s) = %v, want %v",
					tt.from, tt.to, got, tt.expect)
			}
		})
	}
}

func TestJobManager_CreateJob(t *testing.T) {
	// Use a JobManager without Kafka (we only test in-memory state).
	mgr := &JobManager{
		jobs:   make(map[string]*CrawlJob),
		logger: noopLogger(),
	}

	job := &CrawlJob{
		ID:       "job-1",
		Name:     "Test Job",
		SeedURLs: []string{"https://example.com"},
		MaxDepth: 3,
	}

	if err := mgr.CreateJob(job); err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if job.State != JobCreated {
		t.Errorf("expected state Created, got %s", job.State)
	}

	// Duplicate should fail.
	err := mgr.CreateJob(&CrawlJob{ID: "job-1"})
	if err == nil {
		t.Error("expected error for duplicate job ID")
	}
}

func TestJobManager_GetJob(t *testing.T) {
	mgr := &JobManager{
		jobs:   make(map[string]*CrawlJob),
		logger: noopLogger(),
	}

	job := &CrawlJob{ID: "job-1", Name: "Test"}
	_ = mgr.CreateJob(job)

	got, err := mgr.GetJob("job-1")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if got.Name != "Test" {
		t.Errorf("expected name %q, got %q", "Test", got.Name)
	}

	_, err = mgr.GetJob("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent job")
	}
}

func TestJobManager_ListJobs(t *testing.T) {
	mgr := &JobManager{
		jobs:   make(map[string]*CrawlJob),
		logger: noopLogger(),
	}

	_ = mgr.CreateJob(&CrawlJob{ID: "job-1"})
	_ = mgr.CreateJob(&CrawlJob{ID: "job-2"})

	jobs := mgr.ListJobs()
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}
