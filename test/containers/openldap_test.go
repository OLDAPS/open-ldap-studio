//go:build integration

package containers

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The readiness probe waits for exactly this many entries, so a miscount either
// hangs the fixture (too high) or declares it ready while it is still loading
// (too low). It needs no Docker.
func TestCountSeedEntries(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	a := write("00.ldif", strings.Join([]string{
		"# dn: cn=commented,dc=example,dc=org",
		"dn: dc=example,dc=org",
		"objectClass: dcObject",
		"description: the text dn: cn=inside,dc=example,dc=org is a value, not an entry",
		"",
		"dn: ou=people,dc=example,dc=org",
		"objectClass: organizationalUnit",
		"",
	}, "\n"))
	b := write("01.ldif", strings.Join([]string{
		"dn:: Y249Ymlu77+9LGRjPWV4YW1wbGUsZGM9b3Jn", // base64 form
		"cn: x",
		"jpegPhoto:: /9j/4AAQSkZJRgABAQEASABIAAD/",
		" 2wBDAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMDAsKCwsNDhIQDQ4RDgsLEBYQERMUFRUVDA8XGBYUGBIUFRT",
		" dn: this continuation line only looks like an entry",
		"",
		"dn: cn=last,dc=example,dc=org",
	}, "\n")) // no trailing newline

	got, err := countSeedEntries([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if got != 4 {
		t.Errorf("countSeedEntries = %d, want 4 (base, people, the base64 DN, last)", got)
	}

	if _, err := countSeedEntries([]string{filepath.Join(dir, "missing.ldif")}); err == nil {
		t.Error("a missing seed file must be an error, not a count of zero")
	}
}

// The real seed directory agrees with an independent count, and the probe
// script carries that number.
func TestReadyScriptUsesTheRealSeedCount(t *testing.T) {
	dir, err := seedDir()
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.ldif"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no seed files in %s (%v)", dir, err)
	}
	got, err := countSeedEntries(files)
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`(?m)^dn:`)
	want := 0
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		want += len(re.FindAll(b, -1))
	}
	if got != want || got == 0 {
		t.Fatalf("countSeedEntries = %d, independent count = %d", got, want)
	}

	if s := readyScript(got); !strings.Contains(s, "-eq "+strconv.Itoa(got)) {
		t.Errorf("the probe does not wait for %d entries:\n%s", got, s)
	}
}
