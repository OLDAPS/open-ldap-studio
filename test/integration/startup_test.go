//go:build integration

package integration

import (
	"testing"
)

// SC-009 cold start with a stored credential present reaches a usable window in under 3 s with no prompt.
func TestStartupTime(t *testing.T) {
	//
}
