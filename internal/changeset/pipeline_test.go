package changeset

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// fakeDirectory is an in-memory directory. It implements the unexported
// methods of Directory, which it can because the tests live in this package —
// the same reason no code outside it can reach a mutation function.
type fakeDirectory struct {
	mu      sync.Mutex
	entries map[string]ldapx.Entry
	// dispatched records what actually reached the "server", in order.
	dispatched []string
	failWith   error
	failResult ldapx.Result
}

func newFakeDirectory(entries ...ldapx.Entry) *fakeDirectory {
	d := &fakeDirectory{entries: make(map[string]ldapx.Entry, len(entries))}
	for _, e := range entries {
		d.entries[e.DN] = e
	}
	return d
}

func (d *fakeDirectory) ReadEntry(_ context.Context, dn string) (ldapx.Entry, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	entry, ok := d.entries[dn]
	if !ok {
		return ldapx.Entry{}, ldapx.ServerError("read",
			ldapx.NewResult(ldapx.NoSuchObject, "", "no such object"))
	}
	return entry, nil
}

func (d *fakeDirectory) Identity() string { return "ldap.example.com:389" }

func (d *fakeDirectory) record(op, dn string) (ldapx.Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dispatched = append(d.dispatched, op+" "+dn)
	if d.failWith != nil {
		return d.failResult, d.failWith
	}
	return ldapx.NewResult(ldapx.Success, "", ""), nil
}

func (d *fakeDirectory) add(_ context.Context, dn string, _ []ldapx.Attribute) (ldapx.Result, error) {
	return d.record("add", dn)
}

func (d *fakeDirectory) modify(_ context.Context, dn string, _ []ldapx.Change) (ldapx.Result, error) {
	return d.record("modify", dn)
}

func (d *fakeDirectory) rename(_ context.Context, dn, _, _ string, _ bool) (ldapx.Result, error) {
	return d.record("rename", dn)
}

func (d *fakeDirectory) del(_ context.Context, dn string, subtree bool) (ldapx.Result, error) {
	if subtree {
		return d.record("subtreeDelete", dn)
	}
	return d.record("delete", dn)
}

func (d *fakeDirectory) writes() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.dispatched...)
}

type source struct {
	dir     Directory
	profile profiles.Profile
}

func (s source) Directory(string) (Directory, error)      { return s.dir, nil }
func (s source) Profile(string) (profiles.Profile, error) { return s.profile, nil }

func testEntry(dn string, values ...string) ldapx.Entry {
	byteValues := make([][]byte, 0, len(values))
	for _, v := range values {
		byteValues = append(byteValues, []byte(v))
	}
	return ldapx.Entry{
		DN: dn,
		Attributes: []ldapx.Attribute{
			{Type: "mail", Values: byteValues},
			{Type: "entryCSN", Values: [][]byte{[]byte("20260901120000.000000Z#000000#000#000000")}},
		},
	}
}

func writableProfile() profiles.Profile {
	return profiles.Profile{ID: "p1", Name: "corp-dev", Host: "ldap.example.com", Port: 389}
}

func newTestPipeline(t *testing.T, dir Directory, p profiles.Profile) *Pipeline {
	t.Helper()
	s := source{dir: dir, profile: p}
	return NewPipeline(s, s, Guard{RequireProductionConfirmation: true})
}

func replaceMail(dn, value string) Input {
	return Input{
		ProfileID: "p1",
		Kind:      KindModify,
		Ops: []Operation{{
			DN:        dn,
			Type:      OpReplaceAttr,
			Attribute: ldapx.Attribute{Type: "mail", Values: [][]byte{[]byte(value)}},
		}},
	}
}

// Contract C13 — a read-only profile is refused a preview token at the
// changeset boundary, before any UI check.

func TestReadOnlyProfileIsRefusedAPreviewToken(t *testing.T) {
	dir := newFakeDirectory(testEntry("cn=a,dc=example,dc=com", "a@example.com"))
	profile := writableProfile()
	profile.ReadOnly = true

	p := newTestPipeline(t, dir, profile)

	_, err := p.Preview(context.Background(), replaceMail("cn=a,dc=example,dc=com", "b@example.com"), false)

	var readOnly *ErrReadOnly
	if !errors.As(err, &readOnly) {
		t.Fatalf("Preview on a read-only profile returned %v, want ErrReadOnly", err)
	}
	if len(dir.writes()) != 0 {
		t.Errorf("a read-only profile reached the server: %v", dir.writes())
	}
}

func TestProductionProfileNeedsItsExtraConfirmation(t *testing.T) {
	dir := newFakeDirectory(testEntry("cn=a,dc=example,dc=com", "a@example.com"))
	profile := writableProfile()
	profile.Tags = []string{profiles.TagProduction}

	p := newTestPipeline(t, dir, profile)
	in := replaceMail("cn=a,dc=example,dc=com", "b@example.com")

	var unconfirmed *ErrProductionUnconfirmed
	if _, err := p.Preview(context.Background(), in, false); !errors.As(err, &unconfirmed) {
		t.Fatalf("unconfirmed production preview returned %v, want ErrProductionUnconfirmed", err)
	}

	preview, err := p.Preview(context.Background(), in, true)
	if err != nil {
		t.Fatalf("confirmed production preview failed: %v", err)
	}
	if !preview.Production {
		t.Error("the preview does not tell the user this is a production connection")
	}
}

// Contract C4 — Commit rejects an absent, reused, expired, or stale token.

func TestCommitRejectsAnAbsentToken(t *testing.T) {
	dir := newFakeDirectory()
	p := newTestPipeline(t, dir, writableProfile())

	if _, err := p.Commit(context.Background(), "not-a-token", nil); !errors.Is(err, ErrNoToken) {
		t.Fatalf("Commit with an unknown token returned %v, want ErrNoToken", err)
	}
	if len(dir.writes()) != 0 {
		t.Errorf("a commit with no token reached the server: %v", dir.writes())
	}
}

func TestCommitRejectsAReusedToken(t *testing.T) {
	dir := newFakeDirectory(testEntry("cn=a,dc=example,dc=com", "a@example.com"))
	p := newTestPipeline(t, dir, writableProfile())

	preview, err := p.Preview(context.Background(), replaceMail("cn=a,dc=example,dc=com", "b@example.com"), false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Commit(context.Background(), preview.Token.Token, nil); err != nil {
		t.Fatalf("first commit failed: %v", err)
	}
	if _, err := p.Commit(context.Background(), preview.Token.Token, nil); !errors.Is(err, ErrTokenUsed) {
		t.Fatalf("second commit returned %v, want ErrTokenUsed", err)
	}
	if got := dir.writes(); len(got) != 1 {
		t.Errorf("the change was dispatched %d times, want once: %v", len(got), got)
	}
}

func TestCommitRejectsAnExpiredToken(t *testing.T) {
	dir := newFakeDirectory(testEntry("cn=a,dc=example,dc=com", "a@example.com"))
	p := newTestPipeline(t, dir, writableProfile())

	now := time.Now()
	p.clock = func() time.Time { return now }

	preview, err := p.Preview(context.Background(), replaceMail("cn=a,dc=example,dc=com", "b@example.com"), false)
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(TokenTTL + time.Second)
	if _, err := p.Commit(context.Background(), preview.Token.Token, nil); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expired commit returned %v, want ErrTokenExpired", err)
	}
	if len(dir.writes()) != 0 {
		t.Errorf("an expired token reached the server: %v", dir.writes())
	}
}

func TestCommitRejectsAStaleTokenWhenTheEntryChanged(t *testing.T) {
	dn := "cn=a,dc=example,dc=com"
	dir := newFakeDirectory(testEntry(dn, "a@example.com"))
	p := newTestPipeline(t, dir, writableProfile())

	preview, err := p.Preview(context.Background(), replaceMail(dn, "b@example.com"), false)
	if err != nil {
		t.Fatal(err)
	}

	// Somebody else edits the entry between preview and commit.
	dir.mu.Lock()
	changed := testEntry(dn, "someone-else@example.com")
	changed.Attributes[1].Values = [][]byte{[]byte("20260901130000.000000Z#000000#000#000000")}
	dir.entries[dn] = changed
	dir.mu.Unlock()

	var stale *ErrStale
	if _, err := p.Commit(context.Background(), preview.Token.Token, nil); !errors.As(err, &stale) {
		t.Fatalf("stale commit returned %v, want ErrStale", err)
	}
	if len(dir.writes()) != 0 {
		t.Errorf("a stale commit reached the server: %v", dir.writes())
	}
}

func TestAddIsStaleIfSomethingAppearedAtTheTargetDN(t *testing.T) {
	dn := "cn=new,dc=example,dc=com"
	dir := newFakeDirectory()
	p := newTestPipeline(t, dir, writableProfile())

	preview, err := p.Preview(context.Background(), Input{
		ProfileID: "p1",
		Kind:      KindAdd,
		Ops: []Operation{{
			DN:         dn,
			Type:       OpAddEntry,
			Attributes: []ldapx.Attribute{{Type: "cn", Values: [][]byte{[]byte("new")}}},
		}},
	}, false)
	if err != nil {
		t.Fatalf("previewing an add against an absent DN failed: %v", err)
	}

	dir.mu.Lock()
	dir.entries[dn] = testEntry(dn, "occupied@example.com")
	dir.mu.Unlock()

	var stale *ErrStale
	if _, err := p.Commit(context.Background(), preview.Token.Token, nil); !errors.As(err, &stale) {
		t.Fatalf("commit returned %v, want ErrStale — the DN is no longer free", err)
	}
}

func TestDiscardDropsTheTokenAndKeepsTheDraft(t *testing.T) {
	dn := "cn=a,dc=example,dc=com"
	dir := newFakeDirectory(testEntry(dn, "a@example.com"))
	p := newTestPipeline(t, dir, writableProfile())

	preview, err := p.Preview(context.Background(), replaceMail(dn, "b@example.com"), false)
	if err != nil {
		t.Fatal(err)
	}

	// The draft is readable while the token stands.
	if _, err := p.Pending(preview.Token.Token); err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if err := p.Discard(preview.Token.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Commit(context.Background(), preview.Token.Token, nil); !errors.Is(err, ErrNoToken) {
		t.Errorf("a discarded token still committed: %v", err)
	}
}

func TestPreviewShowsTheDiffTheCountAndTheTargetServer(t *testing.T) {
	dn := "cn=a,dc=example,dc=com"
	dir := newFakeDirectory(testEntry(dn, "old@example.com"))
	p := newTestPipeline(t, dir, writableProfile())

	preview, err := p.Preview(context.Background(), replaceMail(dn, "new@example.com"), false)
	if err != nil {
		t.Fatal(err)
	}

	if preview.ServerIdentity == "" || preview.ProfileName == "" {
		t.Error("a preview must name the connection and the server it targets")
	}
	if preview.ChangeSet.AffectedCount != 1 {
		t.Errorf("AffectedCount = %d, want 1", preview.ChangeSet.AffectedCount)
	}

	op := preview.ChangeSet.Ops[0]
	if len(op.Before) != 1 || string(op.Before[0]) != "old@example.com" {
		t.Errorf("Before = %q, want the value currently on the server", op.Before)
	}
	if len(op.After) != 1 || string(op.After[0]) != "new@example.com" {
		t.Errorf("After = %q, want the value being written", op.After)
	}
}

func TestEntryVersionUsesTheServersOwnChangeMarker(t *testing.T) {
	entry := testEntry("cn=a,dc=example,dc=com", "a@example.com")
	if got := EntryVersion(entry); got[:8] != "entryCSN" {
		t.Errorf("EntryVersion = %q, want the server's entryCSN where it publishes one", got)
	}
}

func TestEntryVersionFallsBackToADigestOfTheEntry(t *testing.T) {
	// A server publishing no change markers must still get a guard.
	a := ldapx.Entry{DN: "cn=a", Attributes: []ldapx.Attribute{
		{Type: "mail", Values: [][]byte{[]byte("a@example.com")}},
	}}
	b := ldapx.Entry{DN: "cn=a", Attributes: []ldapx.Attribute{
		{Type: "mail", Values: [][]byte{[]byte("b@example.com")}},
	}}

	if EntryVersion(a) == EntryVersion(b) {
		t.Error("two different entries produced the same version; the guard would not fire")
	}
	if EntryVersion(a) != EntryVersion(a) {
		t.Error("the version is not stable for an unchanged entry")
	}
}

func TestMultiEntryCommitReportsEveryEntry(t *testing.T) {
	dir := newFakeDirectory(
		testEntry("cn=a,dc=example,dc=com", "a@example.com"),
		testEntry("cn=b,dc=example,dc=com", "b@example.com"),
	)
	p := newTestPipeline(t, dir, writableProfile())

	in := Input{ProfileID: "p1", Kind: KindBulkModify, Ops: []Operation{
		{
			DN: "cn=a,dc=example,dc=com", Type: OpReplaceAttr,
			Attribute: ldapx.Attribute{Type: "mail", Values: [][]byte{[]byte("x@example.com")}},
		},
		{
			DN: "cn=b,dc=example,dc=com", Type: OpReplaceAttr,
			Attribute: ldapx.Attribute{Type: "mail", Values: [][]byte{[]byte("y@example.com")}},
		},
	}}

	preview, err := p.Preview(context.Background(), in, false)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ChangeSet.AffectedCount != 2 {
		t.Errorf("AffectedCount = %d, want 2 — the count is enumerated, not estimated",
			preview.ChangeSet.AffectedCount)
	}

	rep := &testReporter{}
	if _, err := p.Commit(context.Background(), preview.Token.Token, rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.outcomes) != 2 {
		t.Errorf("got %d outcomes, want one per entry", len(rep.outcomes))
	}
}

type testReporter struct {
	outcomes []string
}

func (r *testReporter) Outcome(dn, status string, _ *ldapx.Result) {
	r.outcomes = append(r.outcomes, dn+" "+status)
}
func (r *testReporter) Progress(int, int, string) {}
