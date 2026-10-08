# Contract: Runtime Event Stream

**Date**: 2026-08-31 | **Plan**: [../plan.md](../plan.md)

Wails events emitted by the Go core and consumed by the frontend. Events are the mechanism behind
Gate IV: nothing that takes time is a blocking call, so progress, state, and challenges all arrive
here.

**Naming**: `domain:event`. Every payload carries `ts` (RFC 3339) and, where server-scoped,
`profileId`.

---

## Job lifecycle

| Event | Payload | Notes |
|-------|---------|-------|
| `job:started` | `{jobId, kind, mode, total?}` | `mode` is `execute` or `dryRun` |
| `job:progress` | `{jobId, done, total, message}` | Throttled to ≤ 20/s so the event stream cannot itself breach the 100 ms UI budget |
| `job:outcome` | `{jobId, dn, status, result?}` | Per-entry; `status` ∈ `succeeded｜failed｜skipped` (FR-097) |
| `job:finished` | `{jobId, state, summary, reportPath?}` | `state` ∈ `succeeded｜failed｜cancelled｜partiallyComplete` |

`partiallyComplete` is a distinct terminal state, never folded into `succeeded` (FR-050).

## Connection state

| Event | Payload | Notes |
|-------|---------|-------|
| `conn:state` | `{profileId, state, boundDN, serverIdentity, tlsVerified}` | Drives the always-visible status bar (FR-010) |
| `conn:reconnected` | `{profileId, writesRequireConfirmation: true}` | After an idle disconnect; the flag is why FR-015 holds |
| `conn:lost` | `{profileId, duringOperation?, jobId?}` | An operation interrupted here is recorded indeterminate (FR-089) |

## Security challenges

| Event | Payload | Notes |
|-------|---------|-------|
| `trust:challenge` | `{host, port, fingerprint, chainPEM, failureReason, previouslyTrusted}` | The connection is **already refused** when this fires; the user's answer starts a fresh attempt (FR-007) |
| `credential:required` | `{profileId, ref, reason}` | Platform agent unavailable or no stored secret → session-only prompt (FR-005) |
| `credential:storeUnavailable` | `{reason}` | Recoverable and explained, never silent (Constitution II) |

No event ever carries secret material. This is asserted by test E1 below.

## Data change notifications

| Event | Payload | Notes |
|-------|---------|-------|
| `entry:changed` | `{profileId, dn, source}` | `source` ∈ `commit｜refresh｜external`; triggers stale-view warnings in other editors (FR-105) |
| `entry:stale` | `{profileId, dn, openEditors}` | Another editor holds an outdated view |
| `profile:reloaded` | `{profileId, reason: "changedOnDisk"}` | Never a silent overwrite (FR-104) |

## Server capability degradation

| Event | Payload | Notes |
|-------|---------|-------|
| `capability:unavailable` | `{profileId, capability, reason}` | `capability` ∈ `paging｜sorting｜vlv｜schema｜extendedOp｜storedACI｜schemaModify｜configEntries` |

Every unavailable capability is announced with a reason. SC-016 is verified by asserting this event
fires for each one against the capability-poor fixture — the absence of a silent failure is the
thing being tested.

## Schema projects

| Event | Payload | Notes |
|-------|---------|-------|
| `schema:problems` | `{projectId, errors, warnings}` | Errors block export and commit; warnings do not |

---

## Contract tests

| # | Assertion | Serves |
|---|-----------|--------|
| E1 | No emitted event payload contains secret material, across every code path | SC-008 |
| E2 | `job:progress` is throttled; a 100k-entry job emits ≤ 20 events/s | SC-002 |
| E3 | Every job reaching a terminal state emits exactly one `job:finished` | FR-050 |
| E4 | `capability:unavailable` fires for each capability the poor fixture lacks | SC-016 |
| E5 | `trust:challenge` never fires without the connection having been refused first | FR-007 |
