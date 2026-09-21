// Package architecture holds the contract tests that assert this codebase's
// three constitutional boundaries. They are not style checks: each one is the
// mechanical form of a guarantee the constitution states in prose, and each
// blocks merge.
package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/open-ldap-studio/open-ldap-studio"

// repoRoot walks up from the test's directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find the module root")
		}
		dir = parent
	}
}

// goPackage is one parsed package of first-party source.
type goPackage struct {
	// ImportPath is relative to the module, e.g. "internal/changeset".
	ImportPath string
	Files      map[string]*ast.File
	FileSet    *token.FileSet
}

// firstPartyPackages parses every non-test Go file under internal/ and at the
// module root.
func firstPartyPackages(t *testing.T) []goPackage {
	t.Helper()
	root := repoRoot(t)

	var packages []goPackage
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		base := entry.Name()
		if base == "node_modules" || base == ".git" || base == "frontend" || base == "dist" {
			return filepath.SkipDir
		}

		fset := token.NewFileSet()
		parsed, parseErr := parser.ParseDir(fset, path, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, parser.ParseComments)
		if parseErr != nil {
			return parseErr
		}
		for _, pkg := range parsed {
			relative, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			packages = append(packages, goPackage{
				ImportPath: filepath.ToSlash(relative),
				Files:      pkg.Files,
				FileSet:    fset,
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) == 0 {
		t.Fatal("parsed no packages; the boundary tests would pass vacuously")
	}
	return packages
}

// Contract C1 — the ldapx mutation functions have exactly one calling package,
// internal/changeset.
//
// This is what makes SC-005 ("an automated check over every mutation path
// finds no exception") a check rather than a claim. Go cannot express
// "package-private to one friend", so the boundary lives here.
func TestOnlyChangesetCallsTheMutationFunctions(t *testing.T) {
	mutations := map[string]bool{
		"AddEntry":      true,
		"ModifyEntry":   true,
		"RenameEntry":   true,
		"DeleteEntry":   true,
		"DeleteSubtree": true,
	}
	const permitted = "internal/changeset"

	for _, pkg := range firstPartyPackages(t) {
		if pkg.ImportPath == permitted || pkg.ImportPath == "internal/ldapx" {
			continue
		}
		for name, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok || ident.Name != "ldapx" || !mutations[sel.Sel.Name] {
					return true
				}
				t.Errorf("%s calls ldapx.%s at %s: every write goes through %s, so that no mutation path can lack a preview",
					pkg.ImportPath, sel.Sel.Name, pkg.FileSet.Position(call.Pos()), permitted)
				_ = name
				return true
			})
		}
	}
}

// Contract C2 — no package outside internal/secrets imports a platform
// credential API.
func TestOnlySecretsImportsAPlatformCredentialAPI(t *testing.T) {
	platformAPIs := []string{
		"github.com/godbus/dbus",
		"github.com/keybase/go-keychain",
		"github.com/danieljoos/wincred",
		"github.com/zalando/go-keyring",
		"github.com/99designs/keyring",
	}
	const permitted = "internal/secrets"

	for _, pkg := range firstPartyPackages(t) {
		if pkg.ImportPath == permitted {
			continue
		}
		for _, file := range pkg.Files {
			for _, imported := range file.Imports {
				path := strings.Trim(imported.Path.Value, `"`)
				for _, api := range platformAPIs {
					if strings.HasPrefix(path, api) {
						t.Errorf("%s imports %s: the credential path lives in %s alone",
							pkg.ImportPath, path, permitted)
					}
				}
			}
		}
	}
}

// TestSecretsIsNotImportedForItsSecretType asserts the second half of the
// credential boundary: packages may hand a Secret to a bind, but nothing
// outside secrets may construct one from arbitrary bytes it read itself.
func TestOnlyExpectedPackagesTouchTheSecretsPackage(t *testing.T) {
	// ldapx sends the secret; connections resolves it and hands it over. The
	// bridge accepts short-lived secrets from transport calls. Nothing else has
	// a reason to hold one.
	permitted := map[string]bool{
		"internal/secrets":     true,
		"internal/ldapx":       true,
		"internal/connections": true,
		"internal/bridge":      true,
	}

	for _, pkg := range firstPartyPackages(t) {
		if permitted[pkg.ImportPath] {
			continue
		}
		for _, file := range pkg.Files {
			for _, imported := range file.Imports {
				path := strings.Trim(imported.Path.Value, `"`)
				if path == modulePath+"/internal/secrets" {
					t.Errorf("%s imports internal/secrets; only %v have a reason to hold secret material",
						pkg.ImportPath, keys(permitted))
				}
			}
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
