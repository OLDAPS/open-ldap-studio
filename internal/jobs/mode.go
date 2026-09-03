package jobs

import (
	"context"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// Dispatcher is the one call a job makes that reaches a server. Execution mode
// selects the implementation, so the validation path above it is identical in
// both modes — a dry run that skipped validation would prove nothing (FR-047).
type Dispatcher interface {
	Dispatch(ctx context.Context, dn string, apply func(context.Context) (ldapx.Result, error)) (ldapx.Result, error)
	Mode() Mode
	// Recorded returns what a dry run would have sent, in order. It is empty
	// in execute mode.
	Recorded() []string
}

// LiveDispatcher performs the operation.
type LiveDispatcher struct{}

func (LiveDispatcher) Dispatch(ctx context.Context, _ string, apply func(context.Context) (ldapx.Result, error)) (ldapx.Result, error) {
	return apply(ctx)
}
func (LiveDispatcher) Mode() Mode         { return ModeExecute }
func (LiveDispatcher) Recorded() []string { return nil }

// RecordingDispatcher is dry-run mode: everything above it runs unchanged and
// the operation itself is recorded instead of sent.
type RecordingDispatcher struct {
	dns []string
}

func (d *RecordingDispatcher) Dispatch(_ context.Context, dn string, _ func(context.Context) (ldapx.Result, error)) (ldapx.Result, error) {
	d.dns = append(d.dns, dn)
	return ldapx.NewResult(ldapx.Success, "", "dry run: nothing was sent to the server"), nil
}
func (d *RecordingDispatcher) Mode() Mode         { return ModeDryRun }
func (d *RecordingDispatcher) Recorded() []string { return d.dns }

// DispatcherFor returns the dispatcher a mode requires.
func DispatcherFor(m Mode) Dispatcher {
	if m == ModeDryRun {
		return &RecordingDispatcher{}
	}
	return LiveDispatcher{}
}
