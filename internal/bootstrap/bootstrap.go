// Package bootstrap composes the desktop application's concrete dependencies.
package bootstrap

import (
	"fmt"
	"path/filepath"
	"runtime"

	changeapp "github.com/open-ldap-studio/open-ldap-studio/internal/application/changes"
	application "github.com/open-ldap-studio/open-ldap-studio/internal/application/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/bridge"
	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/commands"
	"github.com/open-ldap-studio/open-ldap-studio/internal/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/logging"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
	"github.com/open-ldap-studio/open-ldap-studio/internal/trust"
)

type Options struct {
	Paths                 *logging.Paths
	Secrets               secrets.Provider
	CredentialStoreReason string
}

// eventRelay is connected before any job or application lifecycle is started.
type eventRelay struct {
	target interface{ Emit(string, any) }
}

func (e *eventRelay) Emit(name string, payload any) {
	if e.target != nil {
		e.target.Emit(name, payload)
	}
}

func New(logs *logging.Logs, options Options) (*bridge.Bridge, error) {
	paths := options.Paths
	if paths == nil {
		defaults, err := logging.DefaultPaths()
		if err != nil {
			return nil, err
		}
		paths = &defaults
	}
	provider, reason := options.Secrets, options.CredentialStoreReason
	if provider == nil {
		provider, reason = secrets.Open()
	}
	store, err := profiles.OpenStore(filepath.Join(paths.Data, "connections.json"))
	if err != nil {
		_ = logs.Sink(logging.Errors).Record("profile store: %v", err)
		store, err = profiles.OpenStore(filepath.Join(paths.Data, "connections.json.unreadable"))
		if err != nil {
			return nil, fmt.Errorf("open recovery profile store: %w", err)
		}
	}
	trustStore, err := trust.OpenStore(filepath.Join(paths.Data, "trust.json"))
	if err != nil {
		_ = logs.Sink(logging.Errors).Record("trust store: %v", err)
		trustStore, err = trust.OpenStore(filepath.Join(paths.Data, "trust.json.unreadable"))
		if err != nil {
			return nil, fmt.Errorf("open recovery trust store: %w", err)
		}
	}
	events := &eventRelay{}
	trustPolicy := func(profiles.Profile) ldapx.TrustPolicy { return trustStore.Policy(events) }
	manager := connections.New(events, provider, trustPolicy, logs)
	registry := jobs.NewRegistry(events)
	service := application.New(application.Dependencies{
		Profiles: store, Lifecycle: manager, Directory: manager,
		Tester: connections.Tester{TrustPolicy: trustPolicy}, Jobs: registry,
		Secrets: provider, Trust: trustStore, Events: events,
		Log: logs.Sink(logging.Application), Version: bridge.Version,
		Platform: runtime.GOOS, CredentialStoreReason: reason,
	})
	changes := changeapp.New(changeapp.Dependencies{
		Pipeline: changeset.NewPipeline(manager, store, changeset.Guard{RequireProductionConfirmation: true}),
		Jobs:     registry, Events: events,
		ModificationLog: logs.Sink(logging.Modification), ErrorLog: logs.Sink(logging.Errors),
	})
	app := bridge.New(bridge.Dependencies{
		Application: service, Changes: changes, Jobs: registry, Commands: commands.Default(),
	})
	events.target = app
	return app, nil
}
