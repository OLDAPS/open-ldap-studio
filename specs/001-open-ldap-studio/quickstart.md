# Quickstart: Open LDAP Studio

**Date**: 2026-08-31 | **Plan**: [plan.md](./plan.md)

How to build, run, and validate the application, and which runnable check proves each user story.
This is a validation and run guide — implementation belongs in `tasks.md`.

---

## Prerequisites

| Tool | Version | Verified locally |
|------|---------|------------------|
| Go | 1.26+ | go1.26.2 ✅ |
| Node | 22+ | v24.12.0 ✅ |
| Docker | any recent | 29.7.2 ✅ (integration tests) |
| Wails CLI | v2.15.0 | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0` |

Platform build dependencies: **Linux** `webkit2gtk-4.1` + `libgtk-3-dev`; **macOS** Xcode command
line tools (cgo is required for both the webview and the Keychain binding); **Windows** WebView2
runtime. Run `wails doctor` to confirm.

---

## Run

```bash
wails dev            # live-reload development build
wails build          # production binary into build/bin/
go test ./...        # unit tests (no containers)
go test -tags=integration ./test/integration/...   # container-backed
```

### The gates CI runs, runnable locally

Every one of these blocks merge (plan.md § CI and Quality Gates):

```bash
govulncheck ./...                       # high-severity advisories block merge
npm --prefix frontend audit --audit-level=high
go mod verify                           # lockfiles pinned
go test -run TestImportBoundaries ./... # C1: ldapx mutations have one caller
                                        # C2: secrets is the only platform-credential importer
go test -run TestNoSubprocessInSecrets ./internal/secrets/...
go test -tags=scan ./test/secrets_scan/ # SC-008: no secret in any written file, log, or crash dump
```

`TestImportBoundaries` and `TestNoSubprocessInSecrets` are worth writing on day one. Both are
architectural rather than behavioural, so they cost minutes now and a refactor later.

The application must reach a usable window with **no password prompt of its own** (SC-009). If a
prompt appears at startup, that is a Gate II failure, not a configuration issue.

---

## Test fixtures

Three containers, started by testcontainers-go (`test/containers/`):

| Fixture | Provides | Needed by |
|---------|----------|-----------|
| `openldap` | Baseline; `cn=config` for schema modification and configuration-as-entries | Most stories, US11, US12 |
| `apacheds` | X.500 `prescriptiveACI`, subentries, administrative roles | US10 |
| `poor` | No paging, no readable subschema, no extended operations | Degradation paths, SC-016 |

`test/corpus/` holds the fidelity corpus: binary, non-UTF-8, empty, option-bearing (`;binary`,
language tags), and multi-megabyte values. Every interchange test runs against it.

---

## Validation scenarios

Each row is runnable and maps to a user story and the success criteria it discharges. Details of
the types involved are in [data-model.md](./data-model.md); the methods are in
[contracts/bridge-api.md](./contracts/bridge-api.md).

### Story-level

| Story | Scenario | Passes when |
|-------|----------|-------------|
| **US1** Connect & explore | Connect to `openldap` over plain, StartTLS, and LDAPS; bind anonymous, simple, EXTERNAL, GSSAPI, DIGEST-MD5, CRAM-MD5; read the root DSE; expand a 25,000-child container; `WhoAmI` | Every bind method succeeds; the tree pages; root DSE lists controls and extended operations; the status bar shows server, bind DN, and TLS state throughout |
| **US1** Trust | Present an untrusted certificate; accept for the session; restart; rotate the certificate | Connection refused first; `trust:challenge` carries the chain and reason; the session decision does not persist; a rotated certificate re-challenges |
| **US2** Search | Run a nested boolean + extensible-match filter; compare against `ldapsearch` output | Result sets identical; the filter string on the wire is byte-identical to what was typed |
| **US2** Limits | Search past the server size limit | Partial results shown, labelled truncated-by-server, with the server's result code |
| **US3** Preview | Modify, add, delete; capture what the server received | The preview matches the transmitted operation exactly; cancelling sends nothing; a schema violation returns the server's verbatim code and message |
| **US3** Concurrency | Change an entry on the server between preview and commit | `Commit` refuses the stale token and forces re-confirmation |
| **US4** Fidelity | Export the corpus subtree to LDIF; import into an empty directory | Byte-for-byte identical values, including folding, base64, and attribute options |
| **US4** Formats | Export the same set as DSML, CSV, JSON, XLSX, ODS | Multi-valued and binary attributes survive losslessly |
| **US5** Schema | Browse `openldap` schema; navigate object class → attribute type → syntax; connect to `poor` | Navigation is bidirectional; against `poor` the app stays usable and `capability:unavailable` explains why assistance is off |
| **US6** Values | Set a password under each hash scheme and verify; round-trip a certificate and an image; use `PasswordModify` where advertised | `EncodeValue(DecodeValue(v)) == v` for every kind; no plaintext reaches disk or logs |
| **US7** Move/copy | Rename with keep-old-RDN both ways; copy a subtree between `openldap` and `apacheds` | RDN handling matches the choice; destination entry count, DNs, and values match; an interrupted copy lists what was written |
| **US8** History | Modify, then reverse from history | Every write recorded with request and response; the reversal restores the exact prior state; an uncapturable operation reports why it cannot be reversed |
| **US9** Compare | Seed two directories with known differences; compare; apply the reconciling LDIF | Exactly the seeded differences reported, no false positives; applying eliminates them |
| **US10** ACI | Author an ACI and a subtree specification on `apacheds` through the structured editors, then edit the raw text | Transmitted value is byte-identical to the hand-written equivalent; `RenderACI(ParseACI(raw)) == raw`; an unparseable value is shown raw with the error, never rewritten |
| **US11** Schema project | Create a project from `openldap`; add an object class with a dangling superior; check; export; re-import; commit to the server | The dangling reference, duplicate OIDs, and circular superiors are all reported; errors block export; definitions survive the file round trip; commit previews before writing |
| **US12** Config | Edit a `cn=config` entry on `openldap` | The preview marks the target as server configuration and warns; the change takes effect; a rejection surfaces verbatim |
| **US13** Bulk | Apply one change to 500 entries: dry run, then execute | Dry run writes nothing and projects the same per-entry outcomes as the real run (≥ 99%); cancellation stops at an entry boundary and reports exactly what changed |

### Cross-cutting

| Check | How | Passes when |
|-------|-----|-------------|
| **SC-002** Responsiveness | Instrument the event loop during a 100k search and a bulk write | No block exceeds 100 ms |
| **SC-003** Cancellation | Cancel each `JobID`-returning method mid-flight | Every one stops within 2 s |
| **SC-005** Preview coverage | Static check: callers of `ldapx` mutation functions | Exactly one — `changeset.dispatch` (contract C1) |
| **SC-006** Verbatim errors | Force a rejection on every mutation path | The server's code and diagnostic appear unmodified in each |
| **SC-008** No secrets on disk | Run a session exercising all 13 stories, then scan every written file, log, and crash artefact | Zero matches (contract F3, E1) |
| **SC-009** Startup | Cold start with a stored credential present | Usable window under 3 s, no prompt until the first bind |
| **SC-016** Degradation | Run the full suite against `poor` | Every unavailable capability announced via `capability:unavailable`; nothing fails silently |
| **SC-017** Edge cases | One automated test per edge case in the spec | 100% of the spec's 23 edge cases covered |
| **Error model** | Force each mandated result code against a live server (contracts/error-model.md X5) | Each triggers its required handling; diagnostics byte-identical |
| **Indeterminate** | Kill the connection mid-write | Reported as neither success nor failure; history records it indeterminate (X4) |
| **No vault** | Enumerate the bound bridge surface | No `LockVault`, `ExportVault`, `RevealSecret`, or master-password method exists (C11) |
| **Read-only mode** | Attempt a mutation on a read-only profile | Refused at the `changeset` boundary, before any UI check (C13) |
| **Referral safety** | Follow a referral to an unbound host | Requires an explicit target profile; current credentials are never auto-reused (C12) |
| **LDIF layout vs fidelity** | Export the corpus at three different wrap/fold settings, re-import each | Values identical in all three; files differ. *Value* equality is the assertion, never file equality (research R13) |
| **Constitution II** | Import-boundary check | No package outside `secrets` imports a platform credential API (contract C2) |

---

## Order of work

Stories are independently shippable, but the enforcement machinery is not optional groundwork —
build it first or retrofit it painfully:

1. `secrets` provider + `jobs` registry + `ldapx` connection and bind (Gates II and IV become real)
2. `changeset` preview pipeline (Gate I becomes structural — do this **before** the first write path)
3. US1 → US2 (P1, shippable read-only product)
4. US3 → US4 (P2, writes and interchange)
5. US5 → US7 (P3)
6. US8 → US10 (P4)
7. US11 → US13 (P5)

Contract tests C1–C14, E1–E5, F1–F9, and X1–X8 are written **before** the code they constrain, per
Constitution V. C1, C2, C11, and C13 in particular are cheap on day one and expensive to satisfy
later, since all four are architectural rather than behavioural.

Milestones, their exit criteria, and the risk register are in [plan.md](./plan.md). The screen-level
breakdown for frontend work is [contracts/screens.md](./contracts/screens.md).

### Before building an affected screen

Deviations D1–D9 are changes to the *design*, not the code. Land them in the Claude Design project
before the milestone that builds the screen — **D2** (delete the vault) and **D3** (remove
auto-save) before M0 and M2 respectively, since both change what gets built rather than how it
looks.
