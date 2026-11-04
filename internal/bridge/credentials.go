package bridge

import (
	"errors"

	"github.com/open-ldap-studio/open-ldap-studio/internal/credentials"
)

func (b *Bridge) StoreProfileSecret(id, secret string) error {
	return b.app.StoreProfileSecret(id, secret)
}
func (b *Bridge) ProfileHasSecret(id string) bool     { return b.app.ProfileHasSecret(id) }
func (b *Bridge) ForgetProfileSecret(id string) error { return b.app.StoreProfileSecret(id, "") }
func (b *Bridge) TestBind(id, secret string) (TestResult, error) {
	result, err := b.app.TestBind(b.context(), id, secret)
	return TestResult(result), err
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
