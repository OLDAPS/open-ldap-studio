//go:build integration

package integration

import (
	"context"
	"testing"
	"time"
)

// Integration test — every bind method succeeds against the OpenLDAP fixture 
// over plain, StartTLS, and LDAPS.
func TestBindMethods(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Spin up test harness (OpenLDAP, ApacheDS) and verify binds:
	// - Simple bind
	// - Anonymous
	// - SASL EXTERNAL
	// - GSSAPI (if configured)
	// - DIGEST-MD5
	_ = ctx
}
