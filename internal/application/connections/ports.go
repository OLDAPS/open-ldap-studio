// Package connections owns profile and connection application workflows.
// It depends on capability ports and has no Wails or filesystem dependency.
package connections

import (
	"context"

	lifecycle "github.com/open-ldap-studio/open-ldap-studio/internal/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
	"github.com/open-ldap-studio/open-ldap-studio/internal/trust"
)

type ProfileStore interface {
	List() []profiles.Summary
	Get(string) (profiles.Profile, error)
	Save(profiles.Profile) (profiles.Profile, error)
	Delete(string) error
	Duplicate(string) (profiles.Profile, error)
	Folders() []profiles.Folder
	SaveFolder(profiles.Folder) (profiles.Folder, error)
}

type Lifecycle interface {
	Connect(context.Context, profiles.Profile) (ldapx.Result, error)
	Disconnect(string) error
	State(string) lifecycle.ConnState
	States() []lifecycle.ConnState
	CloseAll()
}

type DirectoryReader interface {
	WhoAmI(context.Context, string) (string, ldapx.Result, error)
	RootDSE(context.Context, string) (ldapx.RootDSE, ldapx.Result, error)
	ListChildren(context.Context, string, string, ldapx.PageRequest) (ldapx.Page, []ldapx.Degradation, error)
	ReadEntry(context.Context, string, string, ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error)
}

type ConnectionTester interface {
	Test(context.Context, profiles.Profile, string, bool) (lifecycle.ProbeResult, error)
}

type JobRegistry interface {
	Start(context.Context, jobs.Kind, jobs.Mode, string, int, jobs.Func) string
	CancelAll()
}

type (
	Emitter        interface{ Emit(string, any) }
	Logger         interface{ Record(string, ...any) error }
	TrustDecisions interface {
		List() []trust.Decision
		Decide(trust.Decision) error
		Revoke(string) error
	}
)

type Dependencies struct {
	Profiles              ProfileStore
	Lifecycle             Lifecycle
	Directory             DirectoryReader
	Tester                ConnectionTester
	Jobs                  JobRegistry
	Secrets               secrets.Provider
	Trust                 TrustDecisions
	Events                Emitter
	Log                   Logger
	Version               string
	Platform              string
	CredentialStoreReason string
}
