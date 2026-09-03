// Package controls builds the request controls the browser and search views
// rely on, and decodes the response controls they come back with.
//
// It sits inside ldapx's boundary: controls are protocol, and no package
// outside internal/ldapx constructs one.
package controls

import (
	"github.com/go-ldap/ldap/v3"
)

// Control OIDs used here that go-ldap does not name.
const (
	OIDVLVRequest    = "2.16.840.1.113730.3.4.9"
	OIDVLVResponse   = "2.16.840.1.113730.3.4.10"
	OIDSubtreeDelete = "1.2.840.113556.1.4.805"
)

// Paging returns the simple paged results control (RFC 2696). The cookie
// continues a previous page; a nil cookie starts a new one.
func Paging(size int, cookie []byte) *ldap.ControlPaging {
	c := ldap.NewControlPaging(uint32(size))
	if len(cookie) > 0 {
		c.SetCookie(cookie)
	}
	return c
}

// SortKey is one server-side sort key (RFC 2891).
type SortKey struct {
	Attribute    string
	MatchingRule string
	Reverse      bool
}

// Sort returns the server-side sorting control. The server may refuse it; a
// refusal is reported as a capability, never worked around by sorting a
// truncated page on the client and calling it sorted (SC-016).
func Sort(keys []SortKey) *ldap.ControlServerSideSorting {
	sortKeys := make([]*ldap.SortKey, 0, len(keys))
	for _, k := range keys {
		sortKeys = append(sortKeys, &ldap.SortKey{
			AttributeType: k.Attribute,
			MatchingRule:  k.MatchingRule,
			Reverse:       k.Reverse,
		})
	}
	return ldap.NewControlServerSideSortingWithSortKeys(sortKeys)
}

// SubtreeDelete returns the tree-delete control. It is attached only to a
// delete the user confirmed as a subtree delete, after the subtree has been
// counted (FR-046).
func SubtreeDelete() *ldap.ControlSubtreeDelete {
	return ldap.NewControlSubtreeDelete()
}

// ManageDsaIT makes referral and alias entries visible as ordinary entries,
// which is how they are edited rather than followed (RFC 3296).
func ManageDsaIT(critical bool) *ldap.ControlManageDsaIT {
	return ldap.NewControlManageDsaIT(critical)
}

// PagingCookie extracts the cookie from a response's paging control. A missing
// control means the server ignored paging — the caller reports that as an
// unavailable capability rather than looping forever on an empty cookie.
func PagingCookie(response []ldap.Control) (cookie []byte, present bool) {
	for _, c := range response {
		if paging, ok := c.(*ldap.ControlPaging); ok {
			return paging.Cookie, true
		}
	}
	return nil, false
}

// SortResult reports the outcome of a server-side sort, if the server answered
// with one. The offending attribute is not returned: go-ldap does not decode
// it, and inventing one would be worse than reporting the code alone.
func SortResult(response []ldap.Control) (code int, present bool) {
	for _, c := range response {
		if sorted, ok := c.(*ldap.ControlServerSideSortingResult); ok {
			return int(sorted.Result), true
		}
	}
	return 0, false
}
