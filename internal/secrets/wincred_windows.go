//go:build windows

package secrets

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/danieljoos/wincred"
)

// Windows binding over the Credential Manager API, in process.
type winCred struct{}

func openPlatform() (Provider, error) { return winCred{}, nil }

func (winCred) Name() string { return "Credential Manager" }

func (winCred) Available() (bool, string) {
	// Listing is the cheapest call that proves the API answers.
	if _, err := wincred.List(); err != nil {
		return false, fmt.Sprintf("the Credential Manager did not answer: %v", err)
	}
	return true, ""
}

func target(ref string) string { return Service + ":" + ref }

func (winCred) Get(ref string) (Secret, error) {
	cred, err := wincred.GetGenericCredential(target(ref))
	if err != nil {
		if errors.Is(err, syscall.Errno(1168)) { // ERROR_NOT_FOUND
			return Secret{}, ErrNotFound
		}
		return Secret{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return NewSecret(cred.CredentialBlob), nil
}

func (winCred) Set(ref string, s Secret) error {
	cred := wincred.NewGenericCredential(target(ref))
	cred.CredentialBlob = s.Bytes()
	cred.UserName = ref
	// LOCAL_MACHINE persistence would survive to other sessions; the secret
	// belongs to this user's session lifetime as the agent defines it.
	cred.Persist = wincred.PersistLocalMachine
	if err := cred.Write(); err != nil {
		return fmt.Errorf("secrets: cannot store the credential: %w", err)
	}
	return nil
}

func (winCred) Delete(ref string) error {
	cred, err := wincred.GetGenericCredential(target(ref))
	if err != nil {
		return nil // absent is not an error
	}
	if err := cred.Delete(); err != nil {
		return fmt.Errorf("secrets: cannot delete the credential: %w", err)
	}
	return nil
}
