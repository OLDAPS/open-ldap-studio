package bridge

import (
	"github.com/open-ldap-studio/open-ldap-studio/internal/jobs"
)

// Every long-running operation is a job with an id, and every id has a Cancel
// that takes effect within two seconds (contract C8, SC-003).

// Cancel stops a running job.
func (b *Bridge) Cancel(jobID string) error { return b.jobs.Cancel(jobID) }

// JobState returns one job's current state.
func (b *Bridge) JobState(jobID string) (jobs.Job, error) { return b.jobs.Get(jobID) }

// ListJobs returns every job, running and finished, for the Progress panel.
func (b *Bridge) ListJobs() []jobs.Job { return b.jobs.List() }

// JobOutcomes returns the per-entry outcomes of a multi-entry job. Every
// entry appears, including the skipped ones (FR-097).
func (b *Bridge) JobOutcomes(jobID string) ([]jobs.Outcome, error) { return b.jobs.Outcomes(jobID) }

// PauseJob and ResumeJob hold a throttled bulk run between batches (FR-096).
func (b *Bridge) PauseJob(jobID string) error  { return b.jobs.Pause(jobID) }
func (b *Bridge) ResumeJob(jobID string) error { return b.jobs.Resume(jobID) }
