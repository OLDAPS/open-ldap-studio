# Contract: Screen Inventory

**Date**: 2026-08-31 | **Plan**: [../plan.md](../plan.md)
**Source**: Claude Design project *LDAP Desktop Application Wireframe* (`LDAP Studio Wireframes.dc.html`)

Every screen in the wireframe, mapped to the user story it serves, the requirements it satisfies,
and the bridge methods behind it. This is the frontend's work breakdown and the traceability link
between the design and the spec: a screen with no requirement is unspecified scope, and a
requirement with no screen has nowhere to live.

**Status column**: `Build` = implement as designed · `Adapt` = implement with a recorded deviation
· `Drop` = not built in v1 · `Gap` = the design promises behaviour the spec does not require yet.

---

## Shell

The application shell is common to every screen and is built once, before any view.

| Element | Design | Requirement |
|---------|--------|-------------|
| Menu bar | 8 top-level menus: File, Edit, Search, LDAP, Schema, Credentials, Preferences, Help. Window menu deliberately removed (`4a`) | FR-102 |
| Activity rail | 6 icons: ⇄ connections · ⊞ DIT browser · ⌕ searches · ◈ schema · ▤ LDIF & files · ⚙ preferences | — |
| Status bar | Connection state, bind DN, target server, entry counts, page N/M, active limits | **FR-010** — always visible, on every screen |
| Bottom panel | Progress · Modification Logs · Search Logs · Errors · Console | FR-089, FR-090 |
| Document tabs | Editors are tabs, not modals — searches and LDIF files are documents | — |

The rail carries **six** icons, not seven: the LDAP Servers perspective (`2c`) is dropped, and
`File › New › Server instance…` is removed from the File menu with it.

---

## Primary workspace (turn 1)

| ID | Screen | Story | Requirements | Bridge | Status |
|----|--------|-------|-------------|--------|--------|
| **1a** | Connections start screen + 4-step wizard (Network → Authentication → Browser options → Edit options) | US1 | FR-001, FR-006, FR-007, FR-012 | `ListProfiles`, `SaveProfile`, `TestConnection`, `RootDSE` | **Adapt** (D5: drop the JNDI/Apache-Directory-API "Provider" selector — it is a Java artefact with no meaning in a Go client) |
| **1b** | DIT browser + entry editor — the primary workspace. Lazy paged children with an explicit "fetch next 100 of 1 842…" node; tree filter box; Searches and Bookmarks nodes under each connection; entry editor tabs *Attributes / LDIF view / Table editor / Object class*; Entry-info side panel (structural + auxiliary classes, last modified, photo preview, "Show in schema browser") | US1, US3 | FR-017–FR-027, FR-041, FR-045 | `ListChildren`, `ReadEntry`, `Preview`, `Commit` | **Adapt** (D3: no auto-save) |
| **1c** | Search editor as a document tab + result grid. Filter builder and raw RFC 4515 field stay in sync; attribute palette drags from schema; filter history; saved searches; virtual-scroll grid with column picker, export, and bulk-modify entry point; selected-row preview | US2 | FR-028–FR-038 | `ValidateFilter`, `StartSearch`, `FetchNextPage`, `SaveSearch` | **Build** |
| **1d** | Schema browser — read-only. Object classes / attribute types / matching rules / syntaxes; class hierarchy diagram; raw definition pane; "Schema diff: dev ↔ prod" tab; **"Used by 1 842 entries · run as search"** | US5 | FR-061, FR-062, FR-065, FR-066 | `ReadSchema`, `CompareSchemas` | **Build** — the schema→search jump is a `SearchDefinition` handoff, no new backend |
| **1e** | LDIF editor — workspace file tree, Outline of change records, target-connection selector, validation gutter with quick fixes, **Diff vs. server** tab, dry-run panel with per-record projected outcome, Import report tab | US4 | FR-053–FR-056, FR-060 | `ValidateLDIF`, `PrepareImport`, `StartImport` | **Build** |

---

## Editing and data (turn 2)

| ID | Screen | Story | Requirements | Bridge | Status |
|----|--------|-------|-------------|--------|--------|
| **2a** | New entry wizard — method (scratch / from existing / template) → object classes with implied superiors → RDN builder (multi-valued RDN supported) → attributes → LDIF preview. Entry context menu: New, New from existing, New context entry, Paste, Copy DN, Rename, Move, Delete subtree | US3 | FR-042, FR-043, FR-044, FR-063 | `Preview`, `Commit`, `ListTemplates` | **Build** |
| **2b** | Access control — administrative points list; ACI item editor (identification tag, precedence, auth level, user classes, protected items with grant/deny matrix); Source tab (RFC 3672); subtree specification with base / chop-before / chop-after / min / max / specification filter and an in-out tree preview; **Effective rights check** | US10 | FR-081–FR-085 | `ParseACI`, `RenderACI`, `PreviewSubtreeSpec` | **Build** + **Gap G1** (effective rights — see below) |
| **2c** | Server management — create / start / stop local server instances, configuration editor, server log console | — | — | — | **Drop** (D1) |
| **2d** | Value editors — password (SSHA-512/SSHA/SHA/MD5/crypt, verify, bind-as-user), certificate (Info/Subject/Extensions/DER tabs, import, export DER/PEM), image, DN picker, generalized time with zone, object class, hex/base64/multi-line text | US6 | FR-072–FR-080 | `DecodeValue`, `EncodeValue`, `HashPassword`, `VerifyPassword` | **Build** |
| **2e** | Compare, copy/move, bulk — two-connection compare with ignore-operational and per-value diff; copy/move dialog with mode (move / copy / copy subtree) and conflict policy; **operation queue with pause**; batch entry points | US7, US9, US13 | FR-049, FR-094–FR-097 | `StartCompare`, `Reconcile`, `PrepareBulk` | **Adapt** (D6: conflict policy — design says ask/overwrite/skip/**rename**, spec FR-034 says skip/overwrite/**merge**; implement the union) |

---

## Preferences and credentials (turns 3, 5)

| ID | Screen | Story | Requirements | Status |
|----|--------|-------|-------------|--------|
| **3a** | Appearance — theme (Dark / Light / **High contrast** / Follow system), accent, LDIF syntax token colours, **colourblind-safe diff pair**, interface and monospace fonts, row density, zoom, value truncation threshold, live preview, settings export | US1 | FR-102, FR-026 | **Build** + **Gap G2** (high contrast and colourblind-safe palettes exceed FR-102's light/dark) |
| **3b** | Credentials Management — credential list by type, per-credential detail, server assignments, test bind, use log, password age | US1 | FR-003–FR-005 | **Adapt heavily** (D2 — see *Credential model* below; also D4: in-app modal, not a separate window) |
| **3c** | Connection wizard step 2 — pick a stored credential or enter inline; "secret held in OS keychain · never written to the connection file"; check-authentication with whoami; **clear-text warning strip** when a credential would cross an unencrypted transport | US1 | FR-003, FR-006, FR-011 | **Adapt** (D7: drop "retry anonymously" as an automatic bind-failure action — a silent privilege downgrade) |
| **4a** | Menu map — reference sheet of all 8 menus and their shortcuts | — | FR-102 | **Adapt** (remove `File › New › Server instance…`; resolve Edit › Undo/Redo per D8) |
| **5a** | Browser & tree — entries per page, fetch on scroll, size/time limits, alias and referral handling, entry label (RDN / full DN / chosen attribute), child sorting, show child count, operational attrs, subentries, expand-on-connect | US1 | FR-018, FR-021, FR-023 | **Build** |
| **5b** | Entry editor — default tab, **save mode**, confirm-before rules, schema check timing, attribute name display, row grouping, multi-value fold threshold, copy-as format | US3 | FR-026, FR-063 | **Adapt** (D3: "auto on focus loss" removed) |
| **5c** | Value editors — syntax OID → editor table, per-attribute-type overrides, fallback for unknown syntax, max inline length | US6 | FR-072, FR-073 | **Build** |
| **5d** | LDIF & text editors — wrap width, base64 fold width, whitespace and record folding display, encoding/EOL, validation timing, token colours | US4 | FR-055 | **Build** (see R15 — these are *output layout* settings and must not be confused with value fidelity) |
| **5e** | Connections & timeouts — connect/response timeouts, keep-alive, auto-reconnect, connection-loss policy, default controls, **modify-request strategy (changed attributes only vs replace whole entry)**, TLS defaults, **open read-only / warn on production tag** | US1 | FR-006, FR-015, FR-022, FR-099 | **Adapt** (D5: drop the provider selector) + **Gap G3** (read-only mode and production tagging are not in the spec but directly serve Principle I) |
| **5f** | Credentials & security | US1 | FR-003–FR-008 | **Adapt heavily** (D2) |
| **5g** | Keyboard shortcuts — searchable command table, keymap presets (Default / Eclipse-style / Custom), live conflict detection, recording field, export | US1 | FR-102 | **Build** + **Gap G4** (a command registry is implied by FR-102 but never specified) |
| **5h** | Updates & about — release channel, check frequency, proxy, version, **extension list** | — | FR-016 | **Adapt** (D9: update checks must default to off per FR-016; the extension list is dropped — a plugin runtime is out of scope) |

---

## Gap-fill screens (turn 6)

| ID | Screen | Story | Requirements | Status |
|----|--------|-------|-------------|--------|
| **6a** | Schema Editor perspective — Projects / Schema (flat or hierarchical) / Hierarchy (type, supertype, subtype) / Problems / Search views; per-element editor with Overview and Source tabs; merge from project; errors block export | US11 | FR-067–FR-071 | **Build** |
| **6b** | Import / export wizard shell — format on page 1 (LDIF, DSML, CSV, Excel, ODF, connections, schema project), source/target and options on page 2, background job with report | US4 | FR-053, FR-057–FR-060 | **Build** |
| **6c** | Batch Operation wizard — entry set (search result / selection / DN file) → operation (modify LDIF fragment, delete, modify DN, compare) → execution (continue on error, dry run, write LDIF log, **batch size and inter-batch pause to throttle the server**) with a 3-record preview and estimated run time | US13 | FR-096, FR-097, FR-047 | **Build** + **Gap G5** (throttling; also batch modify-DN and batch compare exceed FR-096's "single change") |
| **6d** | Properties dialogs — connection, entry, attribute, value, search, bookmark (Alt+Enter); read-only facts beside editable settings | US1, US3 | FR-019, FR-061 | **Build** |
| **6e** | Small dialogs — filter editor with content assist; **Go to DN (accepts an LDAP URL)**; rename with keep/delete old RDN; move with conflict policy; **referral connection chooser** (pick an existing connection, create a new one, remember for session); certificate trust with chain view and trust-once / trust-permanently / reject; **attribute wizard with description options (`;binary`, `;lang-de`, custom)** | US1, US2, US3, US7 | FR-007, FR-008, FR-020, FR-021, FR-029, FR-045, FR-048 | **Build** + **Gap G6** (LDAP URL parsing) + see R12 (referral credentials) |
| **6f** | Progress, logs, outline + remaining value editors — concurrent cancellable operations; modification and search logs as LDIF; outline for entry and LDIF; **address editor** (`$`-separated postalAddress), **OID editor** (resolves and validates dotted-decimal), **in-place grid text editor**, **hex editor**; log rotation (10 MB × 3) | US6, US8 | FR-072, FR-073, FR-089, FR-090, FR-099 | **Build** + **Gap G7** (OID editor is not in FR-072's list) + see R17 (log rotation) |

---

## Flows (turn 7)

Each flow is an end-to-end path with a documented trigger, precondition, success state, and failure
state. They are the acceptance-test skeletons for their stories.

| Flow | Path | Story | Terminal state |
|------|------|-------|----------------|
| **7a** | Connect & authenticate — connections → network → auth → cert trust → browser options → open | US1 | DIT root fetched |
| **7b** | Find an entry & change a value — browse *or* search → editor → value editor → commit → logged | US3 | Modification recorded in the log |
| **7c** | Entry lifecycle — create / duplicate / move / delete, with copy-as-background-job branches | US3, US7 | Tree branch refreshed, entry selected |
| **7d** | Search lifecycle — new search → filter → scope → grid → save → export | US2 | Reusable Search or Bookmark |
| **7e** | Bulk in & out — direction → format → options → validate/dry run → execute → report + rejects file | US4, US13 | Report plus re-runnable rejects LDIF |
| **7f** | Schema project — new → import → edit → hierarchy → resolve Problems → export/apply | US11 | Zero errors, server-acceptable export |

Flow 7e's "rejected records land in a sibling `.ldif` so the run can be repaired and repeated" is a
concrete requirement the spec only implies through FR-056; implement it as designed.

---

## Design capabilities with no spec requirement

The design promises these; the spec does not require them. Each is genuinely useful and none
conflicts with the constitution. **They are not silently added to scope** — they need either a spec
amendment via `/speckit-clarify` or an explicit deferral.

| # | Capability | Screen | Recommendation |
|---|-----------|--------|----------------|
| **G1** | Effective-rights check ("as cn=lchen → userPassword: write ✓") via the Get Effective Rights control | 2b | **Adopt** — it is the only way to answer "can this user actually do that", the question ACI editing exists to serve. Needs a new FR and a capability-degradation path where unsupported |
| **G2** | High-contrast theme and colourblind-safe diff palette | 3a | **Adopt** — small cost, and FR-102 already commits to OS appearance settings |
| **G3** | Per-connection read-only mode and "production" tagging with warnings | 5e | **Adopt** — directly serves Principle I at almost no cost, and read-only mode is the cheapest possible protection against a wrong-window mistake |
| **G4** | Command registry with rebindable keymaps, presets, and conflict detection | 5g | **Adopt** — FR-102 requires full keyboard operability, which implies a command registry; the rebinding UI is the increment |
| **G5** | Batch throttling (batch size + inter-batch pause), batch modify-DN, batch compare | 6c | **Adopt throttling** (a bulk job that overruns a production directory is a real hazard); **defer** batch modify-DN and compare — FR-096 says "a single change" |
| **G6** | Go to DN accepts an LDAP URL (RFC 4516) | 6e | **Adopt** — trivial, and LDAP URLs are how referrals and documentation identify entries |
| **G7** | OID editor that resolves a dotted-decimal OID to its registered name | 6f | **Adopt** — extends FR-072's editor list by one |

---

## Deviations index

Full statements are in [../plan.md](../plan.md). Summarised here so a frontend task can find its
constraint without leaving this document.

| # | Screens | Deviation |
|---|---------|-----------|
| D1 | 2c, 4a, rail | Local server instance management dropped |
| D2 | 3b, 5f, Credentials menu | The entire vault concept is replaced — see plan.md |
| D3 | 1b, 5b, flow 7b | Auto-save removed; every commit previews |
| D4 | 3b | Credentials manager is an in-app modal, not a separate window (Wails v2) |
| D5 | 1a, 5e | JNDI / Apache Directory API "Provider" selector removed |
| D6 | 2e, 6e | Conflict policy is the union of the design's and the spec's options |
| D7 | 3c | "Retry anonymously" removed as an automatic bind-failure action |
| D8 | 4a | Edit › Undo/Redo scoped to editor-local edits only |
| D9 | 5h | Update checks default off; extension list dropped |
