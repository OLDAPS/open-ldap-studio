package bridge

import (
	"errors"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

func (b *Bridge) ListProfiles() []profiles.Summary               { return b.app.ListProfiles() }
func (b *Bridge) GetProfile(id string) (profiles.Profile, error) { return b.app.GetProfile(id) }
func (b *Bridge) SaveProfile(payload string) (profiles.Profile, error) {
	profile, err := profiles.DecodeStrict([]byte(payload))
	if err != nil {
		return profiles.Profile{}, err
	}
	return b.app.SaveProfile(profile)
}
func (b *Bridge) DeleteProfile(id string) error { return b.app.DeleteProfile(id) }
func (b *Bridge) DuplicateProfile(id string) (profiles.Profile, error) {
	return b.app.DuplicateProfile(id)
}
func (b *Bridge) ListFolders() []profiles.Folder { return b.app.ListFolders() }
func (b *Bridge) SaveFolder(folder profiles.Folder) (profiles.Folder, error) {
	return b.app.SaveFolder(folder)
}
func (b *Bridge) Connect(id string) (string, error)   { return b.app.Connect(b.context(), id) }
func (b *Bridge) Disconnect(id string) error          { return b.app.Disconnect(id) }
func (b *Bridge) ConnectionState(id string) ConnState { return b.app.ConnectionState(id) }
func (b *Bridge) ConnectionStates() []ConnState       { return b.app.ConnectionStates() }
func (b *Bridge) WhoAmI(id string) (IdentityResult, error) {
	identity, result, err := b.app.WhoAmI(b.context(), id)
	return IdentityResult{Identity: identity, Result: result}, err
}

func (b *Bridge) RootDSE(id string) (RootDSEResult, error) {
	dse, result, err := b.app.RootDSE(b.context(), id)
	return RootDSEResult{RootDSE: dse, Result: result}, err
}

func (b *Bridge) ValidateFilter(filter string) ldapx.FilterDiagnostic {
	return b.app.ValidateFilter(filter)
}

func (b *Bridge) ListChildren(id, dn string, page ldapx.PageRequest) (ldapx.Page, error) {
	return b.app.ListChildren(b.context(), id, dn, page)
}

func (b *Bridge) ReadEntry(id, dn string, options ldapx.ReadOptions) (EntryResult, error) {
	entry, result, err := b.app.ReadEntry(b.context(), id, dn, options)
	return EntryResult{Entry: entry, Result: result}, err
}

// DeleteFolder deletes a folder by ID.
func (b *Bridge) DeleteFolder(id string) error {
	return errors.New("not implemented")
}

// ExportProfiles exports all profiles (excluding credentials).
func (b *Bridge) ExportProfiles() (string, error) {
	return "", errors.New("not implemented")
}

// ImportProfiles imports profiles from a bundle.
func (b *Bridge) ImportProfiles(payload string) error {
	return errors.New("not implemented")
}

// MoveProfile changes a profile's folder.
func (b *Bridge) MoveProfile(id, folderID string) error {
	return errors.New("not implemented")
}

// RefreshEntry reloads an entry.
func (b *Bridge) RefreshEntry(profileID, dn string, opts ldapx.ReadOptions) (EntryResult, error) {
	return b.ReadEntry(profileID, dn, opts)
}

// GoToDN fetches an entry directly.
func (b *Bridge) GoToDN(profileID, dn string) (EntryResult, error) {
	return b.ReadEntry(profileID, dn, ldapx.ReadOptions{})
}

// CountSubtree counts elements for subtree deletion.
func (b *Bridge) CountSubtree(profileID, dn string) (string, error) {
	return "", errors.New("not implemented")
}
