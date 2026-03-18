package operator

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Reconciler manages the desired vs actual state of CrawlJob resources.
// In a full operator, this would use controller-runtime; here we provide
// the core reconciliation logic that can be integrated with any K8s
// controller framework.
type Reconciler struct {
	mu     sync.RWMutex
	jobs   map[string]*ReconcileState
	logger *slog.Logger
}

// ReconcileState tracks the reconciliation state for a single CrawlJob.
type ReconcileState struct {
	Spec           CrawlJobSpec
	Status         CrawlJobStatus
	DesiredWorkers int
	ActualWorkers  int
	LastReconcile  time.Time
}

// NewReconciler creates a new CrawlJob reconciler.
func NewReconciler(logger *slog.Logger) *Reconciler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Reconciler{
		jobs:   make(map[string]*ReconcileState),
		logger: logger,
	}
}

// ReconcileResult describes the outcome of a reconciliation loop.
type ReconcileResult struct {
	// Requeue indicates whether the reconciler should reprocess this job.
	Requeue bool

	// RequeueAfter specifies the delay before reprocessing.
	RequeueAfter time.Duration

	// Actions lists the actions taken during reconciliation.
	Actions []string
}

// Reconcile processes a CrawlJob and determines what actions to take
// to move from the current state toward the desired state.
func (r *Reconciler) Reconcile(jobID string, spec CrawlJobSpec, currentWorkers int) (*ReconcileResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.jobs[jobID]
	if !exists {
		state = &ReconcileState{
			Spec: spec,
			Status: CrawlJobStatus{
				Phase: JobPhasePending,
			},
		}
		r.jobs[jobID] = state
	}

	state.Spec = spec
	state.ActualWorkers = currentWorkers
	state.LastReconcile = time.Now()

	result := &ReconcileResult{}

	// Handle suspended jobs.
	if spec.Suspend {
		if state.Status.Phase == JobPhaseRunning {
			state.Status.Phase = JobPhasePaused
			state.DesiredWorkers = 0
			result.Actions = append(result.Actions, "scale-down-to-zero")
			r.logger.Info("job suspended", "job_id", jobID)
		}
		return result, nil
	}

	switch state.Status.Phase {
	case JobPhasePending:
		return r.reconcilePending(jobID, state, result)
	case JobPhaseRunning:
		return r.reconcileRunning(jobID, state, result)
	case JobPhasePaused:
		return r.reconcilePaused(jobID, state, result)
	case JobPhaseCompleted, JobPhaseFailed:
		return result, nil
	default:
		return nil, fmt.Errorf("operator: unknown phase %s for job %s", state.Status.Phase, jobID)
	}
}

// reconcilePending transitions a pending job to running.
func (r *Reconciler) reconcilePending(jobID string, state *ReconcileState, result *ReconcileResult) (*ReconcileResult, error) {
	workers := state.Spec.Workers
	if workers <= 0 {
		workers = 3
	}

	state.DesiredWorkers = workers
	state.Status.Phase = JobPhaseRunning
	now := time.Now()
	state.Status.StartTime = &now
	state.Status.Conditions = append(state.Status.Conditions, Condition{
		Type:               "Ready",
		Status:             "True",
		LastTransitionTime: now,
		Reason:             "JobStarted",
		Message:            fmt.Sprintf("Starting %d workers", workers),
	})

	result.Actions = append(result.Actions,
		fmt.Sprintf("create-workers:%d", workers),
		"publish-seed-urls",
	)
	result.Requeue = true
	result.RequeueAfter = 10 * time.Second

	r.logger.Info("job transitioning to running",
		"job_id", jobID,
		"workers", workers,
	)
	return result, nil
}

// reconcileRunning ensures the running job has the correct number of workers.
func (r *Reconciler) reconcileRunning(jobID string, state *ReconcileState, result *ReconcileResult) (*ReconcileResult, error) {
	desired := state.Spec.Workers
	if desired <= 0 {
		desired = 3
	}
	state.DesiredWorkers = desired

	actual := state.ActualWorkers

	switch {
	case actual < desired:
		diff := desired - actual
		result.Actions = append(result.Actions,
			fmt.Sprintf("scale-up:%d->%d", actual, desired),
		)
		r.logger.Info("scaling up workers",
			"job_id", jobID,
			"from", actual,
			"to", desired,
			"diff", diff,
		)
	case actual > desired:
		result.Actions = append(result.Actions,
			fmt.Sprintf("scale-down:%d->%d", actual, desired),
		)
		r.logger.Info("scaling down workers",
			"job_id", jobID,
			"from", actual,
			"to", desired,
		)
	}

	state.Status.ActiveWorkers = desired

	// Requeue for periodic health check.
	result.Requeue = true
	result.RequeueAfter = 30 * time.Second

	return result, nil
}

// reconcilePaused resumes a paused job if it's no longer suspended.
func (r *Reconciler) reconcilePaused(jobID string, state *ReconcileState, result *ReconcileResult) (*ReconcileResult, error) {
	if !state.Spec.Suspend {
		state.Status.Phase = JobPhaseRunning
		result.Actions = append(result.Actions, "resume")
		r.logger.Info("job resumed", "job_id", jobID)
		result.Requeue = true
	}
	return result, nil
}

// GetState returns the current reconciliation state for a job.
func (r *Reconciler) GetState(jobID string) (*ReconcileState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.jobs[jobID]
	return state, ok
}

// MarkCompleted marks a job as completed.
func (r *Reconciler) MarkCompleted(jobID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if state, ok := r.jobs[jobID]; ok {
		state.Status.Phase = JobPhaseCompleted
		now := time.Now()
		state.Status.CompletionTime = &now
		state.DesiredWorkers = 0
	}
}

// MarkFailed marks a job as failed with a reason.
func (r *Reconciler) MarkFailed(jobID string, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if state, ok := r.jobs[jobID]; ok {
		state.Status.Phase = JobPhaseFailed
		now := time.Now()
		state.Status.CompletionTime = &now
		state.DesiredWorkers = 0
		state.Status.Conditions = append(state.Status.Conditions, Condition{
			Type:               "Failed",
			Status:             "True",
			LastTransitionTime: now,
			Reason:             "JobFailed",
			Message:            reason,
		})
	}
}
