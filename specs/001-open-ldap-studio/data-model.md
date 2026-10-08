# Phase 1 Data Model: Open LDAP Studio

**Date**: 2026-08-31 | **Plan**: [plan.md](./plan.md) | **Spec**: [spec.md](./spec.md)

Derived from the spec's Key Entities. Go types live in the package named in each heading; the
wire shape crossing the Wails bridge is in [contracts/bridge-api.md](./contracts/bridge-api.md),
and the on-disk shape in [contracts/file-formats.md](./contracts/file-formats.md).

Types are described in Go-ish shorthand. Field notes carry the validation rule and the requirement
it serves; a rule without a requirement reference is an implementation invariant.

---

## 1. Connections and security

### `profiles.Profile`

| Field | Type | Rules |
|-------|------|-------|
| `ID` | `string` (UUID) | Stable across renames |
| `Name` | `string` | Non-empty, unique within its folder |
| `FolderID` | `string` | Empty = root; must not form a cycle |
| `Host`, `Port` | `string`, `int` | Port 1–65535, defaults 389/636 by encryption |
| `Encryption` | `enum{None, StartTLS, LDAPS}` | Never silently downgraded (FR-009) |
| `TLS` | `TLSPolicy` | See below |
| `BindMethod` | `enum{Anonymous, Simple, External, GSSAPI, DigestMD5, CramMD5}` | FR-002 |
| `BindDN` | `string` | Raw text, never normalised (FR-020) |
| `CredentialID` | `string` | Points at a `Credential` (§1.1), which holds the `SecretRef`. **Never a secret.** A profile containing secret material is a bug CI fails on (FR-003, SC-008). *Corrected in the second pass — see research R10* |
| `ReadOnly` | `bool` | Refuses to issue a preview token at all (research R20) |
| `Tags` | `[]string` | `production` triggers distinct visual treatment and an extra confirmation (R20) |
| `Timeouts` | `{ConnectMS, ReadMS int}` | > 0; per-profile override of the global default (FR-099) |
| `Limits` | `{SizeLimit, TimeLimit, PageSize int}` | 0 = server default |
| `Aliases` | `enum{Never, Search, Find, Always}` | FR-021 |
| `Referrals` | `enum{Follow, Ignore, Ask}` | FR-021 |
| `SchemaVersion` | `int` | Migration contract (FR-108) |

**Invariant**: `Profile` has no field capable of holding a secret. The type is the enforcement
point — there is nowhere to put one.

### `profiles.TLSPolicy`

| Field | Type | Rules |
|-------|------|-------|
| `VerifyCertificate` | `bool` | Defaults **true**; false requires explicit opt-in and is surfaced in the UI whenever connected (FR-006) |
| `VerifyHostname` | `bool` | Defaults **true** |
| `ClientCertRef` | `string` | For SASL EXTERNAL |

### `trust.Decision`

| Field | Type | Rules |
|-------|------|-------|
| `Host`, `Port` | `string`, `int` | |
| `Fingerprint` | `string` (SHA-256) | **The identity of the decision.** A decision never applies to a different certificate (FR-008) |
| `Scope` | `enum{Session, Permanent}` | Session decisions are never persisted |
| `Chain` | `[]byte` (DER) | Retained so the user can review what they accepted |
| `AcceptedAt` | `time.Time` | |
| `Reason` | `string` | The validation failure that was overridden |

**State transition**: `untrusted → (user decision) → trusted(fingerprint)`. A presented certificate
whose fingerprint differs from the stored one returns to `untrusted` and re-raises — it is never
auto-accepted (US1 scenario 5).

### `secrets.Provider` (interface, not data)

```
Get(ref string) (Secret, error)
Set(ref string, s Secret) error
Delete(ref string) error
Available() (bool, reason string)
```

`Secret` holds bytes in memory only, is zeroed on disconnect and exit, and has no marshaller — it
cannot be serialised by accident (FR-003, FR-077).

---

## 2. Directory data

### `ldapx.Entry`

| Field | Type | Rules |
|-------|------|-------|
| `DN` | `string` | **Exactly as returned by the server.** Never re-encoded or normalised (FR-019, III) |
| `Attributes` | `[]Attribute` | Order as received |
| `HasChildren` | `tri-state{Yes,No,Unknown}` | `Unknown` until determined; drives lazy expansion without a false leaf |

### `ldapx.Attribute`

| Field | Type | Rules |
|-------|------|-------|
| `Type` | `string` | Base attribute description |
| `Options` | `[]string` | `binary`, language tags. **Values differing only by option are distinct attributes** (FR-045) |
| `Values` | `[][]byte` | **Bytes, not strings.** Non-UTF-8 must survive intact (SC-007) |
| `IsOperational` | `bool` | From schema where available; else unknown, and the attribute is still shown (FR-027) |

**Rule**: nothing converts `Values` to `string` outside a presentation layer that can round-trip
back to the same bytes. This is the single most consequential invariant in the model — SC-007
depends on it and so does every export path.

### `ldapx.Result`

| Field | Type | Rules |
|-------|------|-------|
| `Code` | `int` | The server's numeric result code, always populated |
| `MatchedDN` | `string` | Verbatim |
| `DiagnosticMessage` | `string` | **Verbatim, never replaced** (FR-051, SC-006) |
| `Interpretation` | `string` | Additive plain-language text; never a substitute |
| `Referrals` | `[]string` | |
| `Controls` | `[]Control` | Response controls, including sort and paging results |

**Invariant**: `Result` is returned on success as well as failure. The bridge has no method that
collapses it to a boolean.

---

## 3. The write path

### `changeset.ChangeSet`

The only representation of a pending mutation. Nothing else may reach `ldapx.Modify`.

| Field | Type | Rules |
|-------|------|-------|
| `ID` | `string` | |
| `ProfileID` | `string` | The target server, displayed on every preview (FR-010) |
| `Kind` | `enum{Add, Modify, ModRDN, Delete, SubtreeDelete, Copy, Move, BulkModify, SchemaCommit, ConfigModify}` | `ConfigModify` and `SchemaCommit` carry extra warnings (FR-087) |
| `Ops` | `[]Operation` | |
| `AffectedCount` | `int` | Enumerated, not estimated, before a token is issued (FR-039) |
| `BeforeState` | `[]Entry` | Captured for reversal; absent ⇒ not reversible, stated as such (FR-092) |
| `Warnings` | `[]Warning` | Schema conflicts are warnings the user may override, never silent blocks (FR-063) |

### `changeset.Operation`

| Field | Type | Rules |
|-------|------|-------|
| `DN` | `string` | Raw |
| `Type` | `enum{AddAttr, DeleteAttr, ReplaceAttr, AddValue, DeleteValue, AddEntry, DeleteEntry, Rename}` | |
| `Attribute` | `string` + options | |
| `Before`, `After` | `[][]byte` | The attribute-level diff the preview renders (FR-039) |
| `KeepOldRDN` | `bool` | `Rename` only; explicitly chosen, never defaulted silently (FR-048) |

### `changeset.PreviewToken`

| Field | Type | Rules |
|-------|------|-------|
| `Token` | `string` (opaque, single-use) | Issued by `Preview`, consumed by `Commit` |
| `ChangeSetID` | `string` | |
| `EntryVersions` | `map[DN]string` | Concurrency guard — re-read at commit; a mismatch aborts and forces re-confirmation (FR-052) |
| `IssuedAt` | `time.Time` | Expires; an expired token forces a fresh preview |

**State machine**: `draft → previewed(token) → committed | cancelled | stale`.
Cancelling discards the token but **keeps the draft** (FR-040). `stale` is reached when
`EntryVersions` no longer match at commit time.

**This type is the mechanism behind SC-005.** `Commit` accepts nothing but a token, so a write with
no preview is not expressible in the API.

---

## 4. Schema

### `schema.Element`

| Field | Type | Rules |
|-------|------|-------|
| `Kind` | `enum{ObjectClass, AttributeType, Syntax, MatchingRule}` | |
| `OID` | `string` | Unique within a project; duplicates are a Problem (FR-069) |
| `Names` | `[]string` | |
| `Description`, `Superiors`, `Syntax`, `Equality`/`Ordering`/`Substring` | | |
| `Usage` | `enum{UserApplications, DirectoryOperation, DistributedOperation, DSAOperation}` | Determines `IsOperational` |
| `SingleValue`, `Obsolete`, `NoUserModification` | `bool` | Drives editor read-only state (FR-063) |
| `RawDefinition` | `string` | The definition text exactly as published or authored (FR-066) |
| `Origin` | `string` | Source schema file or server |

### `schema.Project`

| Field | Type | Rules |
|-------|------|-------|
| `ID`, `Name` | `string` | |
| `Source` | `enum{Empty, FromServer, FromFiles}` | |
| `Elements` | `[]Element` | |
| `Problems` | `[]Problem` | Recomputed on every edit |
| `SchemaVersion` | `int` | Migration contract |

### `schema.Problem`

| Field | Type | Rules |
|-------|------|-------|
| `Severity` | `enum{Error, Warning}` | **Errors block export and server commit; warnings do not** (wireframe `6a`, FR-069) |
| `Kind` | `enum{DanglingReference, DuplicateOID, DuplicateName, CircularSuperior, NoMatchingRuleForSyntax}` | |
| `Elements` | `[]string` | Every element involved, named (FR-069) |
| `Message` | `string` | |

---

## 5. Access control

### `aci.Item`

| Field | Type | Rules |
|-------|------|-------|
| `Raw` | `string` | **The source of truth.** The structured view is derived from it, never the reverse (FR-081, FR-084) |
| `Parsed` | `*ParsedACI` | `nil` when parsing failed |
| `ParseError` | `string` | Shown with the raw text; the value is never rewritten or discarded (FR-084) |
| `Attribute` | `enum{prescriptiveACI, entryACI, subentryACI}` | |

### `aci.SubtreeSpecification`

| Field | Type | Rules |
|-------|------|-------|
| `Raw` | `string` | Source of truth, as above |
| `Base`, `ChopBefore`, `ChopAfter`, `Min`, `Max` | | Structured view |
| `PreviewDNs` | `[]string` | The entries the specification selects, resolved on demand before saving (FR-082) |

**Rule shared with every structured editor**: round-tripping an unmodified value must produce
byte-identical output. A parse-render cycle that changes the text is a defect, not a normalisation.

---

## 6. History and comparison

### `history.Record`

| Field | Type | Rules |
|-------|------|-------|
| `ID`, `Timestamp` | | |
| `ProfileID`, `ServerIdentity`, `BindDN` | `string` | Who did what, where (FR-089) |
| `Request` | `string` (LDIF) | The operation **as transmitted** |
| `Result` | `ldapx.Result` \| `Indeterminate` | An interrupted operation is recorded as indeterminate — never as success or failure (FR-089) |
| `BeforeState` | `[]Entry` (optional) | Absent ⇒ `Reversible = false` |
| `Reversible` | `bool` | Computed at write time |

**Redaction happens at write time, not read time** (FR-093, SC-008): bind credentials, password
values, and attributes marked sensitive never enter the record. A redaction applied on display
would leave the secret on disk.

### `history.SearchRecord`

Request parameters and the server's response summary (FR-090). Same redaction rule.

### `compare.Result`

| Field | Type | Rules |
|-------|------|-------|
| `LeftOnly`, `RightOnly` | `[]DN` | |
| `Differences` | `[]AttributeDifference` | Per-attribute, per-value |
| `IgnoredAttributes` | `[]string` | User-nominated (FR-094) |
| `OperationalExcluded` | `bool` | |

`AttributeDifference` → reconciling LDIF via `compare.Reconcile()`, which produces a `ChangeSet`
and therefore inherits the preview requirement automatically (FR-095).

---

## 7. Jobs

### `jobs.Job`

| Field | Type | Rules |
|-------|------|-------|
| `ID` | `string` | The frontend's cancellation handle |
| `Kind` | `enum{Connect, Search, Export, Import, Copy, Compare, Bulk, SchemaCommit}` | |
| `Mode` | `enum{Execute, DryRun}` | **Dry run runs the full validation path** and replaces dispatch with a recorder (FR-047, SC-014) |
| `State` | `enum{Running, Succeeded, Failed, Cancelled, PartiallyComplete}` | `PartiallyComplete` is never reported as success (FR-050) |
| `Progress` | `{Done, Total int, Message string}` | Streamed as events |
| `Outcomes` | `[]EntryOutcome` | Per-entry: succeeded / failed with verbatim message / skipped (FR-097) |
| `ctx`, `cancel` | | Not serialised |

---

## 8. Workspace objects

| Type | Key fields | Notes |
|------|-----------|-------|
| `SavedSearch` | Base, Scope, Filter (raw string), Attributes, Limits, Aliases, Referrals, Controls | Runnable against **any** profile (FR-035) — so it stores no profile id |
| `Bookmark` | ProfileID, DN, Name | FR-024 |
| `EntryTemplate` | ObjectClasses, prefilled attributes | Shareable as a file (FR-044) |
| `Folder` | ID, Name, ParentID | Nestable, acyclic |

---

## Cross-cutting invariants

1. **Attribute values are `[]byte` end to end.** Any `string` conversion outside presentation is a
   defect (SC-007).
2. **Raw text is the source of truth** for DNs, filters, LDIF, ACIs, subtree specifications, and
   schema definitions. Structured views derive from it and never replace it (Principle III).
3. **`ldapx.Result` is never collapsed** to a boolean or a rewritten message (SC-006).
4. **Writes exist only as a `ChangeSet` with a consumed `PreviewToken`** (SC-005).
5. **Secrets exist only inside `secrets`**, only in memory, and have no marshaller (SC-008).
6. **Everything persisted carries a `SchemaVersion`** from day one (FR-108).

---

# Second-pass additions

Entities the first pass missed, found by reading all 27 wireframe screens. Rationale for each is in
`research.md` R10–R20.

## 1.1 `credentials.Credential` — first-class, replacing `Profile.SecretRef`

A credential is defined once and assigned to many profiles, so revoking it invalidates every
connection that used it (wireframe `3b`). It is a **reference plus metadata** — never a container
for a secret.

| Field | Type | Rules |
|-------|------|-------|
| `ID`, `Name` | `string` | |
| `Type` | `enum{Anonymous, Simple, DigestMD5, CramMD5, GSSAPI, Certificate}` | Groups the list in `3b` |
| `BindDN` | `string` | Raw text |
| `SecretRef` | `string` | Lookup key into the platform credential service. **The only pointer to a secret in the model** |
| `Realm`, `QoP` | `string` | SASL mechanisms that define them |
| `KeytabPath`, `ClientCertRef` | `string` | GSSAPI and certificate types |
| `CreatedAt`, `LastUsedAt`, `SecretAgeDays` | | Drives the rotation reminder in `3b` |
| `UseLog` | `[]{ts, profileID, result}` | Recent-use list; redacted like any log |

**Relationship**: `Profile.CredentialID → Credential.ID → SecretRef → platform store`.
Deleting a credential leaves referring profiles in a `credential missing` state that prompts per
session (FR-005) — it never silently falls back to anonymous (deviation D7).

**Explicitly absent** — each prohibited by Constitution II (research R10): a storage-backend
selector, a master password, lock/unlock state or an auto-lock timer, vault import/export, a
`Reveal` action, and any "remember the password" option.

## 5.1 `prefs.Preferences`

Panes `3a` and `5a`–`5h`. Persisted in `preferences.json`; every pane has Restore Defaults.

| Group | Fields |
|-------|--------|
| `Appearance` | `Theme{Dark,Light,HighContrast,FollowSystem}`, accent, LDIF token colours, `DiffPalette{default, colorblindSafe}`, interface + monospace font and size, line height, row density, zoom, `TruncateValuesAt int` |
| `BrowserTree` | `EntriesPerPage`, `FetchOnScroll`, `SizeLimit`, `TimeLimit`, `Aliases`, `Referrals`, `EntryLabel{RDN,FullDN,Attribute+name}`, `SortChildrenBy`, `ShowChildCount`, `ShowOperational`, `ShowSubentries`, `ExpandOnConnect` |
| `EntryEditor` | `DefaultTab`, `ConfirmOn[]`, `SchemaCheck{whileTyping,onSave,off}`, `AttributeNamesAs`, `GroupRowsBy`, `MultiValueFold int`, `CopyAs` — **no save-mode field** (deviation D3) |
| `ValueEditors` | `ByAttributeType map[attr]editor`, `BySyntaxOID map[oid]editor`, `UnknownSyntaxFallback`, `MaxInlineLength` (see §5.2) |
| `LDIFText` | `WrapAt`, `FoldBase64At`, `ShowLineNumbers/Whitespace/Folding/Minimap`, `SpaceAfterColon`, `BlankLineBetweenRecords`, `Encoding`, `EOL`, `ValidateWhen`, `Flag[]`, token colours — **layout only, never value fidelity** (research R13) |
| `Connections` | `ConnectTimeout`, `ResponseTimeout`, `KeepAlive`, `AutoReconnectTries`, `OnConnectionLoss`, `DefaultControls[]`, `ModifyStrategy{changedOnly,replaceEntry}`, `NewConnectionsReadOnly`, `WarnOnProductionTag` |
| `Security` | Trust-store management, `PlaintextBindPolicy{warn,blockUnlessStartTLS,allow}`, `RotationReminderDays`, `AuditRetentionDays`, `ClearSessionSecretsOnExit` — **no vault group** (R10) |
| `Keyboard` | `Preset{Default,EclipseStyle,Custom}`, `Bindings map[commandID]chord` (see §5.3) |
| `Updates` | `Channel`, `CheckFrequency` — **defaults to never; opt-in per FR-016** (deviation D9) |

## 5.2 `editors.ValueEditorMapping`

Pane `5c`. Resolution order, most specific first: **attribute type → syntax OID → fallback**.

| Field | Type | Rules |
|-------|------|-------|
| `Match` | `{attributeType}` or `{syntaxOID}` | |
| `Editor` | `enum{Text, MultiLineText, Password, Certificate, Image, DN, GeneralizedTime, Boolean, Integer, PostalAddress, ObjectClass, OID, Hex}` | 13 editors — the design's 11 plus hex (FR-073) and the OID editor (gap G7) |
| `Scope` | `enum{All, Connection}` | |

Every editor, whichever is resolved, exposes the raw tab (FR-074). The mapping decides which view
opens **first**, never which views exist.

## 5.3 `commands.Command`

Research R15. Menus, keybindings, context menus, and toolbar are views over this registry.

| Field | Type | Rules |
|-------|------|-------|
| `ID` | `string` | Stable; the key for user bindings |
| `Label`, `Menu`, `Group` | `string` | Placement in the `4a` menu map |
| `DefaultBinding` | `map[platform]chord` | Defaults differ per OS |
| `Enablement` | predicate | e.g. *needs a tree selection*, *needs an open project*, *needs a search or DN list* (`4a`) |
| `Scope` | `enum{Global, Browser, SearchEditor, LDIFEditor, SchemaEditor}` | Determines conflict domains |

Conflicts are detected within a scope and flagged live; OS-reserved chords are refused (`5g`).

## 7.1 `jobs.Job` — additions

| Field | Type | Rules |
|-------|------|-------|
| `BatchSize`, `PauseBetweenBatchesMS` | `int` | Server throttling (`6c`, gap G5) |
| `Paused` | `bool` | The design's operation queue supports pause (`2e`); resume continues at the next entry boundary |
| `RejectsPath` | `string` | Failed records as re-runnable LDIF (flow 7e) |
| `EstimatedDuration` | `time.Duration` | Shown on the batch preview page (`6c`) |

## 2.1 `ldapx.ReferralTarget`

Research R11 — a referral resolves to a *connection*, not a socket.

| Field | Type | Rules |
|-------|------|-------|
| `URL` | `string` | RFC 4516 LDAP URL as returned |
| `ResolvedProfileID` | `string` | Existing profile, or one created for the target |
| `RememberForSession` | `bool` | Dialog `6e` |
| `Depth` | `int` | Bounded; loops reported with the DNs involved |

**Security rule**: credentials are never automatically reused against a host the user has not
explicitly bound them to (R11).

## 6.1 `aci.EffectiveRights` *(spec gap G1 — needs a requirement before build)*

| Field | Type | Rules |
|-------|------|-------|
| `AsSubject` | `string` | DN, or anonymous |
| `Entries` | `[]{DN, attribute, rights}` | From the Get Effective Rights control |
| `Supported` | `bool` + reason | Absent support degrades via `capability:unavailable` |

## 8.1 `workspace.File`

The LDIF editor's file tree and outline (`1e`, `6f`): path, dirty flag, target profile, outline
(change records for LDIF, attributes for an entry).

---

## State machines

**Connection**: `disconnected → connecting → bound → idle ⇄ reconnecting → disconnected`.
`reconnecting` reached only after an idle disconnect; leaving it sets
`writesRequireConfirmation` (FR-015). A `TrustError` or `AuthError` returns to `disconnected` —
never to `bound`.

**ChangeSet**: `draft → previewed(token) → committed | cancelled | stale` (§3). A read-only or
production-tagged profile intercepts at `draft → previewed`: read-only refuses the token entirely,
production requires an additional confirmation (R20).

**Job**: `running ⇄ paused → succeeded | failed | cancelled | partiallyComplete`. Every terminal
state emits exactly one `job:finished` (test E3).

**Credential**: `defined → assigned(n profiles) → rotated | revoked`. `revoked` leaves referring
profiles in `credential missing`, which prompts per session and never falls back to anonymous.
