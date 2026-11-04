package bridge

import (
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// TestResult is what the wizard's "Check network parameter" and "Check
// authentication" report back.
//
// It carries what the server said about itself rather than a verdict: naming
// contexts are the answer to "what base DN should I use", and the vendor
// string is the answer to "am I even talking to the directory I think I am".
type TestResult struct {
	// Reachable means the socket opened and, where the profile asks for TLS,
	// that TLS was negotiated. It says nothing about the bind.
	Reachable bool `json:"reachable"`
	Encrypted bool `json:"encrypted"`
	// TLSVerified is false for a plaintext connection and for one whose
	// certificate the profile chose not to verify. The pair is what lets the
	// UI distinguish "no TLS" from "TLS the user opted out of checking".
	TLSVerified bool `json:"tlsVerified"`

	// Bound is true only when a bind was attempted and accepted.
	Bound   bool   `json:"bound"`
	BoundDN string `json:"boundDn,omitempty"`

	VendorName     string   `json:"vendorName,omitempty"`
	VendorVersion  string   `json:"vendorVersion,omitempty"`
	NamingContexts []string `json:"namingContexts,omitempty"`
	SASLMechanisms []string `json:"saslMechanisms,omitempty"`

	// Result is the server's own answer to the bind, verbatim. A failed test
	// is reported through this rather than through an error string, so the
	// LDAP result code survives the trip to the UI (FR-013).
	Result ldapx.Result `json:"result"`
	// Message explains a failure that happened before the server could answer
	// — DNS, connection refused, a TLS handshake that never completed.
	Message string `json:"message,omitempty"`
}

func (b *Bridge) TestConnection(payload, secret string) (TestResult, error) {
	profile, err := profiles.DecodeStrict([]byte(payload))
	if err != nil {
		return TestResult{}, err
	}
	result, err := b.app.TestConnection(b.context(), profile, secret)
	return TestResult(result), err
}
