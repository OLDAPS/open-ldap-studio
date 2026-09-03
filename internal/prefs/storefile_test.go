package prefs

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type sample struct {
	Name string `json:"name"`
}

// Contract F1 — every written file has schemaVersion as its first key.
func TestSchemaVersionIsTheFirstKeyInTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := Write(path, 1, "profiles", []sample{{Name: "corp-dev"}}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	if _, err := decoder.Token(); err != nil { // opening brace
		t.Fatal(err)
	}
	first, err := decoder.Token()
	if err != nil {
		t.Fatal(err)
	}
	if first != "schemaVersion" {
		t.Fatalf("first key is %v, want schemaVersion — it must be readable before the file is parsed", first)
	}
}

// Contract F2 — a higher version is refused rather than rewritten.
func TestAFileFromANewerVersionIsRefusedAndLeftUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := Write(path, 7, "profiles", []sample{{Name: "from the future"}}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var out []sample
	_, err = Read(path, 1, &out)

	var newer *ErrNewerSchema
	if !errors.As(err, &newer) {
		t.Fatalf("Read returned %v, want ErrNewerSchema", err)
	}
	if newer.Found != 7 || newer.Supported != 1 {
		t.Errorf("error reports found=%d supported=%d, want 7 and 1", newer.Found, newer.Supported)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("the file was modified; a file from a newer version must be left untouched")
	}
}

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	want := []sample{{Name: "corp-dev"}, {Name: "corp-prod"}}
	if err := Write(path, 1, "profiles", want); err != nil {
		t.Fatal(err)
	}

	var got []sample
	version, err := Read(path, 1, &got)
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
	if len(got) != 2 || got[0].Name != "corp-dev" || got[1].Name != "corp-prod" {
		t.Errorf("round trip returned %+v", got)
	}
}

func TestFilesAreOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions do not apply on Windows")
	}
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := Write(path, 1, "profiles", []sample{{Name: "corp-dev"}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions are %o, want 600", perm)
	}
}

func TestWriteReplacesAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")
	if err := Write(path, 1, "profiles", []sample{{Name: "first"}}); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, 1, "profiles", []sample{{Name: "second"}}); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("left %d files behind: %v — the temporary file must not survive", len(entries), names)
	}
}

func TestMissingFileIsNotAnError(t *testing.T) {
	var out []sample
	_, err := Read(filepath.Join(t.TempDir(), "absent.json"), 1, &out)
	if !os.IsNotExist(err) {
		t.Fatalf("Read of a missing file returned %v, want a not-exist error the caller can recognise", err)
	}
}
