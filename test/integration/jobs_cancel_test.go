//go:build integration

package integration

import (
	"testing"
)

// Contract C8 — every JobID-returning method has a Cancel effective within 2 s.
func TestJobsCancellationIsEffectiveWithinTwoSeconds(t *testing.T) {
	// 1. Identify all JobID-returning methods on the Bridge.
	// 2. Dispatch a long-running job using each method against the test containers.
	// 3. Issue a Cancel command and measure the time it takes for the job to reach
	//    a cancelled or terminal state.
	// 4. Fail the test if the delay exceeds 2 seconds.
}
