package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

// A Secret must be impossible to serialise or print by accident. These tests
// are the runtime half of contracts C3, E1 and F3: the type is what stops a
// secret reaching a profile file, a log line, or an event payload.

func TestSecretCannotBeSerialised(t *testing.T) {
	s := NewSecret([]byte("hunter2"))

	if _, err := json.Marshal(s); err == nil {
		t.Fatal("json.Marshal(Secret) succeeded; a Secret must have no working marshaller")
	}
	if _, err := json.Marshal(struct {
		Secret Secret `json:"secret"`
	}{s}); err == nil {
		t.Fatal("a struct embedding a Secret was serialisable; the failure must propagate")
	}
}

func TestSecretIsUnprintable(t *testing.T) {
	s := NewSecret([]byte("hunter2"))
	for _, format := range []string{"%v", "%s", "%q", "%#v", "%+v", "%x"} {
		if got := fmt.Sprintf(format, s); bytes.Contains([]byte(got), []byte("hunter2")) {
			t.Errorf("fmt %s leaked the secret: %s", format, got)
		}
	}
}

func TestSecretZeroWipesTheMaterial(t *testing.T) {
	backing := []byte("hunter2")
	s := NewSecret(backing)
	s.Zero()

	if !s.IsZero() {
		t.Error("Zero left the secret populated")
	}
	if !bytes.Equal(backing, make([]byte, len(backing))) {
		t.Errorf("Zero did not wipe the backing array: %q", backing)
	}
}

// The session store is the fallback that keeps the application usable with no
// credential service at all (FR-005).

func TestSessionStoreRoundTrip(t *testing.T) {
	s := NewSession()

	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) = %v, want ErrNotFound", err)
	}
	if err := s.Set("ref-1", NewSecret([]byte("hunter2"))); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("ref-1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), []byte("hunter2")) {
		t.Errorf("round trip returned %q", got.Bytes())
	}
	if err := s.Delete("ref-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("ref-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after Delete, Get = %v, want ErrNotFound", err)
	}
	// Deleting an absent reference is not an error.
	if err := s.Delete("ref-1"); err != nil {
		t.Errorf("deleting an absent reference returned %v", err)
	}
}

func TestSessionStoreIsAlwaysAvailableAndSaysWhatItIs(t *testing.T) {
	ok, reason := NewSession().Available()
	if !ok {
		t.Fatal("the fallback store reported itself unavailable")
	}
	if reason == "" {
		t.Error("the fallback store must state that secrets last for the session only")
	}
}

func TestSessionZeroWipesEverythingItHolds(t *testing.T) {
	s := NewSession()
	backing := []byte("hunter2")
	if err := s.Set("ref-1", NewSecret(backing)); err != nil {
		t.Fatal(err)
	}
	s.Zero()

	if !bytes.Equal(backing, make([]byte, len(backing))) {
		t.Errorf("Zero left material in memory: %q", backing)
	}
	if _, err := s.Get("ref-1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Zero left the reference resolvable")
	}
}

// TestPlatformProviderRoundTrip exercises the real platform binding. It is
// skipped where no agent is reachable — a locked or absent agent is a
// recoverable state, not a test failure (FR-005).
func TestPlatformProviderRoundTrip(t *testing.T) {
	p, err := openPlatform()
	if err != nil {
		t.Skipf("no platform credential service: %v", err)
	}
	if ok, reason := p.Available(); !ok {
		t.Skipf("credential service unavailable: %s", reason)
	}

	const ref = "test.open-ldap-studio.roundtrip"
	t.Cleanup(func() { _ = p.Delete(ref) })

	if err := p.Set(ref, NewSecret([]byte("hunter2"))); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := p.Get(ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got.Bytes(), []byte("hunter2")) {
		t.Errorf("round trip returned %q", got.Bytes())
	}
	if err := p.Delete(ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := p.Get(ref); !errors.Is(err, ErrNotFound) {
		t.Errorf("after Delete, Get = %v, want ErrNotFound", err)
	}
}

func TestOpenFallsBackToTheSessionStoreWithAReason(t *testing.T) {
	p, reason := Open()
	if p == nil {
		t.Fatal("Open returned no provider; it must always return one")
	}
	if p.Name() == "session only" && reason == "" {
		t.Error("falling back to the session store must state why")
	}
}
