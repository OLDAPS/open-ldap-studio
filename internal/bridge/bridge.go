package bridge

import (
	"context"
	"path/filepath"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/open-ldap-studio/open-ldap-studio/internal/changeset"
	"github.com/open-ldap-studio/open-ldap-studio/internal/commands"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
	"github.com/open-ldap-studio/open-ldap-studio/internal/logging"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
	"github.com/open-ldap-studio/open-ldap-studio/internal/trust"
)

// Version of the application, stamped at build time.
var Version = "0.1.0-dev"

// Bridge is the object Wails binds. Every method on it is callable from the
// webview, which is why the surface is deliberately narrow:
//
//   - There is no Modify, Add, Delete or Rename. Preview and Commit(token) are
//     the only way a change reaches a server (contract C1, bridge-api.md Rule 1).
//   - No method returns secret material, and none of the vault methods the
//     wireframe implied exists (contracts C9, C11, deviation D2).
type Bridge struct {
	ctx  context.Context
	logs *logging.Logs

	jobs     *jobs.Registry
	commands *commands.Registry
	profiles *profiles.Store
	pipeline *changeset.Pipeline

	secrets       secrets.Provider
	secretsReason string
	trust         *trust.Store

	conns *connections

	mu    sync.RWMutex
	ready bool
}

// New builds the bridge. It does no I/O beyond opening the profile store, so a
// failure here cannot stop the window from appearing — the application must
// reach a usable window in under three seconds with no prompt (SC-009).
func New(logs *logging.Logs) *Bridge {
	paths, err := logging.DefaultPaths()
	if err != nil {
		paths = logging.Paths{Data: "."}
	}

	provider, reason := secrets.Open()

	b := &Bridge{
		logs:          logs,
		commands:      commands.Default(),
		secrets:       provider,
		secretsReason: reason,
	}
	b.jobs = jobs.NewRegistry(b)

	store, err := profiles.OpenStore(filepath.Join(paths.Data, "connections.json"))
	if err != nil {
		// A store from a newer version, or an unreadable one, is reported
		// rather than replaced. The application still starts; the connections
		// list explains itself.
		_ = logs.Sink(logging.Errors).Record("profile store: %v", err)
		store, _ = profiles.OpenStore(filepath.Join(paths.Data, "connections.json.unreadable"))
	}
	b.profiles = store

	trustStore, err := trust.OpenStore(filepath.Join(paths.Data, "trust.json"))
	if err != nil {
		_ = logs.Sink(logging.Errors).Record("trust store: %v", err)
		trustStore, _ = trust.OpenStore(filepath.Join(paths.Data, "trust.json.unreadable"))
	}
	b.trust = trustStore

	b.conns = newConnections(b, provider)
	b.pipeline = changeset.NewPipeline(b.conns, store, changeset.Guard{RequireProductionConfirmation: true})

	return b
}

// Startup receives the Wails context. Events cannot be emitted before it
// arrives, so anything queued earlier is dropped rather than panicking.
func (b *Bridge) Startup(ctx context.Context) {
	b.mu.Lock()
	b.ctx = ctx
	b.ready = true
	b.mu.Unlock()

	_ = b.logs.Sink(logging.Application).Record("started, version %s, credential store %q",
		Version, b.secrets.Name())

	if b.secretsReason != "" {
		// An unavailable credential store is recoverable and explained, never
		// silent (Constitution II, FR-005).
		b.Emit("credential:storeUnavailable", map[string]any{"reason": b.secretsReason})
	}
}

// Shutdown cancels every running job and closes every connection.
func (b *Bridge) Shutdown() {
	b.jobs.CancelAll()
	b.conns.closeAll()
	if session, ok := b.secrets.(*secrets.Session); ok {
		session.Zero()
	}
	_ = b.logs.Sink(logging.Application).Record("shutdown")
}

// Emit sends a runtime event to the frontend. It is the jobs.Emitter the
// registry was built with.
func (b *Bridge) Emit(name string, payload any) {
	b.mu.RLock()
	ctx, ready := b.ctx, b.ready
	b.mu.RUnlock()
	if !ready {
		return
	}
	wailsruntime.EventsEmit(ctx, name, payload)
}

// AppInfo is what the About pane and the status bar need at start-up.
type AppInfo struct {
	Version string `json:"version"`
	// CredentialStore names the platform store in use, or "session only".
	CredentialStore string `json:"credentialStore"`
	// CredentialStoreReason explains a fallback, and is empty when the
	// platform store is in use.
	CredentialStoreReason string `json:"credentialStoreReason"`
	// UpdateChecksEnabled is false by default and stays false until the user
	// turns it on (FR-016, deviation D9).
	UpdateChecksEnabled bool   `json:"updateChecksEnabled"`
	Platform            string `json:"platform"`
}

// GetAppInfo returns the application's own state.
func (b *Bridge) GetAppInfo() AppInfo {
	return AppInfo{
		Version:               Version,
		CredentialStore:       b.secrets.Name(),
		CredentialStoreReason: b.secretsReason,
		UpdateChecksEnabled:   false,
		Platform:              string(currentPlatform()),
	}
}

// CredentialStoreStatus reports whether the platform agent is reachable, and
// why not when it is not. It never returns a secret.
func (b *Bridge) CredentialStoreStatus() (bool, string) {
	available, reason := b.secrets.Available()
	if reason == "" {
		reason = b.secretsReason
	}
	return available, reason
}
