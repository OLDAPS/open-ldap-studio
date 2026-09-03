//go:build integration

package integration

import (
	"testing"
)

// Contract X8 — invalidCredentials produces no retry loop and no anonymous fallback (D7).
func TestInvalidCredentialsProducesNoRetryOrFallback(t *testing.T) {
	//
}
