package trust

import (
	"os"
	"slices"
	"sync"
	"time"

	"github.com/open-ldap-studio/open-ldap-studio/internal/prefs"
)

// SchemaVersion of the trust store's on-disk format.
const SchemaVersion = 1

// Store holds certificate trust decisions.
//
// Permanent decisions are written to disk; session decisions live only in
// memory and are gone at exit, which is the whole difference between the two
// (FR-008).
type Store struct {
	mu        sync.RWMutex
	path      string
	permanent []Decision
	session   []Decision
}

// OpenStore loads the permanent decisions at path.
func OpenStore(path string) (*Store, error) {
	s := &Store{path: path}

	var loaded []Decision
	_, err := prefs.Read(path, SchemaVersion, &loaded)
	switch {
	case err == nil:
		s.permanent = loaded
	case os.IsNotExist(err):
	default:
		return nil, err
	}
	return s, nil
}

// Trusted reports whether this exact certificate has been accepted for this
// host and port.
//
// The fingerprint is the identity of the decision. A certificate that differs
// from the accepted one is untrusted even on a host that was trusted a moment
// ago — that is the case renewal and interception both look like, and only the
// user can tell them apart.
func (s *Store) Trusted(host string, port int, fingerprint string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	match := func(d Decision) bool {
		return d.Host == host && d.Port == port && d.Fingerprint == fingerprint
	}
	return slices.ContainsFunc(s.permanent, match) || slices.ContainsFunc(s.session, match)
}

// Decide records the user's answer to a trust challenge.
func (s *Store) Decide(d Decision) error {
	d.AcceptedAt = time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if d.Scope == ScopeSession {
		s.session = append(s.session, d)
		return nil
	}
	s.permanent = append(s.permanent, d)
	return prefs.Write(s.path, SchemaVersion, "trust", s.permanent)
}

// List returns every decision, permanent first.
func (s *Store) List() []Decision {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append(append([]Decision{}, s.permanent...), s.session...)
}

// Revoke removes every decision for a fingerprint, permanent and session
// alike. The next connection to that certificate raises a fresh challenge.
func (s *Store) Revoke(fingerprint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	drop := func(d Decision) bool { return d.Fingerprint == fingerprint }
	s.session = slices.DeleteFunc(s.session, drop)

	before := len(s.permanent)
	s.permanent = slices.DeleteFunc(s.permanent, drop)
	if len(s.permanent) == before {
		return nil
	}
	return prefs.Write(s.path, SchemaVersion, "trust", s.permanent)
}
