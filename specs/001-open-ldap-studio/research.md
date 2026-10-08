# Phase 0 Research: Open LDAP Studio

**Date**: 2026-08-31 | **Plan**: [plan.md](./plan.md) | **Spec**: [spec.md](./spec.md)

Resolves the constitution's `TODO(TECH_STACK)` and every unknown in the plan's Technical Context.
Version facts were verified against the Go module proxy and, where a claim mattered, against the
library source in the local module cache on 2026-08-31.

---

## R1. Desktop framework — Wails v2 vs Wails v3

**Decision**: `github.com/wailsapp/wails/v2 v2.15.0`.

**Rationale**: The user specified Go with Wails, so the question is only which line. v2.15.0 is the
current stable release (2026-08-17). v3 is at `v3.0.0-beta.16` (2026-08-29) — actively developed
but still beta, and this product is a 13-story build against a 108-requirement spec where an API
break mid-flight is expensive. v2 covers everything the wireframe needs except one item.

**The exception, verified**: Wails v2 is single-window. Its runtime package exposes only singleton
`Window*` functions (`WindowSetTitle`, `WindowShow`, …) and no `NewWindow`. Wireframe screen `3b`
specifies Credentials Management "as its own window", which v2 cannot do. Recorded as deviation D4;
the credentials manager becomes an in-app modal surface in v1.

**Alternatives considered**:
- *Wails v3 beta* — would deliver the separate window and better native menus, but a beta dependency
  for a multi-month build risks churn the schedule cannot absorb. **Revisit trigger**: if v3 reaches
  stable before the P4 stories begin, re-evaluate; the bridge layer is deliberately thin to keep
  that migration cheap.
- *Fyne / Gio (pure Go)* — no HTML/CSS, so the wireframe's dense IDE shell, CodeMirror-based source
  editors, and virtualised 100k-row grids would all be hand-built. Rejected on effort.
- *Electron / Tauri* — Tauri means Rust, Electron means Node; both discard the Go requirement.

---

## R2. LDAP client library

**Decision**: `github.com/go-ldap/ldap/v3 v3.4.14`, extended in-house where it falls short.

**Rationale**: The only maintained pure-Go LDAP v3 client. Pure Go matters because it keeps
cross-compilation simple and avoids an OpenLDAP C dependency on three platforms. Verified present
in v3.4.14 by inspecting the module source:

| Capability | Requirement | Status in go-ldap v3.4.14 |
|-----------|-------------|---------------------------|
| Paged results (`1.2.840.113556.1.4.319`) | FR-018, FR-022 | `ControlPaging` — present |
| Server-side sorting (`…4.473`/`.474`) | FR-034 | `ControlServerSideSorting` + `…Result` — present |
| Subtree delete (`…4.805`) | FR-046 | `ControlSubtreeDelete` — present |
| ManageDsaIT | FR-022 | `ControlManageDsaIT` — present |
| StartTLS / LDAPS | FR-006, FR-009 | present |
| SASL EXTERNAL | FR-002 | `ExternalBind` — present |
| GSSAPI | FR-002 | `GSSAPIBind` + gokrb5 client — present |
| DIGEST-MD5 | FR-002 | present in `bind.go` |
| Password Modify (RFC 3062) | FR-076 | `PasswordModify` — present |
| Who Am I (RFC 4532) | FR-011 | `whoami.go` — present |
| Modify DN with new superior | FR-048, FR-049 | `moddn.go` — present |

**Gaps that must be built** (verified absent):

1. **CRAM-MD5 bind** — no implementation. FR-002 requires it. Small, well-specified
   (RFC 2195); implement in `ldapx/bind.go` over the existing SASL exchange.
2. **VLV control** — no `ControlVLV` type; the OID is not even registered. FR-022 requires virtual
   list views. Implement as a BER-encoded control in `ldapx/controls/`.

**Alternatives considered**: cgo bindings to OpenLDAP's `libldap` would give both for free, but drag
a C toolchain and platform libraries into every build and cross-build, and the constitution's
non-blocking requirement is far easier to honour with a Go-native client and `context`.

---

## R3. Credential storage — the constitution's hardest constraint

**Decision**: an in-house `secrets.Provider` interface with three native, in-process bindings:

| Platform | Binding | Mechanism |
|----------|---------|-----------|
| Linux / BSD | `github.com/godbus/dbus/v5 v5.2.2` | freedesktop Secret Service over the session bus |
| macOS | `github.com/keybase/go-keychain v0.0.1` | cgo → Security.framework |
| Windows | `github.com/danieljoos/wincred v1.2.3` | syscall → `wincred.dll` |

**Rationale**: Constitution II mandates "an in-process library binding" and states the application
"MUST NOT shell out to a password-manager CLI on the default path". That single sentence eliminates
the obvious dependency.

**Alternative rejected, with evidence**: `zalando/go-keyring v0.2.8` is the popular choice and would
be one dependency instead of three. Its darwin implementation imports `os/exec` and an internal
`shellescape` package to drive `/usr/bin/security` — verified by reading
`keyring_darwin.go` in the module cache. That is exactly the subprocess-inside-the-security-boundary
pattern the constitution names as the common cause of a GUI session never receiving an unlock
prompt. Rejected. `99designs/keyring` was rejected for the same reason: its backend set includes
CLI-driven providers in the same package, so the prohibited path would be one config flag away.

**Maintenance note**: `keybase/go-keychain` last tagged v0.0.1 (2025-02-27). It is the de facto
standard Go binding for Security.framework and is not archived, so it satisfies the constitution's
"maintained dependency" bar — but it is the thinnest-maintained item in the tree and belongs on the
dependency watch list the constitution requires.

**Design consequence**: the D-Bus, Keychain, and wincred calls all block; every one runs inside the
job registry so a locked agent cannot freeze the UI (Gate IV). A locked or absent agent surfaces as
a recoverable error and falls through to the session-only in-memory path (FR-005).

---

## R4. LDIF — build, do not borrow

**Decision**: implement an RFC 2849 reader/writer in `internal/ldif/`.

**Rationale**: SC-007 requires a byte-for-byte round trip over a corpus that includes binary values,
non-UTF-8 bytes, empty values, folded long lines, and attribute description options. That is a
fidelity contract, not a convenience. `github.com/go-ldap/ldif` is available but sits at a
`v0.0.0-20260814080914-…` pseudo-version with no tagged release and no stated guarantee about
folding policy, base64 thresholds, or option preservation. Depending on it would put the single
most unrecoverable failure mode in the product — silently mangled export data — behind an untagged
third-party module.

The format is small and completely specified. Writing it is a day or two; proving it with the
corpus is the real work either way, and that work is identical whichever path is chosen.

**Alternatives considered**: `go-ldap/ldif` as a starting point to fork — worth reading for its
parser structure, but not worth a dependency edge.

---

## R5. DSML

**Decision**: implement over `encoding/xml` in `internal/dsml/`.

**Rationale**: No maintained Go DSML v2 library exists. The subset the spec needs (FR-057) is entry
and change records, which maps cleanly onto the same internal model LDIF already uses. Scoped to
DSML v2 request/response documents; DSML v1 is not offered.

---

## R6. Frontend stack

**Decision**: React 19 + TypeScript + Vite, with CodeMirror 6, TanStack Virtual, and Zustand.

**Rationale**, component by component:

- **CodeMirror 6** for the LDIF editor, filter editor, and schema source tabs. FR-055 requires
  syntax highlighting and per-line error reporting with line numbers, and the wireframe's `1e`
  specifies gutter diagnostics with quick fixes — that is a gutter-and-lint-source API, which
  CodeMirror provides directly. Monaco was rejected as roughly an order of magnitude larger for a
  feature set aimed at IDE-scale editing this product does not need.
- **TanStack Virtual** for the DIT tree and result grid. SC-004 requires a 100,000-child container
  to start rendering within 2 s and scroll without stutter; that is unreachable without windowing.
- **Zustand** for client state — the interesting state lives in Go, and the frontend mostly mirrors
  it. Redux's ceremony buys nothing here.
- **Design tokens over a component library**. The wireframe is a bespoke dark IDE shell; MUI or
  similar would be fought rather than used. FR-102 also requires honouring the OS light/dark
  setting, which is a token-swap problem.

---

## R7. Concurrency, cancellation, and the IPC boundary

**Decision**: a `jobs.Registry`. Every server-touching call becomes a job with an id, a
`context.Context`, a progress channel, and a mode (`execute` | `dryRun`).

**Rationale**: Wails v2 binds Go methods as promises, which gives no cancellation channel of its
own. FR-099 requires every server operation to be cancellable and SC-003 bounds cancellation at
2 s, so the handle has to be explicit: the frontend calls `StartX(...) → jobID`, subscribes to
progress events, and can call `Cancel(jobID)`. The same registry supplies the dry-run mode bulk
operations need (FR-047) by swapping the dispatch call for a recorder while leaving the full
validation path intact — which is what makes SC-014's "dry run matches real run" claim testable.

**Bulk data never crosses IPC.** A 50,000-entry export streams from `ldapx` to disk inside Go, with
only progress events crossing to the frontend. Marshalling that volume through the webview bridge
would breach the 100 ms budget on its own.

**Connection concurrency**: `go-ldap`'s `Conn` multiplexes by message id, but a single connection is
still a serialisation point under load and, more importantly, a cancelled operation must not
disturb a sibling. One connection per profile with a small pool for parallel work (subtree copy,
comparison), each pool member independently cancellable.

---

## R8. Test strategy and container matrix

**Decision**: `testcontainers-go v0.44.0` driving three server fixtures.

| Fixture | Purpose | Covers |
|---------|---------|--------|
| **OpenLDAP** | Baseline, as the constitution requires | All core stories; `cn=config` gives both schema modification (US11) and configuration-as-entries (US12) |
| **ApacheDS** | X.500 authorisation model | US10 — `prescriptiveACI`, subentries, administrative roles, which OpenLDAP does not express |
| **Capability-poor profile** | Degradation paths | FR-038, FR-047, FR-064, FR-085, SC-016 — no paging, no readable subschema, no extended operations |

**Rationale**: The spec's Dependencies section already requires a server with stored access-control
rules and one without, precisely so the "explain, don't fail" behaviour is exercised rather than
assumed. Docker 29.7.2 is available locally, so this runs on developer machines as well as CI.

Constitution V requires the failing test first for every protocol-facing change, and forbids mocking
the client library. The `ldapx` boundary is therefore never mocked in integration tests; mocks are
confined to pure logic (filter validation, LDIF encoding decisions, diff computation).

---

## R9. Packaging and platform prerequisites

**Decision**: `wails build` per platform; no cross-compilation of the cgo paths.

- **Linux**: webkit2gtk 4.1 at build and run time; ship AppImage plus `.deb`.
- **macOS**: 12+, `.app` bundle, signed and notarised; cgo required for both webview and Keychain.
- **Windows**: 10+, WebView2 runtime (evergreen bootstrapper), `.exe` plus NSIS installer.

**Rationale**: cgo on the macOS Keychain and Linux webkit paths means each platform builds on its
own runner. FR-108 requires a migration path for on-disk format changes, so profile, history, and
schema-project files each carry a schema version from the first release — cheap now, impossible to
retrofit.

---

## Resolved constitution TODO

`TODO(TECH_STACK)` in `.specify/memory/constitution.md` is answered by R1–R3: **Go 1.26 + Wails
v2.15.0**, LDAP via **go-ldap/ldap v3.4.14**, credential store via **native per-platform bindings**
behind a single provider interface, supported matrix **Linux / macOS 12+ / Windows 10+**.

Per the constitution's governance rules this warrants an amendment to the "Security and Data
Protection Standards" section recording the binding stack — a MINOR bump to 1.4.0. That amendment
is a separate pull request against the constitution and is **not** performed by this plan.

---

# Phase 0 Research, second pass

Added after a full read of the wireframe (all 27 screens, not the six sampled on the first pass).
The design surfaced requirements the first research round never considered.

---

## R10. The credential model — replacing the vault

**Decision**: there is no vault. A `Credential` is a **named reference** to an entry in the platform
credential service, plus non-secret metadata, and connection profiles point at it.

**Why this is a redesign rather than a tweak.** Wireframe `3b`, pane `5f`, and the Credentials menu
together specify: storage choice of "OS keychain **or encrypted file · master password**",
"auto-lock after 15 min idle", "Lock vault", "**Reveal**" on a stored password,
"**Import / export vault…**", and a "remember" option for password prompts. Constitution II
prohibits five of those outright:

| Design element | Constitutional text |
|----------------|--------------------|
| "encrypted file · master password" storage | "in an application-managed keyfile or embedded vault, or behind an in-application master password is prohibited without exception" |
| Master password to unlock | "the application MUST never receive, display, or process the master passphrase" |
| Auto-lock timer / "Lock vault" | "Unlock lifetime and caching policy belong to the agent; the application MUST NOT extend, re-prompt around, or cache secrets beyond the agent's TTL" |
| "Import / export vault…" | Writing secrets to an application-owned file |
| Password prompt → "remember" | Application-side caching beyond the agent's TTL |

**What survives, and it is most of the screen.** The valuable idea in `3b` is not the vault — it is
that **a credential is defined once and assigned to many connections**, so revoking it invalidates
every connection that used it. That is a genuine improvement over per-profile passwords and it
needs no vault at all:

```
Credential { ID, Name, Type, BindDN, SecretRef, Realm?, KeytabPath?, ClientCertRef? }
Profile.CredentialID → Credential.ID → SecretRef → platform credential service
```

The list, the type grouping, per-credential detail, **Test bind**, the use log, password age, and
rotation reminders all remain. Removed: the storage selector, the master password, the lock/unlock
controls and countdown, vault import/export, and `Reveal`.

**On `Reveal`**: not prohibited by the letter of the constitution — the application must retrieve a
bind secret to use it — but it turns a shoulder-surf into a credential compromise and serves no
workflow that `Test bind` does not serve better. **Recommendation: drop it.** If it is wanted, it
must require a fresh platform-agent authorisation for each reveal, never a session grant.

**Consequence for the data model**: `Profile.SecretRef string` was wrong. It becomes
`Profile.CredentialID`, with `Credential` as a first-class entity. This is corrected in
`data-model.md`.

---

## R11. Referral chasing needs its own connection and possibly its own credentials

**Decision**: a referral target is resolved to a **connection**, not a socket. Following a referral
either reuses an existing profile that matches the target, creates one, or asks — and the choice can
be remembered for the session.

**Rationale**: dialog `6e` specifies exactly this ("ref: `ldap://dc02.corp.local/ou=people…`" →
pick `corp-stage` · create new connection… · remember for session), and it is correct: a referral
commonly points at a different host, which needs its own TLS trust decision and its own bind. A
client that silently rebinds with the current credentials on another host is leaking those
credentials to a server the user never chose.

**Design**: `ldapx.ReferralResolver` maps an LDAP URL to a profile. Policy per FR-021 —
`follow` uses the resolver, `ignore` returns the referral as data, `ask` raises a chooser. Traversal
depth is bounded and loops are reported with the DNs involved (spec edge case §Directory shape).

**Security rule that is not in the spec and should be**: credentials are never automatically reused
against a host the user has not explicitly bound them to.

---

## R12. Effective rights (design gap G1)

**Decision**: implement using the Get Effective Rights control, behind a capability check.

**Rationale**: screen `2b` shows "as `cn=lchen` → `userPassword`: write ✓ / as anonymous → mail:
read ✗ — uses the *Get effective rights* control". This answers the question ACI editing exists to
serve, and without it the ACI editor can only show what a rule *says*, never what it *does*.

The control is not in `go-ldap` and is not uniformly specified across servers (OpenLDAP
`1.3.6.1.4.1.42.2.27.9.5.2`; the IETF draft `draft-ietf-ldapext-acl-model` differs). It is therefore
implemented as a custom control with per-server support detection, and its absence degrades to
`capability:unavailable` like any other optional capability.

**Scope note**: this is spec gap G1. It needs a functional requirement before it is built.

---

## R13. LDIF formatting preferences vs. byte fidelity

**Decision**: separate **value fidelity** from **document layout**, explicitly, in both code and UI.

**Why it matters**: pane `5d` offers "wrap lines at 78", "fold base64 at 76", "space after colon",
"blank line between records", and an encoding/EOL choice. Read carelessly, those look like they
contradict SC-007's byte-for-byte round trip.

They do not, and the distinction has to be stated or someone will implement it wrong:

- **Value fidelity is absolute.** The bytes of every attribute value survive export → import
  unchanged. Non-negotiable, tested against the corpus.
- **Document layout is user-configurable.** Line wrapping, folding column, and separator style
  change how the same values are *rendered* in the file. Two LDIF files with different wrapping are
  semantically identical, and re-importing either yields identical values.

The round-trip test asserts **value equality after re-import**, not file equality. Asserting file
equality would be both wrong and untestable across layout settings.

**Corollary**: base64 is applied when RFC 2849 *requires* it (leading/trailing whitespace, non-ASCII,
leading colon or less-than), never as a formatting preference. The "fold base64 at 76" setting
controls the fold column, not whether base64 is used.

---

## R14. Wails IPC — what may and may not cross

**Decision**: a hard rule — **entry data crosses in bounded pages; bulk data never crosses at all.**

**Rationale**: Wails v2 marshals bound-method arguments and returns as JSON through the webview
bridge. A 100,000-entry result set or a 50,000-entry export would breach the 100 ms budget in
serialisation alone, before rendering.

| Path | Mechanism |
|------|-----------|
| Tree children, search results | Paged: `ListChildren`/`FetchNextPage` return one page (default 100, per `5a`) |
| Export / import | Streams to and from disk **inside Go**; only `job:progress` events cross |
| Large attribute values | Summarised (`binary · 24 KB`); full bytes fetched only on explicit action (FR-026, and `3a`'s truncation threshold) |
| Progress events | Throttled to ≤ 20/s so the event stream cannot itself block the UI |

Binary values cross as base64 with an explicit `isBinary` flag, never coerced to UTF-8 — a JSON
bridge would otherwise silently replace invalid sequences and break SC-007 at the boundary.

---

## R15. Command registry and keybindings (design gap G4)

**Decision**: every user-invocable action is registered in a command registry with a stable id,
a default binding, and an enablement predicate. Menus, the keyboard layer, context menus, and the
toolbar are all views over that registry.

**Rationale**: FR-102 requires full keyboard operability for all primary flows, and screen `5g`
specifies searchable commands, keymap presets, live conflict detection, and export. Screen `4a`
notes that "commands enable and disable by selection context: entry commands need a tree selection,
schema commands need an open project, batch operations need a search or DN list" — that is an
enablement predicate per command, which only works if commands are first-class.

Building menus and shortcuts separately guarantees they drift. One registry, four renderers.

**Reserved-binding awareness**: `5g` flags OS-reserved shortcuts live, which matters because the
design's defaults (⌘K, ⌘R, ⌘⇧C) differ per platform.

---

## R16. Undo and redo (design gap D8)

**Decision**: `Edit › Undo/Redo` is **editor-local only** — text edits in the LDIF editor, pending
attribute edits in the entry editor before commit. It never reverses a dispatched operation.

**Rationale**: screen `4a` places Undo/Redo directly above "Copy entry / Paste entry / Delete", which
reads as entry-level undo. Entry-level undo cannot be reconciled with Principle I: an undo that
silently dispatches a compensating write is exactly the auto-committing side effect the constitution
forbids.

Reversal of a committed change already exists and is correct — it goes through history (US8,
FR-092), producing a reversing LDIF that is previewed and confirmed like any other write. The menu
must make the two distinguishable: **Undo** (local, free, instant) and **Reverse…** (a server write,
previewed).

---

## R17. Logging, redaction, and rotation

**Decision**: structured logs with redaction applied at write time, rotating at 10 MB × 3 files as
the design specifies (`6f`: "logs: 2 files · 4.2 MB rotate at 10 MB × 3").

Four sinks, matching the wireframe's bottom panel:

| Sink | Contents | Retention |
|------|----------|-----------|
| Modification log | Every write as an LDIF record plus its result | Rotating, per profile |
| Search log | Request parameters and response summary | Rotating, per profile |
| Error log | Failures with full `Result` | Rotating |
| Console | Diagnostic/debug output | Session only |

Redaction is a property of the writer, not of the caller — no call site can forget it. This is what
test X7 pins down.

**Crash output** is included in the SC-008 scan: a panic dump containing a bind password would fail
the criterion just as a log file would.

---

## R18. Accessibility and theming (design gap G2)

**Decision**: adopt the design's high-contrast theme and colourblind-safe diff palette.

**Rationale**: FR-102 commits only to honouring the OS light/dark setting, but `3a` already
specifies "Dark / Light / High contrast / Follow system" and a "colourblind-safe pair" for diff
highlights. Diff colour is not decorative here — it is how comparison results and attribute changes
are read, so encoding meaning in red/green alone would make US9 unusable for a substantial minority
of users. Cheap now; a re-theme later.

Screen-reader certification stays a post-v1 goal per the spec's Assumptions, but semantic markup and
focus management are done as the components are built, since retrofitting them is far more expensive.

---

## R19. CI, quality gates, and supply chain

**Decision**: the constitution's Development Workflow section is encoded as CI jobs, not left to
reviewer memory.

| Constitutional requirement | Mechanism |
|---------------------------|-----------|
| "Dependencies MUST be pinned with a committed lockfile" | `go.sum` + `package-lock.json`, both committed; `go mod verify` in CI |
| "A dependency vulnerability scan MUST run in CI on every pull request. Known high-severity advisories block merge" | `govulncheck` for Go, `npm audit --audit-level=high` for the frontend; both non-advisory gates |
| "Container-backed LDAP integration tests" must pass before merge | testcontainers-go job with the three fixtures (R8) |
| "Any change to a mutation path, a credential path, or TLS configuration requires review by a second maintainer. Self-approval is not permitted" | `CODEOWNERS` covering `internal/changeset/`, `internal/secrets/`, `internal/trust/`, and TLS code in `internal/ldapx/`, with required reviews |
| "tests proving that no secret reaches disk, logs, or crash output" | The SC-008 scan job (contract F3, E1, X7) |
| "the default path performs no subprocess invocation" | A test asserting no `os/exec` import in the `secrets` transitive tree |
| "Releases MUST follow semantic versioning… migration path" | Release workflow + the `schemaVersion` contract in `file-formats.md` |

**Build matrix**: three OS runners, because cgo (macOS Keychain, Linux webkit) rules out
cross-compiling the platform paths. Containers run on the Linux runner only; macOS and Windows
runners cover build, unit tests, and the platform credential-store tests, which cannot be faked.

**Signing**: macOS codesign + notarisation, Windows Authenticode. Both need secrets in CI and both
should be wired before the first release, not after — an unsigned build gets quarantined on both
platforms and creates a bad first impression that is hard to undo.

---

## R20. Read-only mode and production tagging (design gap G3)

**Decision**: adopt both from pane `5e` ("New connections: open read-only · warn on production tag").

**Rationale**: this is the cheapest safety mechanism in the entire design and it is not in the spec.
A profile marked read-only refuses to issue a preview token at all, so the guarantee holds at the
`changeset` boundary rather than in the UI. A profile tagged *production* adds a distinct visual
treatment and an extra confirmation step on every mutation.

Principle I exists because "an accidental write can lock out an organization". The most common
version of that accident is a right command in the wrong window, and neither previews nor dry runs
address it — both assume the user meant this server. Read-only mode does.

**Scope note**: spec gap G3, needs a functional requirement.

---

## R21. Spreadsheet export

**Decision**: `github.com/xuri/excelize/v2 v2.11.0` for XLSX. **ODS is written in-house** as a
zipped flat OpenDocument spreadsheet.

**Rationale**: FR-058 requires spreadsheet workbooks and screen `6b` names both Excel and ODF.
excelize is the maintained Go XLSX library (v2.11.0, 2026-07-06) and supports streaming writes,
which matters because these exports share the constant-memory requirement of every other export
path (FR-060).

No maintained Go ODS **writer** exists — `knieriem/odf` was last released in 2019 and is a reader.
Since ODS content is a single XML document inside a zip, writing it directly is a smaller and safer
commitment than adopting a seven-year-old dependency for a security-sensitive product. Both writers
share the same row-streaming interface as the CSV writer, so the incremental cost is the XML
serialiser only.

**Alternative considered**: `tealeg/xlsx/v3` (v3.3.13, 2025-04) — viable, but excelize has the more
active release cadence and better streaming support.
