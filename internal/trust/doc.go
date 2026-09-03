// Package trust stores certificate trust decisions, keyed by exact SHA-256 fingerprint.
//
// Boundary: a decision never applies to a certificate other than the one it was made for (FR-008).
package trust
