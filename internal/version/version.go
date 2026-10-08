// Package version carries the application's release identity.
//
// Version is the single source of truth for the released version number and is
// rewritten by Release Please on every release pull request — the trailing
// annotation is what marks the line, so do not remove it. Commit and Date are
// empty in a development build and are stamped by the release workflow via
// -ldflags.
package version

// Version is the semantic version of this build.
const Version = "0.1.0" // x-release-please-version

var (
	// Commit is the git SHA this binary was built from, stamped at link time.
	Commit = "unknown"

	// Date is the RFC 3339 build timestamp, stamped at link time.
	Date = "unknown"
)

// String renders the full build identity for the about box and the logs.
func String() string {
	return Version + " (" + Commit + ", built " + Date + ")"
}
