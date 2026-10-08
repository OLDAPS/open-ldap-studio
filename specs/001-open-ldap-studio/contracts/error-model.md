# Contract: Error and Result Model

**Date**: 2026-08-31 | **Plan**: [../plan.md](../plan.md)

Constitution Principle III and IV both turn on one rule: **the server's own words survive to the
user**. SC-006 measures it at 100%, and reviewers are instructed to reject any change that discards
a result code or diagnostic message. That makes error handling a contract, not a style preference.

---

## The one rule

```go
type Result struct {
    Code              int      // ALWAYS populated — 0 (success) is a value, not an absence
    MatchedDN         string   // verbatim
    DiagnosticMessage string   // verbatim, byte-for-byte as received
    Referrals         []string
    Controls          []Control
    Interpretation    string   // ADDITIVE plain-language text — never a replacement
}
```

`Interpretation` is rendered **alongside** `DiagnosticMessage`, never instead of it. A UI that shows
only the interpretation is a defect; so is one that shows only the raw message when an
interpretation exists, since Principle IV asks for both.

**Prohibited transformations**, each of which fails SC-006:

- Collapsing a `Result` to `error` or `bool` anywhere between `ldapx` and the frontend
- Replacing a diagnostic message with a friendlier one
- Trimming, re-casing, or re-wrapping the diagnostic message
- Reporting only the first error of a multi-entry operation
- Swallowing `sizeLimitExceeded` (4) or `timeLimitExceeded` (3) as an empty result set

---

## Error categories

Every failure the user can see falls into exactly one of these. The category determines the
affordance offered, which is why the taxonomy matters beyond tidiness.

| Category | Origin | Carries a `Result`? | Affordance |
|----------|--------|--------------------|------------|
| **ServerError** | The directory returned a non-zero result code | Yes | Show verbatim; the operation is definitively not applied |
| **AuthError** | Bind rejected, credential expired, ticket missing, account locked | Yes | Re-authenticate without losing unsaved work; **never** retried in a loop, and never silently downgraded to anonymous (D7) |
| **TransportError** | Dial failure, DNS, TLS negotiation, connection reset | No | Retry; the profile stays usable |
| **TrustError** | Certificate validation failed | No | `trust:challenge`; connection stays refused until a decision |
| **CredentialStoreError** | Agent locked, unavailable, absent, or the secret is gone | No | Recoverable — falls through to the session-only path (FR-005) |
| **LocalValidationError** | Filter syntax, LDIF parse, schema violation caught before dispatch | No | Position and reason; **nothing was sent to the server** |
| **CapabilityError** | Server does not support a required control or operation | Sometimes | `capability:unavailable`; degrade with the reason stated (SC-016) |
| **Indeterminate** | Connection lost after a write was sent | No | **Neither success nor failure.** Re-read the entry; the history records it as indeterminate (FR-089) |
| **Cancelled** | User cancelled | No | Not an error state; report what was and was not done |

`Indeterminate` is the category most often missing from LDAP clients and the one most likely to
mislead an administrator. It exists because "we sent a modify and the socket died" is genuinely
unknowable from the client, and pretending otherwise is how a directory gets double-modified.

---

## Result codes with mandated handling

The spec names behaviour for these; the rest are surfaced generically with their diagnostic.

| Code | Name | Required handling | Requirement |
|------|------|-------------------|-------------|
| 0 | success | `Result` still returned and logged | FR-051 |
| 3 | timeLimitExceeded | Partial results shown, labelled truncated-by-server | FR-037 |
| 4 | sizeLimitExceeded | As above, plus "fetch next page" where paging is available | FR-037, flow 7d |
| 10 | referral | Follow / ignore / list per policy; chooser dialog on `ask` (6e) | FR-021 |
| 12 | unavailableCriticalExtension | `capability:unavailable`; do **not** silently retry without the control | FR-022 |
| 19 | constraintViolation | Verbatim; entry refreshed to actual server state | FR-051 |
| 20 | attributeOrValueExists | Verbatim; the client does **not** pre-deduplicate to avoid it | Edge cases §Data |
| 32 | noSuchObject | Verbatim; refresh the parent branch | — |
| 49 | invalidCredentials | AuthError; no retry loop, no anonymous fallback | D7 |
| 50 | insufficientAccessRights | Verbatim; offer the effective-rights check where supported (G1) | FR-051 |
| 53 | unwillingToPerform | Verbatim — commonly the response to a schema-modification or config write refusal | FR-071, FR-088 |
| 66 | notAllowedOnNonLeaf | Verbatim; offer subtree delete as a **distinct** confirmed action | FR-046 |
| 80 | other | Verbatim; never re-interpreted | FR-051 |

---

## Multi-entry operations

A bulk job, import, copy, or subtree delete produces **one outcome per entry**, never one verdict
for the run:

```go
type EntryOutcome struct {
    DN     string
    Status string  // succeeded | failed | skipped
    Result *Result // populated for failed — verbatim
}
```

Job terminal states are `succeeded`, `failed`, `cancelled`, and `partiallyComplete`.
`partiallyComplete` is never folded into `succeeded` (FR-050), and the report always names every
skipped DN — the wireframe states this as "no partial-silent state" in flow 7e.

Failed records are written to a sibling `.ldif` so the run can be repaired and repeated (flow 7e).

---

## Redaction

Redaction happens **at write time**, before a record reaches disk (FR-093). A redaction applied at
display time leaves the secret in the file, which is the failure SC-008 scans for.

Redacted everywhere — logs, history, diagnostics, crash output, event payloads:
`userPassword`, any attribute the user marks sensitive, bind credentials, and SASL exchange
material. The DN and the result code are **not** redacted; losing them would defeat the log's
purpose.

---

## Contract tests

| # | Assertion | Serves |
|---|-----------|--------|
| X1 | Every bridge method touching a server returns a populated `Result` on success and failure | SC-006 |
| X2 | No code path between `ldapx` and the bridge converts a `Result` to `bool` or bare `error` | SC-006 |
| X3 | `DiagnosticMessage` is byte-identical to the server's, asserted against a server returning a deliberately odd message | SC-006 |
| X4 | A connection dropped mid-write yields `Indeterminate`, never success or failure | FR-089 |
| X5 | Every code in the table above triggers its mandated handling, forced against a live server | FR-037, FR-046, FR-051 |
| X6 | A multi-entry run with mixed outcomes reports every entry and never reports `succeeded` | FR-050, FR-097 |
| X7 | Redaction is applied before write: the raw history file never contains a password value | FR-093, SC-008 |
| X8 | `invalidCredentials` produces no retry loop and no anonymous fallback | D7 |
