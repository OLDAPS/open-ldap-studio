package jobs

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// recorder captures emitted events for assertion.
type recorder struct {
	mu     sync.Mutex
	events []event
}

type event struct {
	name    string
	payload map[string]any
}

func (r *recorder) Emit(name string, payload any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, _ := payload.(map[string]any)
	r.events = append(r.events, event{name: name, payload: m})
}

func (r *recorder) count(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int
	for _, e := range r.events {
		if e.name == name {
			n++
		}
	}
	return n
}

func (r *recorder) last(name string) (event, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].name == name {
			return r.events[i], true
		}
	}
	return event{}, false
}

// waitFor polls until cond holds or the deadline passes.
func waitFor(t *testing.T, d time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return cond()
}

func TestJobEmitsStartedThenExactlyOneFinished(t *testing.T) {
	// Contract E3 — every job reaching a terminal state emits exactly one
	// job:finished.
	rec := &recorder{}
	r := NewRegistry(rec)

	id := r.Start(context.Background(), KindSearch, ModeExecute, "p1", 3, func(ctx context.Context, rep *Reporter) error {
		rep.Progress(3, 3, "done")
		return nil
	})

	if !waitFor(t, 2*time.Second, func() bool { return rec.count("job:finished") == 1 }) {
		t.Fatalf("got %d job:finished events, want exactly 1", rec.count("job:finished"))
	}
	if rec.count("job:started") != 1 {
		t.Errorf("got %d job:started events, want 1", rec.count("job:started"))
	}

	job, err := r.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if job.State != StateSucceeded {
		t.Errorf("state = %s, want %s", job.State, StateSucceeded)
	}
	if job.EndedAt.IsZero() {
		t.Error("a terminal job must record when it ended")
	}
}

func TestCancelIsEffectiveWithinTwoSeconds(t *testing.T) {
	// Contract C8 / SC-003 — every JobID-returning method has a Cancel that
	// takes effect within 2 s.
	rec := &recorder{}
	r := NewRegistry(rec)

	started := make(chan struct{})
	id := r.Start(context.Background(), KindExport, ModeExecute, "p1", 0, func(ctx context.Context, rep *Reporter) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	<-started

	begin := time.Now()
	if err := r.Cancel(id); err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, 2*time.Second, func() bool {
		job, _ := r.Get(id)
		return job.State.IsTerminal()
	}) {
		t.Fatal("job did not reach a terminal state within 2 s of Cancel")
	}
	if elapsed := time.Since(begin); elapsed > 2*time.Second {
		t.Errorf("cancellation took %s, budget is 2 s", elapsed)
	}

	job, _ := r.Get(id)
	if job.State != StateCancelled {
		t.Errorf("state = %s, want %s", job.State, StateCancelled)
	}
}

func TestCancelUnblocksAPausedJob(t *testing.T) {
	rec := &recorder{}
	r := NewRegistry(rec)

	entered := make(chan struct{})
	id := r.Start(context.Background(), KindBulkModify, ModeExecute, "p1", 100, func(ctx context.Context, rep *Reporter) error {
		close(entered)
		throttle := Throttle{BatchSize: 1, BetweenBatches: time.Hour}
		for i := range 100 {
			if err := throttle.Step(ctx, rep, i); err != nil {
				return err
			}
		}
		return nil
	})
	<-entered

	if err := r.Pause(id); err != nil {
		t.Fatal(err)
	}
	if err := r.Cancel(id); err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, 2*time.Second, func() bool {
		job, _ := r.Get(id)
		return job.State == StateCancelled
	}) {
		job, _ := r.Get(id)
		t.Fatalf("a paused job did not cancel within 2 s; state = %s", job.State)
	}
}

func TestMixedOutcomesArePartiallyCompleteNeverSucceeded(t *testing.T) {
	// Contract X6 / FR-050 — a multi-entry run with mixed outcomes reports
	// every entry and never reports succeeded.
	rec := &recorder{}
	r := NewRegistry(rec)

	failure := ldapx.NewResult(ldapx.InsufficientAccessRights, "", "not authorised")
	id := r.Start(context.Background(), KindBulkModify, ModeExecute, "p1", 3, func(ctx context.Context, rep *Reporter) error {
		rep.Outcome("cn=a", OutcomeSucceeded, nil)
		rep.Outcome("cn=b", OutcomeFailed, &failure)
		rep.Outcome("cn=c", OutcomeSkipped, nil)
		return nil
	})

	if !waitFor(t, 2*time.Second, func() bool {
		job, _ := r.Get(id)
		return job.State.IsTerminal()
	}) {
		t.Fatal("job did not finish")
	}

	job, _ := r.Get(id)
	if job.State != StatePartiallyComplete {
		t.Fatalf("state = %s, want %s — mixed outcomes are never folded into succeeded",
			job.State, StatePartiallyComplete)
	}

	outcomes, err := r.Outcomes(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 3 {
		t.Fatalf("got %d outcomes, want one per entry including the skipped one", len(outcomes))
	}
	// The failing entry keeps the server's verbatim result.
	for _, o := range outcomes {
		if o.Status == OutcomeFailed {
			if o.Result == nil || o.Result.DiagnosticMessage != "not authorised" {
				t.Errorf("failed outcome lost the server's diagnostic: %+v", o.Result)
			}
		}
	}
	if rec.count("job:outcome") != 3 {
		t.Errorf("emitted %d job:outcome events, want 3", rec.count("job:outcome"))
	}
}

func TestAllEntriesFailingIsFailedNotPartiallyComplete(t *testing.T) {
	r := NewRegistry(&recorder{})
	failure := ldapx.NewResult(ldapx.UnwillingToPerform, "", "refused")

	id := r.Start(context.Background(), KindImport, ModeExecute, "p1", 2, func(ctx context.Context, rep *Reporter) error {
		rep.Outcome("cn=a", OutcomeFailed, &failure)
		rep.Outcome("cn=b", OutcomeFailed, &failure)
		return nil
	})

	waitFor(t, 2*time.Second, func() bool { job, _ := r.Get(id); return job.State.IsTerminal() })
	if job, _ := r.Get(id); job.State != StateFailed {
		t.Errorf("state = %s, want %s", job.State, StateFailed)
	}
}

func TestBodyErrorIsFailedAndKeepsItsExplanation(t *testing.T) {
	r := NewRegistry(&recorder{})
	id := r.Start(context.Background(), KindConnect, ModeExecute, "p1", 0, func(ctx context.Context, rep *Reporter) error {
		return errors.New("dial tcp: connection refused")
	})

	waitFor(t, 2*time.Second, func() bool { job, _ := r.Get(id); return job.State.IsTerminal() })
	job, _ := r.Get(id)
	if job.State != StateFailed {
		t.Errorf("state = %s, want %s", job.State, StateFailed)
	}
	if job.Err != "dial tcp: connection refused" {
		t.Errorf("error text = %q, want it preserved verbatim", job.Err)
	}
}

func TestCancelUnknownJob(t *testing.T) {
	r := NewRegistry(&recorder{})
	if err := r.Cancel("nope"); !errors.Is(err, ErrNoSuchJob) {
		t.Errorf("Cancel(unknown) = %v, want ErrNoSuchJob", err)
	}
}

func TestDryRunSendsNothingAndRecordsWhatItWouldHaveSent(t *testing.T) {
	d := DispatcherFor(ModeDryRun)
	if d.Mode() != ModeDryRun {
		t.Fatalf("mode = %s", d.Mode())
	}

	var applied bool
	res, err := d.Dispatch(context.Background(), "cn=a,dc=example,dc=com", func(context.Context) (ldapx.Result, error) {
		applied = true
		return ldapx.NewResult(ldapx.Success, "", ""), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("a dry run reached the dispatch call; it must be replaced by the recorder")
	}
	if got := d.Recorded(); len(got) != 1 || got[0] != "cn=a,dc=example,dc=com" {
		t.Errorf("recorded %v, want the DN a live run would have written", got)
	}
	if res.Code != ldapx.Success {
		t.Errorf("dry run result code = %d, want success", res.Code)
	}
}

func TestPruneDropsOnlyFinishedJobs(t *testing.T) {
	r := NewRegistry(&recorder{})
	// Some platform clocks return the same time for consecutive readings.
	// Control the clock so pruning never relies on a real clock tick.
	var now atomic.Int64
	now.Store(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).UnixNano())
	r.clock = func() time.Time { return time.Unix(0, now.Load()) }
	t.Cleanup(r.CancelAll)
	runningID := r.Start(context.Background(), KindSearch, ModeExecute, "p1", 0, func(ctx context.Context, _ *Reporter) error {
		<-ctx.Done()
		return ctx.Err()
	})
	id := r.Start(context.Background(), KindSearch, ModeExecute, "p1", 0, func(context.Context, *Reporter) error {
		return nil
	})
	if !waitFor(t, 2*time.Second, func() bool { job, _ := r.Get(id); return job.State.IsTerminal() }) {
		t.Fatal("job did not finish")
	}

	if removed := r.Prune(time.Hour); removed != 0 {
		t.Errorf("pruned %d recent jobs, want 0", removed)
	}
	if removed := r.Prune(0); removed != 0 {
		t.Errorf("pruned %d jobs at the cutoff, want 0", removed)
	}
	now.Add(int64(time.Nanosecond))
	if removed := r.Prune(0); removed != 1 {
		t.Errorf("pruned %d jobs, want 1", removed)
	}
	if _, err := r.Get(id); !errors.Is(err, ErrNoSuchJob) {
		t.Error("pruned job is still registered")
	}
	if job, err := r.Get(runningID); err != nil || job.State != StateRunning {
		t.Errorf("running job was pruned or stopped: job = %+v, err = %v", job, err)
	}
}
