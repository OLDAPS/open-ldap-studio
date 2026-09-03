package jobs

import (
	"context"
	"testing"
	"time"
)

// Contract E2 — job:progress is throttled: a 100k-entry job emits at most
// 20 events per second, so the event stream cannot itself breach the 100 ms UI
// budget (SC-002).
func TestProgressIsThrottledToTwentyEventsPerSecond(t *testing.T) {
	rec := &recorder{}
	r := NewRegistry(rec)

	const entries = 100_000
	start := time.Now()

	id := r.Start(context.Background(), KindImport, ModeExecute, "p1", entries, func(ctx context.Context, rep *Reporter) error {
		for i := range entries {
			rep.Progress(i+1, entries, "importing")
		}
		return nil
	})

	if !waitFor(t, 30*time.Second, func() bool {
		job, _ := r.Get(id)
		return job.State.IsTerminal()
	}) {
		t.Fatal("job did not finish")
	}
	elapsed := time.Since(start)

	emitted := rec.count("job:progress")
	// One event may be emitted immediately and one on flush, hence the +2.
	budget := int(elapsed.Seconds()*MaxProgressEventsPerSecond) + 2
	if emitted > budget {
		t.Errorf("emitted %d progress events in %s; the cap is %d/s (budget %d)",
			emitted, elapsed, MaxProgressEventsPerSecond, budget)
	}
	if emitted == 0 {
		t.Error("emitted no progress at all; throttling must coalesce, not silence")
	}
}

func TestFinalProgressValueAlwaysReachesTheUI(t *testing.T) {
	// Coalescing must not swallow the last update: a job that ends at 100%
	// while its final event is still pending would leave the panel at 87%.
	rec := &recorder{}
	r := NewRegistry(rec)

	id := r.Start(context.Background(), KindExport, ModeExecute, "p1", 3, func(ctx context.Context, rep *Reporter) error {
		rep.Progress(1, 3, "one")
		rep.Progress(2, 3, "two")
		rep.Progress(3, 3, "three") // within the throttle window: coalesced
		return nil
	})

	waitFor(t, 2*time.Second, func() bool { job, _ := r.Get(id); return job.State.IsTerminal() })

	last, ok := rec.last("job:progress")
	if !ok {
		t.Fatal("no progress event was emitted")
	}
	if last.payload["done"] != 3 {
		t.Errorf("last progress event reported done=%v, want 3", last.payload["done"])
	}

	job, _ := r.Get(id)
	if job.Done != 3 {
		t.Errorf("job state kept done=%d, want 3", job.Done)
	}
}

func TestThrottlePausesBetweenBatches(t *testing.T) {
	throttle := Throttle{BatchSize: 2, BetweenBatches: 20 * time.Millisecond}
	r := NewRegistry(&recorder{})

	var elapsed time.Duration
	id := r.Start(context.Background(), KindBulkModify, ModeExecute, "p1", 6, func(ctx context.Context, rep *Reporter) error {
		start := time.Now()
		for i := range 6 {
			if err := throttle.Step(ctx, rep, i); err != nil {
				return err
			}
		}
		elapsed = time.Since(start)
		return nil
	})

	waitFor(t, 5*time.Second, func() bool { job, _ := r.Get(id); return job.State.IsTerminal() })

	// Six units at a batch size of two pause twice, after units 2 and 4.
	if elapsed < 40*time.Millisecond {
		t.Errorf("run took %s; two inter-batch pauses of 20 ms were expected", elapsed)
	}
}

func TestEstimatedDurationIncludesThePauses(t *testing.T) {
	throttle := Throttle{BatchSize: 100, BetweenBatches: time.Second}
	got := throttle.EstimatedDuration(1000, 10*time.Millisecond)
	want := 10*time.Second + 10*time.Second // work plus ten pauses
	if got != want {
		t.Errorf("EstimatedDuration = %s, want %s", got, want)
	}
}
