package main

import "github.com/open-ldap-studio/open-ldap-studio/internal/bridge"

// desktopBindings is shared by the native application and Wails' generator so
// the generated API always describes the objects bound into the window.
func desktopBindings(app *bridge.Bridge) []any {
	return []any{app}
}
