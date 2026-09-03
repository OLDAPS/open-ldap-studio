package jobs

import (
	"sync"
	"time"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// MaxProgressEventsPerSecond caps the progress stream.
//
// The 100 ms UI budget (SC-002) can be breached by the event stream itself:
// a 100,000-entry job that emitted per entry would deliver more work to the
// webview than the operation does to the server (research R14).
const MaxProgressEventsPerSecond = 20

// progressInterval is the minimum gap between two emitted progress events.
const progressInterval = time.Second / MaxProgressEventsPerSecond

// Reporter is the job body's channel back to the UI. It coalesces progress,
// records per-entry outcomes, and is the point where a paused job waits.
type Reporter struct {
	registry *Registry
	jobID    string

	mu       sync.Mutex
	lastSent time.Time
	pending  bool
	done     int
	total    int
	message  string

	pauseMu sync.Mutex
	paused  chan struct{} // non-nil while paused; closed on resume
}

func newReporter(r *Registry, jobID string) *Reporter {
	return &Reporter{registry: r, jobID: jobID}
}

// Progress records that done of total units are complete.
//
// It never blocks and never emits more than MaxProgressEventsPerSecond events;
// intermediate values are coalesced, and the final value always reaches the UI
// because flush runs when the job body returns (contract E2).
func (r *Reporter) Progress(done, total int, message string) {
	r.mu.Lock()
	r.done, r.total, r.message = done, total, message

	now := time.Now()
	if now.Sub(r.lastSent) < progressInterval {
		r.pending = true
		r.mu.Unlock()
		return
	}
	r.lastSent = now
	r.pending = false
	payload := r.payloadLocked()
	r.mu.Unlock()

	r.store(done, total, message)
	r.registry.emitter.Emit("job:progress", payload)
}

// flush emits any coalesced final value. It runs once, when the body returns.
func (r *Reporter) flush() {
	r.mu.Lock()
	if !r.pending {
		r.mu.Unlock()
		return
	}
	r.pending = false
	payload := r.payloadLocked()
	done, total, message := r.done, r.total, r.message
	r.mu.Unlock()

	r.store(done, total, message)
	r.registry.emitter.Emit("job:progress", payload)
}

func (r *Reporter) payloadLocked() map[string]any {
	return map[string]any{
		"jobId": r.jobID, "done": r.done, "total": r.total,
		"message": r.message, "ts": time.Now().Format(time.RFC3339Nano),
	}
}

func (r *Reporter) store(done, total int, message string) {
	r.registry.mu.Lock()
	defer r.registry.mu.Unlock()
	if e, ok := r.registry.jobs[r.jobID]; ok {
		e.job.Done, e.job.Message = done, message
		if total > 0 {
			e.job.Total = total
		}
	}
}

// Outcome records one entry's result. Every entry of a multi-entry run gets
// one, including the skipped ones (FR-097, contract X6).
func (r *Reporter) Outcome(dn, status string, result *ldapx.Result) {
	o := Outcome{DN: dn, Status: status, Result: result}

	r.registry.mu.Lock()
	if e, ok := r.registry.jobs[r.jobID]; ok {
		e.outcomes = append(e.outcomes, o)
	}
	r.registry.mu.Unlock()

	r.registry.emitter.Emit("job:outcome", map[string]any{
		"jobId": r.jobID, "dn": dn, "status": status, "result": result,
		"ts": time.Now().Format(time.RFC3339Nano),
	})
}

// SetReportPath records where the run's report and rejects file were written.
func (r *Reporter) SetReportPath(path string) {
	r.registry.mu.Lock()
	defer r.registry.mu.Unlock()
	if e, ok := r.registry.jobs[r.jobID]; ok {
		e.job.ReportPath = path
	}
}

// JobID identifies the job this reporter belongs to.
func (r *Reporter) JobID() string { return r.jobID }

func (r *Reporter) pause() {
	r.pauseMu.Lock()
	defer r.pauseMu.Unlock()
	if r.paused == nil {
		r.paused = make(chan struct{})
	}
}

func (r *Reporter) resume() {
	r.pauseMu.Lock()
	defer r.pauseMu.Unlock()
	if r.paused != nil {
		close(r.paused)
		r.paused = nil
	}
}

// pausedChan returns the channel to wait on, or nil when running.
func (r *Reporter) pausedChan() chan struct{} {
	r.pauseMu.Lock()
	defer r.pauseMu.Unlock()
	return r.paused
}
