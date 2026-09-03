// Package jobs is the cancellable job registry behind every server-touching operation.
//
// Boundary: nothing that takes time blocks the UI: each job owns a context, reports throttled progress, and answers Cancel within 2 s (SC-002, SC-003).
package jobs
