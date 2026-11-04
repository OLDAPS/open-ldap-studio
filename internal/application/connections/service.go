package connections

import (
	"context"
	"errors"
	"fmt"

	lifecycle "github.com/open-ldap-studio/open-ldap-studio/internal/connections"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
	"github.com/open-ldap-studio/open-ldap-studio/internal/trust"
)

type Service struct{ deps Dependencies }

func New(deps Dependencies) *Service { return &Service{deps: deps} }

type AppInfo struct {
	Version               string `json:"version"`
	CredentialStore       string `json:"credentialStore"`
	CredentialStoreReason string `json:"credentialStoreReason"`
	UpdateChecksEnabled   bool   `json:"updateChecksEnabled"`
	Platform              string `json:"platform"`
}

func (s *Service) Info() AppInfo {
	return AppInfo{
		Version: s.deps.Version, CredentialStore: s.deps.Secrets.Name(),
		CredentialStoreReason: s.deps.CredentialStoreReason, Platform: s.deps.Platform,
	}
}

func (s *Service) CredentialStoreStatus() (bool, string) {
	available, reason := s.deps.Secrets.Available()
	if reason == "" {
		reason = s.deps.CredentialStoreReason
	}
	return available, reason
}

func (s *Service) Startup() {
	s.record("started, version %s, credential store %q", s.deps.Version, s.deps.Secrets.Name())
	if s.deps.CredentialStoreReason != "" {
		s.deps.Events.Emit("credential:storeUnavailable", map[string]any{"reason": s.deps.CredentialStoreReason})
	}
}

func (s *Service) Shutdown() {
	s.deps.Jobs.CancelAll()
	s.deps.Lifecycle.CloseAll()
	if ephemeral, ok := s.deps.Secrets.(interface{ Zero() }); ok {
		ephemeral.Zero()
	}
	s.record("shutdown")
}

func (s *Service) record(format string, args ...any) {
	if s.deps.Log != nil {
		_ = s.deps.Log.Record(format, args...)
	}
}

func (s *Service) ListProfiles() []profiles.Summary               { return s.deps.Profiles.List() }
func (s *Service) GetProfile(id string) (profiles.Profile, error) { return s.deps.Profiles.Get(id) }
func (s *Service) SaveProfile(profile profiles.Profile) (profiles.Profile, error) {
	return s.deps.Profiles.Save(profile)
}

func (s *Service) DuplicateProfile(id string) (profiles.Profile, error) {
	return s.deps.Profiles.Duplicate(id)
}
func (s *Service) ListFolders() []profiles.Folder { return s.deps.Profiles.Folders() }
func (s *Service) SaveFolder(folder profiles.Folder) (profiles.Folder, error) {
	return s.deps.Profiles.SaveFolder(folder)
}

func (s *Service) DeleteProfile(id string) error {
	if err := s.deps.Lifecycle.Disconnect(id); err != nil && !errors.Is(err, lifecycle.ErrNotConnected) {
		return err
	}
	if err := s.deps.Secrets.Delete(secretRef(id)); err != nil {
		return err
	}
	return s.deps.Profiles.Delete(id)
}

func (s *Service) Connect(ctx context.Context, profileID string) (string, error) {
	p, err := s.deps.Profiles.Get(profileID)
	if err != nil {
		return "", err
	}
	return s.deps.Jobs.Start(ctx, jobs.KindConnect, jobs.ModeExecute, profileID, 0,
		func(ctx context.Context, reporter *jobs.Reporter) error {
			reporter.Progress(0, 1, "connecting to "+p.Host)
			if _, err := s.deps.Lifecycle.Connect(ctx, p); err != nil {
				return err
			}
			reporter.Progress(1, 1, "connected")
			dse, _, err := s.deps.Directory.RootDSE(ctx, profileID)
			if err == nil {
				s.announce(profileID, dse.MissingCapabilities())
			}
			return nil
		}), nil
}

func (s *Service) Disconnect(id string) error { return s.deps.Lifecycle.Disconnect(id) }

func (s *Service) ConnectionState(id string) lifecycle.ConnState { return s.deps.Lifecycle.State(id) }

func (s *Service) ConnectionStates() []lifecycle.ConnState { return s.deps.Lifecycle.States() }

func (s *Service) WhoAmI(ctx context.Context, id string) (string, ldapx.Result, error) {
	return s.deps.Directory.WhoAmI(ctx, id)
}

func (s *Service) RootDSE(ctx context.Context, id string) (ldapx.RootDSE, ldapx.Result, error) {
	return s.deps.Directory.RootDSE(ctx, id)
}

func (s *Service) ReadEntry(ctx context.Context, id, dn string, options ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error) {
	return s.deps.Directory.ReadEntry(ctx, id, dn, options)
}

func (s *Service) ListChildren(ctx context.Context, id, dn string, page ldapx.PageRequest) (ldapx.Page, error) {
	result, degradations, err := s.deps.Directory.ListChildren(ctx, id, dn, page)
	s.announce(id, degradations)
	return result, err
}

func (s *Service) ValidateFilter(filter string) ldapx.FilterDiagnostic {
	return ldapx.ValidateFilter(filter)
}

func (s *Service) announce(id string, degradations []ldapx.Degradation) {
	for _, degradation := range degradations {
		s.deps.Events.Emit("capability:unavailable", map[string]any{
			"profileId": id, "capability": degradation.Capability, "reason": degradation.Reason,
		})
	}
}

func (s *Service) TestConnection(ctx context.Context, p profiles.Profile, secret string) (lifecycle.ProbeResult, error) {
	return s.deps.Tester.Test(ctx, p, secret, p.BindMethod != "" && p.BindMethod != profiles.BindAnonymous)
}

func (s *Service) TestBind(ctx context.Context, id, secret string) (lifecycle.ProbeResult, error) {
	p, err := s.deps.Profiles.Get(id)
	if err != nil {
		return lifecycle.ProbeResult{}, err
	}
	return s.deps.Tester.Test(ctx, p, secret, true)
}

func secretRef(id string) string { return "profile/" + id }

func (s *Service) StoreProfileSecret(id, secret string) error {
	p, err := s.deps.Profiles.Get(id)
	if err != nil {
		return err
	}
	ref := secretRef(id)
	if secret == "" {
		if err := s.deps.Secrets.Delete(ref); err != nil {
			return err
		}
		if p.CredentialID == ref {
			p.CredentialID = ""
			_, err = s.deps.Profiles.Save(p)
		}
		return err
	}
	material := secrets.NewSecret([]byte(secret))
	if err := s.deps.Secrets.Set(ref, material); err != nil {
		material.Zero()
		return fmt.Errorf("could not store the secret in %s: %w", s.deps.Secrets.Name(), err)
	}
	if p.CredentialID != ref {
		p.CredentialID = ref
		if _, err := s.deps.Profiles.Save(p); err != nil {
			return err
		}
	}
	s.record("stored a bind secret for %q in %s", p.Name, s.deps.Secrets.Name())
	return nil
}

func (s *Service) ProfileHasSecret(id string) bool {
	p, err := s.deps.Profiles.Get(id)
	if err != nil || p.CredentialID == "" {
		return false
	}
	secret, err := s.deps.Secrets.Get(p.CredentialID)
	if err != nil {
		return false
	}
	defer secret.Zero()
	return !secret.IsZero()
}

func (s *Service) ListTrustDecisions() []trust.Decision {
	return s.deps.Trust.List()
}

func (s *Service) DecideTrust(decision trust.Decision) error { return s.deps.Trust.Decide(decision) }

func (s *Service) RevokeTrustDecision(fingerprint string) error {
	return s.deps.Trust.Revoke(fingerprint)
}
