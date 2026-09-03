package profiles

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/open-ldap-studio/open-ldap-studio/internal/prefs"
)

// SchemaVersion of the profile store's on-disk format.
const SchemaVersion = 1

// Store holds connection profiles and their folders, backed by one file.
type Store struct {
	mu       sync.RWMutex
	path     string
	profiles []Profile
	folders  []Folder
}

// OpenStore loads the store at path, treating a missing file as an empty one.
func OpenStore(path string) (*Store, error) {
	s := &Store{path: path}

	var loaded struct {
		Profiles []Profile `json:"profiles"`
		Folders  []Folder  `json:"folders"`
	}
	_, err := prefs.Read(path, SchemaVersion, &loaded)
	switch {
	case err == nil:
		s.profiles, s.folders = loaded.Profiles, loaded.Folders
	case os.IsNotExist(err):
		// An empty store is where every install starts.
	default:
		// A store from a newer version is refused, not rewritten (contract F2).
		return nil, err
	}
	return s, nil
}

// Path is the file backing the store.
func (s *Store) Path() string { return s.path }

func (s *Store) persist() error {
	return prefs.Write(s.path, SchemaVersion, "profiles", struct {
		Profiles []Profile `json:"profiles"`
		Folders  []Folder  `json:"folders"`
	}{Profiles: s.profiles, Folders: s.folders})
}

// List returns the summaries the connections view renders.
func (s *Store) List() []Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Summary, 0, len(s.profiles))
	for _, p := range s.profiles {
		out = append(out, p.Summarise())
	}
	return out
}

// ErrNotFound is returned for an unknown profile or folder id.
var ErrNotFound = errors.New("profiles: no such profile")

// Get returns one profile.
func (s *Store) Get(id string) (Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.profiles {
		if p.ID == id {
			return p, nil
		}
	}
	return Profile{}, fmt.Errorf("%w: %s", ErrNotFound, id)
}

// Profile satisfies the changeset pipeline's ProfileSource.
func (s *Store) Profile(id string) (Profile, error) { return s.Get(id) }

// Save creates or replaces a profile, applying defaults and validating it.
func (s *Store) Save(p Profile) (Profile, error) {
	if err := Validate(p); err != nil {
		return Profile{}, err
	}
	p = applyDefaults(p)

	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if index := slices.IndexFunc(s.profiles, func(existing Profile) bool { return existing.ID == p.ID }); index >= 0 {
		s.profiles[index] = p
	} else {
		s.profiles = append(s.profiles, p)
	}
	if err := s.persist(); err != nil {
		return Profile{}, err
	}
	return p, nil
}

// Delete removes a profile.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := slices.IndexFunc(s.profiles, func(p Profile) bool { return p.ID == id })
	if index < 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	s.profiles = slices.Delete(s.profiles, index, index+1)
	return s.persist()
}

// Duplicate copies a profile under a new id and name.
func (s *Store) Duplicate(id string) (Profile, error) {
	original, err := s.Get(id)
	if err != nil {
		return Profile{}, err
	}
	copied := original
	copied.ID = ""
	copied.Name = original.Name + " (copy)"
	return s.Save(copied)
}

// Folders returns every folder.
func (s *Store) Folders() []Folder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Folder(nil), s.folders...)
}

// SaveFolder creates or replaces a folder, rejecting a cycle.
func (s *Store) SaveFolder(f Folder) (Folder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	if strings.TrimSpace(f.Name) == "" {
		return Folder{}, errors.New("profiles: a folder needs a name")
	}
	if f.ParentID == f.ID {
		return Folder{}, errors.New("profiles: a folder cannot contain itself")
	}

	candidate := append([]Folder(nil), s.folders...)
	if index := slices.IndexFunc(candidate, func(existing Folder) bool { return existing.ID == f.ID }); index >= 0 {
		candidate[index] = f
	} else {
		candidate = append(candidate, f)
	}
	if err := checkCycles(candidate); err != nil {
		return Folder{}, err
	}

	s.folders = candidate
	if err := s.persist(); err != nil {
		return Folder{}, err
	}
	return f, nil
}

func checkCycles(folders []Folder) error {
	parent := make(map[string]string, len(folders))
	for _, f := range folders {
		parent[f.ID] = f.ParentID
	}
	for _, f := range folders {
		seen := map[string]bool{f.ID: true}
		for current := f.ParentID; current != ""; current = parent[current] {
			if seen[current] {
				return fmt.Errorf("profiles: folder %q would form a cycle", f.Name)
			}
			seen[current] = true
		}
	}
	return nil
}

// Validate checks a profile before it is stored.
func Validate(p Profile) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("profiles: a connection needs a name")
	}
	if strings.TrimSpace(p.Host) == "" {
		return errors.New("profiles: a connection needs a host")
	}
	if p.Port < 0 || p.Port > 65535 {
		return fmt.Errorf("profiles: port %d is out of range", p.Port)
	}
	switch p.Encryption {
	case EncryptionNone, EncryptionStartTLS, EncryptionLDAPS, "":
	default:
		return fmt.Errorf("profiles: unknown encryption mode %q", p.Encryption)
	}
	switch p.BindMethod {
	case BindAnonymous, BindSimple, BindExternal, BindGSSAPI, BindDigestMD5, BindCramMD5, "":
	default:
		return fmt.Errorf("profiles: unknown bind method %q", p.BindMethod)
	}
	return nil
}

func applyDefaults(p Profile) Profile {
	if p.Port == 0 {
		p.Port = DefaultPort(p.Encryption)
	}
	if p.Encryption == "" {
		p.Encryption = EncryptionNone
	}
	if p.BindMethod == "" {
		p.BindMethod = BindAnonymous
	}
	if p.Aliases == "" {
		p.Aliases = AliasNever
	}
	if p.Referrals == "" {
		p.Referrals = ReferralIgnore
	}
	if p.Timeouts.ConnectMS == 0 {
		p.Timeouts.ConnectMS = 30_000
	}
	if p.Timeouts.ReadMS == 0 {
		p.Timeouts.ReadMS = 60_000
	}
	if p.Limits.PageSize == 0 {
		p.Limits.PageSize = 100
	}
	// TLS verification is on unless the user explicitly turned it off, and a
	// profile that has never been saved has not turned it off.
	if p.Encryption != EncryptionNone && !p.TLS.VerifyCertificate && !p.TLS.VerifyHostname {
		p.TLS.VerifyCertificate = true
		p.TLS.VerifyHostname = true
	}
	if p.SchemaVersion == 0 {
		p.SchemaVersion = SchemaVersion
	}
	return p
}

// DecodeStrict decodes a profile payload, rejecting any field the type does
// not declare.
//
// This is the runtime half of "a Profile has nowhere to put a secret": a
// payload carrying a password field is refused outright rather than silently
// dropped, so a frontend that tries to store one finds out (bridge-api.md
// § Connections, SC-008).
func DecodeStrict(payload []byte) (Profile, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var p Profile
	if err := decoder.Decode(&p); err != nil {
		return Profile{}, fmt.Errorf(
			"profiles: this payload cannot be stored as a connection profile (%w). "+
				"Secrets in particular belong in the platform credential store, never in a profile", err)
	}
	return p, nil
}
