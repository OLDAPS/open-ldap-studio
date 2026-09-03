package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/credentials"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/logging"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// State is a connection's position in its lifecycle, as the status bar shows
// it. It is always visible, on every screen (FR-010).
type State string

const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateLost         State = "lost"
)

// ConnState is the status bar's payload.
type ConnState struct {
	ProfileID      string `json:"profileId"`
	State          State  `json:"state"`
	BoundDN        string `json:"boundDn"`
	ServerIdentity string `json:"serverIdentity"`
	// TLSVerified is false both for a plaintext connection and for one whose
	// verification the profile turned off. The UI distinguishes them with
	// Encrypted, and badges the second for as long as it is open (FR-006).
	TLSVerified bool `json:"tlsVerified"`
	Encrypted   bool `json:"encrypted"`
	ReadOnly    bool `json:"readOnly"`
	Production  bool `json:"production"`
	// WritesRequireConfirmation is set after a reconnect: the server may have
	// forgotten the session, so the next write is confirmed afresh (FR-015).
	WritesRequireConfirmation bool   `json:"writesRequireConfirmation"`
	Message                   string `json:"message,omitempty"`
}

// connections owns every open connection and is the pipeline's DirectorySource.
type connections struct {
	bridge   *Bridge
	provider secrets.Provider

	mu    sync.RWMutex
	open  map[string]*ldapx.Conn
	state map[string]ConnState
}

func newConnections(b *Bridge, provider secrets.Provider) *connections {
	return &connections{
		bridge:   b,
		provider: provider,
		open:     make(map[string]*ldapx.Conn),
		state:    make(map[string]ConnState),
	}
}

// ErrNotConnected is returned when an operation needs a connection that is not
// open. It is distinct from a failed operation: nothing was attempted.
var ErrNotConnected = errors.New("bridge: that connection is not open")

// Directory implements changeset.DirectorySource.
func (c *connections) Directory(profileID string) (changeset.Directory, error) {
	c.mu.RLock()
	conn, ok := c.open[profileID]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotConnected, profileID)
	}
	return changeset.Live(conn), nil
}

// conn returns the open connection for reads.
func (c *connections) conn(profileID string) (*ldapx.Conn, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, ok := c.open[profileID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotConnected, profileID)
	}
	return conn, nil
}

func (c *connections) setState(s ConnState) {
	c.mu.Lock()
	c.state[s.ProfileID] = s
	c.mu.Unlock()
	c.bridge.Emit("conn:state", s)
}

// stateOf returns the last known state, disconnected by default.
func (c *connections) stateOf(profileID string) ConnState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.state[profileID]; ok {
		return s
	}
	return ConnState{ProfileID: profileID, State: StateDisconnected}
}

func (c *connections) closeAll() {
	c.mu.Lock()
	open := c.open
	c.open = make(map[string]*ldapx.Conn)
	c.mu.Unlock()

	for _, conn := range open {
		_ = conn.Close()
	}
}

// dial opens and binds one connection. It runs inside a job, so every step is
// cancellable and nothing here blocks the UI.
func (c *connections) dial(ctx context.Context, p profiles.Profile, cred *credentials.Credential) (*ldapx.Conn, error) {
	c.setState(ConnState{ProfileID: p.ID, State: StateConnecting, ReadOnly: p.ReadOnly, Production: p.IsProduction()})

	conn, err := ldapx.Dial(ctx, ldapx.DialConfig{
		Profile:   p,
		Trust:     c.bridge.trustPolicy(p),
		KeepAlive: 30 * time.Second,
	})
	if err != nil {
		c.setState(ConnState{
			ProfileID: p.ID, State: StateDisconnected, Message: err.Error(),
			ReadOnly: p.ReadOnly, Production: p.IsProduction(),
		})
		return nil, err
	}

	// A credential that would cross an unencrypted transport is worth saying
	// out loud before it does (screen 3c's clear-text warning strip).
	if p.BindMethod != profiles.BindAnonymous && !conn.Encrypted() {
		c.bridge.Emit("capability:unavailable", map[string]any{
			"profileId":  p.ID,
			"capability": "encryptedTransport",
			"reason":     "this connection is not encrypted; the bind credential will cross the network in the clear",
		})
	}

	secret, err := c.secretFor(p, cred)
	if err != nil {
		_ = conn.Close()
		c.setState(ConnState{ProfileID: p.ID, State: StateDisconnected, Message: err.Error()})
		return nil, err
	}
	defer secret.Zero()

	bindReq := ldapx.BindRequest{Method: p.BindMethod, DN: p.BindDN, Secret: secret}
	if cred != nil {
		bindReq.DN = cred.BindDN
		bindReq.Realm = cred.Realm
	}

	result, err := ldapx.Bind(ctx, conn, bindReq)
	if err != nil {
		_ = conn.Close()
		// A rejected bind is reported with the server's own words. It is not
		// retried, and it never falls back to anonymous (deviation D7).
		c.setState(ConnState{ProfileID: p.ID, State: StateDisconnected, Message: result.String()})
		return nil, err
	}

	c.mu.Lock()
	c.open[p.ID] = conn
	c.mu.Unlock()

	conn.StartKeepAlive(30*time.Second, func(lostErr error) {
		c.setState(ConnState{
			ProfileID: p.ID, State: StateLost, Message: lostErr.Error(),
			ReadOnly: p.ReadOnly, Production: p.IsProduction(),
		})
		c.bridge.Emit("conn:lost", map[string]any{"profileId": p.ID, "reason": lostErr.Error()})
	})

	c.setState(ConnState{
		ProfileID:      p.ID,
		State:          StateConnected,
		BoundDN:        conn.BoundDN(),
		ServerIdentity: conn.ServerIdentity(),
		TLSVerified:    conn.TLSVerified(),
		Encrypted:      conn.Encrypted(),
		ReadOnly:       p.ReadOnly,
		Production:     p.IsProduction(),
	})

	_ = c.bridge.logs.Sink(logging.Application).Record("connected %s as %q", p.Name, conn.BoundDN())
	return conn, nil
}

// secretFor resolves the profile's credential.
//
// A missing or unreachable credential store is recoverable: the caller is
// asked for a session-only secret rather than being refused (FR-005).
func (c *connections) secretFor(p profiles.Profile, cred *credentials.Credential) (secrets.Secret, error) {
	if p.BindMethod == profiles.BindAnonymous || p.BindMethod == profiles.BindExternal {
		return secrets.Secret{}, nil
	}

	ref := p.CredentialID
	if cred != nil && cred.SecretRef != "" {
		ref = cred.SecretRef
	}
	if ref == "" {
		c.bridge.Emit("credential:required", map[string]any{
			"profileId": p.ID,
			"reason":    "this connection has no credential assigned",
		})
		return secrets.Secret{}, fmt.Errorf("bridge: connection %q has no credential assigned", p.Name)
	}

	secret, err := c.provider.Get(ref)
	if err != nil {
		c.bridge.Emit("credential:required", map[string]any{
			"profileId": p.ID,
			"ref":       ref,
			"reason":    err.Error(),
		})
		return secrets.Secret{}, err
	}
	return secret, nil
}
