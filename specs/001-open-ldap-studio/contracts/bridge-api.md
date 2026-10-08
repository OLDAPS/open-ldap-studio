# Contract: Bridge API (Go ↔ frontend)

**Date**: 2026-08-31 | **Plan**: [../plan.md](../plan.md)

The `internal/bridge` package is the application's external interface: the Go methods Wails binds
and exposes to the webview. This is the contract the frontend codes against and the surface the
integration tests drive — every user story is assertable here without touching the UI.

**Shape conventions**

- Methods returning `JobID` are asynchronous: they start a cancellable job and report through the
  events in [events.md](./events.md).
- Everything else is a synchronous promise on the frontend side.
- `Result` is `ldapx.Result` (see [../data-model.md](../data-model.md)) and is returned on success
  **and** failure. There is no method returning a bare boolean for a server operation.
- Byte-valued attribute values cross as base64 with an explicit `isBinary` flag; they are never
  coerced to UTF-8.

---

## Rule 1 — no write without a token

```go
Preview(cs ChangeSetInput) (ChangeSetPreview, error)   // returns diff + AffectedCount + Token
Commit(token string) (Result, error)                    // the ONLY dispatch path
Discard(token string) error                             // drops the token, keeps the draft
```

There is **no** `Modify`, `Add`, `Delete`, or `Rename` method on the bridge. Every mutation is a
`ChangeSetInput` → `Preview` → `Commit`. This is what makes SC-005 ("an automated check over every
mutation path finds no exception") mechanically checkable: the check is that `ldapx.Modify` and its
siblings have exactly one caller, `changeset.dispatch`.

A CI import-boundary test enforces it: no package outside `changeset` may import the mutation
functions of `ldapx`.

---

## Connections

```go
ListProfiles() ([]ProfileSummary, error)
GetProfile(id string) (Profile, error)
SaveProfile(p Profile) (Profile, error)        // rejects any payload carrying secret material
DeleteProfile(id string) error
DuplicateProfile(id string) (Profile, error)
MoveProfile(id, folderID string) error

ListFolders() ([]Folder, error)
SaveFolder(f Folder) (Folder, error)           // rejects cycles
DeleteFolder(id string) error

ExportProfiles(ids []string, path string) error
ImportProfiles(path string) ([]Profile, error)

Connect(profileID string) (JobID, error)       // async: bind may prompt the platform agent
Disconnect(profileID string) error
ConnectionState(profileID string) (ConnState, error)   // state, boundDN, serverIdentity, tls badge
TestConnection(p Profile) (JobID, error)       // real bind, nothing persisted

WhoAmI(profileID string) (string, Result, error)       // FR-011
RootDSE(profileID string) (RootDSE, error)             // FR-012
```

`SaveProfile` returning an error on secret-bearing input is a contract, not a nicety — it is the
runtime half of the invariant that `Profile` has nowhere to put a secret.

## Credentials

Revised in the second pass: a `Credential` is a first-class entity assigned to many profiles, not a
string on a profile (research R10, data-model §1.1).

```go
ListCredentials() ([]Credential, error)        // metadata only — never a secret value
GetCredential(id string) (Credential, error)
SaveCredential(c Credential) (Credential, error)   // secret supplied via the platform prompt
DeleteCredential(id string) error                  // referring profiles enter "credential missing"
AssignCredential(credentialID, profileID string) error
UnassignCredential(credentialID, profileID string) error
CredentialAssignments(id string) ([]ProfileSummary, error)
TestBind(credentialID, profileID string) (JobID, error)
CredentialStoreStatus() (available bool, reason string, err error)
```

No method returns a secret value. `secrets.Provider.Get` is internal and not bound.

**Deliberately absent**, each prohibited by Constitution II (research R10): `LockVault`,
`UnlockVault`, `SetVaultStorage`, `ExportVault`, `ImportVault`, `RevealSecret`, and any
auto-lock or master-password method. Their absence is asserted by contract C11.

## Trust

```go
ListTrustDecisions() ([]TrustDecision, error)
RevokeTrustDecision(fingerprint string) error
DecideTrust(host string, port int, fingerprint string, scope TrustScope) error
```

`DecideTrust` is called only in response to a `trust:challenge` event, and only for the exact
fingerprint that event carried.

## Browsing

```go
ListChildren(profileID, dn string, page PageRequest) (EntryPage, error)
ReadEntry(profileID, dn string, opts ReadOptions) (Entry, Result, error)  // opts.IncludeOperational
RefreshEntry(profileID, dn string) (Entry, Result, error)
CountSubtree(profileID, dn string) (JobID, error)          // for delete confirmation
GoToDN(profileID, dn string) (Entry, Result, error)
```

`EntryPage` carries `entries`, `cookie`, `loadedCount`, `serverLimit`, and `truncatedByServer` —
the last two exist so the UI can state a truncation rather than silently show a short list
(FR-018, FR-037).

## Search

```go
ValidateFilter(filter string) (FilterDiagnostic, error)    // local only, never contacts the server
StartSearch(profileID string, s SearchDefinition) (JobID, error)
FetchNextPage(jobID string) (EntryPage, error)

ListSavedSearches() ([]SavedSearch, error)
SaveSearch(s SavedSearch) (SavedSearch, error)
DeleteSavedSearch(id string) error
SearchHistory(profileID string) ([]SearchRecord, error)

ListBookmarks(profileID string) ([]Bookmark, error)
SaveBookmark(b Bookmark) (Bookmark, error)
DeleteBookmark(id string) error
```

`SearchDefinition.Filter` is a **string** and is transmitted byte-identically. `ValidateFilter`
parses a copy to report `{ok, position, message}`; it never returns a rewritten filter, and there
is no method that does (FR-019, FR-028).

## Schema

```go
ReadSchema(profileID string) (Schema, Result, error)
SchemaAvailable(profileID string) (bool, reason string, error)   // schema-less mode (FR-064)
CompareSchemas(leftProfileID, rightProfileID string) (JobID, error)

ListSchemaProjects() ([]SchemaProjectSummary, error)
CreateSchemaProject(name string, src ProjectSource) (SchemaProject, error)
SaveSchemaElement(projectID string, e SchemaElement) (SchemaProject, error)
DeleteSchemaElement(projectID, oid string) (SchemaProject, error)
CheckSchemaProject(projectID string) ([]Problem, error)
CompareSchemaProject(projectID string, against ProjectOrProfile) (CompareResult, error)
ExportSchemaProject(projectID, path string, format SchemaFormat) (JobID, error)
PrepareSchemaCommit(projectID, profileID string) (ChangeSetPreview, error)   // → Commit(token)
```

`PrepareSchemaCommit` returns a preview token, so a schema commit goes through the same
confirmation as any other write (FR-071). It refuses to issue a token while the project holds any
`Error`-severity problem.

## Interchange

```go
StartExport(profileID string, req ExportRequest) (JobID, error)   // LDIF, DSML, CSV, JSON, XLSX, ODS
StartImport(profileID string, req ImportRequest) (JobID, error)   // LDIF, DSML
PrepareImport(profileID string, req ImportRequest) (ImportPlan, error)  // counts by type, → token
ValidateLDIF(text string) ([]LDIFDiagnostic, error)               // line-numbered, local only
CopyToClipboard(sel Selection, as ClipboardFormat) (string, error) // DN | LDIF | delimited
```

Export and import stream inside Go. **No bulk entry data crosses the bridge** — only progress
events and a final report path (see R7 in [../research.md](../research.md)).

## Values

```go
ReadValue(profileID, dn, attr string, idx int) (ValueBlob, error)
DecodeValue(v ValueBlob, as ValueKind) (DecodedValue, error)   // cert, image, time, postal, ACI…
EncodeValue(d DecodedValue) (ValueBlob, error)                 // must round-trip byte-identically
LoadValueFromFile(path string) (ValueBlob, error)
SaveValueToFile(v ValueBlob, path string) error
HashPassword(plain string, scheme HashScheme) (ValueBlob, error)
VerifyPassword(profileID, dn, candidate string) (bool, Result, error)
PasswordModify(profileID, dn string, old, new string) (ChangeSetPreview, error)  // → Commit(token)
SupportsPasswordModify(profileID string) (bool, error)
```

`EncodeValue(DecodeValue(v)) == v` byte-for-byte for every unmodified value, across every
`ValueKind`. That property is a test, not a comment (data-model §5).

`HashPassword` and `VerifyPassword` never log, persist, or echo the plaintext (FR-077).

## Access control

```go
ParseACI(raw string) (ACIItem, error)              // ParseError is data, not a failure
RenderACI(item ACIItem) (string, error)            // must round-trip unmodified input
ParseSubtreeSpec(raw string) (SubtreeSpec, error)
PreviewSubtreeSpec(profileID string, spec SubtreeSpec) (JobID, error)   // the DNs it selects
SupportsStoredACI(profileID string) (bool, reason string, error)
```

## Comparison, history, bulk

```go
StartCompare(req CompareRequest) (JobID, error)
Reconcile(compareID string, direction Direction) (ChangeSetPreview, error)  // → Commit(token)

History(profileID string, filter HistoryFilter) ([]HistoryRecord, error)
ExportHistoryRecord(id, path string) error
PrepareReplay(id, targetProfileID string) (ChangeSetPreview, error)         // → Commit(token)
PrepareReversal(id string) (ChangeSetPreview, error)                        // → Commit(token)
ReversalAvailable(id string) (bool, reason string, error)

PrepareBulk(profileID string, req BulkRequest) (ChangeSetPreview, error)
StartBulk(token string, mode JobMode) (JobID, error)   // mode: Execute | DryRun
ExportJobReport(jobID, path string) error
```

`PrepareReversal` returning `(false, reason)` from `ReversalAvailable` is the contract behind
FR-092: the API states that a reversal is impossible rather than producing a partial one.

`StartBulk` takes a token, so the bulk path cannot skip the preview either.

## Jobs

```go
Cancel(jobID string) error         // effective within 2s (SC-003)
JobState(jobID string) (Job, error)
ListJobs() ([]JobSummary, error)
```

## Referrals

Research R11 — a referral resolves to a connection, not a socket.

```go
ResolveReferral(url string) (ReferralTarget, error)   // matching profile, if any
FollowReferral(url, profileID string, remember bool) (Entry, Result, error)
ParseLDAPURL(url string) (LDAPURL, error)             // RFC 4516, for Go-to-DN (gap G6)
```

`FollowReferral` requires an explicit `profileID`. There is no method that binds the current
credentials against a host the user has not chosen — the security rule in R11 is enforced by the
signature.

## Effective rights *(spec gap G1 — requires a functional requirement before build)*

```go
SupportsEffectiveRights(profileID string) (bool, reason string, error)
CheckEffectiveRights(profileID, dn, asSubjectDN string) (EffectiveRights, Result, error)
```

## Commands and keybindings

Research R15 — one registry, rendered as menus, shortcuts, context menus, and toolbar.

```go
ListCommands() ([]Command, error)              // id, label, menu placement, enablement, scope
GetKeymap() (Keymap, error)
SetBinding(commandID, chord string) (conflicts []Conflict, err error)
ResetKeymap(preset KeymapPreset) error
ExportKeymap(path string) error
```

`SetBinding` returns conflicts rather than silently overwriting, and refuses OS-reserved chords.

## Preferences and workspace

```go
GetPreferences() (Preferences, error)
SetPreferences(p Preferences) error         // includes value-editor syntax→editor mapping (5c)
PurgeCache(profileID string) error          // one action, incl. orphaned data (FR-100)
ListTemplates() ([]EntryTemplate, error)
SaveTemplate(t EntryTemplate) (EntryTemplate, error)
```

---

## Contract tests

Each of these is a failing-test-first gate before the corresponding story is implemented
(Constitution V):

| # | Assertion | Serves |
|---|-----------|--------|
| C1 | `ldapx` mutation functions have exactly one calling package, `changeset` | SC-005 |
| C2 | No package outside `secrets` imports a platform credential API | Constitution II |
| C3 | `SaveProfile` rejects every payload carrying secret material | SC-008 |
| C4 | `Commit` rejects an absent, reused, expired, or stale token | FR-039, FR-052 |
| C5 | `ValidateFilter` never returns a modified filter string | FR-019 |
| C6 | `EncodeValue(DecodeValue(v)) == v` over the fidelity corpus | SC-007 |
| C7 | Every method contacting a server returns a populated `Result` on success and failure | SC-006 |
| C8 | Every `JobID`-returning method has a `Cancel` that takes effect within 2 s | SC-003 |
| C9 | No bridge method returns a secret value | Constitution II |
| C10 | `RenderACI(ParseACI(raw)) == raw` for unmodified input | FR-084 |
| C11 | No vault method exists on the bridge — asserted by name over the bound surface | Constitution II, R10 |
| C12 | `FollowReferral` cannot be called without an explicit target profile | R11 |
| C13 | A read-only profile is refused a preview token; a production-tagged one demands the extra confirmation | R20 |
| C14 | Every command in the registry has an enablement predicate and a non-conflicting default binding per platform | FR-102, R15 |
