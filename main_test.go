//go:build !bindings

package main

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/open-ldap-studio/open-ldap-studio/internal/bridge"
)

func TestDesktopShellContract(t *testing.T) {
	app := &bridge.Bridge{}
	opts := desktopOptions(app)
	if opts.Title == "" || opts.Width < opts.MinWidth || opts.Height < opts.MinHeight {
		t.Fatal("desktop window must have a title and usable initial dimensions")
	}
	if len(opts.Bind) != 1 || opts.Bind[0] != app {
		t.Fatal("the desktop must bind its Go bridge")
	}
	if opts.OnStartup == nil || opts.OnShutdown == nil {
		t.Fatal("the desktop must start and stop the bridge with its native window")
	}
	if opts.Linux == nil || opts.Mac == nil || opts.Windows == nil {
		t.Fatal("the desktop must configure each supported native backend")
	}
	if opts.AssetServer == nil || opts.AssetServer.Assets == nil {
		t.Fatal("the desktop must host its embedded frontend")
	}
	index, err := fs.ReadFile(opts.AssetServer.Assets, "frontend/dist/index.html")
	if err != nil {
		t.Fatalf("embedded frontend entrypoint: %v", err)
	}
	if !strings.Contains(string(index), `id="root"`) || !strings.Contains(string(index), `type="module"`) {
		t.Fatal("the embedded entrypoint must mount the frontend and load its module")
	}
}
