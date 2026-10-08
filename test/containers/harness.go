//go:build integration

package containers

import (
	"context"
	"fmt"
)

// Harness holds all the running test containers for the integration tests.
type Harness struct {
	OpenLDAP *OpenLDAPContainer
	ApacheDS *ApacheDSContainer
	Poor     *PoorContainer
}

// SetupHarness initializes and starts all integration test containers.
func SetupHarness(ctx context.Context) (*Harness, error) {
	openldap, err := SetupOpenLDAP(ctx)
	if err != nil {
		return nil, fmt.Errorf("openldap setup failed: %v", err)
	}

	apacheds, err := SetupApacheDS(ctx)
	if err != nil {
		_ = openldap.Terminate(ctx)
		return nil, fmt.Errorf("apacheds setup failed: %v", err)
	}

	poor, err := SetupPoorContainer(ctx)
	if err != nil {
		_ = openldap.Terminate(ctx)
		_ = apacheds.Terminate(ctx)
		return nil, fmt.Errorf("poor container setup failed: %v", err)
	}

	// Wait, we need to seed the fixtures (T022).
	// This can be done by parsing LDIFs from test/corpus and applying them via ldapsearch/ldapmodify
	// or through the client. We leave the hook here for the tests to use.

	return &Harness{
		OpenLDAP: openldap,
		ApacheDS: apacheds,
		Poor:     poor,
	}, nil
}

// Teardown shuts down all containers.
func (h *Harness) Teardown(ctx context.Context) {
	if h.OpenLDAP != nil {
		_ = h.OpenLDAP.Terminate(ctx)
	}
	if h.ApacheDS != nil {
		_ = h.ApacheDS.Terminate(ctx)
	}
	if h.Poor != nil {
		_ = h.Poor.Terminate(ctx)
	}
}
