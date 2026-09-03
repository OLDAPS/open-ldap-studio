// Package prefs models and persists preferences and the on-disk file envelope.
//
// Boundary: every written file carries schemaVersion as its first key, and a file written by a newer version is refused rather than rewritten (contracts F1, F2).
package prefs
