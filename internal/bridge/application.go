package bridge

import (
	"context"

	application "github.com/open-ldap-studio/open-ldap-studio/internal/application/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/commands"
	"github.com/open-ldap-studio/open-ldap-studio/internal/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/trust"
)

// ConnectionApplication is the inbound port consumed by the migrated bridge.
type ConnectionApplication interface {
	Startup()
	Shutdown()
	Info() application.AppInfo
	CredentialStoreStatus() (bool, string)
	ListProfiles() []profiles.Summary
	GetProfile(string) (profiles.Profile, error)
	SaveProfile(profiles.Profile) (profiles.Profile, error)
	DeleteProfile(string) error
	DuplicateProfile(string) (profiles.Profile, error)
	ListFolders() []profiles.Folder
	SaveFolder(profiles.Folder) (profiles.Folder, error)
	Connect(context.Context, string) (string, error)
	Disconnect(string) error
	ConnectionState(string) connections.ConnState
	ConnectionStates() []connections.ConnState
	WhoAmI(context.Context, string) (string, ldapx.Result, error)
	RootDSE(context.Context, string) (ldapx.RootDSE, ldapx.Result, error)
	ReadEntry(context.Context, string, string, ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error)
	ListChildren(context.Context, string, string, ldapx.PageRequest) (ldapx.Page, error)
	ValidateFilter(string) ldapx.FilterDiagnostic
	TestConnection(context.Context, profiles.Profile, string) (connections.ProbeResult, error)
	TestBind(context.Context, string, string) (connections.ProbeResult, error)
	StoreProfileSecret(string, string) error
	ProfileHasSecret(string) bool
	ListTrustDecisions() []trust.Decision
	DecideTrust(trust.Decision) error
	RevokeTrustDecision(string) error
}

// ChangeApplication owns mutation workflows behind the Wails adapter.
type ChangeApplication interface {
	Preview(context.Context, changeset.Input, bool) (changeset.Preview, error)
	Pending(string) (changeset.ChangeSet, error)
	Discard(string) error
	Commit(context.Context, string) (ldapx.Result, error)
	CommitAsJob(context.Context, string) (string, error)
}

// Dependencies are composed before binding; constructors are not Wails methods.
type Dependencies struct {
	Application ConnectionApplication
	Changes     ChangeApplication
	Jobs        *jobs.Registry
	Commands    *commands.Registry
}
