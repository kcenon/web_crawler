package operator

import (
	"log/slog"
	"testing"
)

func newTestReconciler() *Reconciler {
	return NewReconciler(slog.New(slog.DiscardHandler))
}

func TestReconcile_PendingToRunning(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		Workers:  5,
	}

	result, err := r.Reconcile("job-1", spec, 0)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if !result.Requeue {
		t.Error("expected Requeue=true for new job")
	}

	if len(result.Actions) < 2 {
		t.Fatalf("expected at least 2 actions, got %d: %v", len(result.Actions), result.Actions)
	}

	state, ok := r.GetState("job-1")
	if !ok {
		t.Fatal("expected state to exist")
	}
	if state.Status.Phase != JobPhaseRunning {
		t.Errorf("expected phase Running, got %s", state.Status.Phase)
	}
	if state.DesiredWorkers != 5 {
		t.Errorf("expected 5 desired workers, got %d", state.DesiredWorkers)
	}
}

func TestReconcile_ScaleUp(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		Workers:  5,
	}

	// First reconcile: pending → running.
	_, _ = r.Reconcile("job-1", spec, 0)

	// Second reconcile: running with fewer workers than desired.
	result, err := r.Reconcile("job-1", spec, 3)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	found := false
	for _, a := range result.Actions {
		if a == "scale-up:3->5" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected scale-up action, got %v", result.Actions)
	}
}

func TestReconcile_ScaleDown(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		Workers:  3,
	}

	_, _ = r.Reconcile("job-1", spec, 0)

	result, err := r.Reconcile("job-1", spec, 5)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	found := false
	for _, a := range result.Actions {
		if a == "scale-down:5->3" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected scale-down action, got %v", result.Actions)
	}
}

func TestReconcile_Suspend(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		Workers:  3,
	}

	_, _ = r.Reconcile("job-1", spec, 0)

	// Suspend the job.
	spec.Suspend = true
	result, err := r.Reconcile("job-1", spec, 3)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	state, _ := r.GetState("job-1")
	if state.Status.Phase != JobPhasePaused {
		t.Errorf("expected phase Paused, got %s", state.Status.Phase)
	}

	found := false
	for _, a := range result.Actions {
		if a == "scale-down-to-zero" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected scale-down-to-zero action, got %v", result.Actions)
	}
}

func TestReconcile_Resume(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		Workers:  3,
	}

	_, _ = r.Reconcile("job-1", spec, 0)

	// Suspend.
	spec.Suspend = true
	_, _ = r.Reconcile("job-1", spec, 3)

	// Resume.
	spec.Suspend = false
	result, err := r.Reconcile("job-1", spec, 0)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	state, _ := r.GetState("job-1")
	if state.Status.Phase != JobPhaseRunning {
		t.Errorf("expected phase Running after resume, got %s", state.Status.Phase)
	}

	found := false
	for _, a := range result.Actions {
		if a == "resume" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected resume action, got %v", result.Actions)
	}
}

func TestMarkCompleted(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		Workers:  1,
	}
	_, _ = r.Reconcile("job-1", spec, 0)

	r.MarkCompleted("job-1")

	state, _ := r.GetState("job-1")
	if state.Status.Phase != JobPhaseCompleted {
		t.Errorf("expected phase Completed, got %s", state.Status.Phase)
	}
	if state.Status.CompletionTime == nil {
		t.Error("expected CompletionTime to be set")
	}
	if state.DesiredWorkers != 0 {
		t.Errorf("expected 0 desired workers after completion, got %d", state.DesiredWorkers)
	}
}

func TestMarkFailed(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		Workers:  1,
	}
	_, _ = r.Reconcile("job-1", spec, 0)

	r.MarkFailed("job-1", "seed URL unreachable")

	state, _ := r.GetState("job-1")
	if state.Status.Phase != JobPhaseFailed {
		t.Errorf("expected phase Failed, got %s", state.Status.Phase)
	}

	if len(state.Status.Conditions) == 0 {
		t.Fatal("expected at least one condition")
	}

	last := state.Status.Conditions[len(state.Status.Conditions)-1]
	if last.Type != "Failed" {
		t.Errorf("expected Failed condition, got %s", last.Type)
	}
}

func TestReconcile_CompletedNoOp(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		Workers:  1,
	}
	_, _ = r.Reconcile("job-1", spec, 0)
	r.MarkCompleted("job-1")

	result, err := r.Reconcile("job-1", spec, 0)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected no actions for completed job, got %v", result.Actions)
	}
	if result.Requeue {
		t.Error("expected no requeue for completed job")
	}
}

func TestReconcile_DefaultWorkers(t *testing.T) {
	r := newTestReconciler()

	spec := CrawlJobSpec{
		SeedURLs: []string{"https://example.com"},
		// Workers not set — should default to 3.
	}

	_, _ = r.Reconcile("job-1", spec, 0)

	state, _ := r.GetState("job-1")
	if state.DesiredWorkers != 3 {
		t.Errorf("expected default 3 workers, got %d", state.DesiredWorkers)
	}
}
