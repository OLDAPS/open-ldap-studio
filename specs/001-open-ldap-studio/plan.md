# Implementation Plan: Open LDAP Studio

**Branch**: `001-open-ldap-studio` | **Date**: 2026-08-31 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-open-ldap-studio/spec.md`

## Summary

A cross-platform desktop LDAP workbench delivering Apache Directory Studio parity plus a safety
and history layer the reference tool lacks. Built as a **Wails v2** application: a Go core that
owns every byte of LDAP protocol traffic, and a React/TypeScript frontend rendering the IDE-style
shell already wireframed in the Claude Design project *LDAP Desktop Application Wireframe*
(25 screens across 6 flows).

The architecture is shaped by three constitutional obligations that are not negotiable and that
drive most of the structure:

1. **No write reaches a server without a preview.** All mutations funnel through a single
   `ChangeSet` pipeline — build → preview → confirm → dispatch. The bridge exposes no method that
   writes without a confirmed preview token, so the guarantee is structural rather than a
   convention reviewers must police.
2. **The application never holds a secret of its own.** A `secrets` provider with three in-process
   platform bindings (Secret Service over D-Bus, macOS Security.framework, Windows Credential
   Manager) is the only code that touches a credential. No subprocess on the default path.
3. **Nothing blocks the UI.** Every server-touching call is a cancellable job in a registry, run
   on a goroutine with a `context.Context`, reporting progress over Wails events. Bulk data
   (exports, imports) streams to and from disk in Go and never crosses the IPC boundary.

Protocol coverage rests on `go-ldap/ldap/v3`, which supplies paging, server-side sorting, subtree
delete, ManageDsaIT, StartTLS, SASL EXTERNAL/DIGEST-MD5/GSSAPI, Password Modify and Who Am I.
Four gaps are filled in-house — CRAM-MD5, the VLV control, RFC 2849 LDIF, and DSML — because the
spec's byte-fidelity requirements exceed what the available libraries guarantee.

## Technical Context

**Language/Version**: Go 1.26 (toolchain go1.26.2 verified locally); TypeScript 5.9 on Node 24.12

**Primary Dependencies**:

- Desktop shell — `github.com/wailsapp/wails/v2 v2.15.0`
- LDAP protocol — `github.com/go-ldap/ldap/v3 v3.4.14`
- Kerberos/GSSAPI — `github.com/jcmturner/gokrb5/v8 v8.4.4` (via go-ldap's GSSAPI client)
- Credential store — `github.com/godbus/dbus/v5 v5.2.2` (Linux/BSD Secret Service),
  `github.com/keybase/go-keychain v0.0.1` (macOS), `github.com/danieljoos/wincred v1.2.3` (Windows)
- Spreadsheet export — `github.com/xuri/excelize/v2 v2.11.0` (XLSX); ODS written in-house (research R21)
- Test harness — `github.com/testcontainers/testcontainers-go v0.44.0`
- Frontend — React 19, Vite 7, CodeMirror 6 (LDIF/filter/schema source editors),
  TanStack Virtual (100k-row grids and trees), Zustand (client state)

**Storage**: Local files only — connection profiles (JSON), certificate trust store, operation
history (append-only JSONL plus reversal LDIF), schema projects, saved searches, bookmarks,
templates, per-profile caches. Secrets live exclusively in the OS credential service; profile
files carry a reference, never a secret.

**Testing**: `go test` for unit-level pure logic; container-backed integration tests via
testcontainers-go against **OpenLDAP** (baseline, `cn=config` for schema modification and
configuration-as-entries) and **ApacheDS** (X.500 `prescriptiveACI`, subentries, administrative
roles), plus a deliberately capability-poor server profile to exercise degradation paths.
Vitest + Testing Library for frontend units. End-to-end flows are asserted through the bridge API
against live containers, which is where the protocol behaviour actually lives.

**Target Platform**: Linux (webkit2gtk 4.1), macOS 12+, Windows 10+ (WebView2) — 64-bit desktop

**Project Type**: Desktop application — single binary embedding a Go core and a web frontend

**Performance Goals**: No UI block exceeding 100 ms (SC-002); first page of a 100,000-child
container visible within 2 s (SC-004); cancellation effective within 2 s (SC-003); 50,000-entry
LDIF export streams with progress and constant memory; 10,000-entry subtree comparison within
30 s (SC-013)

**Constraints**: Zero secret material on disk (SC-008); every mutation previewed and confirmed
(SC-005); server result codes preserved verbatim (SC-006); byte-for-byte LDIF round trip (SC-007);
no network destination other than user-configured LDAP servers; launch to usable window under 3 s
with no password prompt (SC-009)

**Scale/Scope**: 13 user stories, 108 functional requirements, 18 success criteria, ~25 screens
and 6 flows from the existing wireframe, 3 platforms

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

### Gate I — Safety-First Directory Mutations (NON-NEGOTIABLE)

| Check | Design response | Status |
|-------|-----------------|--------|
| Every mutation previews DN, attribute-level diff, affected count | Single `changeset` package; `Preview()` returns the diff and a token, `Commit(token)` is the only dispatch path | PASS |
| Explicit confirmation before dispatch | Bridge exposes no write method that accepts anything but a preview token issued in the same session | PASS |
| Dry run for bulk and recursive operations | `jobs.DryRun` mode runs the full validation path with the dispatch call replaced by a recorder | PASS |
| Non-leaf delete states subtree size, distinct confirmation | `changeset.SubtreeDelete` requires a counted enumeration before a token is issued | PASS |
| No auto-commit as a side effect of navigation/refresh | Editor state is frontend-local until an explicit commit; no bridge method mutates on read | PASS — **but see Deviation D3**, the wireframe currently offers auto-save |

### Gate II — Credential and Transport Security (NON-NEGOTIABLE)

| Check | Design response | Status |
|-------|-----------------|--------|
| Secrets delegated to platform secret service | `secrets.Provider` interface; three in-process platform bindings | PASS |
| No secret in application-owned files | Profiles store `secretRef` only; verified by an automated scan in CI (SC-008) | PASS |
| No subprocess on the default path | `zalando/go-keyring` **rejected** — its darwin path imports `os/exec` and shells to `/usr/bin/security` (verified in source). Native bindings used instead | PASS |
| No startup password prompt, no in-app master password | Unlock is deferred to first bind and raised by the platform agent | PASS — **but see Deviation D2**, the wireframe's security pane offers a master password |
| Usable with no credential store | Session-only in-memory credential path, tested per platform | PASS |
| TLS verification on by default, per-profile opt-out surfaced | `tlsPolicy` per profile, badge rendered whenever that profile is connected | PASS |
| Single provider interface, no other call site touches secrets | Enforced by package boundary and a CI import check | PASS |

### Gate III — Protocol Fidelity Over Abstraction

| Check | Design response | Status |
|-------|-----------------|--------|
| Raw DN, filter, LDIF editable as text | Filter and DN cross the bridge as strings, never as parsed structures | PASS |
| No silent rewrite/normalisation before transmission | Filters are validated by parsing a *copy*; the original string is what is sent | PASS |
| Operational attributes inspectable | Explicit `+` request; never stripped | PASS |
| Exact result code and diagnostic returned on success and failure | `ldapx.Result` carries `resultCode`, `matchedDN`, `diagnosticMessage`; the bridge never collapses them to a bool | PASS |
| LDIF (RFC 2849) as interchange format | In-house reader/writer for byte fidelity | PASS |
| Convenience editors expose an escape hatch | Every structured editor is paired with a raw tab, enforced by a shared editor contract | PASS |

### Gate IV — Non-Blocking, Transparent Desktop UX

| Check | Design response | Status |
|-------|-----------------|--------|
| All network I/O off the UI thread | Go core runs on goroutines; the webview only renders | PASS |
| Every server operation cancellable | `jobs.Registry` keyed by job id; `Cancel(id)` cancels the context | PASS |
| Configurable timeouts | Per-profile connect/read timeouts, defaulted globally | PASS |
| Paged searches, server limits surfaced | Paging control by default; `sizeLimitExceeded` reported, never swallowed | PASS |
| Connection state, bind DN, target server always visible | Status bar bound to a connection-state event stream | PASS |
| Errors shown verbatim alongside interpretation | `ldapx.Result` rendered raw, interpretation is an additive field | PASS |

### Gate V — Test-First Against Real Directories (NON-NEGOTIABLE)

| Check | Design response | Status |
|-------|-----------------|--------|
| Tests written and failing before implementation | Enforced per task in `/speckit-tasks`; every protocol task pairs with a preceding test task | PASS |
| Integration tests against a real containerised server | testcontainers-go; OpenLDAP baseline plus ApacheDS for X.500 ACI and subentries | PASS |
| Mocks only for pure logic | The `ldapx` boundary is never mocked in integration tests | PASS |
| Required coverage: binds, paging, referrals, each mutation type, LDIF round trip | Mapped in `quickstart.md`; each is a gate before its story is complete | PASS |

**Result: all five gates PASS.** Three deviations between the existing wireframe and the
specification are recorded below; each is resolved in favour of the specification and the
constitution, and each requires a wireframe change rather than a code concession.

### Design review — second pass

The first pass read six of the wireframe's screens and sampled the rest by label. This pass read
all 27. The full inventory, with every screen mapped to its story, requirements, and bridge
methods, is in [contracts/screens.md](./contracts/screens.md). Two things came out of it: a set of
deviations three times larger than first recorded, and seven capabilities the design promises that
the spec does not require.

#### Deviations — the design must change, not the code

| # | Screens | Conflict | Resolution |
|---|---------|----------|------------|
| **D1** | `2c`, `4a`, activity rail | Local server instance management vs. spec "Out of Scope for v1" | **Dropped.** The LDAP Servers perspective, its rail icon, and `File › New › Server instance…` are all removed. Rail carries six icons, not seven |
| **D2** | `3b`, `5f`, Credentials menu | **The entire vault concept.** Five separate prohibitions — see below | **Replaced.** Credentials become named references with assignments; the vault is deleted |
| **D3** | `1b`, `5b`, flow 7b | "auto-save on focus loss" vs. Principle I and FR-039 | **Auto-save removed.** Commit always previews; the preference becomes preview *detail*, not whether to preview |
| **D4** | `3b` | Separate window vs. Wails v2 single-window (verified) | In-app modal surface in v1; revisit on Wails v3 |
| **D5** | `1a`, `5e` | "Provider: JNDI ▾ · Apache Directory API" | **Removed.** A Java/Eclipse artefact of the reference tool with no meaning in a Go client — there is one protocol implementation |
| **D6** | `2e`, `6e` | Conflict policy: design says ask/overwrite/skip/**rename**, FR-034 says skip/overwrite/**merge** | Implement the **union** of both; neither source is wrong, each is incomplete |
| **D7** | `3c` | "On bind failure → retry anonymously" | **Removed** as an automatic action. A silent privilege downgrade that makes a failed bind look like a successful one |
| **D8** | `4a` | `Edit › Undo/Redo` sitting above entry commands, implying entry-level undo | **Scoped to editor-local edits only.** Reversing a dispatched write goes through history (US8) and is previewed like any other write. The menu must distinguish **Undo** from **Reverse…** (research R16) |
| **D9** | `5h` | Update channel with start-up checks, plus an extension list | Update checks **default off** per FR-016; the extension list is dropped with the plugin runtime |

#### D2 in full — why the credentials area is a redesign, not an edit

The wireframe's credential surfaces (`3b`, `5f`, and the Credentials menu) specify a vault:
storage choice of "OS keychain **or encrypted file · master password**", "auto-lock after 15 min
idle", "Lock vault", a **Reveal** action on a stored password, **Import / export vault…**, and a
"remember" option on password prompts.

Constitution II prohibits five of those outright:

| Design element | Constitutional text |
|----------------|--------------------|
| "encrypted file · master password" storage | "in an application-managed keyfile or embedded vault, or behind an in-application master password is prohibited without exception" |
| Master password to unlock | "the application MUST never receive, display, or process the master passphrase" |
| Auto-lock timer, Lock vault | "Unlock lifetime and caching policy belong to the agent; the application MUST NOT extend, re-prompt around, or cache secrets beyond the agent's TTL" |
| Import / export vault | Writing secret material to an application-owned file |
| Password prompt → "remember" | Application-side caching beyond the agent's TTL |

**The good idea in `3b` survives intact.** What makes that screen worth having is not the vault —
it is that a credential is defined once and **assigned to many connections**, so revoking it
invalidates every connection that used it. That needs no vault: a `Credential` is a name, a type, a
bind DN, and a reference into the platform store. The list, type grouping, per-credential detail,
**Test bind**, use log, password age, and rotation reminders all stay. Removed: the storage
selector, master password, lock controls and countdown, vault import/export, and `Reveal`.

`Reveal` is not prohibited by the letter of the constitution — the app must retrieve a bind secret
to use it — but it converts a shoulder-surf into a credential compromise and serves no workflow
that `Test bind` does not serve better. **Recommendation: drop it.** If it is wanted, each reveal
must require a fresh platform-agent authorisation, never a session grant.

This correction propagated into the data model: `Profile.SecretRef` was wrong and is now
`Profile.CredentialID` pointing at a first-class `Credential` (data-model §1.1, research R10).

#### Capabilities the design promises that the spec does not require

None conflicts with the constitution and all are useful, but **scope is the spec's to set**, so
none is silently absorbed. Each needs a requirement via `/speckit-clarify` or an explicit deferral.

| # | Capability | Screen | Recommendation |
|---|-----------|--------|----------------|
| **G1** | Effective-rights check via the Get Effective Rights control | `2b` | **Adopt.** It answers "can this user actually do that" — the question ACI editing exists to serve. Without it the editor shows what a rule *says*, never what it *does* |
| **G2** | High-contrast theme, colourblind-safe diff palette | `3a` | **Adopt.** Diff colour is how comparison results are read; red/green alone makes US9 unusable for a substantial minority |
| **G3** | Per-connection read-only mode and "production" tagging | `5e` | **Adopt.** The cheapest safety mechanism in the design — see below |
| **G4** | Command registry with rebindable keymaps and conflict detection | `5g` | **Adopt.** FR-102 already implies a command registry; the rebinding UI is the increment |
| **G5** | Batch throttling (batch size + inter-batch pause) | `6c` | **Adopt throttling**; **defer** batch modify-DN and batch compare, which exceed FR-096's "single change" |
| **G6** | Go to DN accepts an LDAP URL (RFC 4516) | `6e` | **Adopt.** Trivial, and LDAP URLs are how referrals identify entries |
| **G7** | OID editor resolving dotted-decimal to registered name | `6f` | **Adopt.** Extends FR-072's editor list by one |

**G3 deserves emphasis.** Principle I exists because "an accidental write can lock out an
organization", and the most common form of that accident is the right command in the wrong window.
Neither previews nor dry runs help — both assume the user meant *this* server. Read-only mode is
the only mechanism here that does, and it costs a boolean: a read-only profile is refused a preview
token at the `changeset` boundary, so the guarantee holds below the UI.

## Project Structure

### Documentation (this feature)

```text
specs/001-open-ldap-studio/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   ├── bridge-api.md    #   Go↔frontend bound methods
│   ├── events.md        #   Runtime event stream
│   ├── error-model.md   #   Result codes, error taxonomy, redaction
│   ├── file-formats.md  #   On-disk formats and their migration contract
│   └── screens.md       #   Wireframe inventory → stories → requirements
├── checklists/
│   └── requirements.md  # Spec quality checklist (/speckit-specify output)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
main.go                      # Wails bootstrap, window, application menu
wails.json

internal/
├── ldapx/                   # The only package that speaks LDAP
│   ├── conn.go              #   dial, StartTLS/LDAPS, keep-alive, reconnect policy
│   ├── bind.go              #   simple, anonymous, EXTERNAL, GSSAPI, DIGEST-MD5, CRAM-MD5 (in-house)
│   ├── search.go            #   paged search, alias/referral policy, bounded traversal
│   ├── controls/            #   VLV (in-house), sort, paging, subtree delete, ManageDsaIT
│   ├── modify.go            #   add, modify, modrdn, delete — dispatch only, never called directly by the bridge
│   ├── extended.go          #   Password Modify, Who Am I
│   ├── rootdse.go
│   └── result.go            #   verbatim result code + diagnostic, never collapsed
├── changeset/               # Build → preview → confirm → dispatch. The only write path.
│   ├── changeset.go
│   ├── preview.go           #   attribute-level diff, affected count, token issue
│   └── reverse.go           #   reversing LDIF from captured before-state
├── ldif/                    # RFC 2849 reader/writer — byte-fidelity critical
├── dsml/
├── schema/
│   ├── subschema.go         #   parse a server's published schema
│   ├── project.go           #   offline schema project model
│   └── problems.go          #   dangling refs, duplicate OIDs, circular superiors
├── aci/                     # ACI item + subtree specification parse/render, raw-preserving
├── credentials/             # Credential entity + profile assignments (research R10)
├── secrets/                 # The ONLY package that touches a credential
│   ├── provider.go          #   interface; no other call site may import a platform file
│   ├── secretservice_linux.go
│   ├── keychain_darwin.go
│   ├── wincred_windows.go
│   └── session.go           #   in-memory fallback when no store is reachable
├── profiles/                # Profile + folder store; credentialID only, never a secret
├── trust/                   # Certificate trust store, bound to exact certificates
├── history/                 # Append-only operation + search log, redaction at write time
├── compare/                 # Entry, subtree, and schema diff → reconciling LDIF
├── jobs/                    # Cancellable job registry, progress, dry-run, throttling, pause
├── exportx/                 # Streaming CSV/JSON/XLSX/ODS writers
├── commands/                # Command registry: menus, keymaps, enablement (research R15)
├── prefs/                   # Preference model and persistence (panes 3a, 5a–5h)
├── logging/                 # Four sinks, redaction at write time, rotation (research R17)
└── bridge/                  # Wails-bound API — the contract in contracts/bridge-api.md

frontend/
├── src/
│   ├── shell/               # Window chrome, activity rail, menus, status bar (wireframe 1b)
│   ├── views/
│   │   ├── connections/     #   1a, 3c — connection list, wizard
│   │   ├── browser/         #   1b — DIT tree + entry editor
│   │   ├── search/          #   1c — filter builder, result grid
│   │   ├── schema/          #   1d read-only browser, 6a project perspective
│   │   ├── ldif/            #   1e — LDIF editor with dry-run diff
│   │   ├── credentials/     #   3b — modal surface (deviation D4)
│   │   ├── compare/         #   2e — diff, copy/move, bulk
│   │   ├── aci/             #   2b — ACI item + subtree specification
│   │   └── logs/            #   6f — progress, modification and search logs
│   ├── commands/            # Menu bar, keymap layer, context menus over one registry
│   ├── editors/             # 2d, 5c, 6f — value editors, each with a mandatory raw tab
│   ├── dialogs/             # 6e — filter, go-to-DN, rename, move, referral, cert trust
│   ├── preferences/         # 3a, 5a–5f — preference panes
│   ├── bridge/              # Generated Wails bindings + typed wrappers
│   └── i18n/                # Externalised strings (FR-107)
└── tests/

test/
├── integration/             # Container-backed, per user story
├── containers/              # OpenLDAP, ApacheDS, capability-poor fixtures
└── corpus/                  # Binary, non-UTF-8, empty, option-bearing, multi-MB values
```

**Structure Decision**: A single Go module with a web frontend, matching the Wails v2 layout. The
`internal/` split is not arbitrary layering — three of its boundaries are constitutional
enforcement points: `secrets/` is the only package importing a platform credential API,
`changeset/` is the only path to a write, and `ldapx/` is the only package that speaks the
protocol. CI enforces those import boundaries, which turns three review-time obligations into
build-time ones. `bridge/` is deliberately thin: it translates and delegates, holding no logic
that a test would otherwise have to reach through the UI to exercise.

## Delivery Milestones

Stories are independently shippable, but the enforcement machinery is not optional groundwork.
M0 exists because C1, C2, and C13 are cheap to satisfy on day one and expensive to retrofit — each
is architectural rather than behavioural.

| Milestone | Contents | Exit criteria |
|-----------|----------|---------------|
| **M0 — Skeleton and guarantees** | Wails shell, command registry, `jobs`, `secrets` provider (3 platforms), `ldapx` connect/bind, `changeset` preview pipeline, CI with all gates | Contracts C1, C2, C3, C11, C13 pass. No feature yet — but a write with no preview is now inexpressible |
| **M1 — Read-only product (P1)** | US1, US2 · screens `1a`, `1b`, `1c`, `3c`, `5a`, `5e`, `6e` (Go-to-DN, filter, cert trust, referral) | Shippable: connect, browse, search, export nothing. SC-001, SC-004, SC-016 |
| **M2 — Writes and interchange (P2)** | US3, US4 · `2a`, `1e`, `6b`, `6d`, `5b`, `5d` | SC-005, SC-006, SC-007. First release that can damage a directory — the preview pipeline is load-bearing from here |
| **M3 — Fluency (P3)** | US5, US6, US7 · `1d`, `2d`, `5c`, `6f` value editors, `2e` copy/move | SC-011 parity for daily work |
| **M4 — Safety net (P4)** | US8, US9, US10 · `2b`, `2e` compare, logs | SC-012, SC-013. The "and more" half of the product |
| **M5 — Specialist (P5)** | US11, US12, US13 · `6a`, `6c`, config editing | SC-014, full parity per the appendix |
| **M6 — Release** | Signing, notarisation, installers, migrations, accessibility pass | SC-009, SC-018; the Parity Reference reaches 100% resolution |

Deviations D1–D9 are wireframe changes and should land in the design project **before** the
milestone that builds the affected screen — D2 and D3 before M0 and M2 respectively, since both
change what gets built rather than how it looks.

## Risk Register

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| R-1 | **`keybase/go-keychain` stagnates** — last tag v0.0.1, Feb 2025, and it is the thinnest-maintained item in the tree | Medium | High — macOS credential path | On the dependency watch list the constitution requires. The `secrets.Provider` interface means replacing it touches one file. Fallback: a direct cgo binding to Security.framework |
| R-2 | **Wails v2 constrains the design** — single window today (D4); v3 is beta | Medium | Medium | Bridge layer kept deliberately thin so a v3 migration is cheap. Re-evaluate if v3 goes stable before M4 |
| R-3 | **In-house LDIF fidelity bugs** — the corpus is the only thing standing between us and silent data mangling | Medium | **Very high** — SC-007 failures are unrecoverable for users | Corpus-driven tests written *before* the writer. Fuzz the reader against the writer. Treat any fidelity bug as a release blocker |
| R-4 | **Server behaviour diverges from spec** — the constitution's stated reason for demanding real servers | High | Medium | Three container fixtures including a capability-poor one; every degradation path tested, not assumed |
| R-5 | **100 ms budget breached by IPC**, not by the network | Medium | High — SC-002 | Paged crossings only, bulk streams inside Go, progress events throttled to 20/s (research R14). Instrument the event loop in CI |
| R-6 | **ApacheDS fixture is the only X.500 ACI source** — if it proves hard to containerise, US10 loses its test bed | Medium | Medium | Validate the fixture during M0, long before M4 needs it. Fallback: 389 Directory Server for a second ACI model |
| R-7 | **Scope creep via design gaps** — seven capabilities beyond the spec, each individually reasonable | High | Medium | G1–G7 are recorded, not absorbed. Nothing is built until it has a requirement |
| R-8 | **Signing and notarisation deferred to the end** | Medium | Medium | Wire the release pipeline during M1 with a throwaway build. An unsigned binary is quarantined on both macOS and Windows |
| R-9 | **GSSAPI differs per platform** (SSPI on Windows, MIT/Heimdal elsewhere) and is hard to test in CI | High | Low–Medium | Kerberos fixture is a stretch goal; manual verification per platform before M6, with the limitation documented if it does not land |

## CI and Quality Gates

The constitution's Development Workflow section is encoded as jobs, not left to reviewer memory
(research R19).

| Gate | Job | Blocks merge |
|------|-----|--------------|
| Unit tests | `go test ./...` + Vitest | Yes |
| Container-backed integration | `go test -tags=integration` against all three fixtures | Yes |
| Dependency vulnerability scan | `govulncheck` + `npm audit --audit-level=high` | Yes — high severity blocks, per the constitution |
| Lockfiles pinned | `go mod verify`, committed `go.sum` and `package-lock.json` | Yes |
| **No secrets on disk** | The SC-008 scan over every written file, log, and crash artefact (F3, E1, X7) | Yes |
| **No subprocess in the credential path** | Assert no `os/exec` in the `secrets` transitive import tree | Yes |
| **Import boundaries** | C1 (`ldapx` mutations have one caller) and C2 (`secrets` is the only platform-credential importer) | Yes |
| Second-maintainer review | `CODEOWNERS` on `internal/changeset/`, `internal/secrets/`, `internal/trust/`, TLS code in `internal/ldapx/` | Yes — self-approval not permitted on these paths |
| Build matrix | Linux, macOS, Windows runners (cgo rules out cross-compiling the platform paths) | Yes |

## Complexity Tracking

> No constitutional violations require justification — all five gates pass. The table records
> deliberate build-versus-borrow decisions where the simpler path (take the dependency) was
> rejected, since those are the places a reviewer would reasonably ask why.

| Decision | Why needed | Simpler alternative rejected because |
|----------|------------|--------------------------------------|
| In-house RFC 2849 LDIF reader/writer | SC-007 demands byte-for-byte round trip of binary, non-UTF-8, empty, folded, and option-bearing values | `go-ldap/ldif` is at a `v0.0.0-` pseudo-version with no fidelity guarantee for folding, base64 policy, or attribute options — the one format where a silent mangle is unrecoverable |
| In-house `secrets` provider over three native bindings | Constitution II mandates in-process platform bindings and forbids subprocess on the default path | `zalando/go-keyring` shells out to `/usr/bin/security` on macOS (verified in its source); `99designs/keyring` carries CLI-backed backends in the same package |
| In-house VLV control and CRAM-MD5 bind | FR-002 and FR-022 require both; go-ldap ships neither | No maintained Go library provides them; both are small, well-specified, and directly testable against a live server |
| Separate `changeset` package rather than preview logic inside each editor | Makes SC-005 ("no mutation path lacks a preview") checkable by construction | Per-editor previews would make the guarantee a convention, and SC-005 requires an automated check finding zero exceptions |
| Job registry rather than plain goroutines | FR-099 requires every server operation to be cancellable and SC-003 bounds it at 2 s | Ad-hoc goroutines give no handle to cancel, no progress channel, and no place to enforce the dry-run mode bulk operations need |

## Post-Design Constitution Re-Check

*Re-evaluated after Phase 1 artefacts (data-model.md, contracts/, quickstart.md) were produced.*

The design work strengthened three gates from "we will do this" to "the API cannot express the
violation", which is the outcome worth recording:

| Gate | Before design | After design | Evidence |
|------|---------------|--------------|----------|
| **I. Safety-first mutations** | Intent to preview every write | The bridge has **no** `Modify`/`Add`/`Delete`/`Rename` method. `Commit(token)` is the only dispatch path, and `PreviewToken` is single-use, expiring, and carries `EntryVersions` for the concurrency guard | contracts/bridge-api.md Rule 1; contract C1; data-model §3 |
| **II. Credential security** | Intent to use the platform store | `Secret` has no marshaller, `Profile` has no field capable of holding one, and no bound method returns a secret value. Import-boundary test C2 makes the provider a compile-time boundary | contracts C2, C3, C9; F3; E1 |
| **III. Protocol fidelity** | Intent to keep raw access | Attribute values are `[]byte` end to end; `ValidateFilter` parses a copy; `Result` is returned on success as well as failure; `RenderACI(ParseACI(x)) == x` is a test | contracts C5, C6, C7, C10; data-model §2 cross-cutting invariants |
| **IV. Non-blocking UX** | Intent to run off-thread | Every long call returns a `JobID` with a matching `Cancel`; `job:progress` is throttled so the event stream cannot itself breach the frame budget; bulk data never crosses IPC | contracts/events.md E2, E3; contract C8; research R7 |
| **V. Test-first** | Obligation acknowledged | 21 contract tests (C1–C10, E1–E5, F1–F6) are enumerated and assigned to the stories they gate, before any implementation task exists | quickstart.md; all three contract documents |

### What the second-pass design review changed

Reading all 27 wireframe screens rather than six did not weaken any gate, but it did find that
**Gate II was being violated by the design in five places, not one**. The first pass recorded a
master-password field; the full read found an entire vault — encrypted-file storage, an auto-lock
timer, vault import/export, a `Reveal` action, and a "remember the password" prompt option. All five
are now resolved by deletion (deviation D2), and the data model was corrected accordingly:
`Profile.SecretRef` became `Profile.CredentialID` pointing at a first-class `Credential`.

Two gates gained enforcement they did not have:

- **Gate I** gains read-only profiles (research R20, design gap G3). A read-only profile is refused
  a preview token at the `changeset` boundary — protection against the one accident previews and dry
  runs cannot catch, the right command in the wrong window.
- **Gate III** gains the error model (`contracts/error-model.md`): a category taxonomy, a table of
  result codes with mandated handling, and the `Indeterminate` state for a write interrupted by a
  lost connection — neither success nor failure, which is the honest answer and the one most clients
  get wrong.

**No new violations were introduced by the design.** The Complexity Tracking table below is
unchanged: it records build-versus-borrow decisions, not constitutional exceptions.

**One governance action falls out of this plan and is deliberately not performed here**: the
constitution's `TODO(TECH_STACK)` is now answered (research R1–R3), so
`.specify/memory/constitution.md` should be amended to record the binding stack — Go 1.26 + Wails
v2.15.0, go-ldap/ldap v3.4.14, and the three native credential bindings — as a MINOR bump to 1.4.0.
Per the constitution's own Governance section that amendment must arrive as its own pull request
with its rationale, so it is out of scope for `/speckit-plan`.
