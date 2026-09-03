package bridge

import (
	"context"

	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/logging"
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
	preview, err := b.pipeline.Preview(b.context(), input, confirmedProduction)
	if err != nil {
		return changeset.Preview{}, err
	}
	return preview, nil
}

// Pending returns the change set a token would commit, so a preview can be
// redisplayed without rebuilding it.
func (b *Bridge) Pending(token string) (changeset.ChangeSet, error) {
	return b.pipeline.Pending(token)
}

// Discard drops a token and keeps the draft: cancelling a confirmation must
// not throw away the work behind it (FR-040).
func (b *Bridge) Discard(token string) error {
	return b.pipeline.Discard(token)
}

// Commit dispatches the change the token vouches for.
//
// It returns the server's verbatim Result on success and on failure. A commit
// touching more than one entry runs as a job so it is cancellable and reports
// one outcome per entry (FR-097).
func (b *Bridge) Commit(token string) (ldapx.Result, error) {
	cs, err := b.pipeline.Pending(token)
	if err != nil {
		return ldapx.Result{}, err
	}

	result, err := b.pipeline.Commit(b.context(), token, nil)

	// The modification log is written as LDIF, redacted before it reaches the
	// sink (FR-090, FR-093).
	_ = b.logs.Sink(logging.Modification).Record("# %s %s on %s -> %s",
		cs.Kind, cs.DNs(), cs.ProfileID, result.String())

	if err != nil {
		_ = b.logs.Sink(logging.Errors).Record("commit %s: %v", cs.ID, err)
		return result, err
	}

	for _, dn := range cs.DNs() {
		b.Emit("entry:changed", map[string]any{
			"profileId": cs.ProfileID, "dn": dn, "source": "commit",
		})
	}
	return result, nil
}

// CommitAsJob dispatches a multi-entry change as a cancellable job, reporting
// one outcome per entry and ending in partiallyComplete where the outcomes are
// mixed (FR-050).
func (b *Bridge) CommitAsJob(token string) (string, error) {
	cs, err := b.pipeline.Pending(token)
	if err != nil {
		return "", err
	}

	id := b.jobs.Start(b.context(), jobs.KindCommit, jobs.ModeExecute, cs.ProfileID, len(cs.Ops),
		func(ctx context.Context, reporter *jobs.Reporter) error {
			_, err := b.pipeline.Commit(ctx, token, reporter)
			for _, dn := range cs.DNs() {
				b.Emit("entry:changed", map[string]any{
					"profileId": cs.ProfileID, "dn": dn, "source": "commit",
				})
			}
			return err
		})
	return id, nil
}

func (b *Bridge) context() context.Context {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.ctx == nil {
		return context.Background()
	}
	return b.ctx
}
