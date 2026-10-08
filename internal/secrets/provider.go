package secrets

import (
	"errors"
	"fmt"
)

// Service is the collection name every stored secret is filed under.
const Service = "Open LDAP Studio"

// Secret holds bind material in memory and nowhere else.
//
// It has no marshaller — it cannot be serialised by accident — and its String
// and Format methods hide the bytes, so a stray %v in a log line cannot leak
// it. Zero it as soon as the bind is done (FR-003, FR-077, data-model §1).
type Secret struct {
	b []byte
}

// NewSecret takes ownership of b. The caller must not retain or reuse it.
func NewSecret(b []byte) Secret { return Secret{b: b} }

// Bytes exposes the material for exactly one purpose: handing it to a bind.
// Nothing else may call it, and no caller may copy the slice elsewhere.
func (s Secret) Bytes() []byte { return s.b }

// Len reports the length without exposing the value.
func (s Secret) Len() int { return len(s.b) }

// IsZero reports whether the secret is empty.
func (s Secret) IsZero() bool { return len(s.b) == 0 }

// Zero wipes the material. Call it on disconnect and at exit.
func (s *Secret) Zero() {
	clear(s.b)
	s.b = nil
}

// String, GoString and Format make a Secret unprintable by any means fmt
// offers, including %v, %s, %q and %#v.
func (Secret) String() string   { return "«secret»" }
func (Secret) GoString() string { return "secrets.Secret{«secret»}" }
func (Secret) Format(f fmt.State, verb rune) {
	_, _ = f.Write([]byte("«secret»"))
	_ = verb
}

// MarshalJSON and MarshalText exist only to fail. Their presence is what makes
// accidental serialisation a compile-time-visible error rather than a silent
// leak into a profile file, a log, or an event payload (contracts C3, E1, F3).
var errNotSerialisable = errors.New("secrets: a Secret cannot be serialised; it lives in memory and the platform store only")

func (Secret) MarshalJSON() ([]byte, error) { return nil, errNotSerialisable }
func (Secret) MarshalText() ([]byte, error) { return nil, errNotSerialisable }

// Provider is the only interface through which a credential is reached.
//
// No package outside internal/secrets may import a platform credential API,
// and no os/exec may appear in this package's transitive import tree — the
// default path is an in-process binding, never a subprocess (Constitution II,
// contracts C2 and C3, and no_subprocess_test.go).
type Provider interface {
	// Get resolves a reference into secret material.
	Get(ref string) (Secret, error)
	// Set stores material under a reference, replacing anything there.
	Set(ref string, s Secret) error
	// Delete removes the material. Deleting an absent reference is not an error.
	Delete(ref string) error
	// Available reports whether the platform agent can be reached, and why not
	// when it cannot. An unavailable store is recoverable and explained, never
	// silent (FR-005).
	Available() (bool, string)
	// Name identifies the backing store for the UI ("Secret Service",
	// "Keychain", "Credential Manager", "session only").
	Name() string
}

// ErrNotFound is returned by Get when the reference resolves to nothing. It is
// distinct from an unavailable store: one means "no secret", the other means
// "cannot tell".
var ErrNotFound = errors.New("secrets: no secret stored for that reference")

// ErrUnavailable is returned when the platform agent cannot be reached. The
// caller falls through to the session-only path and tells the user why.
var ErrUnavailable = errors.New("secrets: the platform credential service is unavailable")

// Open returns the platform provider, falling back to the session-only store
// when the platform agent cannot be reached.
//
// The fallback is deliberate: the application must remain usable with no
// credential store at all, prompting per session instead (FR-005).
func Open() (Provider, string) {
	p, err := openPlatform()
	if err == nil {
		ok, reason := p.Available()
		if ok {
			return p, ""
		}
		return NewSession(), reason
	}
	return NewSession(), err.Error()
}
