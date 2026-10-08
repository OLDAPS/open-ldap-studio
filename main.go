// Command open-ldap-studio is the Wails v2 entrypoint for Open LDAP Studio.
//
// This file owns exactly three things: the embedded frontend assets, the window,
// and the set of Go objects bound into the webview. Everything else lives behind
// internal/bridge, which is the application's only external interface.
package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/open-ldap-studio/open-ldap-studio/internal/bridge"
	"github.com/open-ldap-studio/open-ldap-studio/internal/logging"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	paths, err := logging.DefaultPaths()
	if err != nil {
		slog.Error("cannot resolve application data directory", "err", err)
		os.Exit(1)
	}

	logs, err := logging.Open(paths)
	if err != nil {
		slog.Error("cannot open log sinks", "err", err)
		os.Exit(1)
	}
	defer func() { _ = logs.Close() }()

	app := bridge.New(logs)

	err = wails.Run(&options.App{
		Title:       "Open LDAP Studio",
		Width:       1440,
		Height:      900,
		MinWidth:    1024,
		MinHeight:   640,
		AssetServer: &assetserver.Options{Assets: assets},
		// The wireframe draws its own title bar (dots, menu bar, context strip),
		// so the window is frameless and the chrome is rendered by the frontend.
		Frameless:        true,
		BackgroundColour: &options.RGBA{R: 0x17, G: 0x17, B: 0x17, A: 1},
		OnStartup: func(ctx context.Context) {
			app.Startup(ctx)
		},
		OnShutdown: func(context.Context) {
			app.Shutdown()
		},
		Bind: []any{app},
		Linux: &linux.Options{
			WindowIsTranslucent: false,
			ProgramName:         "open-ldap-studio",
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "Open LDAP Studio",
				Message: "A cross-platform LDAP workbench.",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			DisableWindowIcon:    false,
		},
	})
	if err != nil {
		slog.Error("application exited with an error", "err", err)
		os.Exit(1)
	}
}
