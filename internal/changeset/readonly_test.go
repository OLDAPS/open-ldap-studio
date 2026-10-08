package changeset

import (
	"testing"

	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// Contract test C13 — a read-only profile is refused a preview token at the
// changeset boundary before any UI check.
func TestReadOnlyProfileIsRefusedPreview(t *testing.T) {
	guard := Guard{RequireProductionConfirmation: true}

	// A read-only profile
	readOnlyProfile := profiles.Profile{
		ID:       "p1",
		Name:     "Test Read-Only",
		ReadOnly: true,
	}

	err := guard.Check(readOnlyProfile, true)
	if err == nil {
		t.Fatal("expected read-only profile to be refused, but got no error")
	}

	if _, ok := err.(*ErrReadOnly); !ok {
		t.Fatalf("expected *ErrReadOnly, got %T: %v", err, err)
	}

	// A production-tagged profile without confirmation
	productionProfile := profiles.Profile{
		ID:       "p2",
		Name:     "Test Prod",
		ReadOnly: false,
		Tags:     []string{"production"},
	}

	err = guard.Check(productionProfile, false)
	if err == nil {
		t.Fatal("expected production profile without confirmation to be refused, but got no error")
	}

	if _, ok := err.(*ErrProductionUnconfirmed); !ok {
		t.Fatalf("expected *ErrProductionUnconfirmed, got %T: %v", err, err)
	}

	// A production-tagged profile with confirmation
	err = guard.Check(productionProfile, true)
	if err != nil {
		t.Fatalf("expected production profile with confirmation to be allowed, but got: %v", err)
	}

	// A regular profile
	regularProfile := profiles.Profile{
		ID:       "p3",
		Name:     "Test Regular",
		ReadOnly: false,
	}

	err = guard.Check(regularProfile, false)
	if err != nil {
		t.Fatalf("expected regular profile to be allowed, but got: %v", err)
	}
}
