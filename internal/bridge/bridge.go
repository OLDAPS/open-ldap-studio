package bridge

import (
	"context"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/open-ldap-studio/open-ldap-studio/internal/commands"
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
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
	ctx context.Context

	jobs     *jobs.Registry
	commands *commands.Registry
	changes  ChangeApplication
	app      ConnectionApplication

	mu    sync.RWMutex
	ready bool
}

// New injects composed services without opening stores or constructing adapters.
func New(deps Dependencies) *Bridge {
	return &Bridge{
		jobs: deps.Jobs, commands: deps.Commands,
		changes: deps.Changes, app: deps.Application,
	}
}

// Startup receives the Wails context. Events cannot be emitted before it
// arrives, so anything queued earlier is dropped rather than panicking.
func (b *Bridge) Startup(ctx context.Context) {
	b.mu.Lock()
	b.ctx = ctx
	b.ready = true
	b.mu.Unlock()

	b.app.Startup()
}

// Shutdown delegates application lifecycle cleanup.
func (b *Bridge) Shutdown() { b.app.Shutdown() }

func (b *Bridge) context() context.Context {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.ctx == nil {
		return context.Background()
	}
	return b.ctx
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
	return AppInfo(b.app.Info())
}

type CredentialStatus struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// CredentialStoreStatus reports whether the platform agent is reachable, and
// why not when it is not. It never returns a secret.
func (b *Bridge) CredentialStoreStatus() CredentialStatus {
	available, reason := b.app.CredentialStoreStatus()
	return CredentialStatus{Available: available, Reason: reason}
}
