//go:build integration

package ldapx

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// The fault proxy delays or drops bytes from a real directory. It never
// generates LDAP responses, so timeout and reconnect tests exercise slapd.
type connectionProxy struct {
	address string
	mu      sync.Mutex
	sockets []net.Conn
}

func proxyDirectory(t *testing.T, address string, delay time.Duration) *connectionProxy {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	proxy := &connectionProxy{address: listener.Addr().String()}
	t.Cleanup(func() {
		cancel()
		_ = listener.Close()
		proxy.drop()
	})
	go func() {
		for {
			client, err := listener.Accept()
			if err != nil {
				return
			}
			server, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
			if err != nil {
				_ = client.Close()
				continue
			}
			proxy.mu.Lock()
			proxy.sockets = append(proxy.sockets, client, server)
			proxy.mu.Unlock()
			go func() {
				defer func() { _ = client.Close() }()
				defer func() { _ = server.Close() }()
				timer := time.NewTimer(delay)
				defer timer.Stop()
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
				}
				go func() { _, _ = io.Copy(server, client) }()
				_, _ = io.Copy(client, server)
			}()
		}
	}()
	return proxy
}

func (p *connectionProxy) drop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, socket := range p.sockets {
		_ = socket.Close()
	}
	p.sockets = nil
}

func directoryForConnections(t *testing.T, encrypted bool) (string, int, int) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	certificate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "LDAP connection tests"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, public, private)
	if err != nil {
		t.Fatal(err)
	}
	key, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})
	req := testcontainers.ContainerRequest{
		Image: "bitnamilegacy/openldap:2.6.10", ExposedPorts: []string{"1389/tcp", "1636/tcp"},
		Env: map[string]string{
			"LDAP_ROOT": "dc=example,dc=org", "LDAP_ADMIN_PASSWORD": "adminpassword",
			"LDAP_ENABLE_TLS": "yes", "LDAP_TLS_CERT_FILE": "/tmp/ldap-test.crt",
			"LDAP_TLS_KEY_FILE": "/tmp/ldap-test.key", "LDAP_TLS_CA_FILE": "/tmp/ldap-test.crt",
			"LDAP_PORT_NUMBER": "1389", "LDAP_LDAPS_PORT_NUMBER": "1636",
		},
		Files: []testcontainers.ContainerFile{
			{Reader: bytes.NewReader(certPEM), ContainerFilePath: "/tmp/ldap-test.crt", FileMode: 0o644},
			{Reader: bytes.NewReader(keyPEM), ContainerFilePath: "/tmp/ldap-test.key", FileMode: 0o644},
		},
	}
	probeURL := "ldaps://127.0.0.1:1636"
	if !encrypted {
		req.Env["LDAP_ENABLE_TLS"] = "no"
		probeURL = "ldap://127.0.0.1:1389"
	}
	// Bitnami starts a temporary slapd during initialization and then restarts
	// it. Require a stable authenticated endpoint, including TLS when enabled.
	probe := fmt.Sprintf(`for i in 1 2 3; do
  LDAPTLS_REQCERT=never ldapwhoami -x -H %s -D cn=admin,dc=example,dc=org -w adminpassword || exit 1
  sleep 1
done`, probeURL)
	req.WaitingFor = wait.ForExec([]string{"sh", "-c", probe}).WithStartupTimeout(time.Minute)
	directory, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := directory.Terminate(context.Background()); err != nil {
			t.Errorf("terminate directory: %v", err)
		}
	})
	host, err := directory.Host(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	plain, err := directory.MappedPort(t.Context(), "1389/tcp")
	if err != nil {
		t.Fatal(err)
	}
	secure, err := directory.MappedPort(t.Context(), "1636/tcp")
	if err != nil {
		t.Fatal(err)
	}
	return host, int(plain.Num()), int(secure.Num())
}

func profileAt(t *testing.T, address string, mode profiles.Encryption) profiles.Profile {
	t.Helper()
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return profiles.Profile{
		Host: host, Port: number, Encryption: mode,
		Timeouts: profiles.Timeouts{ConnectMS: 1000, ReadMS: 1000},
	}
}

func TestLiveConnectionLifecycle(t *testing.T) {
	host, plain, secure := directoryForConnections(t, true)
	for _, endpoint := range []struct {
		mode profiles.Encryption
		port int
	}{
		{profiles.EncryptionLDAPS, secure},
		{profiles.EncryptionStartTLS, plain},
	} {
		t.Run(string(endpoint.mode)+"/connect timeout includes negotiation", func(t *testing.T) {
			proxy := proxyDirectory(t, net.JoinHostPort(host, strconv.Itoa(endpoint.port)), 300*time.Millisecond)
			profile := profileAt(t, proxy.address, endpoint.mode)
			profile.Timeouts.ConnectMS = 50
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			started := time.Now()
			conn, _, err := Dial(ctx, DialConfig{Profile: profile})
			if conn != nil {
				_ = conn.Close()
			}
			if err == nil {
				t.Error("a delayed TLS negotiation exceeded the connect timeout and succeeded")
			}
			if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
				t.Errorf("connect timeout took %s; negotiation must share the 50ms budget", elapsed)
			}
		})
		t.Run(string(endpoint.mode)+"/cancel negotiation", func(t *testing.T) {
			proxy := proxyDirectory(t, net.JoinHostPort(host, strconv.Itoa(endpoint.port)), 300*time.Millisecond)
			ctx, cancel := context.WithCancel(t.Context())
			time.AfterFunc(30*time.Millisecond, cancel)
			defer cancel()
			started := time.Now()
			conn, _, err := Dial(ctx, DialConfig{Profile: profileAt(t, proxy.address, endpoint.mode)})
			if conn != nil {
				_ = conn.Close()
			}
			if !errors.Is(err, ErrCancelled) || time.Since(started) > 200*time.Millisecond {
				t.Fatalf("cancel negotiation took %s: %v", time.Since(started), err)
			}
		})
		t.Run(string(endpoint.mode)+"/certificate refusal", func(t *testing.T) {
			profile := profileAt(t, net.JoinHostPort(host, strconv.Itoa(endpoint.port)), endpoint.mode)
			profile.TLS = profiles.TLSPolicy{VerifyCertificate: true, VerifyHostname: true}
			conn, _, err := Dial(t.Context(), DialConfig{Profile: profile})
			if conn != nil {
				_ = conn.Close()
				t.Fatal("an untrusted certificate established a connection")
			}
			if !errors.Is(err, ErrTrust) {
				t.Fatalf("certificate refusal = %v, want trust error", err)
			}
		})
	}
	for _, mode := range []profiles.Encryption{profiles.EncryptionNone, profiles.EncryptionStartTLS, profiles.EncryptionLDAPS} {
		port := plain
		if mode == profiles.EncryptionLDAPS {
			port = secure
		}
		t.Run(string(mode)+"/read reconnect", func(t *testing.T) {
			testReadReconnect(t, net.JoinHostPort(host, strconv.Itoa(port)), mode)
		})
	}
	address := net.JoinHostPort(host, strconv.Itoa(plain))
	t.Run("keep-alive recovers an idle connection", func(t *testing.T) { testIdleRecovery(t, address) })
	t.Run("keep-alive reports loss once", func(t *testing.T) { testIdleLoss(t, address) })
	t.Run("authentication refusal is terminal", func(t *testing.T) { testReconnectRefusal(t, address) })
	t.Run("writes are never reconnected or replayed", func(t *testing.T) { testNoWriteRetry(t, address) })
	t.Run("read and bind are cancellable", func(t *testing.T) { testOperationCancellation(t, address) })
	t.Run("closing cancels recovery and its waiters", func(t *testing.T) { testRecoveryCancellation(t, address) })
	t.Run("search cancellation interrupts a blocked socket write", func(t *testing.T) { testBlockedSearchCancellation(t, address) })
	t.Run("paging cookies cannot cross a recovered session", func(t *testing.T) { testPagingSession(t, address) })
}

func testBlockedSearchCancellation(t *testing.T, address string) {
	proxy := proxyDirectory(t, address, 5*time.Second)
	conn, _, err := Dial(t.Context(), DialConfig{Profile: profileAt(t, proxy.address, profiles.EncryptionNone)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	conn.profile.Timeouts.ReadMS = 30_000
	if err := conn.activate(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	time.AfterFunc(100*time.Millisecond, cancel)
	started := time.Now()
	_, _, err = Search(ctx, conn, SearchRequest{Filter: "(description=" + strings.Repeat("x", 8<<20) + ")"})
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("cancel blocked search: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("cancelling a socket write took %s", elapsed)
	}
	ctx, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer stop()
	done := make(chan error, 1)
	go func() {
		_, _, err := ReadRootDSE(ctx, conn)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("read succeeded on a stalled transport")
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("a subsequent search ignored its deadline after cancellation")
	}
}

func testPagingSession(t *testing.T, address string) {
	proxy := proxyDirectory(t, address, 0)
	conn := connectedForTest(t, proxy.address, func(ctx context.Context, candidate *Conn) (Result, error) {
		return bindConnectionTest(ctx, candidate, "adminpassword")
	})
	req := SearchRequest{BaseDN: "dc=example,dc=org", Scope: ScopeSubtree, Filter: "(objectClass=*)", Page: PageRequest{Size: 1}}
	first, _, err := Search(t.Context(), conn, req)
	if err != nil || len(first.Cookie) == 0 {
		t.Fatalf("first page: %+v, %v", first, err)
	}
	proxy.drop()
	if _, _, err := ReadRootDSE(t.Context(), conn); err != nil {
		t.Fatal(err)
	}
	req.Page.Cookie = first.Cookie
	req.Page.SessionID = first.SessionID
	if _, _, err := Search(t.Context(), conn, req); !errors.Is(err, ErrLocalValidation) {
		t.Fatalf("old paging cookie reached the replacement session: %v", err)
	}
}

func bindConnectionTest(ctx context.Context, conn *Conn, password string) (Result, error) {
	secret := secrets.NewSecret([]byte(password))
	defer secret.Zero()
	return Bind(ctx, conn, BindRequest{Method: profiles.BindSimple, DN: "cn=admin,dc=example,dc=org", Secret: secret})
}

func connectedForTest(t *testing.T, address string, rebind func(context.Context, *Conn) (Result, error)) *Conn {
	t.Helper()
	conn, _, err := Dial(t.Context(), DialConfig{Profile: profileAt(t, address, profiles.EncryptionNone), Rebind: rebind})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := bindConnectionTest(t.Context(), conn, "adminpassword"); err != nil {
		t.Fatal(err)
	}
	return conn
}

func testIdleRecovery(t *testing.T, address string) {
	proxy := proxyDirectory(t, address, 0)
	rebound := make(chan struct{}, 1)
	conn := connectedForTest(t, proxy.address, func(ctx context.Context, replacement *Conn) (Result, error) {
		result, err := bindConnectionTest(ctx, replacement, "adminpassword")
		rebound <- struct{}{}
		return result, err
	})
	before := conn.SessionID()
	lost := make(chan error, 1)
	conn.StartKeepAlive(t.Context(), 10*time.Millisecond, func(_ Result, err error) { lost <- err })
	proxy.drop()
	select {
	case err := <-lost:
		t.Fatalf("idle recovery failed: %v", err)
	case <-rebound:
	case <-time.After(2 * time.Second):
		t.Fatal("keep-alive did not attempt recovery")
	}
	// Read waits for recovery to finish if it is still authenticating.
	if _, _, err := ReadRootDSE(t.Context(), conn); err != nil {
		t.Fatal(err)
	}
	if conn.SessionID() == before {
		t.Fatal("idle recovery reused the old session identity")
	}
}

func testIdleLoss(t *testing.T, address string) {
	proxy := proxyDirectory(t, address, 0)
	conn := connectedForTest(t, proxy.address, nil)
	lost := make(chan error, 2)
	conn.StartKeepAlive(t.Context(), 5*time.Millisecond, func(_ Result, err error) { lost <- err })
	proxy.drop()
	select {
	case err := <-lost:
		if !errors.Is(err, ErrTransport) {
			t.Fatalf("idle loss = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("idle loss was not reported")
	}
	select {
	case err := <-lost:
		t.Fatalf("idle loss reported twice: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
}

func testReconnectRefusal(t *testing.T, address string) {
	proxy := proxyDirectory(t, address, 0)
	var attempts atomic.Int32
	conn := connectedForTest(t, proxy.address, func(ctx context.Context, replacement *Conn) (Result, error) {
		attempts.Add(1)
		return bindConnectionTest(ctx, replacement, "wrong-password")
	})
	proxy.drop()
	_, result, err := ReadRootDSE(t.Context(), conn)
	if !errors.Is(err, ErrAuth) || result.Code != InvalidCredentials {
		t.Fatalf("rebind refusal = %+v, %v", result, err)
	}
	_, _, _ = ReadRootDSE(t.Context(), conn)
	if attempts.Load() != 1 || !conn.IsClosed() {
		t.Fatalf("authentication refusal retried %d times, closed = %t", attempts.Load(), conn.IsClosed())
	}
}

func testNoWriteRetry(t *testing.T, address string) {
	proxy := proxyDirectory(t, address, 0)
	var attempts atomic.Int32
	conn := connectedForTest(t, proxy.address, func(ctx context.Context, replacement *Conn) (Result, error) {
		attempts.Add(1)
		return bindConnectionTest(ctx, replacement, "adminpassword")
	})
	proxy.drop()
	_, err := ModifyEntry(t.Context(), conn, "dc=example,dc=org", []Change{
		{Operation: ChangeReplace, Attribute: Attribute{Type: "description", Values: [][]byte{[]byte("never replay")}}},
	})
	if !errors.Is(err, ErrTransport) && !errors.Is(err, ErrIndeterminate) {
		t.Fatalf("lost write = %v", err)
	}
	if attempts.Load() != 0 {
		t.Fatalf("a write reconnected %d times", attempts.Load())
	}
}

func testOperationCancellation(t *testing.T, address string) {
	for _, operation := range []string{"bind", "read"} {
		t.Run(operation, func(t *testing.T) {
			proxy := proxyDirectory(t, address, 300*time.Millisecond)
			conn, _, err := Dial(t.Context(), DialConfig{Profile: profileAt(t, proxy.address, profiles.EncryptionNone)})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = conn.Close() }()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			time.AfterFunc(30*time.Millisecond, cancel)
			started := time.Now()
			if operation == "bind" {
				_, err = bindConnectionTest(ctx, conn, "adminpassword")
			} else {
				if err := conn.activate(); err != nil {
					t.Fatal(err)
				}
				_, _, err = ReadRootDSE(ctx, conn)
			}
			if !errors.Is(err, ErrCancelled) || time.Since(started) > 200*time.Millisecond {
				t.Fatalf("%s cancellation took %s: %v", operation, time.Since(started), err)
			}
		})
	}
}

func TestLiveStartTLSRefusalPreservesResult(t *testing.T) {
	host, port, _ := directoryForConnections(t, false)
	profile := profileAt(t, net.JoinHostPort(host, strconv.Itoa(port)), profiles.EncryptionStartTLS)
	conn, result, err := Dial(t.Context(), DialConfig{Profile: profile})
	if conn != nil {
		_ = conn.Close()
		t.Fatal("StartTLS refusal fell back to a connected plaintext transport")
	}
	server := ResultOf(err)
	if server == nil || result.Code == Success || result.Code != server.Code || result.DiagnosticMessage != server.DiagnosticMessage || result.MatchedDN != server.MatchedDN {
		t.Fatalf("StartTLS refusal lost its result: %+v, %v", result, err)
	}
}

func testRecoveryCancellation(t *testing.T, address string) {
	proxy := proxyDirectory(t, address, 0)
	rebinding := make(chan struct{})
	conn := connectedForTest(t, proxy.address, func(ctx context.Context, _ *Conn) (Result, error) {
		close(rebinding)
		<-ctx.Done()
		return Result{}, contextError(ctx, "rebind", ctx.Err())
	})
	proxy.drop()
	done := make(chan error, 1)
	go func() {
		_, _, err := ReadRootDSE(t.Context(), conn)
		done <- err
	}()
	select {
	case <-rebinding:
	case <-time.After(time.Second):
		t.Fatal("recovery never started")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if _, _, err := ReadRootDSE(ctx, conn); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancel recovery waiter: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrCancelled) {
			t.Fatalf("cancel recovery: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("closing the connection did not cancel recovery")
	}
	if !conn.IsClosed() {
		t.Fatal("recovery resurrected a locally closed connection")
	}
}

func testReadReconnect(t *testing.T, address string, mode profiles.Encryption) {
	proxy := proxyDirectory(t, address, 0)
	profile := profileAt(t, proxy.address, mode)
	var rebound atomic.Int32
	conn, _, err := Dial(t.Context(), DialConfig{
		Profile: profile,
		Rebind: func(ctx context.Context, candidate *Conn) (Result, error) {
			rebound.Add(1)
			return bindConnectionTest(ctx, candidate, "adminpassword")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := bindConnectionTest(t.Context(), conn, "adminpassword"); err != nil {
		t.Fatal(err)
	}
	before := conn.SessionID()
	proxy.drop()
	// A continuation cookie belongs to the old session. Recovery may
	// restart a read, but must never replay that cookie on another one.
	if _, _, err := Search(t.Context(), conn, SearchRequest{
		BaseDN: "dc=example,dc=org", Filter: "(objectClass=*)", Page: PageRequest{Size: 1, Cookie: []byte("old-session")},
	}); !errors.Is(err, ErrLocalValidation) || rebound.Load() != 0 {
		t.Fatalf("paged continuation reconnected: attempts = %d, error = %v", rebound.Load(), err)
	}
	var readers sync.WaitGroup
	for range 5 {
		readers.Go(func() {
			_, result, err := ReadEntry(t.Context(), conn, "dc=example,dc=org", ReadOptions{})
			if err != nil || result.Code != Success {
				t.Errorf("read after idle drop = %+v, %v", result, err)
			}
		})
	}
	readers.Wait()
	if rebound.Load() != 1 || conn.SessionID() == before || conn.BoundDN() != "cn=admin,dc=example,dc=org" {
		t.Fatalf("reconnects = %d, session = %s (before %s), identity = %q", rebound.Load(), conn.SessionID(), before, conn.BoundDN())
	}
	if _, err := ModifyEntry(t.Context(), conn, "dc=example,dc=org", nil); !errors.Is(err, ErrLocalValidation) {
		t.Fatalf("unconfirmed write after reconnect = %v", err)
	}
	if _, err := ModifyEntry(WithWriteSession(t.Context(), before), conn, "dc=example,dc=org", nil); !errors.Is(err, ErrLocalValidation) {
		t.Fatalf("write confirmed before reconnect = %v", err)
	}
	// A real write with a fresh confirmation proves the new session is
	// bound with the same identity and that the write gate can reopen.
	result, err := ModifyEntry(WithWriteSession(t.Context(), conn.SessionID()), conn, "dc=example,dc=org", []Change{
		{Operation: ChangeReplace, Attribute: Attribute{Type: "description", Values: [][]byte{[]byte("confirmed reconnect")}}},
	})
	if err != nil || result.Code != Success {
		t.Fatalf("confirmed write = %+v, %v", result, err)
	}
	proxy.drop()
	identity, result, err := WhoAmI(t.Context(), conn)
	if err != nil || result.Code != Success || identity != "dn:cn=admin,dc=example,dc=org" {
		t.Fatalf("Who Am I after disconnect = %q, %+v, %v", identity, result, err)
	}
	proxy.drop()
	matched, result, err := Compare(t.Context(), conn, "dc=example,dc=org", "description", []byte("not the value"))
	if err != nil || matched || result.Code != 5 {
		t.Fatalf("compareFalse after disconnect = %t, %+v, %v", matched, result, err)
	}
	matched, result, err = Compare(t.Context(), conn, "dc=example,dc=org", "description", []byte("confirmed reconnect"))
	if err != nil || !matched || result.Code != 6 {
		t.Fatalf("compareTrue = %t, %+v, %v", matched, result, err)
	}
}
