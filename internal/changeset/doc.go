// Package changeset is the only write path in this application.
//
// Boundary: build -> Preview -> Commit(token) -> dispatch. It is the sole caller of the ldapx mutation functions, and it is the only place that issues a PreviewToken. A write with no preview is not expressible above this package (contracts C1, C4, C13).
package changeset
