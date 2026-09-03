//go:build darwin

package secrets

import (
	"errors"
	"fmt"

	keychain "github.com/keybase/go-keychain"
)

// macOS binding over Security.framework, in process.
//
// zalando/go-keyring was rejected for this platform: its darwin path imports
// os/exec and shells out to /usr/bin/security, which Constitution II forbids
// on the default path (research R2, verified in that library's source).
type macKeychain struct{}

func openPlatform() (Provider, error) { return macKeychain{}, nil }

func (macKeychain) Name() string { return "Keychain" }

func (macKeychain) Available() (bool, string) {
	// A query for a reference that cannot exist tells us whether the keychain
	// answers at all, without touching a real secret.
	q := keychain.NewItem()
	q.SetSecClass(keychain.SecClassGenericPassword)
	q.SetService(Service)
	q.SetAccount("\x00probe")
	q.SetMatchLimit(keychain.MatchLimitOne)
	if _, err := keychain.QueryItem(q); err != nil && !errors.Is(err, keychain.ErrorItemNotFound) {
		return false, fmt.Sprintf("the login keychain did not answer: %v", err)
	}
	return true, ""
}

func item(ref string) keychain.Item {
	i := keychain.NewItem()
	i.SetSecClass(keychain.SecClassGenericPassword)
	i.SetService(Service)
	i.SetAccount(ref)
	return i
}

func (macKeychain) Get(ref string) (Secret, error) {
	q := item(ref)
	q.SetMatchLimit(keychain.MatchLimitOne)
	q.SetReturnData(true)

	results, err := keychain.QueryItem(q)
	if err != nil {
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return Secret{}, ErrNotFound
		}
		return Secret{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if len(results) == 0 {
		return Secret{}, ErrNotFound
	}
	return NewSecret(results[0].Data), nil
}

func (m macKeychain) Set(ref string, s Secret) error {
	i := item(ref)
	i.SetLabel(Service + ": " + ref)
	i.SetData(s.Bytes())
	// Unlock policy belongs to the agent, so the item is accessible whenever
	// the keychain itself is unlocked — we never extend or cache beyond that.
	i.SetAccessible(keychain.AccessibleWhenUnlocked)

	err := keychain.AddItem(i)
	if errors.Is(err, keychain.ErrorDuplicateItem) {
		if err := keychain.UpdateItem(item(ref), i); err != nil {
			return fmt.Errorf("secrets: cannot update the keychain item: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("secrets: cannot store the keychain item: %w", err)
	}
	return nil
}

func (macKeychain) Delete(ref string) error {
	if err := keychain.DeleteItem(item(ref)); err != nil && !errors.Is(err, keychain.ErrorItemNotFound) {
		return fmt.Errorf("secrets: cannot delete the keychain item: %w", err)
	}
	return nil
}
