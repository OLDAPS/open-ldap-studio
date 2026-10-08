//go:build integration

package bridge

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// connectLive saves a profile, stores its secret and opens it, returning the
// profile id. It is the state every tree test starts from.
func connectLive(t *testing.T) (*Bridge, string) {
	t.Helper()
	b := liveBridge(t)

	saved, err := b.SaveProfile(encode(t, liveProfile()))
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if err := b.StoreProfileSecret(saved.ID, livePass); err != nil {
		t.Fatalf("StoreProfileSecret: %v", err)
	}
	if _, err := b.Connect(saved.ID); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	waitForState(t, b, saved.ID, StateConnected)
	return b, saved.ID
}

// The tree's roots come from the Root DSE. Without naming contexts there is
// nothing to draw, so this is the first thing that must work.
func TestLiveTreeRoots(t *testing.T) {
	b, id := connectLive(t)

	dse, _, err := b.RootDSE(id)
	if err != nil {
		t.Fatalf("RootDSE: %v", err)
	}
	if len(dse.NamingContexts) == 0 {
		t.Fatal("no naming contexts; the tree would have no roots")
	}
	t.Logf("naming contexts: %v", dse.NamingContexts)
}

// One level under the base must return the fixture's four containers, each
// reporting whether it has children — the tri-state the tree renders.
func TestLiveTreeChildren(t *testing.T) {
	b, id := connectLive(t)

	page, err := b.ListChildren(id, liveBase, ldapx.PageRequest{})
	if err != nil {
		t.Fatalf("ListChildren: %v", err)
	}
	if len(page.Entries) != 4 {
		t.Errorf("got %d children of the base, want 4", len(page.Entries))
	}

	// The fixture is built so this is checkable: three containers hold
	// entries and ou=archive is deliberately empty.
	want := map[string]ldapx.Children{
		"ou=people," + liveBase:   ldapx.ChildrenYes,
		"ou=groups," + liveBase:   ldapx.ChildrenYes,
		"ou=services," + liveBase: ldapx.ChildrenYes,
		"ou=archive," + liveBase:  ldapx.ChildrenNo,
	}
	for _, e := range page.Entries {
		t.Logf("%-40s hasChildren=%s", e.DN, e.HasChildren)
		if expected, ok := want[e.DN]; ok && e.HasChildren != expected {
			t.Errorf("%s hasChildren = %q, want %q", e.DN, e.HasChildren, expected)
		}
	}
}

// A leaf must report itself as a leaf, or the tree offers a twisty on every
// person and expanding it searches for children that cannot exist.
func TestLiveTreeLeavesAreLeaves(t *testing.T) {
	b, id := connectLive(t)

	page, err := b.ListChildren(id, "ou=people,"+liveBase, ldapx.PageRequest{Size: 5})
	if err != nil {
		t.Fatalf("ListChildren: %v", err)
	}
	if len(page.Entries) == 0 {
		t.Fatal("no people returned")
	}
	for _, e := range page.Entries {
		if e.HasChildren != ldapx.ChildrenNo {
			t.Errorf("%s hasChildren = %q, want %q", e.DN, e.HasChildren, ldapx.ChildrenNo)
		}
	}
}

// The headline behaviour of screen 1b: 205 people, a page size of 100, and an
// explicit "fetch more" row driven by the cookie.
func TestLiveTreePaging(t *testing.T) {
	b, id := connectLive(t)

	people := "ou=people," + liveBase
	first, err := b.ListChildren(id, people, ldapx.PageRequest{Size: 100})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Entries) != 100 {
		t.Fatalf("first page has %d entries, want 100", len(first.Entries))
	}
	if len(first.Cookie) == 0 {
		t.Fatal("no cookie on the first page; the tree could never fetch the rest")
	}

	// Walk every page the way the "fetch more" row does.
	seen := map[string]bool{}
	for _, e := range first.Entries {
		seen[e.DN] = true
	}

	cookie := first.Cookie
	pages := 1
	for len(cookie) > 0 && pages < 10 {
		next, err := b.ListChildren(id, people, ldapx.PageRequest{Size: 100, Cookie: cookie})
		if err != nil {
			t.Fatalf("page %d: %v", pages+1, err)
		}
		for _, e := range next.Entries {
			if seen[e.DN] {
				t.Errorf("page %d repeated %s", pages+1, e.DN)
			}
			seen[e.DN] = true
		}
		cookie = next.Cookie
		pages++
	}

	if len(seen) != 205 {
		t.Errorf("walked %d distinct entries over %d pages, want 205", len(seen), pages)
	}
	t.Logf("%d entries over %d pages", len(seen), pages)
}

// The cookie crosses the bridge as JSON, so it must survive being marshalled
// and handed back — the tree round-trips it verbatim.
func TestLiveTreeCookieSurvivesJSON(t *testing.T) {
	b, id := connectLive(t)
	people := "ou=people," + liveBase

	first, err := b.ListChildren(id, people, ldapx.PageRequest{Size: 100})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}

	wire, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal page: %v", err)
	}

	var decoded ldapx.Page
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatalf("unmarshal page: %v", err)
	}

	// Feed the round-tripped cookie back, exactly as the frontend does.
	second, err := b.ListChildren(id, people, ldapx.PageRequest{Size: 100, Cookie: decoded.Cookie})
	if err != nil {
		t.Fatalf("continuation with a JSON round-tripped cookie: %v", err)
	}
	if len(second.Entries) == 0 {
		t.Fatal("the continuation returned nothing; the cookie did not survive JSON")
	}
}

// What the entry editor renders. Values are bytes on this side and base64 over
// the wire; the awkward ones in the fixture are the point of this test.
func TestLiveTreeReadEntry(t *testing.T) {
	b, id := connectLive(t)

	entry, _, err := b.ReadEntry(id, "cn=jrivera,ou=people,"+liveBase, ldapx.ReadOptions{})
	if err != nil {
		t.Fatalf("ReadEntry: %v", err)
	}
	if entry.DN == "" {
		t.Fatal("entry has no DN")
	}

	if _, ok := entry.Attribute("jpegPhoto"); !ok {
		t.Error("no jpegPhoto; the entry-info photo panel would have nothing to show")
	}
	mail, ok := entry.Attribute("mail")
	if !ok || len(mail.Values) != 2 {
		t.Errorf("mail has %d values, want 2", len(mail.Values))
	}

	// Base64 in, same bytes out.
	wire, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal entry: %v", err)
	}
	var decoded ldapx.Entry
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatalf("unmarshal entry: %v", err)
	}
	photo, _ := decoded.Attribute("jpegPhoto")
	if len(photo.Values) == 0 || len(photo.Values[0]) < 100 {
		t.Error("jpegPhoto did not survive the JSON round trip")
	}
	if photo.Values[0][0] != 0xFF || photo.Values[0][1] != 0xD8 {
		t.Error("jpegPhoto is not a JPEG after the round trip")
	}

	// The non-ASCII entry must arrive intact, not mangled.
	tomas, _, err := b.ReadEntry(id, "cn=tvasquez,ou=people,"+liveBase, ldapx.ReadOptions{})
	if err != nil {
		t.Fatalf("ReadEntry tvasquez: %v", err)
	}
	sn, _ := tomas.Attribute("sn")
	if len(sn.Values) == 0 || string(sn.Values[0]) != "Vásquez" {
		t.Errorf("sn = %q, want %q", string(sn.Values[0]), "Vásquez")
	}

	// The leading-space value must keep its space.
	lchen, _, err := b.ReadEntry(id, "cn=lchen,ou=people,"+liveBase, ldapx.ReadOptions{})
	if err != nil {
		t.Fatalf("ReadEntry lchen: %v", err)
	}
	description, _ := lchen.Attribute("description")
	if len(description.Values) == 0 || !strings.HasPrefix(string(description.Values[0]), " leading space") {
		t.Errorf("description lost its leading space: %q", string(description.Values[0]))
	}
}

// Operational attributes are opt-in and must actually arrive when asked for,
// or the entry-info panel's timestamps are permanently blank.
func TestLiveTreeOperationalAttributes(t *testing.T) {
	b, id := connectLive(t)
	dn := "cn=jrivera,ou=people," + liveBase

	plain, _, err := b.ReadEntry(id, dn, ldapx.ReadOptions{})
	if err != nil {
		t.Fatalf("ReadEntry: %v", err)
	}
	if _, ok := plain.Attribute("modifyTimestamp"); ok {
		t.Error("modifyTimestamp came back without being asked for")
	}

	withOp, _, err := b.ReadEntry(id, dn, ldapx.ReadOptions{IncludeOperational: true})
	if err != nil {
		t.Fatalf("ReadEntry with operational: %v", err)
	}
	if _, ok := withOp.Attribute("modifyTimestamp"); !ok {
		t.Error("modifyTimestamp missing even with operational attributes requested")
	}
}
