// Package changes owns preview, commit, audit and invalidation workflows.
package changes

import (
	"context"

	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// Pipeline is the guarded mutation boundary. The application never dispatches
// LDAP writes itself or accepts a new change payload at commit time.
type Pipeline interface {
	Preview(context.Context, changeset.Input, bool) (changeset.Preview, error)
	Pending(string) (changeset.ChangeSet, error)
	Discard(string) error
	Commit(context.Context, string, changeset.Reporter) (ldapx.Result, error)
}

type JobRegistry interface {
	Start(context.Context, jobs.Kind, jobs.Mode, string, int, jobs.Func) string
}

type Emitter interface{ Emit(string, any) }

// Logger implementations own redaction and persistence.
type Logger interface{ Record(string, ...any) error }

type Dependencies struct {
	Pipeline        Pipeline
	Jobs            JobRegistry
	Events          Emitter
	ModificationLog Logger
	ErrorLog        Logger
}
