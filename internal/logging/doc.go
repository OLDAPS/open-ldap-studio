// Package logging owns the four log sinks, their rotation, and redaction.
//
// Boundary: redaction is applied before a byte reaches a sink, never at display time (FR-093).
package logging
