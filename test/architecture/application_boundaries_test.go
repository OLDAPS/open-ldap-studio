package architecture

import (
	"go/ast"
	"strconv"
	"strings"
	"testing"
)

func TestApplicationUseCasesDoNotImportWailsOrBridge(t *testing.T) {
	for _, pkg := range firstPartyPackages(t) {
		if !strings.HasPrefix(pkg.ImportPath, "internal/application/") {
			continue
		}
		for _, file := range pkg.Files {
			for _, imported := range file.Imports {
				path, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(path, "wailsapp") || strings.HasSuffix(path, "/internal/bridge") {
					t.Errorf("%s imports transport %s", pkg.ImportPath, path)
				}
			}
		}
	}
}

func TestConnectionBridgeDoesNotConstructAdaptersOrSpeakLDAP(t *testing.T) {
	for _, pkg := range firstPartyPackages(t) {
		if pkg.ImportPath != "internal/bridge" {
			continue
		}
		for name, file := range pkg.Files {
			if !strings.HasSuffix(name, "/connect.go") && !strings.HasSuffix(name, "/connecttest.go") && !strings.HasSuffix(name, "/bridge.go") {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				owner, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				if owner.Name == "ldapx" || (selector.Sel.Name == "OpenStore" || selector.Sel.Name == "NewRegistry") {
					t.Errorf("%s performs adapter work at %s", name, pkg.FileSet.Position(call.Pos()))
				}
				return true
			})
		}
	}
}

func TestMutationBridgeDoesNotOwnPipelineOrAudit(t *testing.T) {
	for _, pkg := range firstPartyPackages(t) {
		if pkg.ImportPath != "internal/bridge" {
			continue
		}
		for name, file := range pkg.Files {
			for _, imported := range file.Imports {
				if strings.Trim(imported.Path.Value, `"`) == modulePath+"/internal/logging" {
					t.Errorf("%s imports logging; application use cases own audit coordination", name)
				}
			}
			ast.Inspect(file, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if owner, ok := selector.X.(*ast.Ident); ok && owner.Name == "changeset" &&
					(selector.Sel.Name == "Pipeline" || selector.Sel.Name == "NewPipeline") {
					t.Errorf("%s owns the concrete mutation pipeline at %s", name, pkg.FileSet.Position(selector.Pos()))
				}
				if strings.HasSuffix(name, "/changeset.go") {
					switch selector.Sel.Name {
					case "Start", "Record", "Emit":
						t.Errorf("%s coordinates mutation jobs, audit or refresh at %s", name, pkg.FileSet.Position(selector.Pos()))
					}
				}
				return true
			})
		}
	}
}
