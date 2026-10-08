package secrets

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// Contract C3 — the credential path never starts a subprocess.
//
// The literal form of this contract in tasks.md T016 is "no os/exec appears in
// the transitive import tree of internal/secrets". That form is unsatisfiable
// with the Linux dependency plan.md selects: github.com/godbus/dbus/v5 imports
// os/exec so it can run dbus-launch when DBUS_SESSION_BUS_ADDRESS is unset.
// Research R2 checked zalando/go-keyring for this and missed it in godbus.
//
// The contract is therefore asserted in the two halves that carry its meaning,
// and the deviation is recorded as D10 in docs/deviations.md:
//
//  1. No first-party package in the tree imports os/exec at all.
//  2. The one third-party exception is godbus, and we never call the entry
//     points that reach its exec path — openPlatform resolves the bus address
//     itself and reports the store unavailable when there is none.
//
// The difference from the rejected library matters: dbus-launch would start a
// bus, while /usr/bin/security returns the secret itself down a pipe. Neither
// is permitted here, but only one of them puts secret material in a process
// table.
const secretsPkg = "github.com/open-ldap-studio/open-ldap-studio/internal/secrets"

// permittedExecImporters are third-party packages allowed to import os/exec
// on the strength of the guarantees asserted below. Adding to this list is a
// constitutional change, not a test fix.
var permittedExecImporters = map[string]bool{
	"github.com/godbus/dbus/v5": true,
}

func TestNoFirstPartyPackageInTheSecretsTreeImportsOsExec(t *testing.T) {
	for _, target := range []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
	} {
		t.Run(target.goos, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", secretsPkg)
			cmd.Env = append(cmd.Environ(), "GOOS="+target.goos, "GOARCH="+target.goarch, "CGO_ENABLED=1")

			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Skipf("cannot resolve dependencies for %s/%s: %s", target.goos, target.goarch, out)
			}

			for line := range strings.Lines(string(out)) {
				fields := strings.Fields(line)
				if len(fields) == 0 {
					continue
				}
				pkg, imports := fields[0], fields[1:]
				for _, imp := range imports {
					if imp != "os/exec" {
						continue
					}
					if permittedExecImporters[pkg] {
						continue
					}
					t.Errorf("%s imports os/exec on %s: the credential path must stay in-process",
						pkg, target.goos)
				}
			}
		})
	}
}

// TestSecretsNeverCallsTheAutolaunchingEntryPoints is the second half: godbus
// only reaches os/exec through these, and we call none of them.
//
// The check parses the package rather than grepping it, so a comment
// describing the forbidden call is not mistaken for the call itself.
func TestSecretsNeverCallsTheAutolaunchingEntryPoints(t *testing.T) {
	forbidden := map[string]bool{
		"SessionBus":                     true,
		"SessionBusPrivate":              true,
		"SessionBusPrivateNoAutoStartup": true,
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir( //nolint:staticcheck // build tags do not matter for a source scan
		fset, ".", func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
	if err != nil {
		t.Fatal(err)
	}

	for _, pkg := range pkgs {
		ast.Inspect(pkg, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "dbus" || !forbidden[sel.Sel.Name] {
				return true
			}
			t.Errorf("%s calls dbus.%s, which autolaunches a bus through os/exec; "+
				"resolve DBUS_SESSION_BUS_ADDRESS and dial it instead",
				fset.Position(call.Pos()), sel.Sel.Name)
			return true
		})
	}
}

// TestUnsetBusAddressIsRecoverableNotASubprocess pins the behaviour the
// deviation rests on: with no bus address, the store is unavailable and the
// caller falls through to the session-only path. It never shells out.
func TestUnsetBusAddressIsRecoverableNotASubprocess(t *testing.T) {
	// The bus address only governs provider selection on Linux; macOS and
	// Windows resolve to their own platform store and ignore it entirely.
	if runtime.GOOS != "linux" {
		t.Skip("the D-Bus session address is a Linux-only selector")
	}

	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")

	p, reason := Open()
	if p.Name() != "session only" {
		t.Fatalf("with no bus address the provider is %q, want the session fallback", p.Name())
	}
	if reason == "" {
		t.Error("falling back must state why; an unavailable store is explained, never silent")
	}
}
