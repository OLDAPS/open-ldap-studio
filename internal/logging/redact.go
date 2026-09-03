package logging

import (
	"bytes"
	"regexp"
	"strings"
	"sync"
)

// Redactor removes secret material from a record before it is written.
//
// Redaction is a write-time operation, never a display-time one: a record
// redacted on the way to the screen still leaves the secret in the file, which
// is exactly the failure SC-008 scans for (FR-093, contract X7).
//
// The DN and the result code are deliberately never redacted — losing them
// would defeat the log's purpose (error-model.md § Redaction).
type Redactor struct {
	mu        sync.RWMutex
	sensitive map[string]struct{}
}

// Placeholder is written in place of every redacted value. It is intentionally
// not a fixed-width mask: the length of a secret is itself information.
const Placeholder = "«redacted»"

// alwaysSensitive are redacted regardless of user configuration.
var alwaysSensitive = []string{
	"userpassword",
	"unicodepwd",
	"sambantpassword",
	"sambalmpassword",
	"krbprincipalkey",
	"pwdhistory",
	"authpassword",
}

// jsonKeys matches secret-bearing keys in a JSON record: "password": "...".
var jsonKeys = regexp.MustCompile(
	`(?i)"(password|passwd|pwd|secret|credential|credentials|bindpassword|saslcredentials|token|passphrase|userpassword)"\s*:\s*"(?:[^"\\]|\\.)*"`)

// ldifAttr matches the start of an LDIF attribute line, capturing the
// attribute description (type plus any options) ahead of ':' or '::'.
var ldifAttr = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*)((?:;[^:\s]+)*)(::?)\s?`)

// NewRedactor returns a Redactor covering the always-sensitive attributes plus
// any the user has marked sensitive.
func NewRedactor(userMarked ...string) *Redactor {
	r := &Redactor{sensitive: make(map[string]struct{}, len(alwaysSensitive)+len(userMarked))}
	for _, a := range alwaysSensitive {
		r.sensitive[a] = struct{}{}
	}
	r.Mark(userMarked...)
	return r
}

// Mark adds attribute types the user considers sensitive (FR-093).
func (r *Redactor) Mark(attrs ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range attrs {
		if a = strings.ToLower(strings.TrimSpace(a)); a != "" {
			r.sensitive[a] = struct{}{}
		}
	}
}

// IsSensitive reports whether an attribute description names a secret. Options
// are ignored, so userPassword;binary is as sensitive as userPassword.
func (r *Redactor) IsSensitive(attrDescription string) bool {
	base, _, _ := strings.Cut(attrDescription, ";")
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.sensitive[strings.ToLower(strings.TrimSpace(base))]
	return ok
}

// Redact returns a copy of record with every secret value replaced.
//
// It handles both shapes a record can take on the way to a sink: LDIF lines,
// including folded continuations, and JSON objects.
func (r *Redactor) Redact(record []byte) []byte {
	if len(record) == 0 {
		return record
	}

	var out bytes.Buffer
	out.Grow(len(record))

	lines := bytes.SplitAfter(record, []byte("\n"))
	dropContinuation := false

	for _, line := range lines {
		body := bytes.TrimRight(line, "\r\n")
		ending := line[len(body):]

		// An LDIF value continues on any line beginning with a single space.
		if dropContinuation && len(body) > 0 && body[0] == ' ' {
			continue
		}
		dropContinuation = false

		if m := ldifAttr.FindSubmatch(body); m != nil && r.IsSensitive(string(m[1])) {
			out.Write(m[1])
			out.Write(m[2])
			out.WriteString(": ")
			out.WriteString(Placeholder)
			out.Write(ending)
			dropContinuation = true
			continue
		}

		out.Write(jsonKeys.ReplaceAllFunc(body, func(match []byte) []byte {
			key := match[:bytes.IndexByte(match, ':')]
			return append(append(append([]byte{}, key...), []byte(`: "`)...),
				append([]byte(Placeholder), '"')...)
		}))
		out.Write(ending)
	}

	return out.Bytes()
}
