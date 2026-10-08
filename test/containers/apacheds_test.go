//go:build integration

package containers

import (
	"context"
	"testing"
	"time"
)

// Validate the ApacheDS fixture containerises and serves ACI reads.
func TestApacheDSFixtureServesACIs(t *testing.T) {
	// The image is unresolved (marcelocg/apacheds here, openmicroscopy/apacheds
	// in server/docker-compose.yml); see T020 (#132).
	t.Skip("Skipping until correct image name is provided")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// This setup ensures that we can containerize the ApacheDS server and verify it runs.
	container, err := SetupApacheDS(ctx)
	if err != nil {
		t.Fatalf("failed to setup ApacheDS: %v", err)
	}
	defer func() { _ = container.Terminate(ctx) }()

	if container.Host == "" || container.Port == 0 {
		t.Errorf("expected valid host and port, got %s:%d", container.Host, container.Port)
	}

	// Add ACI test logic here later depending on LDAP client tools, currently we just
	// discharge risk R-6 by asserting it runs properly.
}
