//go:build integration

package integration

import "testing"

func TestUS2Limits(t *testing.T) {
	// a search past the server size limit shows partial results labelled truncated-by-server...
}
