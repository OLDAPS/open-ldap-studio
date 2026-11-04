package trust

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

type (
	Emitter interface{ Emit(string, any) }
	policy  struct {
		store  *Store
		events Emitter
	}
)

func (s *Store) Policy(events Emitter) ldapx.TrustPolicy { return policy{store: s, events: events} }

// VerifyChain decides what happens to a chain the system roots reject.
//
// The order here is the contract: the connection is refused first, and the
// challenge is raised second. A challenge that fired while the handshake
// continued would be a prompt the user could not affect (FR-007, contract E5).
func (t policy) VerifyChain(host string, port int, chain []*x509.Certificate, fingerprint string, verifyErr error) error {
	if t.store.Trusted(host, port, fingerprint) {
		return nil
	}

	previouslyTrusted := false
	for _, d := range t.store.List() {
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

	t.events.Emit("trust:challenge", Challenge{
		Host:              host,
		Port:              port,
		Fingerprint:       fingerprint,
		ChainPEM:          pemChain,
		FailureReason:     verifyErr.Error(),
		PreviouslyTrusted: previouslyTrusted,
	})

	return ldapx.TrustError("connect", verifyErr.Error(), verifyErr)
}
