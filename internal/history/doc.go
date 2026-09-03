// Package history is the append-only operation and search log.
//
// Boundary: redaction happens at write time, before any byte reaches disk (FR-093, contract X7).
package history
