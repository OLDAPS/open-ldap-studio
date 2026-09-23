package bridge

import (
	"context"
	"errors"

	"github.com/open-ldap-studio/open-ldap-studio/internal/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// ListProfiles returns the connection list.
func (b *Bridge) ListProfiles() []profiles.Summary { return b.profiles.List() }

// GetProfile returns one profile. It cannot return a secret, because a Profile
// has no field that could hold one.
func (b *Bridge) GetProfile(id string) (profiles.Profile, error) { return b.profiles.Get(id) }

// SaveProfile stores a connection profile.
//
// It takes the payload as JSON and decodes it strictly, so a payload carrying
// a password field is refused rather than silently dropped. The frontend sends
// JSON.stringify(profile); the strictness is the point, and it is the runtime
// half of "a profile has nowhere to put a secret" (SC-008).
func (b *Bridge) SaveProfile(payload string) (profiles.Profile, error) {
	p, err := profiles.DecodeStrict([]byte(payload))
	if err != nil {
		return profiles.Profile{}, err
	}
	return b.profiles.Save(p)
}

// DeleteProfile removes a connection profile and the secret filed under it.
//
// Dropping the profile without the secret would leave material in the platform
// store that nothing in this application can reach or name any more.
func (b *Bridge) DeleteProfile(id string) error {
	if err := b.Disconnect(id); err != nil && !errors.Is(err, connections.ErrNotConnected) {
		return err
	}
	_ = b.secrets.Delete(secretRef(id))
	return b.profiles.Delete(id)
}

// DuplicateProfile copies a profile, including its credential assignment.
func (b *Bridge) DuplicateProfile(id string) (profiles.Profile, error) {
	return b.profiles.Duplicate(id)
}

// ListFolders returns the connection folders.
func (b *Bridge) ListFolders() []profiles.Folder { return b.profiles.Folders() }

// SaveFolder creates or renames a folder, rejecting a cycle.
func (b *Bridge) SaveFolder(f profiles.Folder) (profiles.Folder, error) {
	return b.profiles.SaveFolder(f)
}

// Connect opens a connection as a cancellable job: the bind may raise a
// platform credential prompt, and that must not block the window (FR-005).
func (b *Bridge) Connect(profileID string) (string, error) {
	p, err := b.profiles.Get(profileID)
	if err != nil {
		return "", err
	}

	id := b.jobs.Start(b.context(), jobs.KindConnect, jobs.ModeExecute, profileID, 0,
		func(ctx context.Context, reporter *jobs.Reporter) error {
			reporter.Progress(0, 1, "connecting to "+p.Host)
			conn, err := b.conns.Dial(ctx, p, nil)
			if err != nil {
				return err
			}
			reporter.Progress(1, 1, "connected")

			// Read the Root DSE once and announce anything the server cannot
			// do, before an operation fails for a reason nobody explained.
			dse, _, err := ldapx.ReadRootDSE(ctx, conn)
			if err == nil {
				for _, missing := range dse.MissingCapabilities() {
					b.Emit("capability:unavailable", map[string]any{
						"profileId":  profileID,
						"capability": missing.Capability,
						"reason":     missing.Reason,
					})
				}
			}
			return nil
		})
	return id, nil
}

// Disconnect closes a connection.
func (b *Bridge) Disconnect(profileID string) error {
	return b.conns.Disconnect(profileID)
}

// ConnectionState returns what the status bar shows for a profile.
func (b *Bridge) ConnectionState(profileID string) ConnState { return b.conns.State(profileID) }

// ConnectionStates returns the state of every profile the session has touched.
func (b *Bridge) ConnectionStates() []ConnState {
	return b.conns.States()
}

// WhoAmI asks the server which identity it believes is bound (FR-011).
func (b *Bridge) WhoAmI(profileID string) (string, ldapx.Result, error) {
	conn, err := b.conns.Conn(profileID)
	if err != nil {
		return "", ldapx.Result{}, err
	}
	return ldapx.WhoAmI(b.context(), conn)
}

// RootDSE returns what the server publishes about itself (FR-012).
func (b *Bridge) RootDSE(profileID string) (ldapx.RootDSE, ldapx.Result, error) {
	conn, err := b.conns.Conn(profileID)
	if err != nil {
		return ldapx.RootDSE{}, ldapx.Result{}, err
	}
	return ldapx.ReadRootDSE(b.context(), conn)
}

// ValidateFilter reports whether a filter parses. It never contacts a server
// and never returns a rewritten filter (FR-028).
func (b *Bridge) ValidateFilter(filter string) ldapx.FilterDiagnostic {
	return ldapx.ValidateFilter(filter)
}

// ListChildren fetches one page of a container's children, reporting a
// server-side truncation rather than showing a short list as complete
// (FR-018).
func (b *Bridge) ListChildren(profileID, dn string, page ldapx.PageRequest) (ldapx.Page, error) {
	conn, err := b.conns.Conn(profileID)
	if err != nil {
		return ldapx.Page{}, err
	}

	result, degradations, err := ldapx.ListChildren(b.context(), conn, dn, page)
	b.announce(profileID, degradations)
	return result, err
}

// ReadEntry reads one entry, optionally with its operational attributes.
func (b *Bridge) ReadEntry(profileID, dn string, opts ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error) {
	conn, err := b.conns.Conn(profileID)
	if err != nil {
		return ldapx.Entry{}, ldapx.Result{}, err
	}
	return ldapx.ReadEntry(b.context(), conn, dn, opts)
}

// announce turns each degradation into a capability:unavailable event. A
// degradation the user is not told about is the failure SC-016 tests for.
func (b *Bridge) announce(profileID string, degradations []ldapx.Degradation) {
	for _, d := range degradations {
		b.Emit("capability:unavailable", map[string]any{
			"profileId": profileID, "capability": d.Capability, "reason": d.Reason,
		})
	}
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
func (b *Bridge) RefreshEntry(profileID, dn string, opts ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error) {
	return b.ReadEntry(profileID, dn, opts)
}

// GoToDN fetches an entry directly.
func (b *Bridge) GoToDN(profileID, dn string) (ldapx.Entry, ldapx.Result, error) {
	return b.ReadEntry(profileID, dn, ldapx.ReadOptions{})
}

// CountSubtree counts elements for subtree deletion.
func (b *Bridge) CountSubtree(profileID, dn string) (string, error) {
	return "", errors.New("not implemented")
}
