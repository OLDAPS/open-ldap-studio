package architecture

import (
	"go/ast"
	"strings"
	"testing"
)

// Contract X2 — no code path between ldapx and the bridge collapses a Result
// to a bool or a bare error.
//
// SC-006 measures "the server's own words survive to the user" at 100%. The
// place that guarantee is usually lost is a helper that decided a bool was
// enough, so the rule is asserted on the signatures themselves: a function
// that takes a connection returns a Result.
func TestEveryServerTouchingFunctionReturnsAResult(t *testing.T) {
	// Compare is the one honest exception: its two success codes are
	// compareTrue and compareFalse, so it returns a bool *and* the Result.
	for _, pkg := range firstPartyPackages(t) {
		if pkg.ImportPath != "internal/ldapx" {
			continue
		}
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !fn.Name.IsExported() || fn.Type.Params == nil {
					continue
				}
				if !takesConnection(fn) {
					continue
				}
				if returnsResult(fn) {
					continue
				}
				t.Errorf("ldapx.%s takes a connection but returns no Result at %s: "+
					"the server's result code and diagnostic must survive to the caller",
					fn.Name.Name, pkg.FileSet.Position(fn.Pos()))
			}
		}
	}
}

func takesConnection(fn *ast.FuncDecl) bool {
	for _, param := range fn.Type.Params.List {
		star, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		if ident, ok := star.X.(*ast.Ident); ok && ident.Name == "Conn" {
			return true
		}
	}
	return false
}

func returnsResult(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, result := range fn.Type.Results.List {
		switch typed := result.Type.(type) {
		case *ast.Ident:
			// Result itself, or a type that embeds one (Page carries it).
			if typed.Name == "Result" || typed.Name == "Page" || typed.Name == "RootDSE" {
				return true
			}
		case *ast.StarExpr:
			if ident, ok := typed.X.(*ast.Ident); ok && ident.Name == "Result" {
				return true
			}
		}
	}
	return false
}

// TestNoDiagnosticMessageIsRewritten asserts the other half of SC-006 at the
// source level: nothing reformats a diagnostic on its way out.
func TestNoDiagnosticMessageIsReformatted(t *testing.T) {
	// Any call that passes DiagnosticMessage through a transforming function
	// is a rewrite. Reading it, comparing it and printing it are all fine.
	transformers := map[string]bool{
		"TrimSpace": true, "Trim": true, "ToLower": true, "ToUpper": true,
		"Title": true, "ReplaceAll": true, "Replace": true, "TrimSuffix": true,
		"TrimPrefix": true,
	}

	for _, pkg := range firstPartyPackages(t) {
		if !strings.HasPrefix(pkg.ImportPath, "internal/") {
			continue
		}
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !transformers[sel.Sel.Name] {
					return true
				}
				for _, arg := range call.Args {
					if argSel, ok := arg.(*ast.SelectorExpr); ok && argSel.Sel.Name == "DiagnosticMessage" {
						t.Errorf("%s passes DiagnosticMessage through %s at %s: "+
							"the server's message is shown byte-for-byte (SC-006)",
							pkg.ImportPath, sel.Sel.Name, pkg.FileSet.Position(call.Pos()))
					}
				}
				return true
			})
		}
	}
}
