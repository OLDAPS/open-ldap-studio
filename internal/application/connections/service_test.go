package connections

import (
	"context"
	"errors"
	"testing"

	lifecycle "github.com/open-ldap-studio/open-ldap-studio/internal/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

type profileMemory struct {
	ProfileStore
	profile profiles.Profile
	deleted bool
}

func (p *profileMemory) Get(string) (profiles.Profile, error) { return p.profile, nil }
func (p *profileMemory) Save(profile profiles.Profile) (profiles.Profile, error) {
	p.profile = profile
	return profile, nil
}
func (p *profileMemory) Delete(string) error { p.deleted = true; return nil }

type connectionMemory struct {
	Lifecycle
	disconnectError error
}

func (c connectionMemory) Disconnect(string) error { return c.disconnectError }

type secretMemory struct {
	secrets.Provider
	deleteError error
}

func (s secretMemory) Delete(string) error { return s.deleteError }

func TestDeleteProfileStopsWhenCleanupFails(t *testing.T) {
	for _, failure := range []string{"disconnect", "secret deletion"} {
		t.Run(failure, func(t *testing.T) {
			cause := errors.New("cleanup unavailable")
			profiles := &profileMemory{}
			connections := connectionMemory{disconnectError: lifecycle.ErrNotConnected}
			secrets := secretMemory{}
			if failure == "disconnect" {
				connections.disconnectError = cause
			} else {
				secrets.deleteError = cause
			}
			service := New(Dependencies{Profiles: profiles, Lifecycle: connections, Secrets: secrets})
			if err := service.DeleteProfile("p1"); !errors.Is(err, cause) || profiles.deleted {
				t.Fatalf("cleanup failure lost: %v, profile deleted = %t", err, profiles.deleted)
			}
		})
	}
}

func TestStoreSecretKeepsOnlyAReferenceInTheProfile(t *testing.T) {
	profiles := &profileMemory{profile: profiles.Profile{ID: "p1"}}
	provider := secrets.NewSession()
	service := New(Dependencies{Profiles: profiles, Secrets: provider})
	if err := service.StoreProfileSecret("p1", "session password"); err != nil {
		t.Fatal(err)
	}
	if profiles.profile.CredentialID == "" || !service.ProfileHasSecret("p1") {
		t.Fatal("secret reference was not assigned")
	}
	if err := service.StoreProfileSecret("p1", ""); err != nil {
		t.Fatal(err)
	}
	if profiles.profile.CredentialID != "" || service.ProfileHasSecret("p1") {
		t.Fatal("forgotten credential survived")
	}
}

type directoryMemory struct {
	DirectoryReader
	read func(context.Context, string, string, ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error)
}

func (d directoryMemory) ReadEntry(ctx context.Context, id, dn string, options ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error) {
	return d.read(ctx, id, dn, options)
}

func TestReadPreservesContextAndVerbatimServerResult(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	want := ldapx.NewResult(ldapx.InsufficientAccessRights, " dc=Odd ", "  Keep Case\n")
	service := New(Dependencies{Directory: directoryMemory{read: func(received context.Context, id, dn string, options ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error) {
		if received != ctx || id != "p1" || dn != "cn=subject" || !options.IncludeOperational {
			t.Error("read intention changed at the application boundary")
		}
		return ldapx.Entry{}, want, ldapx.ServerError("read", want)
	}}})
	_, result, err := service.ReadEntry(ctx, "p1", "cn=subject", ldapx.ReadOptions{IncludeOperational: true})
	if result.Code != want.Code || result.MatchedDN != want.MatchedDN || result.DiagnosticMessage != want.DiagnosticMessage || !errors.Is(err, ldapx.ErrServer) {
		t.Fatalf("server result changed: %+v, %v", result, err)
	}
}
