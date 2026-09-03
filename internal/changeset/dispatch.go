package changeset

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// Commit is the only dispatch path in the application.
//
// It accepts a token and nothing else. Every mutation function in ldapx is
// called from this file and from nowhere else, which is what makes SC-005
// mechanically checkable rather than a convention reviewers must police
// (contract C1, bridge-api.md Rule 1).
//
// Reporter is how a multi-entry commit reports per-entry outcomes; it may be
// nil for a single-entry change.
type Reporter interface {
	Outcome(dn, status string, result *ldapx.Result)
	Progress(done, total int, message string)
}

const (
	outcomeSucceeded = "succeeded"
	outcomeFailed    = "failed"
	outcomeSkipped   = "skipped"
)

// Commit dispatches the change set the token vouches for.
//
// Before anything is sent, every touched entry is re-read and its version
// compared with the one the preview was computed against. A mismatch aborts
// the whole commit: the user approved a diff against a state that no longer
// exists (FR-052).
func (p *Pipeline) Commit(ctx context.Context, token string, reporter Reporter) (ldapx.Result, error) {
	held, err := p.take(token)
	if err != nil {
		return ldapx.Result{}, err
	}
	cs := held.changeSet

	// The guard is checked again at commit, not only at preview: a profile can
	// be switched to read-only while a preview is on screen.
	profile, err := p.profiles.Profile(cs.ProfileID)
	if err != nil {
		return ldapx.Result{}, err
	}
	if profile.ReadOnly {
		return ldapx.Result{}, &ErrReadOnly{ProfileID: profile.ID, Name: profile.Name}
	}

	dir, err := p.conns.Directory(cs.ProfileID)
	if err != nil {
		return ldapx.Result{}, err
	}

	if err := p.checkVersions(ctx, dir, held.token, cs); err != nil {
		return ldapx.Result{}, err
	}

	var last ldapx.Result
	total := len(cs.Ops)

	for i, op := range cs.Ops {
		if err := ctx.Err(); err != nil {
			if reporter != nil {
				for _, remaining := range cs.Ops[i:] {
					reporter.Outcome(remaining.DN, outcomeSkipped, nil)
				}
			}
			return last, ldapx.Cancelled("commit")
		}

		result, err := dispatch(ctx, dir, cs, op)
		last = result

		if reporter != nil {
			status := outcomeSucceeded
			if err != nil {
				status = outcomeFailed
			}
			outcome := result
			reporter.Outcome(op.DN, status, &outcome)
			reporter.Progress(i+1, total, op.DN)
		}

		if err != nil {
			// An indeterminate write stops the run: continuing would apply
			// later operations on top of a state nobody can describe.
			if ldapx.CategoryOf(err) == ldapx.CategoryIndeterminate {
				return result, err
			}
			if reporter == nil {
				return result, err
			}
			// A multi-entry run continues so that every entry gets an outcome,
			// and the job ends partiallyComplete rather than at the first
			// failure (FR-097).
		}
	}

	return last, nil
}

// dispatch routes one operation to the directory.
func dispatch(ctx context.Context, dir Directory, cs ChangeSet, op Operation) (ldapx.Result, error) {
	switch op.Type {
	case OpAddEntry:
		return dir.add(ctx, op.DN, op.Attributes)

	case OpDeleteEntry:
		return dir.del(ctx, op.DN, cs.Kind == KindSubtreeDelete)

	case OpRename:
		return dir.rename(ctx, op.DN, op.NewRDN, op.NewSuperior, op.KeepOldRDN)

	case OpAddAttr, OpAddValue:
		return dir.modify(ctx, op.DN, []ldapx.Change{
			{Operation: ldapx.ChangeAdd, Attribute: op.Attribute},
		})

	case OpDeleteAttr, OpDeleteValue:
		return dir.modify(ctx, op.DN, []ldapx.Change{
			{Operation: ldapx.ChangeDelete, Attribute: op.Attribute},
		})

	case OpReplaceAttr:
		return dir.modify(ctx, op.DN, []ldapx.Change{
			{Operation: ldapx.ChangeReplace, Attribute: op.Attribute},
		})

	default:
		return ldapx.Result{}, ldapx.LocalValidationError("commit",
			fmt.Sprintf("unknown operation type %q", op.Type))
	}
}

// checkVersions re-reads every touched entry and compares it with the state
// the preview was computed against.
func (p *Pipeline) checkVersions(ctx context.Context, dir Directory, token Token, cs ChangeSet) error {
	for dn, expected := range token.EntryVersions {
		entry, err := dir.ReadEntry(ctx, dn)
		if err != nil {
			var ldapErr *ldapx.Error
			if errors.As(err, &ldapErr) && ldapErr.Result != nil && ldapErr.Result.Code == ldapx.NoSuchObject {
				if expected == versionAbsent {
					continue // an add whose target is still absent: as previewed
				}
				return &ErrStale{DN: dn, Expected: expected, Actual: versionAbsent}
			}
			return err
		}

		actual := EntryVersion(entry)
		if expected == versionAbsent {
			// Something was created at this DN between preview and commit.
			return &ErrStale{DN: dn, Expected: versionAbsent, Actual: actual}
		}
		if actual != expected {
			return &ErrStale{DN: dn, Expected: expected, Actual: actual}
		}
	}
	return nil
}

// liveDirectory is the only implementation of Directory that speaks to a
// server, and the only place in the application where ldapx's mutation
// functions are called (contract C1).
type liveDirectory struct {
	conn *ldapx.Conn
}

// Live wraps an open connection as the Directory the pipeline writes through.
func Live(conn *ldapx.Conn) Directory { return liveDirectory{conn: conn} }

func (d liveDirectory) ReadEntry(ctx context.Context, dn string) (ldapx.Entry, error) {
	entry, _, err := ldapx.ReadEntry(ctx, d.conn, dn, ldapx.ReadOptions{IncludeOperational: true})
	return entry, err
}

func (d liveDirectory) Identity() string { return d.conn.ServerIdentity() }

func (d liveDirectory) add(ctx context.Context, dn string, attributes []ldapx.Attribute) (ldapx.Result, error) {
	return ldapx.AddEntry(ctx, d.conn, dn, attributes)
}

func (d liveDirectory) modify(ctx context.Context, dn string, changes []ldapx.Change) (ldapx.Result, error) {
	return ldapx.ModifyEntry(ctx, d.conn, dn, changes)
}

func (d liveDirectory) rename(ctx context.Context, dn, newRDN, newSuperior string, keepOldRDN bool) (ldapx.Result, error) {
	return ldapx.RenameEntry(ctx, d.conn, dn, newRDN, newSuperior, keepOldRDN)
}

func (d liveDirectory) del(ctx context.Context, dn string, subtree bool) (ldapx.Result, error) {
	if subtree {
		return ldapx.DeleteSubtree(ctx, d.conn, dn)
	}
	return ldapx.DeleteEntry(ctx, d.conn, dn)
}
