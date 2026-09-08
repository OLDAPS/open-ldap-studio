# Open LDAP Studio — MVP Plan

This file is the **MVP scope map over the GitHub tracker**. The tracker
(`OLDAPS/open-ldap-studio`) is the source of truth for state; this file is the
source of truth for *order* and for where the MVP line falls.

Hierarchy, on GitHub and here: **Milestone → Epic / User story → Story → Task**.
Epics (`epic`) and user stories (`user-story`) are the roots; their sub-issues
are stories; those stories' sub-issues are tasks (`T001`…`T339`). A story
labelled `is-task` has no sub-tasks and is itself the unit of work.

Every line below carries its issue number, so `#186` here is issue #186 there.

## Milestone order

| # | Milestone | Source on GitHub | Issues | Exit criteria |
|---|---|---|---|---|
| **M0** | Foundation and delivery pipeline | GH `M0` epics #1, #7 + #99 pulled forward from GH `M6` | 59 | `make build` and `make test` green on all three platforms in CI; a tag publishes an artifact and release notes |
| **M1** | Secrets and enforcement machinery | GH `M0` epic #103 | 43 | Secrets live in the OS keychain; the changeset boundary is the only write path and the contract tests prove it |
| **M2** | Connection experience | GH `M1` / US1 stories #15, #16, #17, #20, #112 | 39 | A profile can be created, tested, saved, edited and opened; status and events are visible |
| **M3** | DIT screen | GH `M1` / US1 stories #18, #19, #111 | 16 | The tree browses a large directory without stalling; an entry shows its attributes |
| **M4** | Search screen | GH `M1` / US2 #21 | 26 | A raw RFC 4515 filter runs, results page and export, searches save and re-run |
| **M5** | MVP release | trimmed from GH `M6` (#99 tail, #100, #101) | 8 + 3 new | Installable build, getting-started guide, security review; read-only product ships |
| | **— MVP line —** | | | |
| M6 | Writes and interchange | GH `M2` (US3 #28, US4 #35) | 61 | |
| M7 | Fluency | GH `M3` (US5 #42, US6 #48, US7 #54) | 67 | |
| M8 | Safety net | GH `M4` (US8 #60, US9 #66, US10 #72) | 57 | |
| M9 | Specialist | GH `M5` (US11 #78, US12 #84, US13 #90) | 54 | |
| M10 | General availability | rest of GH `M6` (#97, #98, #102, #200) | 20 | Signed installers, updates, accessibility, localisation, SC verification |

## What moved, and why

The tracker and `MVP.md` disagreed about what "MVP" means. The tracker treats
all thirteen user stories as the product and puts release last, at `M6`. The
MVP file treats a **read-only directory explorer** as the shippable thing:
connect, browse, search, release. This plan takes the MVP file's definition and
re-cuts the tracker around it.

1. **The MVP line now falls after search, not after US13.** `MVP.md`'s M5 "MVP
   Release" arrives once the connection, DIT and search screens work. On the
   tracker that release sat behind writes, fluency, safety-net and specialist
   work — four milestones and 239 issues that the MVP file never asked for.
   Those become M6–M9, unchanged in content.

2. **Secrets is promoted to its own milestone (M1).** `MVP.md` listed Secrets as
   a standalone milestone; the tracker buries it inside `M0`'s epic #103
   alongside the jobs registry and the changeset pipeline. Epic #103 is kept
   whole and lifted out of M0 — it is enforcement machinery, not skeleton, and
   splitting it would break the point of building the write-guard before there
   is anything to guard.

3. **Release automation moves up from GH `M6` to M0.** `MVP.md` put "create a
   tag for each build / publish binary artifact / use google PR release /
   create release notes" in M0. On the tracker that is #99 (T331, T332) in the
   last milestone. Tag-and-publish is cheap and wants to exist before there are
   builds worth keeping; **signing and notarisation (#97) stay late**, in M10,
   because they need keys and a legal entity, not because they are hard.

4. **US1 splits across M2 and M3.** The tracker's US1 bundles the connection
   manager and the DIT browser into one 39-issue story. `MVP.md` keeps
   "Connection Screen" and "DIT Screen" as separate milestones, which is the
   better cut: the connection work is finished and testable long before the
   tree is. Referral handling (#111) follows the DIT, since its surface is
   Go-to-DN and the referral chooser.

## Where MVP.md and the tracker actually disagree

These are scope conflicts, not ordering. Each needs a decision; none is
resolved here.

- **Single connection at a time.** `MVP.md`: "Ensure there is only one open
  connection per time." The tracker assumes the opposite throughout — #271 runs
  a saved search against a *different* connection, US7 copies entries *across
  servers*, US9 compares *two* connections. The single-connection rule is
  implementable for the MVP, but it contradicts M7–M9 and would have to be
  lifted before them. The post-MVP "Profile Workspace" item is exactly that
  lifting.
- **StartTLS is missing from MVP.md.** It lists "No Encryption and LDAPS" only.
  The tracker implements StartTLS in T045 (#157) and tests it in #251. Dropping
  it saves nothing — it is the same code path — so the tracker's coverage is
  kept here.
- **Bind mechanisms.** `MVP.md` asks for Simple and Anonymous. The tracker adds
  SASL EXTERNAL, GSSAPI, DIGEST-MD5 and an in-house CRAM-MD5 (#158, #159), and
  CRAM-MD5 is a stated functional requirement (FR-022). Kept, but they are the
  first thing to cut from M2 if it runs long.
- **The Events panel.** `MVP.md` wants the bottom progress bar *replaced* by an
  Events list showing the last 5 events. T074 (#186) instead specifies a bottom
  panel with Progress, Modification Logs, Search Logs, Errors and Console tabs.
  These are different designs. The tracker's is broader and already has the log
  views behind it (#378, #379); MVP.md's is a single unified feed. **Decide
  before #186 is built.**
- **Connection status sub-screen.** `MVP.md` wants a double-click sub-screen
  showing user, base DN, state, host, encryption and mode. The tracker spreads
  this across the status bar (T073 #185) and the connection properties dialog
  (T122 #250). No dedicated sub-screen issue exists — file one under #15 if the
  design stands.
- **Preference panes (#112) are not in MVP.md at all.** Seven issues, including
  the keyboard-shortcuts pane. Parked in M2 because the connections and
  security panes are needed there; the browser, appearance and shortcuts panes
  could defer.

---

## M0 — Foundation and delivery pipeline

> GH milestone `M0 — Skeleton and guarantees`, epics #1 and #7, plus #99 pulled
> forward from `M6`. Nothing user-facing ships here.

### Epic #1 — Repository skeleton, desktop shell, and build guarantees

#### Story #2 — Scaffold the Go module and repository layout
- [ ] #113 T001 Initialise the Go module and pin the toolchain to Go 1.26 in `go.mod`
- [ ] #115 T003 Create the `internal/` package skeleton with doc.go files declaring each package's boundary in `internal/{ldapx,changeset,ldif,dsml,schema,aci,credentials,secrets,profiles,trust,history,compare,jobs,exportx,commands,prefs,logging,bridge}/doc.go`

#### Story #3 — Bootstrap the desktop shell (Go core + webview)
- [ ] #114 T002 Scaffold the Wails v2 application entrypoint and window in `main.go` and `wails.json`
- [ ] #116 T004 Scaffold the React 19 + Vite 7 + TypeScript 5.9 frontend in `frontend/package.json`, `frontend/vite.config.ts`, `frontend/tsconfig.json`
- [ ] #117 T005 Add Zustand, TanStack Virtual, and CodeMirror 6 dependencies in `frontend/package.json`

#### Story #4 — Continuous integration across Linux, macOS, and Windows
- [ ] #121 T009 Add the CI build matrix for Linux, macOS, and Windows runners in `.github/workflows/ci.yml`
- [ ] #122 T010 Add the quality-gate CI job running govulncheck, `npm audit --audit-level=high`, and `go mod verify` in `.github/workflows/gates.yml`

#### Story #5 — Reproducible developer environment and task runner
- [ ] #118 T006 Configure golangci-lint in `.golangci.yml` and gofumpt formatting in the Makefile
- [ ] #119 T007 Configure ESLint, Prettier, and Vitest in `frontend/.eslintrc.cjs` and `frontend/vitest.config.ts`
- [ ] #120 T008 Create the task runner with `build`, `test`, `test-integration`, `lint`, `fmt`, `run`, and `gates` targets in `Makefile`

#### Story #6 — Contribution standards, templates, and coding conventions
- [ ] #123 T011 Add CODEOWNERS requiring second-maintainer review on `internal/changeset/`, `internal/secrets/`, `internal/trust/`, and TLS code in `internal/ldapx/` in `.github/CODEOWNERS`
- [ ] #124 T012 Document the platform build prerequisites (webkit2gtk-4.1, Xcode CLT, WebView2) in `CONTRIBUTING.md`

#### Story #110 — Application shell: menus, activity rail, status bar, and panels
- [ ] #183 T071 Implement the window chrome and the 8 top-level menus, with the Window menu and `File › New › Server instance…` removed per D1, in `frontend/src/shell/MenuBar.tsx`
- [ ] #184 T072 Implement the 6-icon activity rail — connections, DIT browser, searches, schema, LDIF & files, preferences — in `frontend/src/shell/ActivityRail.tsx`
- [ ] #185 T073 Implement the always-visible status bar showing connection state, bind DN, target server, entry counts, page N/M, and active limits (FR-010) in `frontend/src/shell/StatusBar.tsx`
- [ ] #186 T074 Implement the bottom panel with Progress, Modification Logs, Search Logs, Errors, and Console tabs in `frontend/src/shell/BottomPanel.tsx`
- [ ] #187 T075 Implement the document tab host, so searches and LDIF files are documents rather than modals, in `frontend/src/shell/DocumentTabs.tsx`
- [ ] #188 T076 Implement the generated Wails bindings wrapper and typed frontend client in `frontend/src/bridge/client.ts`
- [ ] #189 T077 Implement the command, menu, keymap, and context-menu layer over the one registry in `frontend/src/commands/CommandProvider.tsx`
- [ ] #190 T078 Implement the event subscription layer for `job:*`, `conn:*`, `trust:*`, and `capability:*` in `frontend/src/bridge/events.ts`

### Epic #7 — LDAP core engine and integration test harness

#### Story #8 — Connection layer: LDAP, LDAPS, and StartTLS
- [ ] #157 T045 Implement dial, LDAPS, StartTLS, keep-alive, and the reconnect policy in `internal/ldapx/conn.go`
- [ ] #160 T048 Implement Root DSE reading — naming contexts, supported controls, extensions, SASL mechanisms — in `internal/ldapx/rootdse.go`

#### Story #9 — Bind mechanisms: simple, SASL EXTERNAL, and GSSAPI
- [ ] #158 T046 Implement simple, anonymous, SASL EXTERNAL, GSSAPI, and DIGEST-MD5 binds in `internal/ldapx/bind.go`
- [ ] #159 T047 Implement the in-house CRAM-MD5 bind mechanism (FR-022, not provided by go-ldap) in `internal/ldapx/cram_md5.go`
- [ ] #166 T054 Integration test — every bind method succeeds against the OpenLDAP fixture over plain, StartTLS, and LDAPS — in `test/integration/bind_test.go`

#### Story #10 — Search primitives: paging, referrals, and streaming results
- [ ] #162 T050 Implement paged search with alias and referral policy and bounded traversal depth in `internal/ldapx/search.go`

#### Story #11 — Controls and extended operations framework
- [ ] #161 T049 Implement the Password Modify and Who Am I extended operations in `internal/ldapx/extended.go`
- [ ] #163 T051 Implement the paging, server-side sort, subtree delete, and ManageDsaIT controls in `internal/ldapx/controls/standard.go`
- [ ] #164 T052 Implement the in-house VLV control (FR-002, not provided by go-ldap) in `internal/ldapx/controls/vlv.go`

#### Story #12 — Integration test harness with containerised directories
- [ ] #131 T019 Define the OpenLDAP baseline container with `cn=config` enabled in `test/containers/openldap.go`
- [ ] #132 T020 Define the ApacheDS container providing X.500 `prescriptiveACI`, subentries, and administrative roles in `test/containers/apacheds.go`
- [ ] #133 T021 Define the capability-poor container — no paging, no readable subschema, no extended operations — in `test/containers/poor.go`
- [ ] #134 T022 Build the testcontainers-go harness with fixture seeding and the `integration` build tag in `test/containers/harness.go`
- [ ] #135 T023 Assemble the fidelity corpus — binary, non-UTF-8, empty, option-bearing (`;binary`, `;lang-de`), and multi-megabyte values — in `test/corpus/`
- [ ] #136 T024 Validate the ApacheDS fixture containerises and serves ACI reads, discharging risk R-6 before M4 depends on it, in `test/containers/apacheds_test.go`

#### Story #13 — Error taxonomy, structured logging, and protocol tracing
- [ ] #137 T025 Implement the four log sinks with rotation at 10 MB × 3 in `internal/logging/sink.go`
- [ ] #138 T026 Implement redaction at write time, applied before any byte reaches a sink, in `internal/logging/redact.go`
- [ ] #139 T027 Contract test F8 — logs rotate at 10 MB × 3 and redact before writing — in `internal/logging/redact_test.go`
- [ ] #140 T028 Implement `ldapx.Result` carrying `resultCode`, `matchedDN`, and `diagnosticMessage`, never collapsed to a bool, in `internal/ldapx/result.go`
- [ ] #141 T029 Implement the error category taxonomy and the `Indeterminate` state in `internal/ldapx/errors.go`
- [ ] #142 T030 Contract test X2 — no code path between `ldapx` and the bridge converts a `Result` to `bool` or a bare `error` — in `test/architecture/result_model_test.go`
- [ ] #143 T031 Contract test X3 — `DiagnosticMessage` is byte-identical to the server's, asserted against a server returning a deliberately odd message — in `test/integration/error_model_test.go`

### From GH `M6` epic #96 — Release: pulled forward

> `MVP.md` puts tag-per-build, artifact publishing, Release Please and release
> notes in M0. Only signing and notarisation stay late.

#### Story #99 — Release automation: versioning, changelog, and publishing

- [ ] #459 T331 Implement file-format migration handling for a lower `schemaVersion` on upgrade in `internal/prefs/migrate.go`
- [ ] #460 T332 Generate the changelog from merged pull requests and embed version information in the binary in `.github/workflows/release.yml`

> **Not on the tracker, delivered in `ci/release-pipelines`** — `MVP.md` asked
> for these and no issue covered them:
> - [x] Tag per release and publish binaries as GitHub Release assets, with `SHA256SUMS` — `.github/workflows/release.yml`
> - [x] Adopt `googleapis/release-please-action` for the version/changelog pull request — `release-please-config.json`, `.release-please-manifest.json`
> - [x] Wire `golangci-lint` into `ci.yml`, gated with `only-new-issues` — T006 (#118) had configured it without running it
> - [x] Conventional-commit enforcement on pull request titles, which Release Please depends on
> - [x] `internal/version` as the single source of release identity, stamped at link time
>
> The repository carries **232 pre-existing golangci-lint findings** (146 revive,
> 64 gofmt, 10 gosec, and one real bug: `EntryVersion(a) != EntryVersion(a)` in
> `internal/changeset/pipeline_test.go:344`). The lint gate is scoped to new
> code so it does not block on that backlog; the cleanup belongs to #118.

---

## M1 — Secrets and enforcement machinery

> GH `M0` epic #103, lifted whole into its own milestone. `MVP.md` tracked
> Secrets as a standalone milestone; this gates every later write.

### Epic #103 — Enforcement machinery: secrets, jobs, and the changeset write pipeline

#### Story #104 — Architectural contract tests and CI import boundaries
- [ ] #125 T013 Contract test C1 — `ldapx` mutation functions have exactly one calling package, `changeset` — in `test/architecture/import_boundaries_test.go`
- [ ] #126 T014 Contract test C2 — no package outside `secrets` imports a platform credential API — in `test/architecture/import_boundaries_test.go`
- [ ] #127 T015 Contract test C11 — no vault method (`LockVault`, `UnlockVault`, `SetVaultStorage`, `ExportVault`, `ImportVault`, `RevealSecret`, master-password) exists on the bound bridge surface — in `test/architecture/bridge_surface_test.go`
- [ ] #128 T016 Contract test — no `os/exec` appears in the transitive import tree of `internal/secrets` — in `internal/secrets/no_subprocess_test.go`
- [ ] #129 T017 Contract test C9 — no bridge method returns a secret value, asserted over the bound surface by reflection — in `test/architecture/bridge_surface_test.go`
- [ ] #130 T018 Contract test C13 — a read-only profile is refused a preview token at the `changeset` boundary before any UI check — in `internal/changeset/readonly_test.go`

#### Story #105 — Secrets provider with three platform credential bindings
- [ ] #144 T032 Define the `secrets.Provider` interface with no marshaller on `Secret` in `internal/secrets/provider.go`
- [ ] #145 T033 Implement the Linux/BSD Secret Service binding over D-Bus in `internal/secrets/secretservice_linux.go`
- [ ] #146 T034 Implement the macOS Keychain Services binding in `internal/secrets/keychain_darwin.go`
- [ ] #147 T035 Implement the Windows Credential Manager binding in `internal/secrets/wincred_windows.go`
- [ ] #148 T036 Implement the in-memory session-only fallback for an unavailable or locked agent in `internal/secrets/session.go`
- [ ] #149 T037 Integration test — each platform binding stores, retrieves, and deletes, and a locked agent surfaces a recoverable error — in `internal/secrets/provider_test.go`

#### Story #106 — Cancellable job registry with progress, dry-run, and throttling
- [ ] #150 T038 Implement the cancellable job registry with `context.Context` propagation in `internal/jobs/registry.go`
- [ ] #151 T039 Implement job progress reporting throttled to ≤ 20 events/s in `internal/jobs/progress.go`
- [ ] #152 T040 Implement the `Execute` and `DryRun` job modes, where dry run replaces the dispatch call with a recorder, in `internal/jobs/mode.go`
- [ ] #153 T041 Implement job throttling, pause, and batch-size control in `internal/jobs/throttle.go`
- [ ] #154 T042 Contract test C8 — every `JobID`-returning method has a `Cancel` effective within 2 s — in `test/integration/jobs_cancel_test.go`
- [ ] #155 T043 Contract test E2 — `job:progress` is throttled; a 100k-entry job emits ≤ 20 events/s — in `internal/jobs/progress_test.go`
- [ ] #156 T044 Contract test E3 — every job reaching a terminal state emits exactly one `job:finished` — in `internal/jobs/registry_test.go`

#### Story #107 — Changeset preview pipeline: the only write path
- [ ] #165 T053 Implement the add, modify, modrdn, and delete dispatch functions — package-private to all callers but `changeset` — in `internal/ldapx/modify.go`
- [ ] #167 T055 Implement the `ChangeSet` and `Operation` model in `internal/changeset/changeset.go`
- [ ] #168 T056 Implement `Preview()` returning the attribute-level diff, affected count, and a single-use expiring `PreviewToken` carrying `EntryVersions` in `internal/changeset/preview.go`
- [ ] #169 T057 Implement `dispatch` as the sole caller of `ldapx` mutation functions in `internal/changeset/dispatch.go`
- [ ] #170 T058 Implement read-only and production-tagged profile refusal at the token-issue boundary in `internal/changeset/readonly.go`
- [ ] #171 T059 Contract test C4 — `Commit` rejects an absent, reused, expired, or stale token — in `internal/changeset/preview_test.go`
- [ ] #172 T060 Contract test X4 — a connection dropped mid-write yields `Indeterminate`, never success or failure — in `test/integration/indeterminate_test.go`

#### Story #108 — On-disk format envelope and the no-secrets-on-disk scan
- [ ] #173 T061 Implement the `schemaVersion`-first file envelope and the refuse-on-higher-version rule in `internal/prefs/storefile.go`
- [ ] #174 T062 Contract test F1 and F2 — every written file has `schemaVersion` as its first key, and a higher version is refused rather than rewritten — in `internal/prefs/storefile_test.go`
- [ ] #175 T063 Implement the SC-008 secrets scan over every written file, log, and crash artefact in `test/secrets_scan/scan_test.go`
- [ ] #176 T064 Contract tests F3 and E1 — no written file and no emitted event payload contains secret material across a session exercising all stories — in `test/secrets_scan/scan_test.go`

#### Story #109 — Bridge skeleton and command registry
- [ ] #177 T065 Implement the Wails-bound bridge skeleton exposing `Preview`, `Commit`, `Discard`, and no `Modify`/`Add`/`Delete`/`Rename` method in `internal/bridge/changeset.go`
- [ ] #178 T066 Implement the jobs bridge methods `Cancel`, `JobState`, `ListJobs` in `internal/bridge/jobs.go`
- [ ] #179 T067 Implement the command registry with ids, labels, menu placement, enablement predicates, and scope in `internal/commands/registry.go`
- [ ] #180 T068 Implement keymap presets, conflict detection, and OS-reserved-chord refusal in `internal/commands/keymap.go`
- [ ] #181 T069 Contract test C14 — every command has an enablement predicate and a non-conflicting default binding per platform — in `internal/commands/registry_test.go`
- [ ] #182 T070 Contract test C7 and X1 — every bridge method contacting a server returns a populated `Result` on success and failure — in `test/architecture/result_model_test.go`

---

## M2 — Connection experience

> GH `M1` / US1 (#14), stories #15, #16, #17, #20, #112. Covers `MVP.md`'s
> "M1 Connection Screen" and "M2 Secrets" user-facing surface.

### User story #14 — Connect to a directory and explore

#### Story #15 — Connection manager: create, test, edit, duplicate, delete
- [ ] #191 T079 Contract test C3 — `SaveProfile` rejects every payload carrying secret material — in `internal/bridge/profiles_test.go`
- [ ] #193 T081 Contract test F4 — a profile bundle round-trips export → import with no field loss — in `internal/profiles/bundle_test.go`
- [ ] #198 T086 Implement `profiles.Profile` and `profiles.TLSPolicy` with no field capable of holding a secret in `internal/profiles/model.go`
- [ ] #199 T087 Implement the profile and folder store with cycle rejection on folders in `internal/profiles/store.go`
- [ ] #216 T088 Implement profile bundle export and import in `internal/profiles/bundle.go`
- [ ] #222 T094 Implement the connection bridge methods `ListProfiles`, `GetProfile`, `SaveProfile`, `DeleteProfile`, `DuplicateProfile`, `MoveProfile` in `internal/bridge/profiles.go`
- [ ] #223 T095 Implement `ListFolders`, `SaveFolder`, `DeleteFolder`, `ExportProfiles`, `ImportProfiles` in `internal/bridge/folders.go`
- [ ] #224 T096 Implement `Connect`, `Disconnect`, `ConnectionState`, `TestConnection` in `internal/bridge/connections.go`
- [ ] #232 T104 Emit `conn:state`, `conn:reconnected` with `writesRequireConfirmation`, and `conn:lost` in `internal/bridge/events_conn.go`
- [ ] #235 T107 Implement the connections start screen and the 4-step wizard — Network, Authentication, Browser options, Edit options — with the JNDI provider selector dropped per D5, in `frontend/src/views/connections/ConnectionWizard.tsx` (screen `1a`)
- [ ] #251 T123 Integration test — connect over plain, StartTLS, and LDAPS; bind anonymous, simple, EXTERNAL, GSSAPI, DIGEST-MD5, CRAM-MD5; read the root DSE; expand a 25,000-child container; call `WhoAmI` — in `test/integration/us1_connect_test.go`

#### Story #16 — Credential storage in the OS keychain
- [ ] #192 T080 Contract test F7 — `credentials.json` contains no secret material and no vault structure — in `internal/credentials/store_test.go`
- [ ] #196 T084 Contract test X8 — `invalidCredentials` produces no retry loop and no anonymous fallback (D7) — in `test/integration/bind_failure_test.go`
- [ ] #197 T085 Integration test — SC-009 cold start with a stored credential present reaches a usable window in under 3 s with no prompt — in `test/integration/startup_test.go`
- [ ] #217 T089 Implement `credentials.Credential` as a first-class entity with many-to-many profile assignments (research R10) in `internal/credentials/model.go`
- [ ] #218 T090 Implement the credential store persisting metadata only, never a secret value, in `internal/credentials/store.go`
- [ ] #226 T098 Implement the credential bridge methods `ListCredentials`, `GetCredential`, `SaveCredential`, `DeleteCredential`, `AssignCredential`, `UnassignCredential`, `CredentialAssignments`, `TestBind`, `CredentialStoreStatus` in `internal/bridge/credentials.go`
- [ ] #234 T106 Emit `credential:required` and `capability:unavailable` in `internal/bridge/events_capability.go`
- [ ] #236 T108 Implement wizard step 2 with stored-credential selection, the "secret held in OS keychain" notice, check-authentication via whoami, and the clear-text warning strip, with "retry anonymously" dropped per D7, in `frontend/src/views/connections/AuthStep.tsx` (screen `3c`)
- [ ] #242 T114 Implement the credentials management modal — credential list by type, detail, server assignments, test bind, use log, password age — as an in-app modal per D4, with every vault affordance absent per D2, in `frontend/src/views/credentials/CredentialsModal.tsx` (screen `3b`)

#### Story #17 — TLS trust configuration and certificate inspection
- [ ] #194 T082 Contract test E5 — `trust:challenge` never fires without the connection having been refused first — in `test/integration/trust_test.go`
- [ ] #219 T091 Implement `trust.Decision` bound to an exact certificate fingerprint in `internal/trust/model.go`
- [ ] #220 T092 Implement the trust store with session and permanent scopes and re-challenge on certificate rotation in `internal/trust/store.go`
- [ ] #227 T099 Implement `ListTrustDecisions`, `RevokeTrustDecision`, `DecideTrust` in `internal/bridge/trust.go`
- [ ] #233 T105 Emit `trust:challenge` with host, port, fingerprint, chain PEM, failure reason, and prior-trust flag, only after the connection has been refused, in `internal/bridge/events_trust.go`
- [ ] #243 T115 Implement the certificate trust dialog with chain view and trust-once / trust-permanently / reject in `frontend/src/dialogs/CertificateTrustDialog.tsx` (screen `6e`)
- [ ] #252 T124 Integration test — present an untrusted certificate, accept for the session, restart, then rotate the certificate, asserting refusal, challenge payload, non-persistence, and re-challenge — in `test/integration/us1_trust_test.go`

#### Story #20 — Root DSE and naming context discovery on connect
- [ ] #225 T097 Implement `WhoAmI` and `RootDSE` in `internal/bridge/connections.go`

#### Story #112 — Preference panes and keyboard shortcuts
- [ ] #230 T102 Implement `GetPreferences`, `SetPreferences`, `PurgeCache` in `internal/bridge/prefs.go`
- [ ] #231 T103 Implement the keymap bridge methods `ListCommands`, `GetKeymap`, `SetBinding`, `ResetKeymap`, `ExportKeymap` in `internal/bridge/commands.go`
- [ ] #246 T118 Implement the Browser & tree preference pane in `frontend/src/preferences/BrowserPane.tsx` (screen `5a`)
- [ ] #247 T119 Implement the Connections & timeouts preference pane including the modify-request strategy and the open-read-only / production-tag settings in `frontend/src/preferences/ConnectionsPane.tsx` (screen `5e`, gap G3)
- [ ] #248 T120 Implement the Credentials & security preference pane with no vault, auto-lock, or master-password affordance per D2 in `frontend/src/preferences/SecurityPane.tsx` (screen `5f`)
- [ ] #249 T121 Implement the keyboard shortcuts preference pane with searchable command table, presets, live conflict detection, and export in `frontend/src/preferences/ShortcutsPane.tsx` (screen `5g`, gap G4)

> **Not yet on the tracker** — from `MVP.md`'s connection screen:
> - [ ] Connection status sub-screen on double-click, showing bind DN, base DN, state, host, encryption and mode as one panel
> - [ ] Refresh/reconnect button on an open connection
> - [ ] Single-open-connection enforcement, if that rule survives the decision above
> - [ ] Events panel showing the last 5 events, if that design wins over T074's tabbed panel

---

## M3 — DIT screen

> GH `M1` / US1 (#14), stories #18, #19, #111.

### User story #14 — Connect to a directory and explore

#### Story #18 — DIT browser tree with lazy loading and virtualised scrolling
- [ ] #228 T100 Implement `ListChildren`, `ReadEntry`, `RefreshEntry`, `GoToDN` with `EntryPage` carrying `cookie`, `loadedCount`, `serverLimit`, and `truncatedByServer` in `internal/bridge/browse.go`
- [ ] #237 T109 Implement the DIT browser tree with lazy paged children, the explicit "fetch next 100 of 1 842…" node, and TanStack Virtual scrolling in `frontend/src/views/browser/DitTree.tsx` (screen `1b`)
- [ ] #238 T110 Implement the tree filter box and the Searches and Bookmarks nodes under each connection in `frontend/src/views/browser/TreeNodes.tsx`
- [ ] #253 T125 Performance test — SC-004, the first page of a 100,000-child container is visible within 2 s — in `test/integration/us1_perf_test.go`

#### Story #19 — Entry detail view with operational attributes and raw mode
- [ ] #239 T111 Implement the read-only entry view with Attributes, LDIF view, Table, and Object class tabs in `frontend/src/views/browser/EntryView.tsx`
- [ ] #240 T112 Implement the entry-info side panel with structural and auxiliary classes, last modified, photo preview, and "Show in schema browser" in `frontend/src/views/browser/EntryInfoPanel.tsx`
- [ ] #241 T113 Implement the operational-attributes toggle and raw-value display in `frontend/src/views/browser/AttributeTable.tsx`
- [ ] #250 T122 Implement the properties dialogs for connection, entry, attribute, and value in `frontend/src/dialogs/PropertiesDialog.tsx` (screen `6d`)

#### Story #111 — Referral resolution and LDAP URL handling
- [ ] #195 T083 Contract test C12 — `FollowReferral` cannot be called without an explicit target profile — in `internal/bridge/referral_test.go`
- [ ] #221 T093 Implement `ldapx.ReferralTarget` and RFC 4516 LDAP URL parsing in `internal/ldapx/referral.go`
- [ ] #229 T101 Implement `ResolveReferral`, `FollowReferral`, `ParseLDAPURL` in `internal/bridge/referral.go`
- [ ] #244 T116 Implement the Go-to-DN dialog accepting an LDAP URL in `frontend/src/dialogs/GoToDnDialog.tsx` (screen `6e`, gap G6)
- [ ] #245 T117 Implement the referral connection chooser — pick existing, create new, remember for session — in `frontend/src/dialogs/ReferralChooser.tsx` (screen `6e`, research R11)

---

## M4 — Search screen

> GH `M1` / US2 (#21), whole.

### User story #21 — Search with raw filters, saved searches, and bookmarks

#### Story #22 — RFC 4515 filter parser with positional error reporting
- [ ] #254 T126 Contract test C5 — `ValidateFilter` never returns a modified filter string — in `internal/bridge/search_test.go`
- [ ] #255 T127 Integration test — the filter string on the wire is byte-identical to what was typed, captured at the protocol layer — in `test/integration/us2_filter_fidelity_test.go`
- [ ] #257 T129 Implement the RFC 4515 filter parser reporting `{ok, position, message}` and parsing a copy, never rewriting the original, in `internal/ldapx/filter.go`
- [ ] #258 T130 Add a fuzz target for the filter parser in `internal/ldapx/filter_fuzz_test.go`
- [ ] #268 T140 Implement the filter editor dialog with content assist in `frontend/src/dialogs/FilterEditorDialog.tsx` (screen `6e`)
- [ ] #270 T142 Integration test — a nested boolean and extensible-match filter returns a result set identical to `ldapsearch` — in `test/integration/us2_search_test.go`

#### Story #23 — Search view: scope, base, attributes, and limits
- [ ] #256 T128 Integration test — a search past the server size limit shows partial results labelled truncated-by-server with the server's result code — in `test/integration/us2_limits_test.go`
- [ ] #259 T131 Implement `ldapx.SearchDefinition` carrying the filter as a string, plus scope, base, attributes, limits, alias and referral policy, and controls, in `internal/ldapx/searchdef.go`
- [ ] #260 T132 Implement `ValidateFilter`, `StartSearch`, and `FetchNextPage` in `internal/bridge/search.go`
- [ ] #265 T137 Implement the search editor document tab with the filter builder and the raw RFC 4515 field kept in sync in `frontend/src/views/search/SearchEditor.tsx` (screen `1c`)
- [ ] #267 T139 Implement the attribute palette dragging from the schema and the filter history list in `frontend/src/views/search/AttributePalette.tsx`

#### Story #24 — Results table with virtualisation, sorting, and column selection
- [ ] #264 T136 Implement server-side sort control negotiation with a stated reason when unsupported in `internal/ldapx/controls/sort.go`
- [ ] #266 T138 Implement the virtual-scroll result grid with column picker, client-side sort, and selected-row preview in `frontend/src/views/search/ResultGrid.tsx`
- [ ] #272 T144 Performance test — SC-002, no UI block exceeds 100 ms during a 100k-result search — in `test/integration/us2_perf_test.go`

#### Story #25 — Saved searches: persist, organise, and re-run
- [ ] #262 T134 Implement saved-search persistence and the `ListSavedSearches`, `SaveSearch`, `DeleteSavedSearch`, `SearchHistory` bridge methods in `internal/bridge/savedsearch.go`
- [ ] #269 T141 Implement saved-search and bookmark management UI under each connection in `frontend/src/views/search/SavedSearches.tsx`
- [ ] #271 T143 Integration test — a saved search runs unchanged against a different connection and reports that server's results or error — in `test/integration/us2_saved_test.go`

#### Story #26 — Bookmarks for DNs and subtrees
- [ ] #263 T135 Implement bookmark persistence and the `ListBookmarks`, `SaveBookmark`, `DeleteBookmark` bridge methods in `internal/bridge/bookmarks.go`

#### Story #27 — Search history with re-run
- [ ] #261 T133 Implement `history.SearchRecord` and the search log in `internal/history/search.go`

---

## M5 — MVP release

> Trimmed from GH `M6`. Ships the read-only product: install it, learn it, trust
> it. Signing, updates, accessibility and localisation are **not** here — they
> are M10.

### From GH `M6` epic #96 — Release: the MVP subset

#### Story #100 — User documentation and getting-started guide

- [ ] #462 T334 Verify SC-018 and close the Parity Reference to 100% resolution in `specs/001-open-ldap-studio/spec.md`
- [ ] #463 T335 Write the getting-started guide covering install, first connection, and first search in `docs/getting-started.md`
- [ ] #464 T336 Write reference documentation for each of the 13 user stories in `docs/reference/`

#### Story #101 — Security review, threat model, and disclosure policy

- [ ] #465 T337 Write the threat model covering credential storage, TLS trust, update delivery, and local persistence in `docs/security/threat-model.md`
- [ ] #466 T338 Write the disclosure policy and response times in `SECURITY.md`
- [ ] #467 T339 Amend the constitution to record the binding tech stack — Go 1.26 + Wails v2.15.0, go-ldap/ldap v3.4.14, and the three native credential bindings — as a MINOR bump to 1.4.0, in its own pull request, in `.specify/memory/constitution.md`

> Also required to close M5:
> - [ ] Unsigned but installable builds for Linux, macOS and Windows (a subset of #97 — the pipeline without the keys)
> - [ ] Known-issues section in the release notes
> - [ ] Tag `v0.1.0` and publish

---

# Post-MVP

Content unchanged from the tracker; only the milestone numbers shift. Stories
are listed with task counts — the tasks themselves live on the issues.

## M6 — Writes and interchange
> GH `M2`. Every write staged, diffed and confirmed before it reaches the server.

### User story #28 — Modify entries with a previewed, confirmed write · 28 issues

- **#29** Attribute editor: add, edit, and remove values — 5 tasks
- **#30** Change staging model and modification builder — 2 tasks
- **#31** Diff preview of the exact LDIF to be sent — 4 tasks
- **#32** Apply with confirmation and per-attribute result reporting — 4 tasks
- **#33** Create-entry wizard driven by objectClass selection — 3 tasks
- **#34** Delete entry and guarded recursive subtree delete — 4 tasks

### User story #35 — Import and export LDIF and other formats · 31 issues

- **#36** RFC 2849 LDIF parser — 6 tasks
- **#37** LDIF writer with deterministic output — 5 tasks
- **#38** Streaming import runner with per-record error policy — 6 tasks
- **#39** Export from a search result, a selection, or a subtree — 4 tasks
- **#40** CSV, TSV, and JSON export — 3 tasks
- **#41** DSML interchange — 1 task

## M7 — Fluency
> GH `M3`. Schema awareness, specialised value editors, move/rename/copy.

### User story #42 — Inspect the schema and edit with schema awareness · 20 issues

- **#43** Schema fetch and parse from the subschema subentry — 5 tasks
- **#44** Schema browser with cross-references and search — 7 tasks
- **#45** Schema-driven attribute editing — 1 task
- **#46** Client-side syntax and constraint validation — 1 task
- **#47** Schema cache with per-connection invalidation — 1 task

### User story #48 — Edit specialised attribute values safely · 27 issues

- **#49** Password editor with hashing schemes — 5 tasks
- **#50** Certificate and binary value viewer and editor — 4 tasks
- **#51** Generalized time and DN value editors — 4 tasks
- **#52** objectClass editor with required-attribute prompting — 1 task
- **#53** Value editor registry keyed by syntax OID — 8 tasks

### User story #54 — Move, rename, copy, and duplicate entries across servers · 17 issues

- **#55** Modify DN: rename with deleteOldRDN and preview — 2 tasks
- **#56** Subtree move with server-side and client-side paths — 3 tasks
- **#57** Copy, paste, and duplicate within a connection — 3 tasks
- **#58** Cross-server copy with schema compatibility checking — 3 tasks
- **#59** Conflict resolution policy for existing targets — 1 task

## M8 — Safety net
> GH `M4`. History with replay and undo, subtree compare, access control.

### User story #60 — Review, replay, and reverse past operations · 19 issues

- **#61** Operation journal: durable, structured write records — 5 tasks
- **#62** History browser with filtering — 3 tasks
- **#63** Replay selected operations against a chosen connection — 1 task
- **#64** Generate and apply reverse operations — 4 tasks
- **#65** Export the journal as LDIF change records — 1 task

### User story #66 — Compare two directories or subtrees · 16 issues

- **#67** Subtree comparison engine — 5 tasks
- **#68** Configurable ignore rules — 1 task
- **#69** Diff results view — 2 tasks
- **#70** Generate a synchronisation LDIF from a diff — 2 tasks
- **#71** Apply a diff with per-change selection and a dry run — 1 task

### User story #72 — Author access control and administrative policy · 19 issues

- **#73** Parse and render olcAccess rules — 6 tasks
- **#74** ACL rule builder with grammar assistance — 6 tasks
- **#75** Password policy editor — `is-task`
- **#76** Limits and overlay policy editors — `is-task`
- **#77** Effective rights probe — 2 tasks

## M9 — Specialist
> GH `M5`. Offline schema projects, `cn=config`, bulk operations.

### User story #78 — Author schemas offline in a project · 21 issues

- **#79** Schema project model and workspace layout — 3 tasks
- **#80** Offline editors for object classes and attribute types — 2 tasks
- **#81** Project validation: OIDs, cycles, and dangling references — 4 tasks
- **#82** Import an existing server schema into a project — 3 tasks
- **#83** Export to OpenLDAP schema files and cn=config LDIF — 4 tasks

### User story #84 — Edit server configuration held as entries · 14 issues

- **#85** cn=config browser with olc attribute awareness — 4 tasks
- **#86** Guarded editing with restart and reload warnings — 4 tasks
- **#87** Overlay and module management — `is-task`
- **#88** Database and backend configuration editors — `is-task`
- **#89** Configuration snapshot and restore — 1 task

### User story #90 — Run bulk operations with a dry run · 16 issues

- **#91** Bulk target selection by search, selection, or DN file — 1 task
- **#92** Operation templates with per-entry substitution — 2 tasks
- **#93** Dry run producing a full report — 3 tasks
- **#94** Batched execution with rate limiting and resume — 2 tasks
- **#95** Failure report and re-run of failed records — 3 tasks

## M10 — General availability
> The rest of GH `M6`: what a public release needs beyond the MVP.

### From GH `M6` epic #96 — the remainder

- **#97** Signed installers for Linux, macOS, and Windows — 2 tasks
- **#98** Update channel with signature verification — 1 task
- **#102** Accessibility and localisation pass — 5 tasks
- **#200** Cross-cutting success-criteria verification — 8 tasks

---

## Applied to GitHub

This re-cut has been applied to the tracker. Milestone numbers below are the
GitHub milestone ids.

| Milestone | id | Action taken | Issues |
|---|---|---|---|
| M0 — Foundation and delivery pipeline | 1 | renamed from `M0 — Skeleton and guarantees`; #103 subtree out, #99 subtree in | 59 |
| M1 — Secrets and enforcement machinery | 8 | created; epic #103 subtree moved in | 43 |
| M2 — Connection experience | 2 | renamed from `M1 — Read-only product (P1)`; DIT and search subtrees moved out | 40 |
| M3 — DIT screen | 9 | created; US1 stories #18, #19, #111 + subtrees moved in | 16 |
| M4 — Search screen | 10 | created; US2 #21 subtree moved in | 26 |
| M5 — MVP release | 11 | created; #100, #101 subtrees moved in | 8 |
| M6 — Writes and interchange | 3 | renamed from `M2`, contents unchanged | 61 |
| M7 — Fluency | 4 | renamed from `M3`, contents unchanged | 67 |
| M8 — Safety net | 5 | renamed from `M4`, contents unchanged | 57 |
| M9 — Specialist | 6 | renamed from `M5`, contents unchanged | 54 |
| M10 — General availability | 7 | renamed from `M6 — Release`; #99, #100, #101 subtrees moved out | 21 |

96 issues were re-milestoned; the other 356 kept their milestone and changed
only its name. All 452 are accounted for.

Two parents now span milestones, by design:

- **US1 (#14)** stays on M2 with the connection stories; its DIT stories
  (#18, #19, #111) sit on M3.
- **The release epic (#96)** stays on M10; its documentation and security
  stories (#100, #101) sit on M5, and release automation (#99) on M0.

Sub-issue links are unaffected — GitHub tracks parentage independently of
milestones, so both hierarchies still render on the issues themselves.
