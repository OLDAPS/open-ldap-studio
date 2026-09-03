package changeset

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// TokenTTL bounds how long a preview stays committable. A token older than
// this forces a fresh preview, because the diff the user approved is no longer
// evidence about the server's current state.
const TokenTTL = 5 * time.Minute

// Token is issued by Preview and consumed by Commit.
//
// It is opaque, single-use and expiring, and it carries the entry versions the
// preview was computed against. This type is the mechanism behind SC-005:
// Commit accepts nothing else, so a write with no preview is not expressible
// in the API (data-model §3).
type Token struct {
	Token       string `json:"token"`
	ChangeSetID string `json:"changeSetId"`
	// EntryVersions is the concurrency guard: each DN's version is re-read at
	// commit, and a mismatch aborts and forces re-confirmation (FR-052).
	EntryVersions map[string]string `json:"entryVersions"`
	IssuedAt      time.Time         `json:"issuedAt"`
	ExpiresAt     time.Time         `json:"expiresAt"`
}

// Preview is what the user confirms: the target, the diff, the count, and the
// token that will commit exactly this.
type Preview struct {
	ChangeSet ChangeSet `json:"changeSet"`
	// ProfileName and ServerIdentity are displayed on every preview, because
	// "which server is this" is the question a preview exists to answer
	// (FR-010).
	ProfileName    string `json:"profileName"`
	ServerIdentity string `json:"serverIdentity"`
	ReadOnly       bool   `json:"readOnly"`
	Production     bool   `json:"production"`
	Token          Token  `json:"token"`
	// Reversible states plainly whether this change can be undone later, so a
	// user is never left believing in an undo that does not exist (FR-092).
	Reversible bool `json:"reversible"`
}

// Directory is the subset of the protocol the write path needs.
//
// The only implementation that touches ldapx is liveDirectory, in dispatch.go,
// inside this package. The seam exists so the write path can be tested
// without a server; it does not widen the boundary, because a caller outside
// changeset still has no way to reach a mutation function (contract C1).
type Directory interface {
	// ReadEntry reads one entry with its operational attributes, for the diff
	// and for the concurrency guard.
	ReadEntry(ctx context.Context, dn string) (ldapx.Entry, error)
	// Identity is what the preview shows as the target server.
	Identity() string

	add(ctx context.Context, dn string, attributes []ldapx.Attribute) (ldapx.Result, error)
	modify(ctx context.Context, dn string, changes []ldapx.Change) (ldapx.Result, error)
	rename(ctx context.Context, dn, newRDN, newSuperior string, keepOldRDN bool) (ldapx.Result, error)
	del(ctx context.Context, dn string, subtree bool) (ldapx.Result, error)
}

// DirectorySource resolves a profile id to its open connection.
type DirectorySource interface {
	Directory(profileID string) (Directory, error)
}

// ProfileSource resolves a profile id to its stored profile.
type ProfileSource interface {
	Profile(profileID string) (profiles.Profile, error)
}

// Pipeline is the whole write path: build, preview, confirm, dispatch.
type Pipeline struct {
	conns    DirectorySource
	profiles ProfileSource
	guard    Guard

	mu     sync.Mutex
	issued map[string]*issuedToken
	clock  func() time.Time
	ttl    time.Duration
}

type issuedToken struct {
	token     Token
	changeSet ChangeSet
	used      bool
}

// NewPipeline builds the pipeline. There is no second constructor and no way
// to obtain a dispatch without going through it.
func NewPipeline(conns DirectorySource, profileSource ProfileSource, guard Guard) *Pipeline {
	return &Pipeline{
		conns:    conns,
		profiles: profileSource,
		guard:    guard,
		issued:   make(map[string]*issuedToken),
		clock:    time.Now,
		ttl:      TokenTTL,
	}
}

// Errors Commit returns for a token it will not accept. Each is distinct
// because each needs a different affordance: re-preview, re-confirm, or
// nothing at all.
var (
	ErrNoToken      = errors.New("changeset: no such preview token; every commit requires a preview")
	ErrTokenUsed    = errors.New("changeset: this preview has already been committed")
	ErrTokenExpired = errors.New("changeset: this preview has expired; preview the change again")
)

// ErrStale is returned when an entry changed between preview and commit. The
// draft survives; the user is asked to look at the new diff (FR-052).
type ErrStale struct {
	DN       string
	Expected string
	Actual   string
}

func (e *ErrStale) Error() string {
	return fmt.Sprintf("changeset: %s changed on the server since the preview; review the change again", e.DN)
}

// Preview computes the diff and issues a token.
//
// It is refused outright for a read-only profile, and for a production-tagged
// profile without its extra confirmation. That refusal happens here rather
// than in the UI so that no code path can route around it (contract C13).
func (p *Pipeline) Preview(ctx context.Context, in Input, confirmedProduction bool) (Preview, error) {
	profile, err := p.profiles.Profile(in.ProfileID)
	if err != nil {
		return Preview{}, err
	}
	if err := p.guard.Check(profile, confirmedProduction); err != nil {
		return Preview{}, err
	}

	dir, err := p.conns.Directory(in.ProfileID)
	if err != nil {
		return Preview{}, err
	}

	cs := ChangeSet{
		ID:        uuid.NewString(),
		ProfileID: in.ProfileID,
		Kind:      in.Kind,
		Ops:       in.Ops,
	}

	// Capture the current state of every DN the change touches. It is both the
	// diff's left-hand side and the material a reversal would need.
	versions := make(map[string]string, len(cs.Ops))
	for _, dn := range cs.DNs() {
		entry, err := dir.ReadEntry(ctx, dn)
		switch {
		case err == nil:
			cs.BeforeState = append(cs.BeforeState, entry)
			versions[dn] = EntryVersion(entry)
		case in.Kind == KindAdd:
			// An add targets a DN that should not exist yet. Its absence is
			// the expected state, and is itself the version to check at
			// commit: if something appears there in the meantime, the commit
			// must stop.
			versions[dn] = versionAbsent
		default:
			return Preview{}, err
		}
	}

	cs.Ops = fillDiff(cs.Ops, cs.BeforeState)
	cs.AffectedCount = len(cs.DNs())
	cs.Warnings = append(cs.Warnings, p.warningsFor(cs, profile)...)

	token := Token{
		Token:         newTokenString(),
		ChangeSetID:   cs.ID,
		EntryVersions: versions,
		IssuedAt:      p.clock(),
		ExpiresAt:     p.clock().Add(p.ttl),
	}

	p.mu.Lock()
	p.issued[token.Token] = &issuedToken{token: token, changeSet: cs}
	p.mu.Unlock()

	return Preview{
		ChangeSet:      cs,
		ProfileName:    profile.Name,
		ServerIdentity: dir.Identity(),
		ReadOnly:       profile.ReadOnly,
		Production:     profile.IsProduction(),
		Token:          token,
		Reversible:     len(cs.BeforeState) > 0 && cs.Kind != KindSubtreeDelete,
	}, nil
}

// versionAbsent is the version of a DN that does not exist yet.
const versionAbsent = "absent"

// warningsFor states what the user should know before confirming. These are
// warnings, not blocks: the user may have a reason, and a client that refuses
// to send what the user asked for is a client they will work around (FR-063).
func (p *Pipeline) warningsFor(cs ChangeSet, profile profiles.Profile) []Warning {
	var out []Warning
	if profile.IsProduction() {
		out = append(out, Warning{
			Severity: SeverityWarning,
			Message:  "this connection is tagged production",
		})
	}
	switch cs.Kind {
	case KindSubtreeDelete:
		out = append(out, Warning{
			Severity: SeverityWarning,
			Message:  "a subtree delete removes every entry beneath the target and cannot be reversed from history",
		})
	case KindSchemaCommit:
		out = append(out, Warning{
			Severity: SeverityWarning,
			Message:  "a schema change affects every entry the modified definitions apply to",
		})
	case KindConfigModify:
		out = append(out, Warning{
			Severity: SeverityWarning,
			Message:  "a configuration change can take the directory out of service",
		})
	}
	return out
}

// Discard drops a token but keeps the draft: cancelling a confirmation must
// not throw away the work that produced it (FR-040).
func (p *Pipeline) Discard(token string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.issued[token]; !ok {
		return ErrNoToken
	}
	delete(p.issued, token)
	return nil
}

// Pending returns the change set a token would commit, for redisplay.
func (p *Pipeline) Pending(token string) (ChangeSet, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	held, ok := p.issued[token]
	if !ok {
		return ChangeSet{}, ErrNoToken
	}
	return held.changeSet, nil
}

// take validates a token and marks it used. A token is consumed even when the
// commit then fails: the diff it vouched for has been acted on, and a second
// attempt needs a fresh look at the server.
func (p *Pipeline) take(token string) (*issuedToken, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	held, ok := p.issued[token]
	if !ok {
		return nil, ErrNoToken
	}
	if held.used {
		return nil, ErrTokenUsed
	}
	if p.clock().After(held.token.ExpiresAt) {
		delete(p.issued, token)
		return nil, ErrTokenExpired
	}
	held.used = true
	return held, nil
}

// EntryVersion computes the version used as the concurrency guard.
//
// It prefers the server's own change markers where they exist, and falls back
// to a digest of the entry. The fallback matters: a server that publishes
// neither entryCSN nor modifyTimestamp would otherwise have no guard at all.
func EntryVersion(entry ldapx.Entry) string {
	for _, name := range []string{"entryCSN", "modifyTimestamp", "entryUUID"} {
		if attr, ok := entry.Attribute(name); ok && len(attr.Values) > 0 {
			return name + "=" + string(attr.Values[0])
		}
	}

	digest := sha256.New()
	digest.Write([]byte(entry.DN))

	descriptions := make([]string, 0, len(entry.Attributes))
	index := make(map[string]ldapx.Attribute, len(entry.Attributes))
	for _, a := range entry.Attributes {
		descriptions = append(descriptions, a.Description())
		index[a.Description()] = a
	}
	sort.Strings(descriptions)

	for _, d := range descriptions {
		digest.Write([]byte{0})
		digest.Write([]byte(d))
		values := index[d].Values
		sorted := make([][]byte, len(values))
		copy(sorted, values)
		sort.Slice(sorted, func(i, j int) bool { return string(sorted[i]) < string(sorted[j]) })
		for _, v := range sorted {
			digest.Write([]byte{1})
			digest.Write(v)
		}
	}
	return "sha256=" + hex.EncodeToString(digest.Sum(nil))
}

// fillDiff populates each operation's Before from the captured state, so the
// preview renders an attribute-level diff rather than a list of intentions.
func fillDiff(ops []Operation, before []ldapx.Entry) []Operation {
	byDN := make(map[string]ldapx.Entry, len(before))
	for _, e := range before {
		byDN[e.DN] = e
	}

	out := make([]Operation, 0, len(ops))
	for _, op := range ops {
		entry, ok := byDN[op.DN]
		if ok && op.Before == nil {
			if existing, found := entry.Attribute(op.Attribute.Description()); found {
				op.Before = existing.Values
			}
		}
		switch op.Type {
		case OpReplaceAttr, OpAddAttr:
			if op.After == nil {
				op.After = op.Attribute.Values
			}
		case OpDeleteAttr:
			op.After = nil
		case OpAddValue:
			op.After = append(append([][]byte{}, op.Before...), op.Attribute.Values...)
		case OpDeleteValue:
			op.After = withoutValues(op.Before, op.Attribute.Values)
		case OpDeleteEntry:
			op.After = nil
		}
		out = append(out, op)
	}
	return out
}

func withoutValues(from, remove [][]byte) [][]byte {
	if len(from) == 0 {
		return nil
	}
	drop := make(map[string]struct{}, len(remove))
	for _, v := range remove {
		drop[string(v)] = struct{}{}
	}
	out := make([][]byte, 0, len(from))
	for _, v := range from {
		if _, ok := drop[string(v)]; ok {
			continue
		}
		out = append(out, v)
	}
	return out
}

func newTokenString() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand does not fail in practice; if it did, a predictable
		// token would be worse than no commit at all.
		panic("changeset: cannot generate a preview token: " + err.Error())
	}
	return hex.EncodeToString(buf)
}
