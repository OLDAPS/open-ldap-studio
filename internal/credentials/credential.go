package credentials

import "time"

// Kind groups credentials in the management list (screen 3b).
type Kind string

const (
	KindSimple    Kind = "simple"
	KindDigestMD5 Kind = "digestMD5"
	KindCramMD5   Kind = "cramMD5"
	KindGSSAPI    Kind = "gssapi"
	KindExternal  Kind = "external" // client certificate; SecretRef names the key
)

// Credential is a named, assignable reference to a secret the platform holds.
//
// It is defined once and assigned to many profiles, so revoking it invalidates
// every connection that used it. That — not a vault — is what makes screen 3b
// worth having (deviation D2, research R10).
//
// Invariant: no field holds secret material. SecretRef is a lookup key handed
// to internal/secrets, which is the only package that can resolve it.
type Credential struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind Kind   `json:"kind"`
	// BindDN is raw text, never normalised (FR-020).
	BindDN string `json:"bindDn,omitempty"`
	// Realm applies to DIGEST-MD5 and GSSAPI.
	Realm string `json:"realm,omitempty"`
	// SecretRef is the handle into the platform credential service. It is not
	// a secret, and resolving it is internal/secrets' job alone.
	SecretRef string `json:"secretRef,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	// RotatedAt drives the password-age display and rotation reminders.
	RotatedAt time.Time `json:"rotatedAt,omitzero"`
	// RotateAfterDays of 0 disables the reminder.
	RotateAfterDays int `json:"rotateAfterDays,omitempty"`

	// StoreAvailable records whether the platform agent could be reached the
	// last time this credential was used. A missing store is recoverable and
	// explained, never silent (Constitution II, FR-005).
	StoreAvailable bool   `json:"storeAvailable"`
	StoreReason    string `json:"storeReason,omitempty"`
}

// Age reports how long since the secret was last rotated. A zero RotatedAt
// means "never rotated since it was created".
func (c Credential) Age(now time.Time) time.Duration {
	from := c.RotatedAt
	if from.IsZero() {
		from = c.CreatedAt
	}
	return now.Sub(from)
}

// RotationDue reports whether the rotation reminder should fire.
func (c Credential) RotationDue(now time.Time) bool {
	if c.RotateAfterDays <= 0 {
		return false
	}
	return c.Age(now) >= time.Duration(c.RotateAfterDays)*24*time.Hour
}

// Assignment binds a credential to a profile. Deleting a credential leaves its
// assignments pointing at nothing, and those profiles enter "credential
// missing" rather than falling back to anonymous.
type Assignment struct {
	CredentialID string `json:"credentialId"`
	ProfileID    string `json:"profileId"`
}
