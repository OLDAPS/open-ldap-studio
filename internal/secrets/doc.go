// Package secrets is the only package that touches a credential.
//
// Boundary: no package outside this one may import a platform credential API, and no os/exec may appear in this package's transitive import tree (contracts C2, C3, and internal/secrets/no_subprocess_test.go).
package secrets
