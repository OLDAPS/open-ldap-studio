package bridge

import (
	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// The write surface, in full.
//
// Preview, Commit and Discard are the only methods here, and Commit takes a
// token and nothing else. There is no Modify, Add, Delete or Rename on this
// bridge, which is what makes SC-005 checkable by construction rather than by
// review (bridge-api.md Rule 1, contract C1).

// Preview builds the diff, counts what would be affected, and issues a
// single-use token.
//
// It is refused outright for a read-only connection, and for a production
// connection without its extra confirmation — refused here, below the UI, so
// no frontend path can route around it (contract C13).
func (b *Bridge) Preview(input changeset.Input, confirmedProduction bool) (changeset.Preview, error) {
	return b.changes.Preview(b.context(), input, confirmedProduction)
}

// Pending returns the change set a token would commit, so a preview can be
// redisplayed without rebuilding it.
func (b *Bridge) Pending(token string) (changeset.ChangeSet, error) {
	return b.changes.Pending(token)
}

// Discard drops a token and keeps the draft: cancelling a confirmation must
// not throw away the work behind it (FR-040).
func (b *Bridge) Discard(token string) error {
	return b.changes.Discard(token)
}

// Commit dispatches the change the token vouches for.
//
// It returns the server's verbatim Result on success and on failure. A commit
// touching more than one entry runs as a job so it is cancellable and reports
// one outcome per entry (FR-097).
func (b *Bridge) Commit(token string) (ldapx.Result, error) {
	return b.changes.Commit(b.context(), token)
}

// CommitAsJob dispatches a multi-entry change as a cancellable job, reporting
// one outcome per entry and ending in partiallyComplete where the outcomes are
// mixed (FR-050).
func (b *Bridge) CommitAsJob(token string) (string, error) {
	return b.changes.CommitAsJob(b.context(), token)
}
