package trust

import "time"

// Scope decides whether a decision outlives the session.
type Scope string

const (
	// ScopeSession decisions are never persisted.
	ScopeSession Scope = "session"
	// ScopePermanent decisions are written to the trust store.
	ScopePermanent Scope = "permanent"
)

// Decision records that the user accepted one specific certificate.
//
// The fingerprint is the identity of the decision: a presented certificate
// whose fingerprint differs returns to untrusted and re-raises the challenge.
// It is never auto-accepted because "the host was trusted before" (FR-008).
type Decision struct {
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Fingerprint string    `json:"fingerprint"` // SHA-256, lowercase hex
	Scope       Scope     `json:"scope"`
	Chain       [][]byte  `json:"chain,omitempty"` // DER, retained so the user can review what they accepted
	AcceptedAt  time.Time `json:"acceptedAt"`
	// Reason is the validation failure that was overridden.
	Reason string `json:"reason"`
}

// Challenge is the payload of the trust:challenge event. The connection is
// already refused when it fires; the user's answer starts a fresh attempt
// (FR-007, contract E5).
type Challenge struct {
	Host              string   `json:"host"`
	Port              int      `json:"port"`
	Fingerprint       string   `json:"fingerprint"`
	ChainPEM          []string `json:"chainPem"`
	FailureReason     string   `json:"failureReason"`
	PreviouslyTrusted bool     `json:"previouslyTrusted"`
}
