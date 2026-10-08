package bridge

import (
	"encoding/json"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/logging"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// The fixture in server/docker-compose.yml. The test skips when it is not up,
// so this is a check you can run, not a check that blocks a build on a machine
// with no Docker. CI sets REQUIRE_LIVE_DIRECTORY so that a fixture that failed
// to start fails the build instead of turning every live test into a skip.
const (
	liveHost = "localhost"
	livePort = 1389
	liveBase = "dc=example,dc=org"
	liveDN   = "cn=admin,dc=example,dc=org"
	livePass = "adminpassword"
)

func liveBridge(t *testing.T) *Bridge {
	t.Helper()

	address := net.JoinHostPort(liveHost, "1389")
	conn, err := net.DialTimeout("tcp", address, 750*time.Millisecond)
	if err != nil {
		if os.Getenv("REQUIRE_LIVE_DIRECTORY") != "" {
			t.Fatalf("no directory on %s, and REQUIRE_LIVE_DIRECTORY is set", address)
		}
		t.Skipf("no directory on %s — start it with: cd server && docker compose up -d", address)
	}
	_ = conn.Close()

	// Everything this bridge writes goes to the test's own directory, so a run
	// cannot disturb the developer's real connection list.
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	paths, err := logging.DefaultPaths()
	if err != nil {
		t.Fatalf("paths: %v", err)
	}
	logs, err := logging.Open(paths)
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	t.Cleanup(func() { _ = logs.Close() })

	b := New(logs)
	t.Cleanup(b.Shutdown)
	return b
}

func liveProfile() profiles.Profile {
	return profiles.Profile{
		Name:       "local-dev",
		Host:       liveHost,
		Port:       livePort,
		Encryption: profiles.EncryptionNone,
		BindMethod: profiles.BindSimple,
		BindDN:     liveDN,
		BaseDN:     liveBase,
		Timeouts:   profiles.Timeouts{ConnectMS: 5000, ReadMS: 5000},
		Limits:     profiles.Limits{PageSize: 100},
		Aliases:    profiles.AliasFind,
		Referrals:  profiles.ReferralFollow,
	}
}

func encode(t *testing.T, p profiles.Profile) string {
	t.Helper()
	payload, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(payload)
}

// TestConnection must dial and bind without saving anything — that is the
// promise the wizard makes on every page ("nothing is persisted until Finish").
func TestLiveTestConnection(t *testing.T) {
	b := liveBridge(t)

	result, err := b.TestConnection(encode(t, liveProfile()), livePass)
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if !result.Reachable {
		t.Fatalf("not reachable: %s", result.Message)
	}
	if !result.Bound {
		t.Fatalf("bind refused: %s (%s)", result.Message, result.Result.DiagnosticMessage)
	}
	if result.BoundDN == "" {
		t.Error("bound with no DN reported")
	}
	if len(result.NamingContexts) == 0 {
		t.Error("no naming contexts read from the root DSE — the wizard has no base DN to offer")
	}

	if got := len(b.ListProfiles()); got != 0 {
		t.Errorf("TestConnection persisted %d profile(s); it must persist none", got)
	}
}

// A wrong password must come back as the server's own result, not as a
// transport error and not as a silent anonymous bind (deviation D7).
func TestLiveTestConnectionRejectsBadPassword(t *testing.T) {
	b := liveBridge(t)

	result, err := b.TestConnection(encode(t, liveProfile()), "not-the-password")
	if err != nil {
		t.Fatalf("TestConnection returned a Go error for a rejected bind: %v", err)
	}
	if result.Bound {
		t.Fatal("a wrong password bound successfully")
	}
	if !result.Reachable {
		t.Error("the server was reachable; only the bind should have failed")
	}
	if result.Result.Code != 49 {
		t.Errorf("result code = %d, want 49 (invalidCredentials); message %q",
			result.Result.Code, result.Result.DiagnosticMessage)
	}
}

// The whole point of the feature: save a profile, store its secret, connect,
// and end up with a connection the rest of the app can read through.
func TestLiveSaveStoreConnect(t *testing.T) {
	b := liveBridge(t)

	saved, err := b.SaveProfile(encode(t, liveProfile()))
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("saved profile has no id")
	}

	if b.ProfileHasSecret(saved.ID) {
		t.Error("a freshly saved profile already has a secret")
	}
	if err := b.StoreProfileSecret(saved.ID, livePass); err != nil {
		t.Fatalf("StoreProfileSecret: %v", err)
	}
	if !b.ProfileHasSecret(saved.ID) {
		t.Fatal("secret was stored but ProfileHasSecret says otherwise")
	}

	// The secret must not have landed in the profile file.
	reloaded, err := b.GetProfile(saved.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	payload, _ := json.Marshal(reloaded)
	if strings.Contains(string(payload), livePass) {
		t.Fatalf("the bind password appears in the stored profile: %s", payload)
	}

	if _, err := b.Connect(saved.ID); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	state := waitForState(t, b, saved.ID, StateConnected)
	if state.BoundDN == "" {
		t.Error("connected with no bound DN")
	}
	if state.ServerIdentity == "" {
		t.Error("connected with no server identity for the status bar")
	}

	// A connection is only useful if something can be read through it.
	page, err := b.ListChildren(saved.ID, liveBase, ldapx.PageRequest{Size: 100})
	if err != nil {
		t.Fatalf("ListChildren through the new connection: %v", err)
	}
	if len(page.Entries) == 0 {
		t.Error("no children under the base DN; the fixture should have four")
	}

	if err := b.Disconnect(saved.ID); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if got := b.ConnectionState(saved.ID).State; got != StateDisconnected {
		t.Errorf("state after disconnect = %q, want %q", got, StateDisconnected)
	}
}

// Deleting a profile must take its secret with it, or material is left in the
// platform store that nothing can name any more.
func TestLiveDeleteProfileDropsSecret(t *testing.T) {
	b := liveBridge(t)

	saved, err := b.SaveProfile(encode(t, liveProfile()))
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if err := b.StoreProfileSecret(saved.ID, livePass); err != nil {
		t.Fatalf("StoreProfileSecret: %v", err)
	}
	if err := b.DeleteProfile(saved.ID); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	if _, err := b.secrets.Get(secretRef(saved.ID)); err == nil {
		t.Error("the secret outlived the profile it belonged to")
	}
}

// SaveProfile must refuse a payload carrying a password rather than dropping
// it silently (SC-008).
func TestLiveSaveProfileRefusesASecret(t *testing.T) {
	b := liveBridge(t)

	if _, err := b.SaveProfile(`{"name":"x","host":"h","port":389,"password":"hunter2"}`); err == nil {
		t.Fatal("a profile payload carrying a password was accepted")
	}
}

func waitForState(t *testing.T, b *Bridge, profileID string, want State) ConnState {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	var last ConnState
	for time.Now().Before(deadline) {
		last = b.ConnectionState(profileID)
		if last.State == want {
			return last
		}
		if last.State == StateDisconnected && last.Message != "" {
			t.Fatalf("connect failed: %s", last.Message)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("state %q not reached in time (last %q: %s)", want, last.State, last.Message)
	return last
}
