package bridge

import "github.com/open-ldap-studio/open-ldap-studio/internal/trust"

// ListTrustDecisions returns every accepted certificate.
func (b *Bridge) ListTrustDecisions() []trust.Decision {
	return b.app.ListTrustDecisions()
}

// DecideTrust records the user's answer to a challenge.
//
// It is called only in response to a trust:challenge event, and only for the
// exact fingerprint that event carried. The answer does not retry the
// connection: the user starts a fresh attempt, which now finds the decision.
func (b *Bridge) DecideTrust(host string, port int, fingerprint string, scope trust.Scope, reason string) error {
	return b.app.DecideTrust(trust.Decision{
		Host: host, Port: port, Fingerprint: fingerprint, Scope: scope, Reason: reason,
	})
}

// RevokeTrustDecision removes a decision; the next connection re-challenges.
func (b *Bridge) RevokeTrustDecision(fingerprint string) error {
	return b.app.RevokeTrustDecision(fingerprint)
}
