package changes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

type pipelineMemory struct {
	Pipeline
	changeSet  changeset.ChangeSet
	pendingErr error
	commit     func(context.Context, string, changeset.Reporter) (ldapx.Result, error)
}

func (p pipelineMemory) Pending(string) (changeset.ChangeSet, error) {
	return p.changeSet, p.pendingErr
}

func (p pipelineMemory) Commit(ctx context.Context, token string, reporter changeset.Reporter) (ldapx.Result, error) {
	return p.commit(ctx, token, reporter)
}

type logMemory struct{ records []string }

func (l *logMemory) Record(format string, args ...any) error {
	l.records = append(l.records, fmt.Sprintf(format, args...))
	return nil
}

type event struct {
	name    string
	payload any
}

type eventMemory struct {
	mu       sync.Mutex
	events   []event
	finished chan struct{}
}

func (e *eventMemory) Emit(name string, payload any) {
	e.mu.Lock()
	e.events = append(e.events, event{name, payload})
	e.mu.Unlock()
	if name == "job:finished" {
		e.finished <- struct{}{}
	}
}

func (e *eventMemory) changes() []any {
	e.mu.Lock()
	defer e.mu.Unlock()
	var changed []any
	for _, event := range e.events {
		if event.name == "entry:changed" {
			changed = append(changed, event.payload)
		}
	}
	return changed
}

func pendingChange() changeset.ChangeSet {
	return changeset.ChangeSet{
		ID: "change-1", ProfileID: "p1", Kind: changeset.KindModify,
		Ops: []changeset.Operation{
			{DN: "cn=one", Type: changeset.OpReplaceAttr},
			{DN: "cn=one", Type: changeset.OpAddAttr},
			{DN: "cn=two", Type: changeset.OpReplaceAttr},
		},
	}
}

func expectedChanges() []any {
	return []any{
		map[string]any{"profileId": "p1", "dn": "cn=one", "source": "commit"},
		map[string]any{"profileId": "p1", "dn": "cn=two", "source": "commit"},
	}
}

func TestCommitPreservesResultsAndCoordinatesAuditAndRefresh(t *testing.T) {
	for _, rejected := range []bool{false, true} {
		t.Run(fmt.Sprintf("rejected=%t", rejected), func(t *testing.T) {
			ctx := t.Context()
			want := ldapx.NewResult(ldapx.Success, " dc=Odd ", "  Keep Case\n")
			var cause error
			if rejected {
				want.Code = ldapx.InsufficientAccessRights
				cause = ldapx.ServerError("commit", want)
			}
			modifications, failures := &logMemory{}, &logMemory{}
			events := &eventMemory{}
			service := New(Dependencies{
				Pipeline: pipelineMemory{changeSet: pendingChange(), commit: func(received context.Context, token string, reporter changeset.Reporter) (ldapx.Result, error) {
					if received != ctx || token != "preview-token" || reporter == nil {
						t.Error("commit context, token or reporting mode changed")
					}
					if !rejected {
						reporter.Outcome("cn=one", jobs.OutcomeSucceeded, &want)
						reporter.Outcome("cn=two", jobs.OutcomeSucceeded, &want)
					}
					return want, cause
				}},
				Events: events, ModificationLog: modifications, ErrorLog: failures,
			})

			result, err := service.Commit(ctx, "preview-token")
			if !reflect.DeepEqual(result, want) || !errors.Is(err, cause) {
				t.Fatalf("server result changed: %+v, %v", result, err)
			}
			if len(modifications.records) == 0 || !strings.Contains(modifications.records[0], want.String()) {
				t.Fatalf("missing commit audit: %v", modifications.records)
			}
			if rejected {
				if !strings.Contains(modifications.records[0], "error: "+cause.Error()) {
					t.Fatalf("failed commit overview omitted its error: %v", modifications.records)
				}
				if len(failures.records) != 1 || !strings.Contains(failures.records[0], cause.Error()) {
					t.Fatalf("missing failure audit: %v", failures.records)
				}
				if len(events.changes()) != 0 {
					t.Fatal("rejected commit announced successful entry changes")
				}
			} else if len(failures.records) != 0 || !reflect.DeepEqual(events.changes(), expectedChanges()) {
				t.Fatalf("successful commit did not refresh each distinct entry: %v, %v", events.changes(), failures.records)
			}
		})
	}
}

func TestUnknownTokenNeverCommitsOrStartsAJob(t *testing.T) {
	registry := jobs.NewRegistry(nil)
	called := false
	service := New(Dependencies{
		Pipeline: pipelineMemory{pendingErr: changeset.ErrNoToken, commit: func(context.Context, string, changeset.Reporter) (ldapx.Result, error) {
			called = true
			return ldapx.Result{}, nil
		}},
		Jobs: registry,
	})
	if _, err := service.Commit(t.Context(), "unknown"); !errors.Is(err, changeset.ErrNoToken) {
		t.Fatal(err)
	}
	if _, err := service.CommitAsJob(t.Context(), "unknown"); !errors.Is(err, changeset.ErrNoToken) {
		t.Fatal(err)
	}
	if called || len(registry.List()) != 0 {
		t.Fatal("an unknown token reached the mutation pipeline or created a job")
	}
}

func TestCommitJobPreservesOutcomesCancellationAndRefresh(t *testing.T) {
	for _, terminal := range []jobs.State{jobs.StatePartiallyComplete, jobs.StateFailed, jobs.StateCancelled} {
		t.Run(string(terminal), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			events := &eventMemory{finished: make(chan struct{}, 1)}
			registry := jobs.NewRegistry(events)
			modifications, failures := &logMemory{}, &logMemory{}
			started := make(chan struct{})
			denied := ldapx.NewResult(ldapx.InsufficientAccessRights, "cn=two", "  denied\n")
			service := New(Dependencies{
				Pipeline: pipelineMemory{changeSet: pendingChange(), commit: func(received context.Context, token string, reporter changeset.Reporter) (ldapx.Result, error) {
					close(started)
					if token != "preview-token" || reporter == nil {
						t.Error("job lost the preview token or per-entry reporter")
						return ldapx.Result{}, errors.New("missing commit context")
					}
					switch terminal {
					case jobs.StateCancelled:
						<-received.Done()
						return ldapx.Result{}, received.Err()
					case jobs.StateFailed:
						return denied, ldapx.ServerError("commit", denied)
					default:
						reporter.Outcome("cn=one", jobs.OutcomeSucceeded, nil)
						reporter.Outcome("cn=two", jobs.OutcomeFailed, &denied)
						reporter.Progress(3, 3, "complete")
						return denied, nil
					}
				}},
				Jobs: registry, Events: events,
				ModificationLog: modifications, ErrorLog: failures,
			})
			id, err := service.CommitAsJob(ctx, "preview-token")
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("commit job did not start")
			}
			if terminal == jobs.StateCancelled {
				cancel()
			}
			select {
			case <-events.finished:
			case <-time.After(2 * time.Second):
				t.Fatal("commit job did not finish")
			}
			job, err := registry.Get(id)
			if err != nil || job.Kind != jobs.KindCommit || job.Mode != jobs.ModeExecute || job.ProfileID != "p1" || job.Total != 3 || job.State != terminal {
				t.Fatalf("job metadata or terminal state changed: %+v, %v", job, err)
			}
			var expected []any
			if terminal == jobs.StatePartiallyComplete {
				expected = expectedChanges()[:1]
			}
			if !reflect.DeepEqual(events.changes(), expected) {
				t.Fatalf("job refreshed the wrong entries: %v, want %v", events.changes(), expected)
			}
			if len(modifications.records) == 0 {
				t.Fatal("mutation job omitted the modification audit")
			}
			if terminal == jobs.StatePartiallyComplete {
				if len(modifications.records) != 3 || !strings.Contains(modifications.records[1], "succeeded cn=one") || !strings.Contains(modifications.records[2], "failed cn=two") {
					t.Fatalf("partial job lost per-operation audit: %v", modifications.records)
				}
				if len(failures.records) != 0 {
					t.Fatalf("partial result was logged as a whole-job failure: %v", failures.records)
				}
				outcomes, err := registry.Outcomes(id)
				if err != nil || len(outcomes) != 2 || !reflect.DeepEqual(outcomes[1].Result, &denied) {
					t.Fatalf("job lost per-entry diagnostics: %+v, %v", outcomes, err)
				}
			}
		})
	}
}

func TestPartialSynchronousFailureRefreshesOnlyAppliedEntries(t *testing.T) {
	denied := ldapx.NewResult(ldapx.InsufficientAccessRights, "cn=two", "denied")
	events := &eventMemory{}
	modifications, failures := &logMemory{}, &logMemory{}
	service := New(Dependencies{
		Pipeline: pipelineMemory{changeSet: pendingChange(), commit: func(_ context.Context, _ string, reporter changeset.Reporter) (ldapx.Result, error) {
			reporter.Outcome("cn=one", jobs.OutcomeSucceeded, nil)
			reporter.Outcome("cn=two", jobs.OutcomeFailed, &denied)
			reporter.Outcome("cn=three", jobs.OutcomeSkipped, nil)
			return denied, ldapx.ServerError("commit", denied)
		}},
		Events: events, ModificationLog: modifications, ErrorLog: failures,
	})
	result, err := service.Commit(t.Context(), "preview-token")
	if !errors.Is(err, ldapx.ErrServer) || !reflect.DeepEqual(result, denied) {
		t.Fatalf("partial commit lost the server result: %+v, %v", result, err)
	}
	if !reflect.DeepEqual(events.changes(), expectedChanges()[:1]) {
		t.Fatalf("partial commit failed to refresh its applied entry: %v", events.changes())
	}
	if len(modifications.records) != 4 || len(failures.records) != 1 {
		t.Fatalf("partial commit lost operation or failure audit: %v, %v", modifications.records, failures.records)
	}
}

func TestIndeterminateSynchronousWriteRefreshesItsTarget(t *testing.T) {
	events := &eventMemory{}
	service := New(Dependencies{
		Pipeline: pipelineMemory{changeSet: pendingChange(), commit: func(_ context.Context, _ string, reporter changeset.Reporter) (ldapx.Result, error) {
			reporter.Outcome("cn=one", jobs.OutcomeIndeterminate, nil)
			return ldapx.Result{}, ldapx.Indeterminate("modify", context.Canceled)
		}},
		Events: events,
	})
	if _, err := service.Commit(t.Context(), "preview-token"); !errors.Is(err, ldapx.ErrIndeterminate) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events.changes(), expectedChanges()[:1]) {
		t.Fatalf("indeterminate write did not invalidate its target: %v", events.changes())
	}
}
