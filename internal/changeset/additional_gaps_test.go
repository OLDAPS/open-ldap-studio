package changeset

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

func TestUnchangedUUIDCannotHideChangedValues(t *testing.T) {
	const dn = "cn=a,dc=example"
	entry := ldapx.Entry{DN: dn, Attributes: []ldapx.Attribute{
		{Type: "entryUUID", Values: [][]byte{[]byte("stable-id")}},
		{Type: "mail", Values: [][]byte{[]byte("old@example.org")}},
	}}
	dir := newFakeDirectory(entry)
	pipeline := newTestPipeline(t, dir, writableProfile())
	preview, err := pipeline.Preview(t.Context(), replaceMail(dn, "ours@example.org"), false)
	if err != nil {
		t.Fatal(err)
	}
	changed := entry
	changed.Attributes = append([]ldapx.Attribute(nil), entry.Attributes...)
	changed.Attributes[1].Values = [][]byte{[]byte("theirs@example.org")}
	dir.mu.Lock()
	dir.entries[dn] = changed
	dir.mu.Unlock()
	var stale *ErrStale
	if _, err := pipeline.Commit(t.Context(), preview.Token.Token, nil); !errors.As(err, &stale) {
		t.Fatalf("changed content with stable UUID committed: %v", err)
	}
	if len(dir.writes()) != 0 {
		t.Fatalf("stale content reached the server: %v", dir.writes())
	}
}

func TestEntryDigestFramesValuesAndCounts(t *testing.T) {
	one := ldapx.Entry{DN: "cn=a", Attributes: []ldapx.Attribute{{
		Type: "mail", Values: [][]byte{[]byte{'a', 1, 'b'}},
	}}}
	two := ldapx.Entry{DN: "cn=a", Attributes: []ldapx.Attribute{{
		Type: "mail", Values: [][]byte{[]byte("a"), []byte("b")},
	}}}
	if EntryVersion(one) == EntryVersion(two) {
		t.Fatal("one binary value collided with two separate values")
	}
	withMarker := one
	withMarker.Attributes = append(append([]ldapx.Attribute(nil), one.Attributes...), ldapx.Attribute{
		Type: "modifyTimestamp", Values: [][]byte{[]byte("20261103120000Z")},
	})
	changed := withMarker
	changed.Attributes = append([]ldapx.Attribute(nil), withMarker.Attributes...)
	changed.Attributes[0].Values = [][]byte{[]byte("changed")}
	if EntryVersion(withMarker) == EntryVersion(changed) {
		t.Fatal("an unchanged timestamp hid changed entry content")
	}
}

func TestSubtreeDeleteIssuesNoTokenWithoutEnumeration(t *testing.T) {
	root := testEntry("ou=people,dc=example", "root")
	child := testEntry("cn=a,ou=people,dc=example", "child")
	dir := newFakeDirectory(root, child)
	pipeline := newTestPipeline(t, dir, writableProfile())
	preview, err := pipeline.Preview(t.Context(), Input{
		ProfileID: "p1", Kind: KindSubtreeDelete,
		Ops: []Operation{{DN: root.DN, Type: OpDeleteEntry}},
	}, false)
	if !errors.Is(err, ErrSubtreePreviewUnavailable) || preview.Token.Token != "" {
		t.Fatalf("uncounted subtree received a confirmation token: %+v, %v", preview, err)
	}
	if len(pipeline.issued) != 0 || len(dir.writes()) != 0 {
		t.Fatal("subtree delete was staged or dispatched despite the missing count")
	}
}

type mutableSource struct {
	mu      sync.Mutex
	dir     Directory
	profile profiles.Profile
}

func (s *mutableSource) Directory(string) (Directory, error) { return s.dir, nil }
func (s *mutableSource) Profile(string) (profiles.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.profile, nil
}

func TestProductionPolicyChangeRequiresFreshPreview(t *testing.T) {
	const dn = "cn=a,dc=example"
	for _, initialProduction := range []bool{false, true} {
		profile := writableProfile()
		if initialProduction {
			profile.Tags = []string{profiles.TagProduction}
		}
		dir := newFakeDirectory(testEntry(dn, "old@example.org"))
		source := &mutableSource{dir: dir, profile: profile}
		pipeline := NewPipeline(source, source, Guard{RequireProductionConfirmation: true})
		preview, err := pipeline.Preview(t.Context(), replaceMail(dn, "new@example.org"), initialProduction)
		if err != nil {
			t.Fatal(err)
		}
		source.mu.Lock()
		if initialProduction {
			source.profile.Tags = nil
		} else {
			source.profile.Tags = []string{profiles.TagProduction}
		}
		source.mu.Unlock()
		if _, err := pipeline.Commit(t.Context(), preview.Token.Token, nil); !errors.Is(err, ErrPolicyChanged) {
			t.Fatalf("production policy changed from %t but commit returned %v", initialProduction, err)
		}
		if len(dir.writes()) != 0 {
			t.Fatal("commit reached the server after its confirmation policy changed")
		}
	}
}

type failSecondDirectory struct{ *fakeDirectory }

func (d *failSecondDirectory) modify(ctx context.Context, dn string, changes []ldapx.Change) (ldapx.Result, error) {
	if dn == "cn=two" {
		result := ldapx.NewResult(ldapx.InsufficientAccessRights, dn, "denied")
		d.mu.Lock()
		d.dispatched = append(d.dispatched, "modify "+dn)
		d.mu.Unlock()
		return result, ldapx.ServerError("modify", result)
	}
	return d.fakeDirectory.modify(ctx, dn, changes)
}

type stoppingReporter struct{ testReporter }

func (*stoppingReporter) StopOnError() bool { return true }

func TestSynchronousOutcomeTrackingStopsAfterFirstRejectedWrite(t *testing.T) {
	dir := &failSecondDirectory{newFakeDirectory(
		testEntry("cn=one", "a"), testEntry("cn=two", "b"), testEntry("cn=three", "c"),
	)}
	pipeline := newTestPipeline(t, dir, writableProfile())
	input := Input{ProfileID: "p1", Kind: KindBulkModify, Ops: []Operation{
		replaceMail("cn=one", "a2").Ops[0],
		replaceMail("cn=two", "b2").Ops[0],
		replaceMail("cn=three", "c2").Ops[0],
	}}
	preview, err := pipeline.Preview(t.Context(), input, false)
	if err != nil {
		t.Fatal(err)
	}
	reporter := &stoppingReporter{}
	if _, err := pipeline.Commit(t.Context(), preview.Token.Token, reporter); !errors.Is(err, ldapx.ErrServer) {
		t.Fatalf("rejected second write returned %v", err)
	}
	if got := dir.writes(); len(got) != 2 {
		t.Fatalf("synchronous commit dispatched after rejection: %v", got)
	}
	want := []string{"cn=one succeeded", "cn=two failed", "cn=three skipped"}
	if got := reporter.outcomes; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("operation outcomes = %v, want %v", got, want)
	}
}
