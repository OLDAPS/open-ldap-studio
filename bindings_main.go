//go:build bindings

package main

import (
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"

	"github.com/open-ldap-studio/open-ldap-studio/internal/bridge"
)

// Wails runs this entrypoint to reflect the API. Generation must work on a
// fresh clone without frontend assets, logs, a credential store, or a display.
func main() {
	if err := wails.Run(&options.App{Bind: desktopBindings(&bridge.Bridge{})}); err != nil {
		slog.Error("cannot generate desktop bindings", "err", err)
		os.Exit(1)
	}
}
