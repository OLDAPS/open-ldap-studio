// Package bridge is the Wails-bound surface: the application's external interface.
//
// Boundary: it translates and delegates, holding no logic a test would have to reach through the UI to exercise. It exposes no Modify/Add/Delete/Rename method and no method returning a secret (contracts C9, C11).
package bridge
