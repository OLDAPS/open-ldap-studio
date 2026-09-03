// Package ldif reads and writes RFC 2849 LDIF with byte fidelity.
//
// Boundary: values are []byte end to end; folding, base64 policy and attribute options round-trip unchanged (SC-007).
package ldif
