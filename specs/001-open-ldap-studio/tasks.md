---

description: "Task list for Open LDAP Studio implementation"
---

# Tasks: Open LDAP Studio

**Input**: Design documents from `/specs/001-open-ldap-studio/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Test tasks are **mandatory** in this feature. Constitution V (Test-First Against Real
Directories) is NON-NEGOTIABLE, and quickstart.md requires contract tests C1–C14, E1–E5, F1–F9,
and X1–X8 to be written **before** the code they constrain. Contract tests are therefore listed
first in every phase and must be failing before implementation begins.

**Organization**: Tasks are grouped by user story. Each of the 13 stories is independently
implementable, testable, and shippable, matching the M0–M6 milestones in plan.md.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US13)
- Exact file paths are given in every task

## Path Conventions

Single Go module with an embedded web frontend (Wails v2), per plan.md § Project Structure:

- Go core: `internal/<package>/`, entrypoint `main.go`
- Frontend: `frontend/src/`
- Tests: `test/integration/`, `test/containers/`, `test/corpus/`, `test/secrets_scan/`

## Architectural invariants that shape these tasks

Three package boundaries are constitutional enforcement points, not layering preferences. Tasks
that would cross them are errors, not shortcuts:

1. `internal/ldapx/` is the only package that speaks LDAP.
2. `internal/changeset/` is the only caller of `ldapx` mutation functions — the only write path.
3. `internal/secrets/` is the only package importing a platform credential API.

T013–T016 make all three checkable by CI on day one. Write them first.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: A buildable, testable, lintable skeleton on all three platforms.

- [ ] T001 Initialise the Go module and pin the toolchain to Go 1.26 in `go.mod`
- [x] T002 Scaffold the Wails v2 application entrypoint and window in `main.go` and `wails.json`
- [ ] T003 [P] Create the `internal/` package skeleton with doc.go files declaring each package's boundary in `internal/{ldapx,changeset,ldif,dsml,schema,aci,credentials,secrets,profiles,trust,history,compare,jobs,exportx,commands,prefs,logging,bridge}/doc.go`
- [x] T004 [P] Scaffold the React 19 + Vite 7 + TypeScript 5.9 frontend in `frontend/package.json`, `frontend/vite.config.ts`, `frontend/tsconfig.json`
- [ ] T005 [P] Add Zustand, TanStack Virtual, and CodeMirror 6 dependencies in `frontend/package.json`
- [ ] T006 [P] Configure golangci-lint in `.golangci.yml` and gofumpt formatting in the Makefile
- [ ] T007 [P] Configure ESLint, Prettier, and Vitest in `frontend/.eslintrc.cjs` and `frontend/vitest.config.ts`
- [ ] T008 Create the task runner with `build`, `test`, `test-integration`, `lint`, `fmt`, `run`, and `gates` targets in `Makefile`
- [ ] T009 [P] Add the CI build matrix for Linux, macOS, and Windows runners in `.github/workflows/ci.yml`
- [ ] T010 [P] Add the quality-gate CI job running govulncheck, `npm audit --audit-level=high`, and `go mod verify` in `.github/workflows/gates.yml`
- [ ] T011 [P] Add CODEOWNERS requiring second-maintainer review on `internal/changeset/`, `internal/secrets/`, `internal/trust/`, and TLS code in `internal/ldapx/` in `.github/CODEOWNERS`
- [ ] T012 [P] Document the platform build prerequisites (webkit2gtk-4.1, Xcode CLT, WebView2) in `CONTRIBUTING.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Milestone M0. The enforcement machinery, the protocol layer, the write pipeline, and
the shell. No user story work can begin until this phase is complete.

**⚠️ CRITICAL**: This phase exists because contracts C1, C2, C11, and C13 are cheap to satisfy on
day one and expensive to retrofit. Each is architectural rather than behavioural. When this phase
ends, **a write with no preview is inexpressible**, not merely discouraged.

### Architectural contract tests (write these first — they must fail before anything else is built)

- [ ] T013 [P] Contract test C1 — `ldapx` mutation functions have exactly one calling package, `changeset` — in `test/architecture/import_boundaries_test.go`
- [ ] T014 [P] Contract test C2 — no package outside `secrets` imports a platform credential API — in `test/architecture/import_boundaries_test.go`
- [ ] T015 [P] Contract test C11 — no vault method (`LockVault`, `UnlockVault`, `SetVaultStorage`, `ExportVault`, `ImportVault`, `RevealSecret`, master-password) exists on the bound bridge surface — in `test/architecture/bridge_surface_test.go`
- [ ] T016 [P] Contract test — no `os/exec` appears in the transitive import tree of `internal/secrets` — in `internal/secrets/no_subprocess_test.go`
- [ ] T017 [P] Contract test C9 — no bridge method returns a secret value, asserted over the bound surface by reflection — in `test/architecture/bridge_surface_test.go`
- [ ] T018 [P] Contract test C13 — a read-only profile is refused a preview token at the `changeset` boundary before any UI check — in `internal/changeset/readonly_test.go`

### Test harness and fixtures

- [ ] T019 [P] Define the OpenLDAP baseline container with `cn=config` enabled in `test/containers/openldap.go`
- [ ] T020 [P] Define the ApacheDS container providing X.500 `prescriptiveACI`, subentries, and administrative roles in `test/containers/apacheds.go`
- [ ] T021 [P] Define the capability-poor container — no paging, no readable subschema, no extended operations — in `test/containers/poor.go`
- [ ] T022 Build the testcontainers-go harness with fixture seeding and the `integration` build tag in `test/containers/harness.go`
- [ ] T023 [P] Assemble the fidelity corpus — binary, non-UTF-8, empty, option-bearing (`;binary`, `;lang-de`), and multi-megabyte values — in `test/corpus/`
- [ ] T024 Validate the ApacheDS fixture containerises and serves ACI reads, discharging risk R-6 before M4 depends on it, in `test/containers/apacheds_test.go`

### Logging and redaction (must exist before anything writes a file)

- [ ] T025 [P] Implement the four log sinks with rotation at 10 MB × 3 in `internal/logging/sink.go`
- [ ] T026 Implement redaction at write time, applied before any byte reaches a sink, in `internal/logging/redact.go`
- [ ] T027 [P] Contract test F8 — logs rotate at 10 MB × 3 and redact before writing — in `internal/logging/redact_test.go`

### Error and result model

- [ ] T028 [P] Implement `ldapx.Result` carrying `resultCode`, `matchedDN`, and `diagnosticMessage`, never collapsed to a bool, in `internal/ldapx/result.go`
- [ ] T029 [P] Implement the error category taxonomy and the `Indeterminate` state in `internal/ldapx/errors.go`
- [ ] T030 [P] Contract test X2 — no code path between `ldapx` and the bridge converts a `Result` to `bool` or a bare `error` — in `test/architecture/result_model_test.go`
- [ ] T031 [P] Contract test X3 — `DiagnosticMessage` is byte-identical to the server's, asserted against a server returning a deliberately odd message — in `test/integration/error_model_test.go`

### Secrets provider (Gate II)

- [ ] T032 Define the `secrets.Provider` interface with no marshaller on `Secret` in `internal/secrets/provider.go`
- [ ] T033 [P] Implement the Linux/BSD Secret Service binding over D-Bus in `internal/secrets/secretservice_linux.go`
- [ ] T034 [P] Implement the macOS Keychain Services binding in `internal/secrets/keychain_darwin.go`
- [ ] T035 [P] Implement the Windows Credential Manager binding in `internal/secrets/wincred_windows.go`
- [ ] T036 Implement the in-memory session-only fallback for an unavailable or locked agent in `internal/secrets/session.go`
- [ ] T037 [P] Integration test — each platform binding stores, retrieves, and deletes, and a locked agent surfaces a recoverable error — in `internal/secrets/provider_test.go`

### Jobs registry (Gate IV)

- [ ] T038 Implement the cancellable job registry with `context.Context` propagation in `internal/jobs/registry.go`
- [ ] T039 Implement job progress reporting throttled to ≤ 20 events/s in `internal/jobs/progress.go`
- [ ] T040 [P] Implement the `Execute` and `DryRun` job modes, where dry run replaces the dispatch call with a recorder, in `internal/jobs/mode.go`
- [ ] T041 [P] Implement job throttling, pause, and batch-size control in `internal/jobs/throttle.go`
- [ ] T042 [P] Contract test C8 — every `JobID`-returning method has a `Cancel` effective within 2 s — in `test/integration/jobs_cancel_test.go`
- [ ] T043 [P] Contract test E2 — `job:progress` is throttled; a 100k-entry job emits ≤ 20 events/s — in `internal/jobs/progress_test.go`
- [ ] T044 [P] Contract test E3 — every job reaching a terminal state emits exactly one `job:finished` — in `internal/jobs/registry_test.go`

### LDAP core — connection and bind

- [ ] T045 Implement dial, LDAPS, StartTLS, keep-alive, and the reconnect policy in `internal/ldapx/conn.go`
- [ ] T046 Implement simple, anonymous, SASL EXTERNAL, GSSAPI, and DIGEST-MD5 binds in `internal/ldapx/bind.go`
- [ ] T047 Implement the in-house CRAM-MD5 bind mechanism (FR-022, not provided by go-ldap) in `internal/ldapx/cram_md5.go`
- [ ] T048 [P] Implement Root DSE reading — naming contexts, supported controls, extensions, SASL mechanisms — in `internal/ldapx/rootdse.go`
- [ ] T049 [P] Implement the Password Modify and Who Am I extended operations in `internal/ldapx/extended.go`
- [ ] T050 Implement paged search with alias and referral policy and bounded traversal depth in `internal/ldapx/search.go`
- [ ] T051 [P] Implement the paging, server-side sort, subtree delete, and ManageDsaIT controls in `internal/ldapx/controls/standard.go`
- [ ] T052 Implement the in-house VLV control (FR-002, not provided by go-ldap) in `internal/ldapx/controls/vlv.go`
- [ ] T053 Implement the add, modify, modrdn, and delete dispatch functions — package-private to all callers but `changeset` — in `internal/ldapx/modify.go`
- [ ] T054 [P] Integration test — every bind method succeeds against the OpenLDAP fixture over plain, StartTLS, and LDAPS — in `test/integration/bind_test.go`

### Changeset pipeline (Gate I — build before any write path exists)

- [ ] T055 Implement the `ChangeSet` and `Operation` model in `internal/changeset/changeset.go`
- [ ] T056 Implement `Preview()` returning the attribute-level diff, affected count, and a single-use expiring `PreviewToken` carrying `EntryVersions` in `internal/changeset/preview.go`
- [ ] T057 Implement `dispatch` as the sole caller of `ldapx` mutation functions in `internal/changeset/dispatch.go`
- [ ] T058 Implement read-only and production-tagged profile refusal at the token-issue boundary in `internal/changeset/readonly.go`
- [ ] T059 [P] Contract test C4 — `Commit` rejects an absent, reused, expired, or stale token — in `internal/changeset/preview_test.go`
- [ ] T060 [P] Contract test X4 — a connection dropped mid-write yields `Indeterminate`, never success or failure — in `test/integration/indeterminate_test.go`

### On-disk formats and migration contract

- [ ] T061 Implement the `schemaVersion`-first file envelope and the refuse-on-higher-version rule in `internal/prefs/storefile.go`
- [ ] T062 [P] Contract test F1 and F2 — every written file has `schemaVersion` as its first key, and a higher version is refused rather than rewritten — in `internal/prefs/storefile_test.go`
- [ ] T063 [P] Implement the SC-008 secrets scan over every written file, log, and crash artefact in `test/secrets_scan/scan_test.go`
- [ ] T064 [P] Contract tests F3 and E1 — no written file and no emitted event payload contains secret material across a session exercising all stories — in `test/secrets_scan/scan_test.go`

### Bridge and command registry

- [ ] T065 Implement the Wails-bound bridge skeleton exposing `Preview`, `Commit`, `Discard`, and no `Modify`/`Add`/`Delete`/`Rename` method in `internal/bridge/changeset.go`
- [ ] T066 [P] Implement the jobs bridge methods `Cancel`, `JobState`, `ListJobs` in `internal/bridge/jobs.go`
- [ ] T067 Implement the command registry with ids, labels, menu placement, enablement predicates, and scope in `internal/commands/registry.go`
- [ ] T068 Implement keymap presets, conflict detection, and OS-reserved-chord refusal in `internal/commands/keymap.go`
- [ ] T069 [P] Contract test C14 — every command has an enablement predicate and a non-conflicting default binding per platform — in `internal/commands/registry_test.go`
- [ ] T070 [P] Contract test C7 and X1 — every bridge method contacting a server returns a populated `Result` on success and failure — in `test/architecture/result_model_test.go`

### Application shell (built once, before any view)

- [ ] T071 Implement the window chrome and the 8 top-level menus, with the Window menu and `File › New › Server instance…` removed per D1, in `frontend/src/shell/MenuBar.tsx`
- [ ] T072 [P] Implement the 6-icon activity rail — connections, DIT browser, searches, schema, LDIF & files, preferences — in `frontend/src/shell/ActivityRail.tsx`
- [ ] T073 Implement the always-visible status bar showing connection state, bind DN, target server, entry counts, page N/M, and active limits (FR-010) in `frontend/src/shell/StatusBar.tsx`
- [ ] T074 [P] Implement the bottom panel with Progress, Modification Logs, Search Logs, Errors, and Console tabs in `frontend/src/shell/BottomPanel.tsx`
- [ ] T075 [P] Implement the document tab host, so searches and LDIF files are documents rather than modals, in `frontend/src/shell/DocumentTabs.tsx`
- [ ] T076 Implement the generated Wails bindings wrapper and typed frontend client in `frontend/src/bridge/client.ts`
- [ ] T077 [P] Implement the command, menu, keymap, and context-menu layer over the one registry in `frontend/src/commands/CommandProvider.tsx`
- [ ] T078 [P] Implement the event subscription layer for `job:*`, `conn:*`, `trust:*`, and `capability:*` in `frontend/src/bridge/events.ts`

**Checkpoint**: M0 exit criteria. Contracts C1, C2, C3, C11, and C13 pass. No feature exists yet,
but the bridge cannot express a write without a preview, and no package outside `secrets` can
reach a platform credential API.

---

## Phase 3: User Story 1 — Connect to a directory and explore it (Priority: P1) 🎯 MVP

**Goal**: Define, store, and connect profiles; browse the DIT; read entries including operational
attributes; establish certificate trust; know who you are bound as — with no editing capability
present anywhere in the build.

**Independent test**: Define a profile against the OpenLDAP fixture over plain LDAP, StartTLS, and
LDAPS; connect using each supported bind method; browse from the root DSE down three levels; open
an entry and confirm every user and operational attribute is displayed.

**Screens**: `1a`, `1b` (read-only half), `3b`, `3c`, `5a`, `5e`, `5f`, `5g`, `6d`, `6e`

### Contract tests for this story (write first)

- [ ] T079 [P] [US1] Contract test C3 — `SaveProfile` rejects every payload carrying secret material — in `internal/bridge/profiles_test.go`
- [ ] T080 [P] [US1] Contract test F7 — `credentials.json` contains no secret material and no vault structure — in `internal/credentials/store_test.go`
- [ ] T081 [P] [US1] Contract test F4 — a profile bundle round-trips export → import with no field loss — in `internal/profiles/bundle_test.go`
- [ ] T082 [P] [US1] Contract test E5 — `trust:challenge` never fires without the connection having been refused first — in `test/integration/trust_test.go`
- [ ] T083 [P] [US1] Contract test C12 — `FollowReferral` cannot be called without an explicit target profile — in `internal/bridge/referral_test.go`
- [ ] T084 [P] [US1] Contract test X8 — `invalidCredentials` produces no retry loop and no anonymous fallback (D7) — in `test/integration/bind_failure_test.go`
- [ ] T085 [P] [US1] Integration test — SC-009 cold start with a stored credential present reaches a usable window in under 3 s with no prompt — in `test/integration/startup_test.go`

### Data model and stores

- [ ] T086 [P] [US1] Implement `profiles.Profile` and `profiles.TLSPolicy` with no field capable of holding a secret in `internal/profiles/model.go`
- [ ] T087 [US1] Implement the profile and folder store with cycle rejection on folders in `internal/profiles/store.go`
- [ ] T088 [P] [US1] Implement profile bundle export and import in `internal/profiles/bundle.go`
- [ ] T089 [P] [US1] Implement `credentials.Credential` as a first-class entity with many-to-many profile assignments (research R10) in `internal/credentials/model.go`
- [ ] T090 [US1] Implement the credential store persisting metadata only, never a secret value, in `internal/credentials/store.go`
- [ ] T091 [P] [US1] Implement `trust.Decision` bound to an exact certificate fingerprint in `internal/trust/model.go`
- [ ] T092 [US1] Implement the trust store with session and permanent scopes and re-challenge on certificate rotation in `internal/trust/store.go`
- [ ] T093 [P] [US1] Implement `ldapx.ReferralTarget` and RFC 4516 LDAP URL parsing in `internal/ldapx/referral.go`

### Bridge methods

- [ ] T094 [US1] Implement the connection bridge methods `ListProfiles`, `GetProfile`, `SaveProfile`, `DeleteProfile`, `DuplicateProfile`, `MoveProfile` in `internal/bridge/profiles.go`
- [ ] T095 [P] [US1] Implement `ListFolders`, `SaveFolder`, `DeleteFolder`, `ExportProfiles`, `ImportProfiles` in `internal/bridge/folders.go`
- [ ] T096 [US1] Implement `Connect`, `Disconnect`, `ConnectionState`, `TestConnection` in `internal/bridge/connections.go`
- [ ] T097 [P] [US1] Implement `WhoAmI` and `RootDSE` in `internal/bridge/connections.go`
- [ ] T098 [US1] Implement the credential bridge methods `ListCredentials`, `GetCredential`, `SaveCredential`, `DeleteCredential`, `AssignCredential`, `UnassignCredential`, `CredentialAssignments`, `TestBind`, `CredentialStoreStatus` in `internal/bridge/credentials.go`
- [ ] T099 [P] [US1] Implement `ListTrustDecisions`, `RevokeTrustDecision`, `DecideTrust` in `internal/bridge/trust.go`
- [ ] T100 [US1] Implement `ListChildren`, `ReadEntry`, `RefreshEntry`, `GoToDN` with `EntryPage` carrying `cookie`, `loadedCount`, `serverLimit`, and `truncatedByServer` in `internal/bridge/browse.go`
- [ ] T101 [P] [US1] Implement `ResolveReferral`, `FollowReferral`, `ParseLDAPURL` in `internal/bridge/referral.go`
- [ ] T102 [P] [US1] Implement `GetPreferences`, `SetPreferences`, `PurgeCache` in `internal/bridge/prefs.go`
- [ ] T103 [P] [US1] Implement the keymap bridge methods `ListCommands`, `GetKeymap`, `SetBinding`, `ResetKeymap`, `ExportKeymap` in `internal/bridge/commands.go`

### Events

- [ ] T104 [US1] Emit `conn:state`, `conn:reconnected` with `writesRequireConfirmation`, and `conn:lost` in `internal/bridge/events_conn.go`
- [ ] T105 [P] [US1] Emit `trust:challenge` with host, port, fingerprint, chain PEM, failure reason, and prior-trust flag, only after the connection has been refused, in `internal/bridge/events_trust.go`
- [ ] T106 [P] [US1] Emit `credential:required` and `capability:unavailable` in `internal/bridge/events_capability.go`

### Frontend

- [ ] T107 [US1] Implement the connections start screen and the 4-step wizard — Network, Authentication, Browser options, Edit options — with the JNDI provider selector dropped per D5, in `frontend/src/views/connections/ConnectionWizard.tsx` (screen `1a`)
- [ ] T108 [P] [US1] Implement wizard step 2 with stored-credential selection, the "secret held in OS keychain" notice, check-authentication via whoami, and the clear-text warning strip, with "retry anonymously" dropped per D7, in `frontend/src/views/connections/AuthStep.tsx` (screen `3c`)
- [ ] T109 [US1] Implement the DIT browser tree with lazy paged children, the explicit "fetch next 100 of 1 842…" node, and TanStack Virtual scrolling in `frontend/src/views/browser/DitTree.tsx` (screen `1b`)
- [ ] T110 [P] [US1] Implement the tree filter box and the Searches and Bookmarks nodes under each connection in `frontend/src/views/browser/TreeNodes.tsx`
- [ ] T111 [US1] Implement the read-only entry view with Attributes, LDIF view, Table, and Object class tabs in `frontend/src/views/browser/EntryView.tsx`
- [ ] T112 [P] [US1] Implement the entry-info side panel with structural and auxiliary classes, last modified, photo preview, and "Show in schema browser" in `frontend/src/views/browser/EntryInfoPanel.tsx`
- [ ] T113 [P] [US1] Implement the operational-attributes toggle and raw-value display in `frontend/src/views/browser/AttributeTable.tsx`
- [ ] T114 [P] [US1] Implement the credentials management modal — credential list by type, detail, server assignments, test bind, use log, password age — as an in-app modal per D4, with every vault affordance absent per D2, in `frontend/src/views/credentials/CredentialsModal.tsx` (screen `3b`)
- [ ] T115 [P] [US1] Implement the certificate trust dialog with chain view and trust-once / trust-permanently / reject in `frontend/src/dialogs/CertificateTrustDialog.tsx` (screen `6e`)
- [ ] T116 [P] [US1] Implement the Go-to-DN dialog accepting an LDAP URL in `frontend/src/dialogs/GoToDnDialog.tsx` (screen `6e`, gap G6)
- [ ] T117 [P] [US1] Implement the referral connection chooser — pick existing, create new, remember for session — in `frontend/src/dialogs/ReferralChooser.tsx` (screen `6e`, research R11)
- [ ] T118 [P] [US1] Implement the Browser & tree preference pane in `frontend/src/preferences/BrowserPane.tsx` (screen `5a`)
- [ ] T119 [P] [US1] Implement the Connections & timeouts preference pane including the modify-request strategy and the open-read-only / production-tag settings in `frontend/src/preferences/ConnectionsPane.tsx` (screen `5e`, gap G3)
- [ ] T120 [P] [US1] Implement the Credentials & security preference pane with no vault, auto-lock, or master-password affordance per D2 in `frontend/src/preferences/SecurityPane.tsx` (screen `5f`)
- [ ] T121 [P] [US1] Implement the keyboard shortcuts preference pane with searchable command table, presets, live conflict detection, and export in `frontend/src/preferences/ShortcutsPane.tsx` (screen `5g`, gap G4)
- [ ] T122 [P] [US1] Implement the properties dialogs for connection, entry, attribute, and value in `frontend/src/dialogs/PropertiesDialog.tsx` (screen `6d`)

### Story validation

- [ ] T123 [US1] Integration test — connect over plain, StartTLS, and LDAPS; bind anonymous, simple, EXTERNAL, GSSAPI, DIGEST-MD5, CRAM-MD5; read the root DSE; expand a 25,000-child container; call `WhoAmI` — in `test/integration/us1_connect_test.go`
- [ ] T124 [US1] Integration test — present an untrusted certificate, accept for the session, restart, then rotate the certificate, asserting refusal, challenge payload, non-persistence, and re-challenge — in `test/integration/us1_trust_test.go`
- [ ] T125 [US1] Performance test — SC-004, the first page of a 100,000-child container is visible within 2 s — in `test/integration/us1_perf_test.go`

**Checkpoint**: M1 half-done. US1 is independently shippable as a read-only browser.

---

## Phase 4: User Story 2 — Search with raw filters, saved searches, and bookmarks (Priority: P1)

**Goal**: Author and run raw RFC 4515 filters transmitted byte-identically, present results in a
virtualised configurable grid, and persist searches and bookmarks.

**Independent test**: Author a filter with nested boolean and extensible-match components, run it
against the OpenLDAP fixture, verify the result set matches an equivalent `ldapsearch` exactly,
save it, reopen it, and re-run it unchanged.

**Screens**: `1c`, `6e` (filter editor)

### Contract tests for this story (write first)

- [ ] T126 [P] [US2] Contract test C5 — `ValidateFilter` never returns a modified filter string — in `internal/bridge/search_test.go`
- [ ] T127 [P] [US2] Integration test — the filter string on the wire is byte-identical to what was typed, captured at the protocol layer — in `test/integration/us2_filter_fidelity_test.go`
- [ ] T128 [P] [US2] Integration test — a search past the server size limit shows partial results labelled truncated-by-server with the server's result code — in `test/integration/us2_limits_test.go`

### Implementation

- [ ] T129 [US2] Implement the RFC 4515 filter parser reporting `{ok, position, message}` and parsing a copy, never rewriting the original, in `internal/ldapx/filter.go`
- [ ] T130 [P] [US2] Add a fuzz target for the filter parser in `internal/ldapx/filter_fuzz_test.go`
- [ ] T131 [P] [US2] Implement `ldapx.SearchDefinition` carrying the filter as a string, plus scope, base, attributes, limits, alias and referral policy, and controls, in `internal/ldapx/searchdef.go`
- [ ] T132 [US2] Implement `ValidateFilter`, `StartSearch`, and `FetchNextPage` in `internal/bridge/search.go`
- [ ] T133 [P] [US2] Implement `history.SearchRecord` and the search log in `internal/history/search.go`
- [ ] T134 [P] [US2] Implement saved-search persistence and the `ListSavedSearches`, `SaveSearch`, `DeleteSavedSearch`, `SearchHistory` bridge methods in `internal/bridge/savedsearch.go`
- [ ] T135 [P] [US2] Implement bookmark persistence and the `ListBookmarks`, `SaveBookmark`, `DeleteBookmark` bridge methods in `internal/bridge/bookmarks.go`
- [ ] T136 [P] [US2] Implement server-side sort control negotiation with a stated reason when unsupported in `internal/ldapx/controls/sort.go`
- [ ] T137 [US2] Implement the search editor document tab with the filter builder and the raw RFC 4515 field kept in sync in `frontend/src/views/search/SearchEditor.tsx` (screen `1c`)
- [ ] T138 [P] [US2] Implement the virtual-scroll result grid with column picker, client-side sort, and selected-row preview in `frontend/src/views/search/ResultGrid.tsx`
- [ ] T139 [P] [US2] Implement the attribute palette dragging from the schema and the filter history list in `frontend/src/views/search/AttributePalette.tsx`
- [ ] T140 [P] [US2] Implement the filter editor dialog with content assist in `frontend/src/dialogs/FilterEditorDialog.tsx` (screen `6e`)
- [ ] T141 [P] [US2] Implement saved-search and bookmark management UI under each connection in `frontend/src/views/search/SavedSearches.tsx`

### Story validation

- [ ] T142 [US2] Integration test — a nested boolean and extensible-match filter returns a result set identical to `ldapsearch` — in `test/integration/us2_search_test.go`
- [ ] T143 [US2] Integration test — a saved search runs unchanged against a different connection and reports that server's results or error — in `test/integration/us2_saved_test.go`
- [ ] T144 [US2] Performance test — SC-002, no UI block exceeds 100 ms during a 100k-result search — in `test/integration/us2_perf_test.go`

**Checkpoint**: **M1 complete.** SC-001, SC-004, SC-016. Shippable read-only product: connect,
browse, search. Nothing in the build can write to a directory.

---

## Phase 5: User Story 3 — Modify entries with a previewed, confirmed write (Priority: P2)

**Goal**: The first write path. Edit attributes, create entries, delete entries and subtrees —
every one through `Preview` → confirm → `Commit(token)`.

**Independent test**: Modify, add, and delete entries against the OpenLDAP fixture; verify for each
that the preview matches the operation the server actually received, that cancelling sends nothing,
and that a schema-violating change returns the server's verbatim result code and diagnostic.

**Screens**: `1b` (editing half), `2a`, `5b`, `6d`, `6e` (rename, attribute wizard)

**⚠️ D3 must land in the design project before this phase**: the wireframe's auto-save contradicts
Constitution I. Remove it from the design first, then build.

### Contract tests for this story (write first)

- [ ] T145 [P] [US3] Integration test — the preview matches the transmitted operation exactly, captured at the protocol layer, for modify, add, and delete — in `test/integration/us3_preview_fidelity_test.go`
- [ ] T146 [P] [US3] Integration test — cancelling a preview sends nothing to the server — in `test/integration/us3_cancel_test.go`
- [ ] T147 [P] [US3] Integration test — an entry changed on the server between preview and commit causes `Commit` to refuse the stale token — in `test/integration/us3_concurrency_test.go`
- [ ] T148 [P] [US3] Contract test X5 — every result code in the error model's mandated-handling table triggers its handling, forced against a live server — in `test/integration/error_codes_test.go`
- [ ] T149 [P] [US3] Integration test — attributes differing only by description option (`;binary`, `;lang-de`) are not merged and each option is separately editable — in `test/integration/us3_options_test.go`

### Implementation

- [ ] T150 [US3] Implement the attribute-level diff producing per-attribute before and after values in `internal/changeset/diff.go`
- [ ] T151 [P] [US3] Implement the modify-request strategy selector — changed attributes only versus replace whole entry — in `internal/changeset/strategy.go`
- [ ] T152 [US3] Implement `changeset.SubtreeDelete` requiring a counted enumeration before a token is issued in `internal/changeset/subtreedelete.go`
- [ ] T153 [P] [US3] Implement `CountSubtree` returning a `JobID` for delete confirmation in `internal/bridge/browse.go`
- [ ] T154 [P] [US3] Implement entry templates and the `ListTemplates`, `SaveTemplate` bridge methods in `internal/bridge/templates.go`
- [ ] T155 [US3] Implement the entry editor with pending-change staging held frontend-local until explicit commit, with no auto-save per D3, in `frontend/src/views/browser/EntryEditor.tsx` (screen `1b`)
- [ ] T156 [P] [US3] Implement the preview dialog listing each add, replace, and delete with old and new values and the affected count in `frontend/src/views/browser/PreviewDialog.tsx`
- [ ] T157 [P] [US3] Implement the verbatim result display showing the server's code and diagnostic alongside any plain-language interpretation in `frontend/src/views/browser/ResultBanner.tsx`
- [ ] T158 [US3] Implement the new-entry wizard — method, object classes with implied superiors, RDN builder with multi-valued RDN, attributes, LDIF preview — in `frontend/src/views/browser/NewEntryWizard.tsx` (screen `2a`)
- [ ] T159 [P] [US3] Implement the entry context menu — New, New from existing, New context entry, Paste, Copy DN, Rename, Move, Delete subtree — in `frontend/src/views/browser/EntryContextMenu.tsx`
- [ ] T160 [P] [US3] Implement the subtree-delete confirmation stating the subtree size as a distinct choice from single-entry delete in `frontend/src/dialogs/DeleteSubtreeDialog.tsx`
- [ ] T161 [P] [US3] Implement the pending-edits prompt on navigate, refresh, and collapse, committing nothing as a side effect, in `frontend/src/views/browser/PendingEditsGuard.tsx`
- [ ] T162 [P] [US3] Implement the attribute wizard with description options `;binary`, `;lang-de`, and custom in `frontend/src/dialogs/AttributeWizard.tsx` (screen `6e`)
- [ ] T163 [P] [US3] Implement the Entry editor preference pane with the "auto on focus loss" save mode removed per D3 in `frontend/src/preferences/EntryEditorPane.tsx` (screen `5b`)

### Story validation

- [ ] T164 [US3] Integration test — a schema-violating change is rejected with the server's verbatim result code and diagnostic and the entry is refreshed to its actual server state — in `test/integration/us3_reject_test.go`
- [ ] T165 [US3] Integration test — a dry run of a recursive delete runs full validation and writes nothing — in `test/integration/us3_dryrun_test.go`
- [ ] T166 [US3] Verify SC-005 — the static check over callers of `ldapx` mutation functions finds exactly one, `changeset.dispatch` — in `test/architecture/import_boundaries_test.go`

**Checkpoint**: The first release that can damage a directory. The preview pipeline is load-bearing
from here.

---

## Phase 6: User Story 4 — Import and export LDIF and other interchange formats (Priority: P2)

**Goal**: Byte-faithful LDIF in and out, plus DSML, CSV, JSON, XLSX, and ODS export, all streaming
inside Go so no bulk data crosses the bridge.

**Independent test**: Export a subtree containing binary values, non-ASCII characters, and folded
long lines; re-import into an empty directory; verify a byte-faithful round trip of every attribute
value and a per-record success report.

**Screens**: `1e`, `6b`, `5d`, `6d`

**Risk R-3 applies**: LDIF fidelity bugs are unrecoverable for users. Corpus-driven tests are
written **before** the writer, and any fidelity bug is a release blocker.

### Contract tests for this story (write first)

- [ ] T167 [P] [US4] Write the LDIF fidelity corpus tests before the writer exists, over binary, non-UTF-8, empty, folded, and option-bearing values, in `internal/ldif/fidelity_test.go`
- [ ] T168 [P] [US4] Contract test C6 — `EncodeValue(DecodeValue(v)) == v` over the fidelity corpus — in `internal/bridge/values_test.go`
- [ ] T169 [P] [US4] Fuzz the LDIF reader against the writer in `internal/ldif/fuzz_test.go`
- [ ] T170 [P] [US4] Integration test — the corpus exported at three different wrap and fold settings re-imports to byte-identical *values*, asserting value equality and never file equality (research R13) — in `test/integration/us4_layout_test.go`
- [ ] T171 [P] [US4] Contract test F9 — a rejects `.ldif` from a failed run re-executes cleanly — in `test/integration/us4_rejects_test.go`
- [ ] T172 [P] [US4] Contract test X6 — a multi-entry run with mixed outcomes reports every entry and never reports `succeeded` — in `test/integration/us4_partial_test.go`

### Implementation

- [ ] T173 [US4] Implement the RFC 2849 LDIF reader — content and change records, folding, base64, `<` URL references, comments, version headers, CRLF — in `internal/ldif/reader.go`
- [ ] T174 [US4] Implement the RFC 2849 LDIF writer with configurable wrap and base64 fold width and a strict base64-only-when-required policy in `internal/ldif/writer.go`
- [ ] T175 [P] [US4] Implement line-numbered LDIF diagnostics in `internal/ldif/diagnostics.go`
- [ ] T176 [P] [US4] Implement the DSML v2 reader and writer with XML entity expansion and external entity resolution disabled in `internal/dsml/dsml.go`
- [ ] T177 [P] [US4] Implement the streaming CSV and JSON writers in `internal/exportx/delimited.go`
- [ ] T178 [P] [US4] Implement the streaming XLSX writer via excelize and the in-house ODS writer (research R21) in `internal/exportx/spreadsheet.go`
- [ ] T179 [US4] Implement `StartExport` streaming inside Go with progress and cancellation in `internal/bridge/export.go`
- [ ] T180 [US4] Implement `PrepareImport` returning counts by type and a preview token, and `StartImport` in `internal/bridge/import.go`
- [ ] T181 [P] [US4] Implement `ValidateLDIF` as a local-only line-numbered check in `internal/bridge/ldif.go`
- [ ] T182 [P] [US4] Implement `CopyToClipboard` for DN, LDIF, and delimited formats in `internal/bridge/clipboard.go`
- [ ] T183 [P] [US4] Implement the per-record error policy — halt on first failure or skip and log — and the rejects `.ldif` sibling file in `internal/jobs/records.go`
- [ ] T184 [US4] Implement the LDIF editor with workspace file tree, change-record outline, target-connection selector, and validation gutter with quick fixes in `frontend/src/views/ldif/LdifEditor.tsx` (screen `1e`)
- [ ] T185 [P] [US4] Implement the "Diff vs. server" tab and the dry-run panel with per-record projected outcome in `frontend/src/views/ldif/DryRunPanel.tsx`
- [ ] T186 [P] [US4] Implement the import report tab in `frontend/src/views/ldif/ImportReport.tsx`
- [ ] T187 [US4] Implement the import/export wizard shell — format page, source/target and options page, background job with report — in `frontend/src/views/ldif/ImportExportWizard.tsx` (screen `6b`)
- [ ] T188 [P] [US4] Implement the LDIF & text editors preference pane, keeping output-layout settings visibly distinct from value fidelity, in `frontend/src/preferences/LdifPane.tsx` (screen `5d`)

### Story validation

- [ ] T189 [US4] Integration test — export the corpus subtree to LDIF and import into an empty directory with byte-for-byte identical values including folding, base64, and attribute options — in `test/integration/us4_fidelity_test.go`
- [ ] T190 [US4] Integration test — export the same set as DSML, CSV, JSON, XLSX, and ODS with multi-valued and binary attributes surviving losslessly — in `test/integration/us4_formats_test.go`
- [ ] T191 [US4] Performance test — a 50,000-entry LDIF export streams with progress in constant memory — in `test/integration/us4_perf_test.go`

**Checkpoint**: **M2 complete.** SC-005, SC-006, SC-007.

---

## Phase 7: User Story 5 — Inspect the schema and edit with schema awareness (Priority: P3)

**Goal**: Read and browse the server's published schema, and use it to drive editing assistance —
while remaining fully usable against a server that publishes no readable subschema.

**Independent test**: Browse the OpenLDAP fixture's schema, navigate object class → attribute type
→ syntax → matching rules and back, and confirm entry editing enumerates required and optional
attributes consistently with that schema. Then connect to the `poor` fixture and confirm the
application stays usable with assistance disabled and the reason stated.

**Screens**: `1d`, `6d`

### Contract tests for this story (write first)

- [ ] T192 [P] [US5] Contract test E4 — `capability:unavailable` fires for each capability the `poor` fixture lacks — in `test/integration/us5_degradation_test.go`
- [ ] T193 [P] [US5] Integration test — against the `poor` fixture the application remains fully usable in schema-less mode with the reason stated — in `test/integration/us5_schemaless_test.go`

### Implementation

- [ ] T194 [US5] Implement RFC 4512 subschema parsing for object classes, attribute types, syntaxes, matching rules, DIT content rules, and name forms in `internal/schema/subschema.go`
- [ ] T195 [P] [US5] Implement `schema.Element` with inheritance resolution and effective MUST/MAY computation for an object class combination in `internal/schema/element.go`
- [ ] T196 [P] [US5] Tolerate vendor extensions and malformed definitions with a warning rather than a hard failure in `internal/schema/lenient.go`
- [ ] T197 [P] [US5] Implement the per-connection schema cache invalidating on the subschema subentry's modify timestamp in `internal/schema/cache.go`
- [ ] T198 [US5] Implement `ReadSchema`, `SchemaAvailable`, and `CompareSchemas` in `internal/bridge/schema.go`
- [ ] T199 [P] [US5] Implement schema-driven validation of syntax and single-value constraints with a deliberate, preview-recorded override in `internal/schema/validate.go`
- [ ] T200 [US5] Implement the read-only schema browser with object classes, attribute types, matching rules, and syntaxes in `frontend/src/views/schema/SchemaBrowser.tsx` (screen `1d`)
- [ ] T201 [P] [US5] Implement bidirectional cross-reference navigation with a traversable back path in `frontend/src/views/schema/ElementNavigator.tsx`
- [ ] T202 [P] [US5] Implement the class hierarchy diagram and the raw definition pane in `frontend/src/views/schema/HierarchyDiagram.tsx`
- [ ] T203 [P] [US5] Implement the "Schema diff: dev ↔ prod" tab in `frontend/src/views/schema/SchemaDiff.tsx`
- [ ] T204 [P] [US5] Implement the "Used by 1 842 entries · run as search" handoff, passing a `SearchDefinition` with no new backend, in `frontend/src/views/schema/UsedBySearch.tsx`
- [ ] T205 [US5] Wire schema-derived required, permitted, single-valued, and read-only indicators into the entry editor in `frontend/src/views/browser/EntryEditor.tsx`

### Story validation

- [ ] T206 [US5] Integration test — schema navigation is bidirectional and comparing two connections lists elements unique to each side and elements defined differently — in `test/integration/us5_schema_test.go`

**Checkpoint**: US5 is independently shippable. Editing is now checked rather than guessed.

---

## Phase 8: User Story 6 — Edit specialised attribute values safely (Priority: P3)

**Goal**: A registry of value editors keyed by syntax OID, each paired with a mandatory raw tab, and
a password path where plaintext never reaches disk or a log.

**Independent test**: Set a password using each supported hash scheme and verify each against the
stored value; upload and download a binary certificate and confirm the bytes are unchanged; edit a
DN-syntax attribute through the picker and confirm the transmitted value is the exact DN chosen.

**Screens**: `2d`, `5c`, `6f` (remaining editors)

### Contract tests for this story (write first)

- [ ] T207 [P] [US6] Integration test — no plaintext password reaches disk, logs, or diagnostic exports on any password path — in `test/integration/us6_password_secrecy_test.go`
- [ ] T208 [P] [US6] Extend contract test C6 — `EncodeValue(DecodeValue(v)) == v` across every `ValueKind`, not only LDIF values — in `internal/bridge/values_test.go`
- [ ] T209 [P] [US6] Contract test — every structured editor exposes a raw tab, asserted over the shared editor contract — in `frontend/tests/editors/rawTab.test.tsx`

### Implementation

- [ ] T210 [US6] Implement `editors.ValueEditorMapping` resolving syntax OID, attribute name, and matching rule with a documented precedence in `internal/prefs/editors.go`
- [ ] T211 [US6] Implement `ReadValue`, `DecodeValue`, `EncodeValue`, `LoadValueFromFile`, `SaveValueToFile` in `internal/bridge/values.go`
- [ ] T212 [US6] Implement `HashPassword` and `VerifyPassword` for SSHA-512, SSHA, SHA, MD5, and crypt, never logging, persisting, or echoing plaintext, in `internal/bridge/password.go`
- [ ] T213 [P] [US6] Implement `PasswordModify` returning a preview token, and `SupportsPasswordModify`, in `internal/bridge/password.go`
- [ ] T214 [P] [US6] Implement X.509 decoding for subject, issuer, validity window, and fingerprints, preserving the raw DER, in `internal/ldapx/certdecode.go`
- [ ] T215 [P] [US6] Implement `;binary` transfer-option handling on read and write in `internal/ldapx/options.go`
- [ ] T216 [US6] Implement the password editor with scheme selection, verify, and bind-as-user in `frontend/src/editors/PasswordEditor.tsx` (screen `2d`)
- [ ] T217 [P] [US6] Implement the certificate editor with Info, Subject, Extensions, and DER tabs plus import and DER/PEM export in `frontend/src/editors/CertificateEditor.tsx`
- [ ] T218 [P] [US6] Implement the image editor with inline preview in `frontend/src/editors/ImageEditor.tsx`
- [ ] T219 [P] [US6] Implement the DN picker editor validating existence against the tree in `frontend/src/editors/DnEditor.tsx`
- [ ] T220 [P] [US6] Implement the generalized time editor with zone handling, showing the stored UTC value beside the local rendering, in `frontend/src/editors/TimeEditor.tsx`
- [ ] T221 [P] [US6] Implement the object class editor prompting for newly required attributes on add and warning on remove in `frontend/src/editors/ObjectClassEditor.tsx`
- [ ] T222 [P] [US6] Implement the hex, base64, and multi-line text editors as the fallback for unrecognised syntaxes in `frontend/src/editors/RawEditors.tsx`
- [ ] T223 [P] [US6] Implement the postal address editor for `$`-separated `postalAddress` in `frontend/src/editors/AddressEditor.tsx` (screen `6f`)
- [ ] T224 [P] [US6] Implement the OID editor resolving and validating dotted-decimal values in `frontend/src/editors/OidEditor.tsx` (screen `6f`, gap G7)
- [ ] T225 [P] [US6] Implement the in-place grid text editor in `frontend/src/editors/InlineGridEditor.tsx` (screen `6f`)
- [ ] T226 [US6] Implement the shared editor contract enforcing a raw tab on every structured editor in `frontend/src/editors/EditorContract.ts`
- [ ] T227 [P] [US6] Implement the Value editors preference pane — syntax OID to editor table, per-attribute-type overrides, unknown-syntax fallback, max inline length — in `frontend/src/preferences/ValueEditorsPane.tsx` (screen `5c`)

### Story validation

- [ ] T228 [US6] Integration test — set a password under each hash scheme and verify; round-trip a certificate and an image; use `PasswordModify` where advertised — in `test/integration/us6_values_test.go`

**Checkpoint**: US6 is independently shippable.

---

## Phase 9: User Story 7 — Move, rename, copy, and duplicate entries across servers (Priority: P3)

**Goal**: Modify DN with explicit old-RDN handling, subtree moves, and cross-server copy with a
conflict policy chosen in advance.

**Independent test**: Rename an entry and confirm RDN attribute handling matches the chosen option;
copy a subtree between the OpenLDAP and ApacheDS fixtures and verify entry count, DNs, and attribute
values at the destination.

**Screens**: `2e` (copy/move half), `6e` (rename, move)

### Contract tests for this story (write first)

- [ ] T229 [P] [US7] Integration test — a copy interrupted partway lists the entries already written, states the cause verbatim, and can be resumed or reversed — in `test/integration/us7_interrupted_test.go`

### Implementation

- [ ] T230 [US7] Implement Modify DN with explicit `deleteOldRDN` in `internal/ldapx/modrdn.go`
- [ ] T231 [US7] Implement subtree move using the server-side path where available and a copy-then-delete fallback that names which path was taken in `internal/changeset/move.go`
- [ ] T232 [US7] Implement cross-server copy streaming entries without materialising the subtree in `internal/changeset/copy.go`
- [ ] T233 [P] [US7] Implement the conflict policy as the union of skip, overwrite, merge, and rename per D6, applied consistently and reported per entry, in `internal/changeset/conflict.go`
- [ ] T234 [P] [US7] Implement the carry-over choice for operational attributes in `internal/changeset/copyopts.go`
- [ ] T235 [P] [US7] Implement schema compatibility checking between source and destination before any write in `internal/schema/compat.go`
- [ ] T236 [US7] Implement the copy/move dialog with mode — move, copy, copy subtree — and conflict policy in `frontend/src/views/compare/CopyMoveDialog.tsx` (screen `2e`)
- [ ] T237 [P] [US7] Implement the rename dialog with keep or delete old RDN in `frontend/src/dialogs/RenameDialog.tsx` (screen `6e`)
- [ ] T238 [P] [US7] Implement the move dialog with conflict policy in `frontend/src/dialogs/MoveDialog.tsx` (screen `6e`)
- [ ] T239 [P] [US7] Implement the operation queue with pause in `frontend/src/views/compare/OperationQueue.tsx` (screen `2e`)

### Story validation

- [ ] T240 [US7] Integration test — rename with keep-old-RDN both ways and copy a subtree between OpenLDAP and ApacheDS, asserting destination entry count, DNs, and values — in `test/integration/us7_move_test.go`

**Checkpoint**: **M3 complete.** SC-011 parity for daily work.

---

## Phase 10: User Story 8 — Review, replay, and reverse past operations (Priority: P4)

**Goal**: A durable per-profile history of every write and every search, redacted at write time,
from which any write can be exported, replayed, or reversed.

**Independent test**: Perform a series of modifications, confirm each appears in the history with
its request and response, generate the reversing LDIF, apply it, and verify the directory returns
to its original state.

**Screens**: `6f` (logs)

### Contract tests for this story (write first)

- [ ] T241 [P] [US8] Contract test F5 and X7 — redaction happens at write time; the raw history file never contains a password value — in `internal/history/redact_test.go`
- [ ] T242 [P] [US8] Integration test — an operation whose before-state was not fully captured reports that it cannot be reversed and why, rather than producing a partial reversal — in `test/integration/us8_irreversible_test.go`

### Implementation

- [ ] T243 [US8] Implement `history.Record` carrying the operation as transmitted, the server's result code and diagnostic, bind DN, server identity, and timestamp in `internal/history/record.go`
- [ ] T244 [US8] Implement the append-only JSONL store at `history/<profileId>.jsonl` with redaction applied before write in `internal/history/store.go`
- [ ] T245 [P] [US8] Capture the before-state using the Pre-Read control where the server supports it in `internal/changeset/preread.go`
- [ ] T246 [US8] Implement reversing-LDIF generation from the captured before-state for modify, add, delete, and modrdn in `internal/changeset/reverse.go`
- [ ] T247 [P] [US8] Implement `ReversalAvailable` returning `(false, reason)` rather than a partial reversal in `internal/bridge/history.go`
- [ ] T248 [US8] Implement `History`, `ExportHistoryRecord`, `PrepareReplay`, and `PrepareReversal`, each returning a preview token where it writes, in `internal/bridge/history.go`
- [ ] T249 [P] [US8] Record an interrupted write as `Indeterminate` in the history rather than as success or failure in `internal/history/indeterminate.go`
- [ ] T250 [US8] Implement the modification and search log views rendered as LDIF in `frontend/src/views/logs/ModificationLog.tsx` (screen `6f`)
- [ ] T251 [P] [US8] Implement history filtering by connection, DN or subtree, operation type, result, and time range in `frontend/src/views/logs/HistoryFilter.tsx`
- [ ] T252 [P] [US8] Implement the record detail view showing the full before-and-after diff with replay and reverse actions in `frontend/src/views/logs/RecordDetail.tsx`
- [ ] T253 [P] [US8] Implement the concurrent cancellable operations progress view in `frontend/src/views/logs/ProgressView.tsx` (screen `6f`)

### Story validation

- [ ] T254 [US8] Integration test — modify, then reverse from history, asserting the reversal restores the exact prior state through the standard preview and confirmation — in `test/integration/us8_history_test.go`

**Checkpoint**: US8 is independently shippable. SC-012.

---

## Phase 11: User Story 9 — Compare two directories or subtrees (Priority: P4)

**Goal**: A structural diff between two entries, subtrees, or search results across connections,
with generated reconciling LDIF that is never applied without the standard preview.

**Independent test**: Seed two containerised directories with a known set of differences, run the
comparison, and verify exactly those differences are reported and that the generated reconciling
LDIF, when applied, eliminates them.

**Screens**: `2e` (compare half)

### Contract tests for this story (write first)

- [ ] T255 [P] [US9] Integration test — a comparison of two identical subtrees reports zero differences, establishing the no-false-positive baseline — in `test/integration/us9_baseline_test.go`

### Implementation

- [ ] T256 [US9] Implement the streaming entry and subtree comparison engine producing `compare.Result` in `internal/compare/subtree.go`
- [ ] T257 [P] [US9] Implement ignore rules for operational attributes, value ordering, and nominated attributes, honouring each attribute's matching rule for case and space handling, in `internal/compare/ignore.go`
- [ ] T258 [P] [US9] Implement schema comparison between two connections or a project and a server in `internal/compare/schema.go`
- [ ] T259 [US9] Implement reconciling-LDIF generation in both directions, respecting the ignore rules used for the comparison, in `internal/compare/reconcile.go`
- [ ] T260 [US9] Implement `StartCompare` and `Reconcile` returning a preview token in `internal/bridge/compare.go`
- [ ] T261 [US9] Implement the two-connection compare view with ignore-operational and per-value diff in `frontend/src/views/compare/CompareView.tsx` (screen `2e`)
- [ ] T262 [P] [US9] Implement the change-class summary and filtering by class and DN in `frontend/src/views/compare/DiffSummary.tsx`
- [ ] T263 [P] [US9] Implement per-change selection with a dry run before applying in `frontend/src/views/compare/ReconcilePanel.tsx`

### Story validation

- [ ] T264 [US9] Integration test — seed two directories with known differences, compare, and apply the reconciling LDIF, asserting exactly the seeded differences and their elimination — in `test/integration/us9_compare_test.go`
- [ ] T265 [US9] Performance test — SC-013, a 10,000-entry subtree comparison completes within 30 s in bounded memory — in `test/integration/us9_perf_test.go`

**Checkpoint**: US9 is independently shippable. SC-013.

---

## Phase 12: User Story 10 — Author access control and administrative policy (Priority: P4)

**Goal**: Structured editors for access-control items and subtree specifications that never become
the only way to express a rule, and that never silently rewrite a value they cannot parse.

**Independent test**: Against the ApacheDS fixture, create a subentry with a subtree specification
and an access-control item through the structured editors, confirm the transmitted value is
byte-identical to the equivalent hand-written value, then edit it as raw text and confirm the
structured view reflects the change.

**Screens**: `2b`

**Note**: `CheckEffectiveRights` (gap G1) is designed but has no functional requirement. Do not
build T273 or T277 until a requirement exists — record the gap, do not absorb it.

### Contract tests for this story (write first)

- [ ] T266 [P] [US10] Contract test C10 — `RenderACI(ParseACI(raw)) == raw` for unmodified input — in `internal/aci/roundtrip_test.go`
- [ ] T267 [P] [US10] Integration test — a value the structured editor cannot parse is shown raw with the error and is never rewritten or discarded — in `test/integration/us10_unparseable_test.go`

### Implementation

- [ ] T268 [US10] Implement `aci.Item` parsing and rendering, preserving the raw string, with `ParseError` returned as data rather than as a failure, in `internal/aci/item.go`
- [ ] T269 [US10] Implement `aci.SubtreeSpecification` parsing for base, chop-before, chop-after, min, max, and specification filter per RFC 3672 in `internal/aci/subtree.go`
- [ ] T270 [P] [US10] Implement `SupportsStoredACI` reporting the reason when unsupported in `internal/bridge/aci.go`
- [ ] T271 [US10] Implement `ParseACI`, `RenderACI`, `ParseSubtreeSpec`, and `PreviewSubtreeSpec` returning the DNs a specification selects in `internal/bridge/aci.go`
- [ ] T272 [P] [US10] Implement OpenLDAP `olcAccess` rule parsing and ordered rendering, keeping an unparseable rule as editable raw text, in `internal/aci/olcaccess.go`
- [ ] T273 [US10] **Blocked on gap G1** — implement `SupportsEffectiveRights` and `CheckEffectiveRights` in `internal/bridge/effectiverights.go`, only once a functional requirement exists
- [ ] T274 [US10] Implement the administrative points list and the ACI item editor — identification tag, precedence, auth level, user classes, protected items with a grant/deny matrix — in `frontend/src/views/aci/AciItemEditor.tsx` (screen `2b`)
- [ ] T275 [P] [US10] Implement the Source tab exposing the exact stored string as directly editable in `frontend/src/views/aci/AciSourceTab.tsx`
- [ ] T276 [P] [US10] Implement the subtree specification editor with the in-out tree preview of matching entries in `frontend/src/views/aci/SubtreeSpecEditor.tsx`
- [ ] T277 [US10] **Blocked on gap G1** — implement the effective rights check panel in `frontend/src/views/aci/EffectiveRights.tsx`
- [ ] T278 [P] [US10] Implement the unavailable-editor state stating the reason, with generic entry editing unaffected, in `frontend/src/views/aci/AciUnavailable.tsx`

### Story validation

- [ ] T279 [US10] Integration test — author an ACI and a subtree specification on ApacheDS through the structured editors, then edit the raw text, asserting byte-identical transmission and round-trip stability — in `test/integration/us10_aci_test.go`

**Checkpoint**: **M4 complete.** SC-012, SC-013. The "and more" half of the product.

---

## Phase 13: User Story 11 — Author schemas offline in a project (Priority: P5)

**Goal**: A local schema project that can be created empty, from a server, or from schema files;
edited entirely offline; checked for structural errors; exported; and committed to a server through
the standard preview.

**Independent test**: Create a project from the OpenLDAP fixture's schema, add an object class
referencing a new attribute type, run the error check to catch a deliberately dangling reference,
export to schema files, re-import into a fresh project, and confirm the definitions are unchanged.

**Screens**: `6a`

### Contract tests for this story (write first)

- [ ] T280 [P] [US11] Integration test — `PrepareSchemaCommit` refuses to issue a token while the project holds any `Error`-severity problem — in `internal/bridge/schemaproject_test.go`
- [ ] T281 [P] [US11] Integration test — definitions survive an export → re-import file round trip unchanged — in `internal/schema/project_roundtrip_test.go`

### Implementation

- [ ] T282 [US11] Implement `schema.Project` as a directory of plain diffable files with a manifest recording the owned OID namespace, at `projects/<id>/project.json`, in `internal/schema/project.go`
- [ ] T283 [US11] Implement `schema.Problem` detection for dangling references, duplicate OIDs, duplicate names, and circular superior chains in `internal/schema/problems.go`
- [ ] T284 [P] [US11] Implement project creation from empty, from a connected server, and from schema definition files in `internal/schema/projectsource.go`
- [ ] T285 [P] [US11] Record the origin of every imported definition and keep standard schemas as read-only references in `internal/schema/imports.go`
- [ ] T286 [P] [US11] Implement export to OpenLDAP `.schema` files and to `cn=config` LDIF, excluding read-only imports, in `internal/schema/export.go`
- [ ] T287 [US11] Implement `ListSchemaProjects`, `CreateSchemaProject`, `SaveSchemaElement`, `DeleteSchemaElement`, `CheckSchemaProject`, `CompareSchemaProject`, `ExportSchemaProject` in `internal/bridge/schemaproject.go`
- [ ] T288 [US11] Implement `PrepareSchemaCommit` returning a preview token so a schema commit goes through the standard confirmation in `internal/bridge/schemaproject.go`
- [ ] T289 [P] [US11] Emit the schema-project events for check results and commit outcomes in `internal/bridge/events_schema.go`
- [ ] T290 [US11] Implement the Schema Editor perspective with Projects, Schema (flat and hierarchical), Hierarchy, Problems, and Search views in `frontend/src/views/schema/SchemaProjectPerspective.tsx` (screen `6a`)
- [ ] T291 [P] [US11] Implement the per-element editor with Overview and Source tabs covering OID, names, description, superior, syntax, matching rules, usage, and the single-valued and obsolete flags in `frontend/src/views/schema/ElementEditor.tsx`
- [ ] T292 [P] [US11] Implement the Problems view with navigation to the offending definition and errors blocking export in `frontend/src/views/schema/ProblemsView.tsx`
- [ ] T293 [P] [US11] Implement merge-from-project and project-versus-server comparison in `frontend/src/views/schema/MergeFromProject.tsx`

### Story validation

- [ ] T294 [US11] Integration test — create a project from OpenLDAP, add an object class with a dangling superior, check, export, re-import, and commit to the server, asserting that all three problem classes are reported, errors block export, and the commit previews before writing — in `test/integration/us11_project_test.go`
- [ ] T295 [US11] Integration test — a server that does not permit schema modification reports the refusal with its own message and leaves the project unchanged — in `test/integration/us11_refusal_test.go`

**Checkpoint**: US11 is independently shippable. SC-014.

---

## Phase 14: User Story 12 — Edit server configuration held as directory entries (Priority: P5)

**Goal**: Browse and edit a configuration naming context through the same tree, editors, preview,
and history as any other part of the directory, with the heightened blast radius made explicit.

**Independent test**: Against the OpenLDAP fixture, browse the `cn=config` subtree, change a
setting, and confirm the preview shows the attribute-level diff and that the change takes effect.

**Screens**: `1b` (config context), `6c` (config editing entry points)

### Contract tests for this story (write first)

- [ ] T296 [P] [US12] Integration test — a configuration change the server rejects surfaces the verbatim result code and diagnostic and presents no partial state as success — in `test/integration/us12_reject_test.go`

### Implementation

- [ ] T297 [US12] Implement configuration naming context discovery from the root DSE in `internal/ldapx/configcontext.go`
- [ ] T298 [US12] Mark a configuration target in the preview and attach the immediate-effect warning at the `changeset` boundary in `internal/changeset/configtarget.go`
- [ ] T299 [P] [US12] Implement `olc*` attribute rendering driven by the configuration schema in `internal/schema/olcattrs.go`
- [ ] T300 [P] [US12] Implement configuration snapshot capture and restore reusing the compare engine for snapshot diffs in `internal/compare/configsnapshot.go`
- [ ] T301 [US12] Render the configuration context in the standard tree with database, overlay, and module entries recognised by type in `frontend/src/views/browser/ConfigContext.tsx`
- [ ] T302 [P] [US12] Implement the configuration preview warning banner in `frontend/src/views/browser/ConfigWarning.tsx`
- [ ] T303 [P] [US12] Implement the unavailable state for a directory that exposes no configuration context in `frontend/src/views/browser/ConfigUnavailable.tsx`

### Story validation

- [ ] T304 [US12] Integration test — edit a `cn=config` entry on OpenLDAP, asserting the preview marks the target as server configuration, warns, and that the change takes effect — in `test/integration/us12_config_test.go`

**Checkpoint**: US12 is independently shippable.

---

## Phase 15: User Story 13 — Run bulk operations with a dry run (Priority: P5)

**Goal**: One change applied to every entry a search matches, previewed in full, dry-runnable, then
executed with progress, cancellation, throttling, and a per-entry report.

**Independent test**: Select 500 entries by filter, apply an attribute change, confirm the dry run
writes nothing and reports the same per-entry outcomes the real run later produces, then execute
and verify every entry.

**Screens**: `6c`, `2e` (bulk entry points)

### Contract tests for this story (write first)

- [ ] T305 [P] [US13] Integration test — a dry run writes nothing, verified against a read-only connection, and projects the same per-entry outcomes as the real run for at least 99% of entries — in `test/integration/us13_dryrun_test.go`
- [ ] T306 [P] [US13] Integration test — cancellation stops at an entry boundary and reports precisely which entries were changed and which were not — in `test/integration/us13_cancel_test.go`

### Implementation

- [ ] T307 [US13] Implement the bulk target set from a search result, a manual selection, or a DN file, resolving unresolvable DNs before execution in `internal/jobs/bulktarget.go`
- [ ] T308 [US13] Implement the bulk operation templates — modify LDIF fragment, delete, modify DN, compare — with per-entry substitution from the entry's own attributes in `internal/jobs/bulktemplate.go`
- [ ] T309 [US13] Implement `PrepareBulk` returning a preview token and `StartBulk(token, mode)` so the bulk path cannot skip the preview in `internal/bridge/bulk.go`
- [ ] T310 [P] [US13] Implement batch size and inter-batch pause throttling (gap G5) in `internal/jobs/batch.go`
- [ ] T311 [P] [US13] Implement `ExportJobReport` distinguishing succeeded, failed with verbatim message, and skipped in `internal/bridge/bulk.go`
- [ ] T312 [P] [US13] Emit `job:outcome` per entry with status `succeeded｜failed｜skipped` in `internal/jobs/outcome.go`
- [ ] T313 [US13] Implement the Batch Operation wizard — entry set, operation, execution options with continue-on-error, dry run, and LDIF log — with a 3-record preview and estimated run time in `frontend/src/views/compare/BatchWizard.tsx` (screen `6c`)
- [ ] T314 [P] [US13] Implement the per-entry report view with export and a re-run of the failed set in `frontend/src/views/compare/BulkReport.tsx`

### Story validation

- [ ] T315 [US13] Integration test — apply one change to 500 entries, dry run then execute, verifying every entry — in `test/integration/us13_bulk_test.go`

**Checkpoint**: **M5 complete.** SC-014, full parity per the spec's Parity Reference appendix.

---

## Phase 16: Polish & Cross-Cutting Concerns (Milestone M6 — Release)

**Purpose**: Signing, distribution, accessibility, localisation, and the cross-cutting success
criteria that can only be verified once every story exists.

### Cross-cutting verification

- [ ] T316 [P] Verify SC-002 — instrument the event loop during a 100k search and a bulk write, asserting no block exceeds 100 ms — in `test/integration/sc002_responsiveness_test.go`
- [ ] T317 [P] Verify SC-003 — cancel every `JobID`-returning method mid-flight, asserting each stops within 2 s — in `test/integration/sc003_cancellation_test.go`
- [ ] T318 [P] Verify SC-006 — force a rejection on every mutation path, asserting the server's code and diagnostic appear unmodified in each — in `test/integration/sc006_verbatim_test.go`
- [ ] T319 [P] Verify SC-008 — run a session exercising all 13 stories, then scan every written file, log, and crash artefact for secret material — in `test/secrets_scan/full_session_test.go`
- [ ] T320 [P] Verify SC-016 — run the full suite against the `poor` fixture, asserting every unavailable capability is announced and nothing fails silently — in `test/integration/sc016_degradation_test.go`
- [ ] T321 Verify SC-017 — one automated test per edge case, covering 100% of the spec's 23 edge cases — in `test/integration/edgecases_test.go`
- [ ] T322 [P] Contract test F6 — `PurgeCache` removes orphaned profile caches too — in `internal/prefs/cache_test.go`

### Accessibility, theming, and localisation

- [ ] T323 [P] Implement the Appearance preference pane — theme including high contrast and follow-system, accent, LDIF token colours, colourblind-safe diff pair, fonts, row density, zoom, truncation threshold, live preview, settings export — in `frontend/src/preferences/AppearancePane.tsx` (screen `3a`, gap G2)
- [ ] T324 [P] Implement keyboard-only operability across every view, verified against the command registry, in `frontend/src/commands/KeyboardNav.tsx`
- [ ] T325 [P] Add accessible names and roles to every interactive element and verify with a screen reader in `frontend/tests/a11y/screenreader.test.tsx`
- [ ] T326 [P] Verify WCAG 2.1 AA colour contrast in light, dark, and high-contrast themes in `frontend/tests/a11y/contrast.test.ts`
- [ ] T327 [P] Externalise all user-facing strings and ship one additional locale as proof (FR-107) in `frontend/src/i18n/`
- [ ] T328 [P] Implement the Updates & about preference pane with update checks defaulting to off per D9 and the extension list dropped in `frontend/src/preferences/UpdatesPane.tsx` (screen `5h`)

### Packaging and release

- [ ] T329 Implement the release pipeline producing signed and notarised macOS builds, Authenticode-signed Windows builds, and a Linux package in `.github/workflows/release.yml`
- [ ] T330 [P] Wire signing keys as CI secrets with no key material in the repository in `.github/workflows/release.yml`
- [ ] T331 [P] Implement file-format migration handling for a lower `schemaVersion` on upgrade in `internal/prefs/migrate.go`
- [ ] T332 [P] Generate the changelog from merged pull requests and embed version information in the binary in `.github/workflows/release.yml`
- [ ] T333 Verify SC-009 — cold start with a stored credential present reaches a usable window under 3 s with no prompt until the first bind — in `test/integration/sc009_startup_test.go`
- [ ] T334 [P] Verify SC-018 and close the Parity Reference to 100% resolution in `specs/001-open-ldap-studio/spec.md`

### Documentation and governance

- [ ] T335 [P] Write the getting-started guide covering install, first connection, and first search in `docs/getting-started.md`
- [ ] T336 [P] Write reference documentation for each of the 13 user stories in `docs/reference/`
- [ ] T337 [P] Write the threat model covering credential storage, TLS trust, update delivery, and local persistence in `docs/security/threat-model.md`
- [ ] T338 [P] Write the disclosure policy and response times in `SECURITY.md`
- [ ] T339 Amend the constitution to record the binding tech stack — Go 1.26 + Wails v2.15.0, go-ldap/ldap v3.4.14, and the three native credential bindings — as a MINOR bump to 1.4.0, in its own pull request, in `.specify/memory/constitution.md`

**Checkpoint**: **M6 complete.** SC-009, SC-018. Release.

---

## Dependencies

### Phase dependencies

```text
Phase 1 Setup
    ↓
Phase 2 Foundational (M0) ── blocks every story
    ↓
    ├─→ Phase 3 US1 (P1) ──┬─→ Phase 4 US2 (P1)          [M1]
    │                       │
    │                       ↓
    ├─────────────────────→ Phase 5 US3 (P2) ─→ Phase 6 US4 (P2)   [M2]
    │                            ↓                   ↓
    │                       ┌────┴───────────────────┴────┐
    │                       ↓            ↓                ↓
    │                  Phase 7 US5   Phase 8 US6    Phase 9 US7     [M3]
    │                       ↓            ↓                ↓
    │                  Phase 10 US8  Phase 11 US9   Phase 12 US10   [M4]
    │                       ↓            ↓                ↓
    │                  Phase 13 US11 Phase 14 US12  Phase 15 US13   [M5]
    │                                    ↓
    └──────────────────────────────→ Phase 16 Polish (M6)
```

### Story dependencies

| Story | Hard prerequisite | Why |
|-------|-------------------|-----|
| US1 | Phase 2 | Nothing connects without `ldapx`, `secrets`, and `jobs` |
| US2 | US1 | Search needs a connection and the entry view |
| US3 | Phase 2, US1 | The `changeset` pipeline exists in M0; the editor needs the entry view |
| US4 | US3 | Import applies change records through the write pipeline |
| US5 | US1 | Schema reading needs a connection; editing assistance needs the editor from US3 |
| US6 | US5 | The editor registry keys off syntax information |
| US7 | US3 | Move and copy dispatch through `changeset` |
| US8 | US3 | There is nothing to journal until writes exist |
| US9 | US4 | Reconciling output is LDIF |
| US10 | US3, US5 | ACI editing is schema-aware entry editing |
| US11 | US5 | The project model reuses the schema parser |
| US12 | US3 | Configuration editing is entry editing with a warning |
| US13 | US2, US3, US8 | Target selection, the write pipeline, and journalling |

US5, US6, and US7 are mutually independent once US3 lands, as are US8, US9, and US10, and US11,
US12, and US13. Each group can be built by parallel teams.

### Within a story

Contract tests → data model → Go implementation → bridge methods → frontend → story validation.
Tasks marked [P] touch different files and have no ordering constraint between them.

---

## Parallel execution examples

**Phase 2, architectural gates** — six independent test files, one developer each:

```text
T013, T014, T015, T016, T017, T018
```

**Phase 2, platform credential bindings** — three platforms, three files, no shared state:

```text
T033 (Linux Secret Service) · T034 (macOS Keychain) · T035 (Windows Credential Manager)
```

**Phase 2, test fixtures** — three containers built concurrently:

```text
T019 (openldap) · T020 (apacheds) · T021 (poor)
```

**Phase 3 (US1), frontend views** — twelve independent components:

```text
T110, T112, T113, T114, T115, T116, T117, T118, T119, T120, T121, T122
```

**Phase 6 (US4), format writers** — four independent output formats:

```text
T176 (DSML) · T177 (CSV/JSON) · T178 (XLSX/ODS) · T175 (diagnostics)
```

**Phase 8 (US6), value editors** — nine editors, one file each:

```text
T217, T218, T219, T220, T221, T222, T223, T224, T225
```

**Phase 16, cross-cutting verification** — six success criteria in parallel:

```text
T316, T317, T318, T319, T320, T322
```

**Across teams after M2** — three stories in parallel:

```text
Team A: Phase 7 (US5) → Phase 13 (US11)
Team B: Phase 8 (US6) → Phase 11 (US9)
Team C: Phase 9 (US7) → Phase 12 (US10) → Phase 15 (US13)
```

---

## Implementation strategy

### MVP scope

**Phase 1 + Phase 2 + Phase 3 (US1)** is the minimum shippable product: a read-only directory
browser with connection management, OS-keychain credentials, certificate trust, and a DIT tree that
handles a hundred thousand children. It contains no code path capable of writing to a directory,
which makes it safe to put in front of an administrator on day one.

Adding **Phase 4 (US2)** completes M1 and delivers the full read-only product described in
plan.md — the point at which the tool replaces the read-only half of a practitioner's daily work.

### Incremental delivery

| Increment | Phases | Delivers |
|-----------|--------|----------|
| M0 | 1–2 | Nothing user-visible. A write with no preview becomes inexpressible |
| M1 | 3–4 | Shippable read-only browser and search |
| M2 | 5–6 | Writes and interchange. First release that can damage a directory |
| M3 | 7–9 | Schema fluency, value editors, restructuring |
| M4 | 10–12 | History, comparison, access control — the "and more" half |
| M5 | 13–15 | Schema projects, configuration, bulk operations. Full parity |
| M6 | 16 | Signed, documented, accessible release |

### What must not be reordered

Phase 2 is not optional groundwork that can be deferred under schedule pressure. Contracts C1, C2,
C11, and C13 are architectural: each costs minutes on day one and a refactor of every call site
later. The `changeset` pipeline in particular must exist **before** the first write path, or
Constitution I becomes a convention reviewers must police rather than a property the API enforces.

### Design deviations to land before their milestone

Deviations D1–D9 are changes to the design project, not the code. Two change *what gets built*:

- **D2** — delete the credential vault, auto-lock, `Reveal`, and master password from the design.
  Land before **M0** (Phase 2), since T032–T037 and T114 build the replacement.
- **D3** — remove auto-save from the entry editor. Land before **M2** (Phase 5), since T155 and
  T163 build the staging model that replaces it.

### Blocked tasks

T273 and T277 (effective rights) are designed but have no functional requirement. They are recorded
as gap G1 and must not be built until a requirement exists. The same discipline applies to gaps
G2–G7: each is recorded in contracts/screens.md, and none is absorbed silently.
