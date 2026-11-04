package connections

import (
	"context"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// Tester owns ephemeral protocol checks separately from registered sessions.
type Tester struct{ TrustPolicy TrustPolicyFactory }

// ProbeResult is what the wizard's "Check network parameter" and "Check
// authentication" report back.
//
// It carries what the server said about itself rather than a verdict: naming
// contexts are the answer to "what base DN should I use", and the vendor
// string is the answer to "am I even talking to the directory I think I am".
type ProbeResult struct {
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

func (t Tester) Test(ctx context.Context, p profiles.Profile, secret string, wantBind bool) (ProbeResult, error) {
	if p.Timeouts.ConnectMS == 0 {
		// A test that hangs for the default thirty seconds is a test nobody
		// runs twice.
		p.Timeouts.ConnectMS = 10_000
	}

	conn, dialResult, err := ldapx.Dial(ctx, ldapx.DialConfig{Profile: p, Trust: t.TrustPolicy(p)})
	if err != nil {
		return ProbeResult{Result: dialResult, Message: err.Error()}, nil
	}
	defer func() { _ = conn.Close() }()

	out := ProbeResult{
		Reachable:   true,
		Encrypted:   conn.Encrypted(),
		TLSVerified: conn.TLSVerified(),
	}

	if wantBind {
		material := secrets.NewSecret([]byte(secret))
		defer material.Zero()

		result, bindErr := ldapx.Bind(ctx, conn, ldapx.BindRequest{
			Method: p.BindMethod,
			DN:     p.BindDN,
			Secret: material,
		})
		out.Result = result
		if bindErr != nil {
			// Not an error return: the server answered, and its answer is the
			// useful part. Returning it as a Go error would flatten a result
			// code into a string on the way to the UI.
			out.Message = bindErr.Error()
			return out, nil
		}
		out.Bound = true
		out.BoundDN = conn.BoundDN()
	} else {
		// Dial leaves the raw transport available for SASL. Activate the LDAP
		// client through an explicit anonymous bind before reading the Root
		// DSE; a network-only check otherwise searches an unopened client.
		result, bindErr := ldapx.Bind(ctx, conn, ldapx.BindRequest{Method: profiles.BindAnonymous})
		out.Result = result
		if bindErr != nil {
			out.Message = bindErr.Error()
			return out, nil
		}
	}

	// The Root DSE is readable anonymously on most servers, so it is worth
	// asking for even when no bind was attempted — it is where the base DN
	// the next wizard page needs comes from.
	dse, result, dseErr := ldapx.ReadRootDSE(ctx, conn)
	if dseErr == nil {
		out.VendorName = dse.VendorName
		out.VendorVersion = dse.VendorVersion
		out.NamingContexts = dse.NamingContexts
		out.SASLMechanisms = dse.SupportedSASLMechs
	} else if !wantBind {
		// Only surface this when there is nothing better to report; after a
		// successful bind, a missing Root DSE is a footnote, not the headline.
		out.Result = result
		out.Message = dseErr.Error()
	}

	return out, nil
}
