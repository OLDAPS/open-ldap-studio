package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-ldap-studio/open-ldap-studio/internal/prefs"
)

func TestTLSVerificationDefaultsOnlyWhenOmitted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, example := range []struct {
		name, tls             string
		certificate, hostname bool
	}{
		{"both disabled", `"tls":{"verifyCertificate":false,"verifyHostname":false}`, false, false},
		{"one disabled", `"tls":{"verifyCertificate":false}`, false, true},
		{"omitted", `"baseDn":"dc=example"`, true, true},
	} {
		t.Run(example.name, func(t *testing.T) {
			payload := []byte(`{"name":"local","host":"localhost","encryption":"startTLS",` + example.tls + `}`)
			profile, err := DecodeStrict(payload)
			if err != nil {
				t.Fatal(err)
			}
			saved, err := store.Save(profile)
			if err != nil {
				t.Fatal(err)
			}
			if saved.TLS.VerifyCertificate != example.certificate || saved.TLS.VerifyHostname != example.hostname {
				t.Fatalf("TLS policy changed during save: %+v", saved.TLS)
			}
		})
	}
	loaded, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, summary := range loaded.List() {
		profile, err := loaded.Get(summary.ID)
		if err != nil {
			t.Fatal(err)
		}
		if profile.TLS.VerifyCertificate || profile.TLS.VerifyHostname {
			continue
		}
		duplicate, err := loaded.Duplicate(profile.ID)
		if err != nil {
			t.Fatal(err)
		}
		if duplicate.TLS.VerifyCertificate || duplicate.TLS.VerifyHostname {
			t.Fatal("duplicating a loaded opt-out silently re-enabled TLS verification")
		}
		return
	}
	t.Fatal("did not find the saved TLS opt-out")
}

func TestFailedPersistenceDoesNotPublishProfileOrFolderChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.Save(Profile{Name: "original", Host: "ldap.example.org"})
	if err != nil {
		t.Fatal(err)
	}
	folder, err := store.SaveFolder(Folder{Name: "original folder"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, path+".saved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}

	changed := profile
	changed.ReadOnly = true
	if _, err := store.Save(changed); err == nil {
		t.Fatal("profile save unexpectedly persisted")
	}
	if _, err := store.Save(Profile{Name: "new", Host: "ldap.example.org"}); err == nil {
		t.Fatal("new profile unexpectedly persisted")
	}
	if err := store.Delete(profile.ID); err == nil {
		t.Fatal("profile deletion unexpectedly persisted")
	}
	changedFolder := folder
	changedFolder.Name = "changed folder"
	if _, err := store.SaveFolder(changedFolder); err == nil {
		t.Fatal("folder save unexpectedly persisted")
	}
	if _, err := store.SaveFolder(Folder{Name: "new folder"}); err == nil {
		t.Fatal("new folder unexpectedly persisted")
	}

	got, err := store.Get(profile.ID)
	if err != nil || got.ReadOnly || got.Name != profile.Name || len(store.List()) != 1 {
		t.Fatalf("failed persistence changed live profiles: %+v, %v", got, err)
	}
	folders := store.Folders()
	if len(folders) != 1 || folders[0] != folder {
		t.Fatalf("failed persistence changed live folders: %+v", folders)
	}
}

func TestReturnedProfilesCannotChangeLivePolicyWithoutSaving(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	input := Profile{Name: "local", Host: "ldap.example.org", Tags: []string{TagProduction}}
	saved, err := store.Save(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Tags[0] = "development"
	saved.Tags[0] = "development"
	read, err := store.Get(saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	read.Tags[0] = "development"
	summary := store.List()[0]
	summary.Tags[0] = "development"
	again, err := store.Get(saved.ID)
	if err != nil || !again.IsProduction() {
		t.Fatalf("returned profile data changed the stored production policy: %+v, %v", again, err)
	}
}

func TestLegacyProfileWithoutTLSFieldsLoadsWithVerification(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	if err := prefs.Write(path, SchemaVersion, "profiles", map[string]any{
		"profiles": []map[string]any{{
			"id": "legacy", "name": "legacy", "host": "ldap.example.org",
			"encryption": "ldaps",
		}},
		"folders": []any{},
	}); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.Get("legacy")
	if err != nil || !profile.TLS.VerifyCertificate || !profile.TLS.VerifyHostname {
		t.Fatalf("omitted legacy TLS settings disabled verification: %+v, %v", profile.TLS, err)
	}
}
