package bridge

import (
	"errors"
	"fmt"

	"github.com/open-ldap-studio/open-ldap-studio/internal/credentials"
	"github.com/open-ldap-studio/open-ldap-studio/internal/logging"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// secretRef is the reference a profile's own bind secret is filed under.
//
// The profile stores this string in CredentialID, which connections.secretFor
// already resolves through the provider. So a connection that carries its own
// password needs no Credential record standing between it and the platform
// store — the shared-credential model (screen 3b) layers on top of this rather
// than replacing it.
func secretRef(profileID string) string { return "profile/" + profileID }

// StoreProfileSecret puts a bind secret in the platform credential store and
// points the profile at it.
//
// The secret arrives as a string because that is what a webview can send, and
// it is turned into a secrets.Secret immediately: from here on it is a type
// that cannot be serialised, logged or put in an event payload. It is never
// written to the profile — the profile only learns the reference.
//
// An empty secret removes the stored one, which is how a connection is moved
// back to prompting.
func (b *Bridge) StoreProfileSecret(profileID, secret string) error {
	p, err := b.profiles.Get(profileID)
	if err != nil {
		return err
	}

	ref := secretRef(profileID)

	if secret == "" {
		_ = b.secrets.Delete(ref)
		if p.CredentialID == ref {
			p.CredentialID = ""
			_, err = b.profiles.Save(p)
		}
		return err
	}

	// Not zeroed afterwards: the session provider stores the Secret by value
	// and shares its backing slice, so wiping here would wipe what was stored.
	// Ownership passes to the provider at Set.
	if err := b.secrets.Set(ref, secrets.NewSecret([]byte(secret))); err != nil {
		return fmt.Errorf("could not store the secret in %s: %w", b.secrets.Name(), err)
	}

	if p.CredentialID != ref {
		p.CredentialID = ref
		if _, err := b.profiles.Save(p); err != nil {
			return err
		}
	}

	_ = b.logs.Sink(logging.Application).Record("stored a bind secret for %q in %s", p.Name, b.secrets.Name())
	return nil
}

// ProfileHasSecret reports whether a bind secret is on file, without returning
// it or saying anything about its contents.
func (b *Bridge) ProfileHasSecret(profileID string) bool {
	p, err := b.profiles.Get(profileID)
	if err != nil || p.CredentialID == "" {
		return false
	}
	s, err := b.secrets.Get(p.CredentialID)
	if err != nil {
		return false
	}
	defer s.Zero()
	return !s.IsZero()
}

// ForgetProfileSecret removes a stored secret, leaving the profile in place.
func (b *Bridge) ForgetProfileSecret(profileID string) error {
	return b.StoreProfileSecret(profileID, "")
}

// TestBind binds an already-saved profile with a secret supplied for this
// attempt only, and closes the connection again.
//
// Nothing is stored and no connection is left open: this answers "would these
// credentials work" without making them the ones in use.
func (b *Bridge) TestBind(profileID, secret string) (TestResult, error) {
	p, err := b.profiles.Get(profileID)
	if err != nil {
		return TestResult{}, err
	}
	return b.probe(p, secret, true)
}

// ListCredentials returns metadata for shared credentials.
//
// The shared-credential model of screen 3b is not built (deviation D2): a
// connection currently carries its own secret through StoreProfileSecret. This
// returns empty rather than failing, because an empty list is the truth.
func (b *Bridge) ListCredentials() ([]credentials.Credential, error) {
	return []credentials.Credential{}, nil
}

func (b *Bridge) GetCredential(id string) (credentials.Credential, error) {
	return credentials.Credential{}, errors.New("shared credentials are not implemented; a connection holds its own secret")
}

func (b *Bridge) SaveCredential(c credentials.Credential) (credentials.Credential, error) {
	return c, errors.New("shared credentials are not implemented; use StoreProfileSecret")
}

func (b *Bridge) DeleteCredential(id string) error {
	return errors.New("shared credentials are not implemented")
}

func (b *Bridge) AssignCredential(profileID, credID string) error {
	return errors.New("shared credentials are not implemented")
}

func (b *Bridge) UnassignCredential(profileID string) error {
	return errors.New("shared credentials are not implemented")
}

func (b *Bridge) CredentialAssignments(credID string) ([]string, error) {
	return []string{}, nil
}
