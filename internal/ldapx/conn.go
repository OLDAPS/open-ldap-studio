package ldapx

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// Capability names something a server may not support. Every unavailable one
// is announced with a reason — the absence of a silent failure is what SC-016
// tests (events.md, capability:unavailable).
type Capability string

const (
	CapPaging        Capability = "paging"
	CapSorting       Capability = "sorting"
	CapVLV           Capability = "vlv"
	CapSchema        Capability = "schema"
	CapExtendedOp    Capability = "extendedOp"
	CapStoredACI     Capability = "storedACI"
	CapSchemaModify  Capability = "schemaModify"
	CapConfigEntries Capability = "configEntries"
)

// TrustPolicy decides what to do with a certificate chain the system roots do
// not vouch for. It is deliberately expressed in plain types so ldapx does not
// depend on the trust store's own package.
//
// A nil error accepts the chain; any error refuses the connection. The
// implementation is responsible for raising trust:challenge — and only after
// the connection has been refused (FR-007, contract E5).
type TrustPolicy interface {
	VerifyChain(host string, port int, chain []*x509.Certificate, fingerprint string, verifyErr error) error
}

// DialConfig is everything needed to open one connection.
type DialConfig struct {
	Profile profiles.Profile
	Trust   TrustPolicy
	// KeepAlive of 0 disables the idle probe.
	KeepAlive time.Duration
}

// Conn is one connection to one directory.
//
// It owns the socket from dial to close: TLS negotiation, the SASL handshakes
// go-ldap does not implement, and the go-ldap client that speaks everything
// else all sit behind this type.
type Conn struct {
	mu       sync.RWMutex
	ldap     *ldap.Conn
	profile  profiles.Profile
	address  string
	boundDN  string
	tlsState *tls.ConnectionState
	// serverIdentity is what the status bar shows beside the bind DN: the
	// certificate subject where TLS is in use, otherwise host:port.
	serverIdentity string
	closed         bool
	stopKeepAlive  chan struct{}
	// rawConn is the socket before go-ldap takes it over. It is retained so a
	// SASL mechanism go-ldap does not implement can run on it first.
	rawConn net.Conn
}

// Address returns host:port as dialled.
func (c *Conn) Address() string { return c.address }

// BoundDN returns the DN the connection is authenticated as, empty for an
// anonymous bind.
func (c *Conn) BoundDN() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.boundDN
}

// ServerIdentity is the certificate subject, or host:port without TLS.
func (c *Conn) ServerIdentity() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverIdentity
}

// TLSVerified reports whether the transport is encrypted and its certificate
// verified against the system roots. A profile that opted out of verification
// reports false, and the UI badges it for as long as it is connected (FR-006).
func (c *Conn) TLSVerified() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsState != nil && c.profile.TLS.VerifyCertificate
}

// Encrypted reports whether the transport carries TLS at all. It is the
// question the clear-text warning strip asks before a credential crosses it.
func (c *Conn) Encrypted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsState != nil
}

// Profile returns the profile this connection was opened from.
func (c *Conn) Profile() profiles.Profile {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.profile
}

// LDAP exposes the underlying client to the rest of this package only. No
// package outside ldapx may reach it.
func (c *Conn) client() *ldap.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ldap
}

// Dial opens a transport to the profile's server and applies its encryption
// policy. It does not bind — that is Bind's job, and the split exists because
// a failed bind must not look like a failed connection.
func Dial(ctx context.Context, cfg DialConfig) (*Conn, error) {
	p := cfg.Profile
	port := p.Port
	if port == 0 {
		port = profiles.DefaultPort(p.Encryption)
	}
	address := net.JoinHostPort(p.Host, strconv.Itoa(port))

	timeout := time.Duration(p.Timeouts.ConnectMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	dialer := &net.Dialer{Timeout: timeout, KeepAlive: cfg.KeepAlive}
	raw, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, TransportError("connect", err)
	}

	conn := &Conn{profile: p, address: address, serverIdentity: address}

	switch p.Encryption {
	case profiles.EncryptionLDAPS:
		tlsConn, err := conn.wrapTLS(ctx, raw, cfg)
		if err != nil {
			_ = raw.Close()
			return nil, err
		}
		raw = tlsConn
	case profiles.EncryptionStartTLS:
		// StartTLS is negotiated here rather than through go-ldap so that the
		// socket is still ours when a SASL mechanism go-ldap does not
		// implement has to run over it (see cram_md5.go).
		if err := startTLSRequest(raw, timeout); err != nil {
			_ = raw.Close()
			return nil, err
		}
		tlsConn, err := conn.wrapTLS(ctx, raw, cfg)
		if err != nil {
			_ = raw.Close()
			// The profile asked for StartTLS and did not get it. It is never
			// silently downgraded to plaintext (FR-009).
			return nil, err
		}
		raw = tlsConn
	case profiles.EncryptionNone, "":
		// Plaintext. The UI states this on every screen the connection appears.
	default:
		_ = raw.Close()
		return nil, LocalValidationError("connect", fmt.Sprintf("unknown encryption mode %q", p.Encryption))
	}

	conn.rawConn = raw
	return conn, nil
}

// wrapTLS performs the handshake and applies the trust policy.
func (c *Conn) wrapTLS(ctx context.Context, raw net.Conn, cfg DialConfig) (net.Conn, error) {
	p := cfg.Profile

	// No ClientSessionCache is set, so sessions never resume and every handshake
	// runs VerifyPeerCertificate; that is what G123 asks about.
	tlsCfg := &tls.Config{
		ServerName: p.Host,
		MinVersion: tls.VersionTLS12,
		// Verification is performed inside VerifyPeerCertificate so that a
		// failure can be offered to the trust store — which pins an exact
		// certificate — rather than being an unconditional refusal or an
		// unconditional acceptance.
		InsecureSkipVerify: true, //nolint:gosec // verified in VerifyPeerCertificate below
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error { //nolint:gosec // G123: no ClientSessionCache, so no resumption (see above)
			chain := make([]*x509.Certificate, 0, len(rawCerts))
			for _, der := range rawCerts {
				cert, err := x509.ParseCertificate(der)
				if err != nil {
					return TrustError("connect", "the server sent a certificate that could not be parsed", err)
				}
				chain = append(chain, cert)
			}
			if len(chain) == 0 {
				return TrustError("connect", "the server sent no certificate", nil)
			}

			fingerprint := Fingerprint(chain[0])
			verifyErr := verifyChain(chain, p)

			if verifyErr == nil {
				return nil
			}
			if !p.TLS.VerifyCertificate {
				// An explicit, per-profile opt-out. The badge stays lit for as
				// long as this connection is open.
				return nil
			}
			if cfg.Trust == nil {
				return TrustError("connect", verifyErr.Error(), verifyErr)
			}
			return cfg.Trust.VerifyChain(p.Host, p.Port, chain, fingerprint, verifyErr)
		},
	}

	tlsConn := tls.Client(raw, tlsCfg)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		var trustErr *Error
		if errors.As(err, &trustErr) {
			return nil, trustErr
		}
		return nil, TransportError("tls handshake", err)
	}

	state := tlsConn.ConnectionState()
	c.mu.Lock()
	c.tlsState = &state
	if len(state.PeerCertificates) > 0 {
		c.serverIdentity = state.PeerCertificates[0].Subject.String()
	}
	c.mu.Unlock()

	return tlsConn, nil
}

// verifyChain performs the standard verification the profile asked for.
func verifyChain(chain []*x509.Certificate, p profiles.Profile) error {
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	intermediates := x509.NewCertPool()
	for _, cert := range chain[1:] {
		intermediates.AddCert(cert)
	}

	opts := x509.VerifyOptions{Roots: roots, Intermediates: intermediates}
	if p.TLS.VerifyHostname {
		opts.DNSName = p.Host
	}
	_, err = chain[0].Verify(opts)
	return err
}

// Fingerprint is the SHA-256 of a certificate's DER, lowercase hex. It is the
// identity of a trust decision: a decision never applies to a different
// certificate (FR-008).
func Fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

// Close unbinds and closes the socket.
func (c *Conn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	client, raw, stop := c.ldap, c.rawConn, c.stopKeepAlive
	c.stopKeepAlive = nil
	c.mu.Unlock()

	if stop != nil {
		close(stop)
	}
	if client != nil {
		_ = client.Close()
		return nil
	}
	if raw != nil {
		return raw.Close()
	}
	return nil
}

// IsClosed reports whether the connection has been closed locally or by the
// server.
func (c *Conn) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return true
	}
	return c.ldap != nil && c.ldap.IsClosing()
}

// activate hands the socket to go-ldap and starts its message loop. It runs
// after any in-house SASL handshake, so go-ldap never sees a message it did
// not send.
func (c *Conn) activate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ldap != nil {
		return nil
	}
	if c.rawConn == nil {
		return TransportError("connect", errors.New("the connection has no socket"))
	}

	client := ldap.NewConn(c.rawConn, c.tlsState != nil)
	client.Start()

	readTimeout := time.Duration(c.profile.Timeouts.ReadMS) * time.Millisecond
	if readTimeout > 0 {
		client.SetTimeout(readTimeout)
	}
	c.ldap = client
	return nil
}

// StartKeepAlive probes the connection every interval so an idle drop is
// noticed by us rather than discovered mid-write. The probe is a Root DSE read
// of nothing: the cheapest request that proves the session is alive.
//
// onLost is called once, off the caller's goroutine, when a probe fails. FR-015
// hangs off it: a reconnected session requires confirmation before the next
// write, because the server may have forgotten the bind.
func (c *Conn) StartKeepAlive(interval time.Duration, onLost func(error)) {
	if interval <= 0 {
		return
	}
	stop := make(chan struct{})

	c.mu.Lock()
	if c.stopKeepAlive != nil {
		close(c.stopKeepAlive)
	}
	c.stopKeepAlive = stop
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if err := c.probe(); err != nil {
					if onLost != nil {
						onLost(err)
					}
					return
				}
			}
		}
	}()
}

// probe issues a base-scoped search for no attributes against the Root DSE.
func (c *Conn) probe() error {
	client := c.client()
	if client == nil || client.IsClosing() {
		return TransportError("keep-alive", errors.New("the connection is closed"))
	}
	req := ldap.NewSearchRequest("", ldap.ScopeBaseObject, ldap.NeverDerefAliases,
		1, 5, false, "(objectClass=*)", []string{"1.1"}, nil)
	if _, err := client.Search(req); err != nil {
		return translateError("keep-alive", err)
	}
	return nil
}

// startTLSRequest sends the StartTLS extended operation and reads its answer
// on the raw socket, leaving the socket ready for a TLS handshake.
//
// go-ldap has its own StartTLS, but using it would mean the LDAP client owns
// the socket from that point on, and the in-house SASL mechanisms need the
// socket for one more exchange first.
func startTLSRequest(raw net.Conn, timeout time.Duration) error {
	const startTLSOID = "1.3.6.1.4.1.1466.20037"

	if timeout > 0 {
		_ = raw.SetDeadline(time.Now().Add(timeout))
		defer func() { _ = raw.SetDeadline(time.Time{}) }()
	}

	if err := writeExtendedRequest(raw, 1, startTLSOID, nil); err != nil {
		return TransportError("startTLS", err)
	}
	result, _, err := readExtendedResponse(raw)
	if err != nil {
		return TransportError("startTLS", err)
	}
	if result.Code != Success {
		// The profile asked for an encrypted transport and the server refused.
		// Continuing in the clear is exactly the silent downgrade FR-009 bans.
		return ServerError("startTLS", result)
	}
	return nil
}

// translateError maps a go-ldap error onto our categorised model, keeping the
// server's own words where there are any.
func translateError(op string, err error) error {
	if err == nil {
		return nil
	}
	var ldapErr *ldap.Error
	if errors.As(err, &ldapErr) {
		if ldapErr.ResultCode == ldap.ErrorNetwork {
			return TransportError(op, err)
		}
		result := Result{
			Code:              int(ldapErr.ResultCode),
			MatchedDN:         ldapErr.MatchedDN,
			DiagnosticMessage: ldapErr.Err.Error(),
			Interpretation:    interpretations[int(ldapErr.ResultCode)],
		}
		return ServerError(op, result)
	}
	if errors.Is(err, context.Canceled) {
		return Cancelled(op)
	}
	return TransportError(op, err)
}

// resultFrom extracts the verbatim Result from a go-ldap error, or a success
// Result when there was none. Every server-touching call in this package ends
// here, which is how Result reaches the bridge on success as well as failure
// (contracts C7, X1).
func resultFrom(err error) Result {
	if err == nil {
		return NewResult(Success, "", "")
	}
	if r := ResultOf(err); r != nil {
		return *r
	}
	var ldapErr *ldap.Error
	if errors.As(err, &ldapErr) {
		return NewResult(int(ldapErr.ResultCode), ldapErr.MatchedDN, ldapErr.Err.Error())
	}
	return NewResult(Other, "", err.Error())
}
