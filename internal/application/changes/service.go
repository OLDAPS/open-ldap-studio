package changes

import (
	"context"

	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

type Service struct{ deps Dependencies }

func New(deps Dependencies) *Service { return &Service{deps: deps} }

func (s *Service) Preview(ctx context.Context, input changeset.Input, confirmedProduction bool) (changeset.Preview, error) {
	return s.deps.Pipeline.Preview(ctx, input, confirmedProduction)
}

func (s *Service) Pending(token string) (changeset.ChangeSet, error) {
	return s.deps.Pipeline.Pending(token)
}

func (s *Service) Discard(token string) error { return s.deps.Pipeline.Discard(token) }

// Commit preserves the pipeline's structured result while coordinating the
// audit and refresh work independently of the desktop transport.
func (s *Service) Commit(ctx context.Context, token string) (ldapx.Result, error) {
	cs, err := s.deps.Pipeline.Pending(token)
	if err != nil {
		return ldapx.Result{}, err
	}
	return s.execute(ctx, token, cs, nil)
}

// CommitAsJob keeps cancellation and per-entry outcomes in the job registry.
func (s *Service) CommitAsJob(ctx context.Context, token string) (string, error) {
	cs, err := s.deps.Pipeline.Pending(token)
	if err != nil {
		return "", err
	}

	id := s.deps.Jobs.Start(ctx, jobs.KindCommit, jobs.ModeExecute, cs.ProfileID, len(cs.Ops),
		func(ctx context.Context, reporter *jobs.Reporter) error {
			_, err := s.execute(ctx, token, cs, reporter)
			return err
		})
	return id, nil
}

type operationOutcome struct {
	dn     string
	status string
	result *ldapx.Result
}

type commitObserver struct {
	job         *jobs.Reporter
	stopOnError bool
	outcomes    []operationOutcome
}

func (o *commitObserver) Outcome(dn, status string, result *ldapx.Result) {
	o.outcomes = append(o.outcomes, operationOutcome{dn: dn, status: status, result: result})
	if o.job != nil {
		o.job.Outcome(dn, status, result)
	}
}

func (o *commitObserver) Progress(done, total int, message string) {
	if o.job != nil {
		o.job.Progress(done, total, message)
	}
}

func (o *commitObserver) StopOnError() bool { return o.stopOnError }

func (s *Service) execute(ctx context.Context, token string, cs changeset.ChangeSet, job *jobs.Reporter) (ldapx.Result, error) {
	observer := &commitObserver{job: job, stopOnError: job == nil}
	result, err := s.deps.Pipeline.Commit(ctx, token, observer)
	if s.deps.ModificationLog != nil {
		overview := result.String()
		if err != nil {
			overview = "error: " + err.Error() + "; result: " + overview
		}
		_ = s.deps.ModificationLog.Record("# %s %s on %s -> %s",
			cs.Kind, cs.DNs(), cs.ProfileID, overview)
		for _, outcome := range observer.outcomes {
			answer := "no server result"
			if outcome.result != nil {
				answer = outcome.result.String()
			}
			_ = s.deps.ModificationLog.Record("# %s %s on %s -> %s", outcome.status, outcome.dn, cs.ProfileID, answer)
		}
	}
	if err != nil && s.deps.ErrorLog != nil {
		_ = s.deps.ErrorLog.Record("commit %s: %v", cs.ID, err)
	}
	s.invalidate(cs.ProfileID, observer.outcomes)
	return result, err
}

func (s *Service) invalidate(profileID string, outcomes []operationOutcome) {
	seen := make(map[string]bool, len(outcomes))
	for _, outcome := range outcomes {
		if outcome.status != jobs.OutcomeSucceeded && outcome.status != jobs.OutcomeIndeterminate || seen[outcome.dn] {
			continue
		}
		seen[outcome.dn] = true
		s.deps.Events.Emit("entry:changed", map[string]any{
			"profileId": profileID, "dn": outcome.dn, "source": "commit",
		})
	}
}
