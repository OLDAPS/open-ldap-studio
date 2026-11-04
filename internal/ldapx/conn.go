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
	"github.com/google/uuid"

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
	// KeepAlive is the TCP keep-alive interval; <= 0 disables OS probes.
	// StartKeepAlive controls the separate LDAP idle probe.
	KeepAlive time.Duration
	// Rebind re-authenticates a replacement transport with fresh credentials.
	// A nil callback disables reconnect. Credentials are never retained here.
	Rebind func(context.Context, *Conn) (Result, error)
	// OnReconnect reports recovery's outcome after all session locks are released.
	OnReconnect func(*Conn, Result, error)
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
	stopKeepAlive  context.CancelFunc
	config         DialConfig
	lifetime       context.Context
	cancel         context.CancelFunc
	reconnecting   chan struct{}
	reconnectError error
	sessionID      string
	reconnected    bool
	writes         int
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
func Dial(ctx context.Context, cfg DialConfig) (*Conn, Result, error) {
	p := cfg.Profile
	if p.Port == 0 {
		p.Port = profiles.DefaultPort(p.Encryption)
	}
	if p.Host == "" || p.Port < 1 || p.Port > 65535 {
		return nil, Result{}, LocalValidationError("connect", "a host and a port between 1 and 65535 are required")
	}
	switch p.Encryption {
	case profiles.EncryptionNone, "", profiles.EncryptionLDAPS, profiles.EncryptionStartTLS:
	default:
		return nil, Result{}, LocalValidationError("connect", fmt.Sprintf("unknown encryption mode %q", p.Encryption))
	}
	address := net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
	cfg.Profile = p

	timeout := time.Duration(p.Timeouts.ConnectMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// The budget covers DNS, TCP, StartTLS and the TLS handshake together.
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	keepAlive := cfg.KeepAlive
	if keepAlive <= 0 {
		keepAlive = -1
	}
	dialer := &net.Dialer{KeepAlive: keepAlive}
	raw, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, Result{}, contextError(ctx, "connect", err)
	}
	stop := closeOnCancel(ctx, raw)
	defer stop()
	deadline, _ := ctx.Deadline()
	if err := raw.SetDeadline(deadline); err != nil {
		_ = raw.Close()
		return nil, Result{}, TransportError("connect", err)
	}

	conn := &Conn{profile: p, config: cfg, address: address, serverIdentity: address, sessionID: uuid.NewString()}
	result := NewResult(Success, "", "")

	switch p.Encryption {
	case profiles.EncryptionLDAPS:
		tlsConn, err := conn.wrapTLS(ctx, raw, cfg)
		if err != nil {
			_ = raw.Close()
			return nil, result, err
		}
		raw = tlsConn
	case profiles.EncryptionStartTLS:
		// StartTLS is negotiated here rather than through go-ldap so that the
		// socket is still ours when a SASL mechanism go-ldap does not
		// implement has to run over it (see cram_md5.go).
		result, err = startTLSRequest(ctx, raw)
		if err != nil {
			_ = raw.Close()
			return nil, result, err
		}
		tlsConn, err := conn.wrapTLS(ctx, raw, cfg)
		if err != nil {
			_ = raw.Close()
			// The profile asked for StartTLS and did not get it. It is never
			// silently downgraded to plaintext (FR-009).
			return nil, result, err
		}
		raw = tlsConn
	case profiles.EncryptionNone, "":
		// Plaintext. The UI states this on every screen the connection appears.
	}

	stop()
	if err := ctx.Err(); err != nil {
		_ = raw.Close()
		return nil, result, contextError(ctx, "connect", err)
	}
	if err := raw.SetDeadline(time.Time{}); err != nil {
		_ = raw.Close()
		return nil, result, TransportError("connect", err)
	}
	conn.lifetime, conn.cancel = context.WithCancel(context.Background())
	conn.rawConn = raw
	return conn, result, nil
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
			if err := cfg.Trust.VerifyChain(p.Host, p.Port, chain, fingerprint, verifyErr); err != nil {
				return TrustError("connect", err.Error(), err)
			}
			return nil
		},
	}

	tlsConn := tls.Client(raw, tlsCfg)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		var trustErr *Error
		if errors.As(err, &trustErr) {
			return nil, trustErr
		}
		return nil, contextError(ctx, "tls handshake", err)
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

// Close cancels background work and closes the socket. It is idempotent.
func (c *Conn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	client, raw, stop := c.ldap, c.rawConn, c.stopKeepAlive
	c.stopKeepAlive = nil
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()

	if stop != nil {
		stop()
	}
	// Interrupt a blocked socket write before asking go-ldap's message loop to
	// exit; that loop cannot process its quit message while Write is blocked.
	var closeErr error
	if raw != nil {
		_ = raw.SetDeadline(time.Now())
		closeErr = raw.Close()
	}
	if client != nil {
		_ = client.Close()
		return nil
	}
	return closeErr
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

	if c.closed {
		return TransportError("connect", net.ErrClosed)
	}
	if c.ldap != nil {
		return nil
	}
	if c.rawConn == nil {
		return TransportError("connect", errors.New("the connection has no socket"))
	}

	client := ldap.NewConn(c.rawConn, c.tlsState != nil)
	client.Start()

	readTimeout := time.Duration(c.profile.Timeouts.ReadMS) * time.Millisecond
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	client.SetTimeout(readTimeout)
	c.ldap = client
	return nil
}

// StartKeepAlive probes the connection every interval so an idle drop is
// noticed by us rather than discovered mid-write. The probe is a Root DSE read
// of nothing: the cheapest request that proves the session is alive.
//
// onLost is called once when recovery fails. Local cancellation, replacement
// of the probe loop and Close never report a server loss.
func (c *Conn) StartKeepAlive(ctx context.Context, interval time.Duration, onLost func(Result, error)) {
	if interval <= 0 {
		return
	}
	ctx, cancel := c.operationContextWithoutTimeout(ctx)

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		cancel()
		return
	}
	if c.stopKeepAlive != nil {
		c.stopKeepAlive()
	}
	c.stopKeepAlive = cancel
	c.mu.Unlock()

	go func() {
		defer cancel()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.RLock()
				writing := c.writes > 0
				c.mu.RUnlock()
				if writing {
					continue
				}
				result, err := c.probe(ctx)
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					if onLost != nil {
						onLost(result, err)
					}
					return
				}
			}
		}
	}()
}

// probe issues a base-scoped search for no attributes against the Root DSE.
func (c *Conn) probe(ctx context.Context) (Result, error) {
	req := ldap.NewSearchRequest("", ldap.ScopeBaseObject, ldap.NeverDerefAliases,
		1, 5, false, "(objectClass=*)", []string{"1.1"}, nil)
	_, _, err := c.search(ctx, req, true, "")
	return resultFrom(err), translateError("keep-alive", err)
}

// startTLSRequest sends the StartTLS extended operation and reads its answer
// on the raw socket, leaving the socket ready for a TLS handshake.
//
// go-ldap has its own StartTLS, but using it would mean the LDAP client owns
// the socket from that point on, and the in-house SASL mechanisms need the
// socket for one more exchange first.
func startTLSRequest(ctx context.Context, raw net.Conn) (Result, error) {
	const startTLSOID = "1.3.6.1.4.1.1466.20037"

	if err := writeExtendedRequest(raw, 1, startTLSOID, nil); err != nil {
		return Result{}, contextError(ctx, "startTLS", err)
	}
	result, _, err := readExtendedResponse(raw)
	if err != nil {
		return Result{}, contextError(ctx, "startTLS", err)
	}
	if result.Code != Success {
		// The profile asked for an encrypted transport and the server refused.
		// Continuing in the clear is exactly the silent downgrade FR-009 bans.
		return result, ServerError("startTLS", result)
	}
	return result, nil
}

// closeOnCancel interrupts raw exchanges without leaving a goroutine or a
// cancellation callback able to close a successfully handed-off socket.
func closeOnCancel(ctx context.Context, raw net.Conn) func() {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		_ = raw.SetDeadline(time.Now())
		_ = raw.Close()
		close(done)
	})
	var once sync.Once
	return func() {
		once.Do(func() {
			if !stop() {
				<-done
			}
		})
	}
}

func contextError(ctx context.Context, op string, err error) error {
	err = translateError(op, err)
	if ResultOf(err) != nil {
		// A response already received is authoritative even if the caller
		// cancelled at the same time. Preserve its diagnostic and category.
		return err
	}
	if ctx.Err() != nil {
		return translateError(op, ctx.Err())
	}
	return err
}

func (c *Conn) operationContextWithoutTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	stop := func() bool { return true }
	if c.lifetime != nil {
		stop = context.AfterFunc(c.lifetime, cancel)
	}
	return ctx, func() {
		stop()
		cancel()
	}
}

func (c *Conn) operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, stop := c.operationContextWithoutTimeout(ctx)
	timeout := time.Duration(c.profile.Timeouts.ReadMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	return ctx, func() {
		cancel()
		stop()
	}
}

// search retries a read once, only after a lost transport, and never carries a
// paging cookie onto a different session. Writes have no retry path.
func (c *Conn) search(ctx context.Context, req *ldap.SearchRequest, retry bool, expectedSession string) (*ldap.SearchResult, string, error) {
	ctx, cancel := c.operationContext(ctx)
	defer cancel()
	client, raw, sessionID := c.readSession()
	if !retry && (expectedSession == "" || expectedSession != sessionID) {
		return nil, sessionID, LocalValidationError("search", "the paging session changed; refresh the results before requesting another page")
	}
	result, err := searchContext(ctx, client, raw, req)
	if err == nil || !retry || client == nil || !client.IsClosing() || !errors.Is(translateError("search", err), ErrTransport) || ctx.Err() != nil {
		return result, sessionID, err
	}
	if _, err := c.reconnect(ctx, client); err != nil {
		return nil, sessionID, err
	}
	client, raw, sessionID = c.readSession()
	result, err = searchContext(ctx, client, raw, req)
	return result, sessionID, err
}

func (c *Conn) readSession() (*ldap.Conn, net.Conn, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ldap, c.rawConn, c.sessionID
}

func searchContext(ctx context.Context, client *ldap.Conn, raw net.Conn, req *ldap.SearchRequest) (*ldap.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client == nil || raw == nil || client.IsClosing() {
		return nil, TransportError("search", net.ErrClosed)
	}
	// SearchAsync checks context while waiting for replies, but acquiring a
	// message ID or writing a request can block its worker before that check.
	// Closing the transport interrupts both and lets later requests terminate.
	stop := closeOnCancel(ctx, raw)
	defer stop()
	type outcome struct {
		result *ldap.SearchResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := collectSearch(ctx, client, req)
		done <- outcome{result, err}
	}()
	select {
	case response := <-done:
		return response.result, contextError(ctx, "search", response.err)
	case <-ctx.Done():
		// Prefer a server response already received over simultaneous cancellation.
		select {
		case response := <-done:
			return response.result, contextError(ctx, "search", response.err)
		default:
			return nil, ctx.Err()
		}
	}
}

func collectSearch(ctx context.Context, client *ldap.Conn, req *ldap.SearchRequest) (*ldap.SearchResult, error) {
	result := &ldap.SearchResult{}
	response := client.SearchAsync(ctx, req, 0)
	for response.Next() {
		if entry := response.Entry(); entry != nil {
			result.Entries = append(result.Entries, entry)
		}
		if referral := response.Referral(); referral != "" {
			result.Referrals = append(result.Referrals, referral)
		}
		result.Controls = append(result.Controls, response.Controls()...)
	}
	if err := response.Err(); err != nil {
		return result, err
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if client.IsClosing() {
		return result, TransportError("search", net.ErrClosed)
	}
	return result, nil
}

// reconnect serializes recovery so concurrent readers authenticate a single
// replacement. Waiters and the dial/bind all observe cancellation and Close.
func (c *Conn) reconnect(ctx context.Context, failed *ldap.Conn) (Result, error) {
	for {
		c.mu.Lock()
		switch {
		case c.closed || c.config.Rebind == nil || c.writes > 0:
			c.mu.Unlock()
			return Result{}, TransportError("reconnect", net.ErrClosed)
		case c.reconnectError != nil:
			err := c.reconnectError
			c.mu.Unlock()
			return resultFrom(err), err
		case c.ldap != failed && c.ldap != nil && !c.ldap.IsClosing():
			c.mu.Unlock()
			return NewResult(Success, "", ""), nil
		case c.reconnecting != nil:
			done := c.reconnecting
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return Result{}, contextError(ctx, "reconnect", ctx.Err())
			case <-done:
				continue
			}
		}
		c.reconnecting = make(chan struct{})
		c.mu.Unlock()
		result, err := c.replaceSession(ctx)
		c.mu.Lock()
		if err != nil && !errors.Is(err, ErrTransport) && !errors.Is(err, ErrCancelled) {
			// Trust and authentication refusals are terminal for this session;
			// another read must not start a credential retry loop.
			c.reconnectError = err
		}
		close(c.reconnecting)
		c.reconnecting = nil
		c.mu.Unlock()
		notify := ctx.Err() == nil
		if err != nil && !errors.Is(err, ErrTransport) && !errors.Is(err, ErrCancelled) {
			_ = c.Close()
		}
		if c.config.OnReconnect != nil && notify {
			c.config.OnReconnect(c, result, err)
		}
		return result, err
	}
}

func (c *Conn) replaceSession(ctx context.Context) (Result, error) {
	replacement, result, err := Dial(ctx, c.config)
	if err != nil {
		return result, err
	}
	result, err = c.config.Rebind(ctx, replacement)
	if err != nil {
		_ = replacement.Close()
		return result, err
	}
	if result.Code != Success {
		_ = replacement.Close()
		return result, ServerError("rebind", result)
	}
	c.mu.Lock()
	if c.closed || ctx.Err() != nil || c.writes > 0 {
		c.mu.Unlock()
		_ = replacement.Close()
		return Result{}, contextError(ctx, "reconnect", net.ErrClosed)
	}
	oldClient, oldRaw := c.ldap, c.rawConn
	c.ldap, c.rawConn = replacement.ldap, replacement.rawConn
	c.tlsState, c.serverIdentity = replacement.tlsState, replacement.serverIdentity
	c.boundDN, c.sessionID = replacement.boundDN, replacement.sessionID
	c.reconnected = true
	c.mu.Unlock()
	replacement.cancel()
	if oldClient != nil {
		_ = oldClient.Close()
	} else if oldRaw != nil {
		_ = oldRaw.Close()
	}
	return result, nil
}

// SessionID identifies the authenticated transport. A preview captures it so
// reconnecting invalidates the previous confirmation even if data is unchanged.
func (c *Conn) SessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

type writeSessionKey struct{}

// WithWriteSession carries the session whose preview was explicitly confirmed.
// It is checked atomically with obtaining the client, immediately before I/O.
func WithWriteSession(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, writeSessionKey{}, sessionID)
}

func (c *Conn) writeClient(ctx context.Context, op string) (*ldap.Conn, func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, nil, contextError(ctx, op, err)
	}
	session, _ := ctx.Value(writeSessionKey{}).(string)
	if (c.reconnected || session != "") && session != c.sessionID {
		return nil, nil, LocalValidationError(op, "the connection was reconnected; preview and confirm this write again")
	}
	if c.closed || c.ldap == nil || c.ldap.IsClosing() || c.reconnecting != nil {
		return nil, nil, TransportError(op, net.ErrClosed)
	}
	c.writes++
	return c.ldap, func() {
		c.mu.Lock()
		c.writes--
		c.mu.Unlock()
	}, nil
}

func (c *Conn) write(ctx context.Context, op string, send func(*ldap.Conn) error) (Result, error) {
	ctx, cancel := c.operationContext(ctx)
	defer cancel()
	client, release, err := c.writeClient(ctx, op)
	if err != nil {
		return Result{}, err
	}
	defer release()
	c.mu.RLock()
	raw := c.rawConn
	c.mu.RUnlock()
	stop := closeOnCancel(ctx, raw)
	defer stop()
	err = send(client)
	stop()
	return finishWrite(ctx, op, err)
}

// readOperation covers the small synchronous read-only extended operations.
// Cancelling them closes their transport; no request goroutine outlives them.
func (c *Conn) readOperation(ctx context.Context, op string, send func(*ldap.Conn) error) error {
	ctx, cancel := c.operationContext(ctx)
	defer cancel()
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return contextError(ctx, op, err)
		}
		c.mu.RLock()
		client, raw := c.ldap, c.rawConn
		c.mu.RUnlock()
		if client == nil || raw == nil {
			return TransportError(op, net.ErrClosed)
		}
		var err error
		if client.IsClosing() {
			err = TransportError(op, net.ErrClosed)
		} else {
			stop := closeOnCancel(ctx, raw)
			err = send(client)
			stop()
		}
		if ctx.Err() != nil {
			return contextError(ctx, op, err)
		}
		if err == nil || attempt > 0 || !client.IsClosing() || !errors.Is(translateError(op, err), ErrTransport) {
			return translateError(op, err)
		}
		if _, err := c.reconnect(ctx, client); err != nil {
			return err
		}
	}
}

// translateError maps a go-ldap error onto our categorised model, keeping the
// server's own words where there are any.
func translateError(op string, err error) error {
	if err == nil {
		return nil
	}
	var categorized *Error
	if errors.As(err, &categorized) {
		return err
	}
	if errors.Is(err, context.Canceled) {
		return Cancelled(op)
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
