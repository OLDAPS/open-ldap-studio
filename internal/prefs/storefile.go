package prefs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Every file this application writes carries the same envelope, and
// schemaVersion is its first key.
//
// The rule exists so that a file written by a newer version is recognisable
// before it is parsed, and can be refused rather than rewritten. Silently
// downgrading a profile store is how a user loses connections they still have
// in another install (contracts F1, F2, FR-108).
type Envelope struct {
	// SchemaVersion is declared first, so encoding/json writes it first.
	SchemaVersion int `json:"schemaVersion"`
	// Kind names what the file holds, so a misplaced file is diagnosable.
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

// ErrNewerSchema is returned when a file was written by a newer version of the
// application. The caller must leave the file alone and say so.
type ErrNewerSchema struct {
	Path      string
	Found     int
	Supported int
}

func (e *ErrNewerSchema) Error() string {
	return fmt.Sprintf(
		"%s was written by a newer version of Open LDAP Studio (schema %d; this build understands %d). It has been left untouched.",
		e.Path, e.Found, e.Supported)
}

// Write serialises v into the envelope and writes it atomically with
// owner-only permissions.
//
// The write is atomic because a store truncated by a crash is worse than a
// stale one: profiles, trust decisions and history all live in these files.
func Write(path string, schemaVersion int, kind string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("prefs: cannot serialise %s: %w", kind, err)
	}

	encoded, err := json.MarshalIndent(Envelope{
		SchemaVersion: schemaVersion,
		Kind:          kind,
		Data:          data,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("prefs: cannot serialise the %s envelope: %w", kind, err)
	}
	encoded = append(encoded, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("prefs: cannot create %s: %w", filepath.Dir(path), err)
	}

	temp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("prefs: cannot write %s: %w", path, err)
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	if err := temp.Chmod(0o600); err != nil {
		return fmt.Errorf("prefs: cannot set permissions on %s: %w", path, err)
	}
	if _, err := temp.Write(encoded); err != nil {
		return fmt.Errorf("prefs: cannot write %s: %w", path, err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("prefs: cannot flush %s: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("prefs: cannot close %s: %w", path, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("prefs: cannot replace %s: %w", path, err)
	}
	return nil
}

// Read loads a file into v, refusing anything written by a newer version.
//
// A missing file is not an error: it means the store is empty, which is the
// state every install starts in. The caller distinguishes it with os.IsNotExist
// on the wrapped error.
func Read(path string, supportedVersion int, v any) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return 0, fmt.Errorf("prefs: %s is not a readable store file: %w", path, err)
	}
	if envelope.SchemaVersion > supportedVersion {
		return envelope.SchemaVersion, &ErrNewerSchema{
			Path: path, Found: envelope.SchemaVersion, Supported: supportedVersion,
		}
	}
	if len(envelope.Data) == 0 {
		return envelope.SchemaVersion, errors.New("prefs: the store file carries no data")
	}
	if err := json.Unmarshal(envelope.Data, v); err != nil {
		return envelope.SchemaVersion, fmt.Errorf("prefs: cannot read %s: %w", path, err)
	}
	return envelope.SchemaVersion, nil
}
