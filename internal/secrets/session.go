package secrets

import "sync"

// Session is the in-memory, session-only fallback used when no platform
// credential service is reachable.
//
// It is not a vault: nothing is written to disk, nothing survives exit, and
// there is no unlock step. The application stays usable with no credential
// store at all, at the cost of prompting once per session (FR-005).
type Session struct {
	mu      sync.Mutex
	secrets map[string]Secret
}

// NewSession returns an empty session store.
func NewSession() *Session {
	return &Session{secrets: make(map[string]Secret)}
}

func (s *Session) Get(ref string) (Secret, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sec, ok := s.secrets[ref]
	if !ok {
		return Secret{}, ErrNotFound
	}
	return sec, nil
}

func (s *Session) Set(ref string, sec Secret) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.secrets[ref]; ok {
		old.Zero()
	}
	s.secrets[ref] = sec
	return nil
}

func (s *Session) Delete(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.secrets[ref]; ok {
		old.Zero()
		delete(s.secrets, ref)
	}
	return nil
}

// Available is always true — the session store is the fallback, so it cannot
// itself be unavailable. The reason explains what the user is getting.
func (s *Session) Available() (bool, string) {
	return true, "secrets are held for this session only and are never written to disk"
}

func (s *Session) Name() string { return "session only" }

// Zero wipes every held secret. Called on exit and when the last connection
// closes.
func (s *Session) Zero() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ref, sec := range s.secrets {
		sec.Zero()
		delete(s.secrets, ref)
	}
}
