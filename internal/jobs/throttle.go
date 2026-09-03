package jobs

import (
	"context"
	"time"
)

// Throttle paces a bulk run so it cannot overrun a production directory: a
// batch size and a pause between batches, plus the pause/resume the user can
// apply mid-run (FR-096, design gap G5).
type Throttle struct {
	// BatchSize of 0 means no batching.
	BatchSize int `json:"batchSize"`
	// BetweenBatches is the pause after each batch.
	BetweenBatches time.Duration `json:"betweenBatches"`
	// ContinueOnError decides whether one failed entry stops the run (FR-097).
	ContinueOnError bool `json:"continueOnError"`
}

// Step is called by a job body once per unit of work, before the unit runs.
//
// It applies the inter-batch pause, honours a user pause, and returns the
// context's error the moment cancellation arrives — a paused job cancels
// immediately rather than waiting out its pause (SC-003).
func (t Throttle) Step(ctx context.Context, r *Reporter, index int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if paused := r.pausedChan(); paused != nil {
		select {
		case <-paused:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if t.BatchSize > 0 && index > 0 && index%t.BatchSize == 0 && t.BetweenBatches > 0 {
		timer := time.NewTimer(t.BetweenBatches)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return ctx.Err()
}

// EstimatedDuration projects how long a run of n units will take, so the batch
// wizard can state it before the user commits (screen 6c).
func (t Throttle) EstimatedDuration(n int, perUnit time.Duration) time.Duration {
	total := time.Duration(n) * perUnit
	if t.BatchSize > 0 && t.BetweenBatches > 0 {
		batches := n / t.BatchSize
		total += time.Duration(batches) * t.BetweenBatches
	}
	return total
}
