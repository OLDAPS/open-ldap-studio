package connections

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

// Emitter publishes lifecycle events without coupling the manager to Wails.
type Emitter interface {
	Emit(name string, payload any)
}

// TrustPolicyFactory binds a connection profile to its certificate policy.
type TrustPolicyFactory func(profiles.Profile) ldapx.TrustPolicy

// Manager owns every open connection and is the pipeline's DirectorySource.
type Manager struct {
	emitter     Emitter
	provider    secrets.Provider
	trustPolicy TrustPolicyFactory
	logs        *logging.Logs

	mu    sync.RWMutex
	open  map[string]*ldapx.Conn
	state map[string]ConnState
}

// New creates an empty connection lifecycle manager.
func New(emitter Emitter, provider secrets.Provider, trustPolicy TrustPolicyFactory, logs *logging.Logs) *Manager {
	return &Manager{
		emitter:     emitter,
		provider:    provider,
		trustPolicy: trustPolicy,
		logs:        logs,
		open:        make(map[string]*ldapx.Conn),
		state:       make(map[string]ConnState),
	}
}

// ErrNotConnected is returned when an operation needs a connection that is not
// open. It is distinct from a failed operation: nothing was attempted.
var ErrNotConnected = errors.New("bridge: that connection is not open")

// Directory implements changeset.DirectorySource.
func (c *Manager) Directory(profileID string) (changeset.Directory, error) {
	c.mu.RLock()
	conn, ok := c.open[profileID]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotConnected, profileID)
	}
	return changeset.Live(conn), nil
}

// Conn returns the open connection for reads.
func (c *Manager) Conn(profileID string) (*ldapx.Conn, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, ok := c.open[profileID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotConnected, profileID)
	}
	return conn, nil
}

// Disconnect removes and closes one live connection.
func (c *Manager) Disconnect(profileID string) error {
	conn, err := c.Conn(profileID)
	if err != nil {
		return err
	}

	c.mu.Lock()
	delete(c.open, profileID)
	c.mu.Unlock()

	closeErr := conn.Close()
	c.setState(ConnState{ProfileID: profileID, State: StateDisconnected})
	return closeErr
}

func (c *Manager) setState(s ConnState) {
	c.mu.Lock()
	c.state[s.ProfileID] = s
	c.mu.Unlock()
	c.emitter.Emit("conn:state", s)
}

// State returns the last known state, disconnected by default.
func (c *Manager) State(profileID string) ConnState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.state[profileID]; ok {
		return s
	}
	return ConnState{ProfileID: profileID, State: StateDisconnected}
}

// States returns snapshots for every profile touched during this session.
func (c *Manager) States() []ConnState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ConnState, 0, len(c.state))
	for _, state := range c.state {
		out = append(out, state)
	}
	return out
}

// CloseAll closes every open connection.
func (c *Manager) CloseAll() {
	c.mu.Lock()
	open := c.open
	c.open = make(map[string]*ldapx.Conn)
	c.mu.Unlock()

	for _, conn := range open {
		_ = conn.Close()
	}
}

// Dial opens and binds one connection. It runs inside a job, so every step is
// cancellable and nothing here blocks the UI.
func (c *Manager) Dial(ctx context.Context, p profiles.Profile, cred *credentials.Credential) (*ldapx.Conn, error) {
	c.setState(ConnState{ProfileID: p.ID, State: StateConnecting, ReadOnly: p.ReadOnly, Production: p.IsProduction()})

	conn, err := ldapx.Dial(ctx, ldapx.DialConfig{
		Profile:   p,
		Trust:     c.trustPolicy(p),
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
		c.emitter.Emit("capability:unavailable", map[string]any{
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
		c.emitter.Emit("conn:lost", map[string]any{"profileId": p.ID, "reason": lostErr.Error()})
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

	_ = c.logs.Sink(logging.Application).Record("connected %s as %q", p.Name, conn.BoundDN())
	return conn, nil
}

// secretFor resolves the profile's credential.
//
// A missing or unreachable credential store is recoverable: the caller is
// asked for a session-only secret rather than being refused (FR-005).
func (c *Manager) secretFor(p profiles.Profile, cred *credentials.Credential) (secrets.Secret, error) {
	if p.BindMethod == profiles.BindAnonymous || p.BindMethod == profiles.BindExternal {
		return secrets.Secret{}, nil
	}

	ref := p.CredentialID
	if cred != nil && cred.SecretRef != "" {
		ref = cred.SecretRef
	}
	if ref == "" {
		c.emitter.Emit("credential:required", map[string]any{
			"profileId": p.ID,
			"reason":    "this connection has no credential assigned",
		})
		return secrets.Secret{}, fmt.Errorf("bridge: connection %q has no credential assigned", p.Name)
	}

	secret, err := c.provider.Get(ref)
	if err != nil {
		c.emitter.Emit("credential:required", map[string]any{
			"profileId": p.ID,
			"ref":       ref,
			"reason":    err.Error(),
		})
		return secrets.Secret{}, err
	}
	return secret, nil
}
