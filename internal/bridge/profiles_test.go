package bridge

import (
	"testing"
)

// Contract C3 — SaveProfile rejects every payload carrying secret material.
//
// The requirement is that the bridge strictly enforces this. Even if a caller
// tries to pass a password in the payload, the strict json decoding in SaveProfile
// must reject it.
func TestSaveProfileRejectsSecretMaterial(t *testing.T) {
	// A new bridge is fine to test method parsing logic, although we might
	// need a mock profile store. But we can test it just expects a failure.
	b := &Bridge{}

	forbidden := []string{
		`{"id":"p1", "password": "secure"}`,
		`{"id":"p1", "secret": "hunter2"}`,
		`{"id":"p1", "tls": {"password": "test"}}`,
	}

	for _, payload := range forbidden {
		_, err := b.SaveProfile(payload)
		if err == nil {
			t.Errorf("expected SaveProfile to reject payload %s, but got nil error", payload)
		}
		// In profiles.DecodeStrict (from profiles package), unknown fields cause
		// an unmarshal error. We just ensure it failed.
	}
}
