package bridge

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/trust"
)

// trustPolicy binds the trust store to one profile's connection attempt.
type trustPolicy struct {
	bridge  *Bridge
	profile profiles.Profile
}

func (b *Bridge) trustPolicy(p profiles.Profile) ldapx.TrustPolicy {
	return trustPolicy{bridge: b, profile: p}
}

// VerifyChain decides what happens to a chain the system roots reject.
//
// The order here is the contract: the connection is refused first, and the
// challenge is raised second. A challenge that fired while the handshake
// continued would be a prompt the user could not affect (FR-007, contract E5).
func (t trustPolicy) VerifyChain(host string, port int, chain []*x509.Certificate, fingerprint string, verifyErr error) error {
	if t.bridge.trust.Trusted(host, port, fingerprint) {
		return nil
	}

	previouslyTrusted := false
	for _, d := range t.bridge.trust.List() {
		if d.Host == host && d.Port == port {
			// The host is known and the certificate is not the one accepted.
			previouslyTrusted = true
			break
		}
	}

	pemChain := make([]string, 0, len(chain))
	for _, cert := range chain {
		pemChain = append(pemChain, string(pem.EncodeToMemory(&pem.Block{
			Type: "CERTIFICATE", Bytes: cert.Raw,
		})))
	}

	t.bridge.Emit("trust:challenge", trust.Challenge{
		Host:              host,
		Port:              port,
		Fingerprint:       fingerprint,
		ChainPEM:          pemChain,
		FailureReason:     verifyErr.Error(),
		PreviouslyTrusted: previouslyTrusted,
	})

	return ldapx.TrustError("connect", verifyErr.Error(), verifyErr)
}

// ListTrustDecisions returns every accepted certificate.
func (b *Bridge) ListTrustDecisions() []trust.Decision {
	return b.trust.List()
}

// DecideTrust records the user's answer to a challenge.
//
// It is called only in response to a trust:challenge event, and only for the
// exact fingerprint that event carried. The answer does not retry the
// connection: the user starts a fresh attempt, which now finds the decision.
func (b *Bridge) DecideTrust(host string, port int, fingerprint string, scope trust.Scope, reason string) error {
	return b.trust.Decide(trust.Decision{
		Host: host, Port: port, Fingerprint: fingerprint, Scope: scope, Reason: reason,
	})
}

// RevokeTrustDecision removes a decision; the next connection re-challenges.
func (b *Bridge) RevokeTrustDecision(fingerprint string) error {
	return b.trust.Revoke(fingerprint)
}
