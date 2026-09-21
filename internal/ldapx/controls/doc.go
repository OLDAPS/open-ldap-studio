// Package controls encodes and decodes LDAP request and response controls.
//
// It contains protocol-focused implementations and helpers for paging,
// server-side sorting, virtual list view, subtree delete, ManageDsaIT, and other
// controls needed by a directory workbench. Unknown response controls remain
// visible to callers rather than being silently discarded.
//
// The package does not choose control policy, open connections, issue requests,
// or model UI pagination. ldapx selects controls from caller options and owns
// the request lifecycle; this package only handles their wire representation.
package controls
