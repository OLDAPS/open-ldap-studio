// Package secrets_scan holds the SC-008 scan: no secret material reaches any
// file this application writes.
//
// The scan exists because redaction applied at display time leaves the secret
// in the file. It is the check that catches that mistake, so it runs over the
// real writers rather than over a mock of them (contracts F3, E1, X7).
package secrets_scan

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-ldap-studio/open-ldap-studio/internal/logging"
	"github.com/open-ldap-studio/open-ldap-studio/internal/prefs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
	"github.com/open-ldap-studio/open-ldap-studio/internal/trust"
)

// canary is a value that must never appear in a file. It is distinctive so a
// hit is unambiguous.
const canary = "sc008-canary-Hunter2-Never-On-Disk"

// TestNoWrittenFileContainsSecretMaterial exercises every writer the
// application has and then scans everything it produced.
func TestNoWrittenFileContainsSecretMaterial(t *testing.T) {
	dir := t.TempDir()

	// 1. The profile store. A Profile has no field for a secret, and the
	//    strict decoder refuses a payload that carries one.
	store, err := profiles.OpenStore(filepath.Join(dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := store.Save(profiles.Profile{
		Name: "corp-dev", Host: "ldap.example.com", Port: 389,
		BindMethod: profiles.BindSimple, BindDN: "cn=admin,dc=example,dc=com",
		CredentialID: "cred-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" {
		t.Fatal("the profile was not saved")
	}

	// A payload carrying a password is refused, not silently stripped.
	payload := `{"name":"corp-dev","host":"ldap.example.com","port":389,"password":"` + canary + `"}`
	if _, err := profiles.DecodeStrict([]byte(payload)); err == nil {
		t.Error("a profile payload carrying a password was accepted; it must be refused")
	}

	// 2. The trust store.
	trustStore, err := trust.OpenStore(filepath.Join(dir, "trust.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := trustStore.Decide(trust.Decision{
		Host: "ldap.example.com", Port: 636, Fingerprint: "ab12cd34",
		Scope: trust.ScopePermanent, Reason: "self-signed certificate",
	}); err != nil {
		t.Fatal(err)
	}

	// 3. The logs, written with material that must be redacted on the way in.
	logs, err := logging.Open(logging.Paths{Data: dir, Logs: filepath.Join(dir, "logs")})
	if err != nil {
		t.Fatal(err)
	}
	modification := logs.Sink(logging.Modification)
	if _, err := modification.Write([]byte(
		"dn: cn=admin,dc=example,dc=com\nchangetype: modify\nreplace: userPassword\nuserPassword: " +
			canary + "\n-\nresult: 0\n")); err != nil {
		t.Fatal(err)
	}
	if err := logs.Sink(logging.Application).Record(
		`{"event":"bind","bindDN":"cn=admin","password":"%s"}`, canary); err != nil {
		t.Fatal(err)
	}
	if err := logs.Sink(logging.Errors).Record("bind failed for %v",
		secrets.NewSecret([]byte(canary))); err != nil {
		t.Fatal(err)
	}
	if err := logs.Close(); err != nil {
		t.Fatal(err)
	}

	// 4. An arbitrary store file, in case a future writer forgets the rule.
	if err := prefs.Write(filepath.Join(dir, "history.json"), 1, "history", map[string]string{
		"dn": "cn=admin,dc=example,dc=com", "result": "0",
	}); err != nil {
		t.Fatal(err)
	}

	scan(t, dir)
}

// scan walks every file under dir and fails on any hit.
func scan(t *testing.T, dir string) {
	t.Helper()

	var scanned int
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++

		if bytes.Contains(content, []byte(canary)) {
			t.Errorf("%s contains secret material:\n%s", path, content)
		}
		// A base64 of the canary would be just as bad.
		if bytes.Contains(content, []byte("c2MwMDgtY2FuYXJ5")) {
			t.Errorf("%s contains base64-encoded secret material", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Fatal("scanned no files; the test would pass vacuously")
	}
	t.Logf("scanned %d written files", scanned)
}

// Contract E1 — no emitted event payload contains secret material.
//
// The event payloads are maps, so the check is that serialising one that was
// handed a Secret fails rather than succeeding with the bytes in it.
func TestAnEventPayloadCarryingASecretCannotBeSerialised(t *testing.T) {
	payload := map[string]any{
		"profileId": "p1",
		"secret":    secrets.NewSecret([]byte(canary)),
	}
	if _, err := json.Marshal(payload); err == nil {
		t.Fatal("an event payload carrying a Secret was serialisable; it must fail instead")
	}
}

// TestRedactionIsAppliedBeforeTheBytesReachTheSink is contract X7 stated
// directly: the raw history file never contains a password value, even
// momentarily.
func TestRedactionIsAppliedBeforeTheBytesReachTheSink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "modification.log")

	sink, err := logging.OpenSink(path, logging.NewRedactor())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sink.Write([]byte("userPassword:: " + canary + "\n")); err != nil {
		t.Fatal(err)
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), canary) {
		t.Fatalf("the secret reached disk: %s", content)
	}
	if !strings.Contains(string(content), logging.Placeholder) {
		t.Errorf("the line was dropped rather than redacted: %s", content)
	}
}
