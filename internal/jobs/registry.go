package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// State is a job's lifecycle position. The four terminal states are distinct:
// partiallyComplete is never folded into succeeded, because a bulk run that
// changed 900 of 1000 entries did not succeed (FR-050, events.md).
type State string

const (
	StateRunning           State = "running"
	StateSucceeded         State = "succeeded"
	StateFailed            State = "failed"
	StateCancelled         State = "cancelled"
	StatePartiallyComplete State = "partiallyComplete"
)

// IsTerminal reports whether a job in this state will not change again.
func (s State) IsTerminal() bool { return s != StateRunning }

// Mode selects execution or a dry run. In DryRun the dispatch call is replaced
// by a recorder: the full validation path runs, nothing is sent (FR-047).
type Mode string

const (
	ModeExecute Mode = "execute"
	ModeDryRun  Mode = "dryRun"
)

// Kind names what a job is doing, for the progress panel's label.
type Kind string

const (
	KindConnect    Kind = "connect"
	KindTestBind   Kind = "testBind"
	KindSearch     Kind = "search"
	KindCount      Kind = "countSubtree"
	KindCommit     Kind = "commit"
	KindImport     Kind = "import"
	KindExport     Kind = "export"
	KindCompare    Kind = "compare"
	KindBulkModify Kind = "bulkModify"
	KindCopyMove   Kind = "copyMove"
)

// Outcome is one entry's result inside a multi-entry job. A bulk job, import,
// copy or subtree delete produces one of these per entry, never one verdict
// for the run (FR-097, error-model.md § Multi-entry operations).
type Outcome struct {
	DN     string        `json:"dn"`
	Status string        `json:"status"` // succeeded | failed | skipped
	Result *ldapx.Result `json:"result,omitempty"`
}

const (
	OutcomeSucceeded = "succeeded"
	OutcomeFailed    = "failed"
	OutcomeSkipped   = "skipped"
)

// Job is the observable state of one cancellable operation.
type Job struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Mode      Mode      `json:"mode"`
	State     State     `json:"state"`
	ProfileID string    `json:"profileId,omitempty"`
	Total     int       `json:"total"`
	Done      int       `json:"done"`
	Message   string    `json:"message"`
	StartedAt time.Time `json:"startedAt"`
	// EndedAt is zero while the job runs.
	EndedAt time.Time `json:"endedAt,omitzero"`
	Summary string    `json:"summary,omitempty"`
	// ReportPath names the file holding per-entry outcomes for a bulk run, and
	// the sibling rejects LDIF where one was written (flow 7e).
	ReportPath string `json:"reportPath,omitempty"`
	// Err is our own explanation; a server failure also carries a Result on
	// the relevant Outcome.
	Err string `json:"error,omitempty"`
	// Paused reports whether a throttled job is currently held (FR-096).
	Paused bool `json:"paused"`
}

// Emitter delivers runtime events to the frontend. The registry depends on the
// interface, not on Wails, so every event assertion in the contract tests runs
// without a webview.
type Emitter interface {
	Emit(name string, payload any)
}

// Func is the body of a job. It must return promptly once ctx is done — SC-003
// bounds cancellation at 2 s.
type Func func(ctx context.Context, r *Reporter) error

// Registry owns every running job. Nothing that touches a server runs outside
// it: a plain goroutine gives no handle to cancel, no progress channel, and no
// place to enforce dry-run mode.
type Registry struct {
	mu      sync.RWMutex
	jobs    map[string]*entry
	emitter Emitter
	clock   func() time.Time
}

type entry struct {
	job      Job
	cancel   context.CancelFunc
	outcomes []Outcome
	reporter *Reporter
}

// NewRegistry returns a registry emitting through e.
func NewRegistry(e Emitter) *Registry {
	if e == nil {
		e = nopEmitter{}
	}
	return &Registry{jobs: make(map[string]*entry), emitter: e, clock: time.Now}
}

type nopEmitter struct{}

func (nopEmitter) Emit(string, any) {}

// ErrNoSuchJob is returned for an unknown job id.
var ErrNoSuchJob = errors.New("jobs: no such job")

// Start runs fn on its own goroutine and returns immediately with the job id.
//
// The parent context governs application shutdown; Cancel cancels this job
// alone.
func (r *Registry) Start(parent context.Context, kind Kind, mode Mode, profileID string, total int, fn Func) string {
	id := uuid.NewString()
	ctx, cancel := context.WithCancel(parent)

	job := Job{
		ID:        id,
		Kind:      kind,
		Mode:      mode,
		State:     StateRunning,
		ProfileID: profileID,
		Total:     total,
		StartedAt: r.clock(),
	}
	reporter := newReporter(r, id)

	r.mu.Lock()
	r.jobs[id] = &entry{job: job, cancel: cancel, reporter: reporter}
	r.mu.Unlock()

	r.emitter.Emit("job:started", map[string]any{
		"jobId": id, "kind": kind, "mode": mode, "total": total,
		"profileId": profileID, "ts": r.clock().Format(time.RFC3339Nano),
	})

	go func() {
		defer cancel()
		err := fn(ctx, reporter)
		reporter.flush()
		r.finish(ctx, id, err)
	}()

	return id
}

// finish computes the terminal state and emits exactly one job:finished
// (contract E3).
func (r *Registry) finish(ctx context.Context, id string, err error) {
	r.mu.Lock()
	e, ok := r.jobs[id]
	if !ok || e.job.State.IsTerminal() {
		r.mu.Unlock()
		return
	}

	e.job.State = terminalState(ctx, err, e.outcomes)
	e.job.EndedAt = r.clock()
	if err != nil && !errors.Is(err, context.Canceled) {
		e.job.Err = err.Error()
	}
	e.job.Summary = summarise(e.job.State, e.outcomes, e.job.Done, e.job.Total)
	finished := e.job
	r.mu.Unlock()

	r.emitter.Emit("job:finished", map[string]any{
		"jobId": id, "state": finished.State, "summary": finished.Summary,
		"reportPath": finished.ReportPath, "error": finished.Err,
		"ts": r.clock().Format(time.RFC3339Nano),
	})
}

func terminalState(ctx context.Context, err error, outcomes []Outcome) State {
	if errors.Is(err, context.Canceled) || (ctx.Err() != nil && err != nil) {
		return StateCancelled
	}
	if err != nil {
		return StateFailed
	}

	var succeeded, failed, skipped int
	for _, o := range outcomes {
		switch o.Status {
		case OutcomeSucceeded:
			succeeded++
		case OutcomeFailed:
			failed++
		case OutcomeSkipped:
			skipped++
		}
	}
	switch {
	case failed+skipped == 0:
		return StateSucceeded
	case succeeded == 0 && skipped == 0:
		return StateFailed
	default:
		// Mixed outcomes are partiallyComplete and are reported as such. The
		// report names every skipped DN — there is no partial-silent state.
		return StatePartiallyComplete
	}
}

func summarise(state State, outcomes []Outcome, done, total int) string {
	if len(outcomes) == 0 {
		return fmt.Sprintf("%s — %d of %d", state, done, total)
	}
	var succeeded, failed, skipped int
	for _, o := range outcomes {
		switch o.Status {
		case OutcomeSucceeded:
			succeeded++
		case OutcomeFailed:
			failed++
		case OutcomeSkipped:
			skipped++
		}
	}
	return fmt.Sprintf("%s — %d succeeded, %d failed, %d skipped of %d",
		state, succeeded, failed, skipped, len(outcomes))
}

// Cancel cancels a running job. It is idempotent, and cancelling a finished
// job is not an error.
func (r *Registry) Cancel(id string) error {
	r.mu.RLock()
	e, ok := r.jobs[id]
	r.mu.RUnlock()
	if !ok {
		return ErrNoSuchJob
	}
	e.cancel()
	e.reporter.resume() // a paused job must not sit through its cancellation
	return nil
}

// Get returns a snapshot of one job.
func (r *Registry) Get(id string) (Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.jobs[id]
	if !ok {
		return Job{}, ErrNoSuchJob
	}
	return e.job, nil
}

// Outcomes returns the per-entry outcomes recorded so far.
func (r *Registry) Outcomes(id string) ([]Outcome, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.jobs[id]
	if !ok {
		return nil, ErrNoSuchJob
	}
	return append([]Outcome(nil), e.outcomes...), nil
}

// List returns every job the registry knows, running and finished.
func (r *Registry) List() []Job {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Job, 0, len(r.jobs))
	for _, e := range r.jobs {
		out = append(out, e.job)
	}
	return out
}

// Pause and Resume hold a throttled job between batches (FR-096, gap G5).
func (r *Registry) Pause(id string) error  { return r.setPaused(id, true) }
func (r *Registry) Resume(id string) error { return r.setPaused(id, false) }

func (r *Registry) setPaused(id string, paused bool) error {
	r.mu.Lock()
	e, ok := r.jobs[id]
	if !ok {
		r.mu.Unlock()
		return ErrNoSuchJob
	}
	e.job.Paused = paused
	reporter := e.reporter
	r.mu.Unlock()

	if paused {
		reporter.pause()
	} else {
		reporter.resume()
	}
	return nil
}

// CancelAll cancels every running job, for shutdown.
func (r *Registry) CancelAll() {
	r.mu.RLock()
	entries := make([]*entry, 0, len(r.jobs))
	for _, e := range r.jobs {
		entries = append(entries, e)
	}
	r.mu.RUnlock()
	for _, e := range entries {
		e.cancel()
		e.reporter.resume()
	}
}

// Prune drops terminal jobs older than age, so the panel does not grow without
// bound across a long session.
func (r *Registry) Prune(age time.Duration) int {
	cutoff := r.clock().Add(-age)
	r.mu.Lock()
	defer r.mu.Unlock()
	var removed int
	for id, e := range r.jobs {
		if e.job.State.IsTerminal() && e.job.EndedAt.Before(cutoff) {
			delete(r.jobs, id)
			removed++
		}
	}
	return removed
}
