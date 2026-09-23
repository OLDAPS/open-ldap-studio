# Open LDAP Studio Architecture Migration Plan

Status: accepted

Scope: current Go, Wails, React, TypeScript, and Zustand application

Migration style: incremental modular-monolith refactor; no rewrite

## 1. Objective

Evolve Open LDAP Studio into a modular desktop application that combines:

- MVVM-style presentation in React;
- application use cases for user-visible operations;
- hexagonal boundaries around LDAP, credentials, trust, files, and persistence;
- a CQRS-like separation between queries and commands; and
- one guarded pipeline for every LDAP mutation.

The architecture must make unsafe actions difficult to express, keep LDAP and
Wails details out of presentation logic, preserve server diagnostics, and allow
the core behavior to be tested without a webview or live directory server.

## 2. Architectural decision

Use a **modular monolith** with four core layers—presentation, application,
domain, and infrastructure—with Wails acting as the inbound transport adapter
between presentation and application:

```text
React Views
    |
    v
Feature ViewModels and frontend application state
    |
    v
Wails transport adapter (typed requests, responses, and events)
    |
    v
Go application use cases (queries and commands)
    |
    v
Domain modules and consumer-owned ports
    |
    v
LDAP, keychain, trust, file, profile, history, and logging adapters
```

This is not a request to create generic `domain`, `service`, and `repository`
packages. Packages should continue to be organized around capabilities such as
connections, searches, changes, schema, jobs, credentials, and history.

MVVM governs presentation only. Hexagonal boundaries govern dependencies in the
Go core. CQRS here means separate query and command paths, not separate services,
databases, or deployment units.

## 3. Current state

The project already has several important architectural foundations:

- `frontend/src/bridge/client.ts` is the only frontend wrapper around Wails.
- `internal/bridge` exposes the desktop API and performs composition.
- `internal/ldapx` owns LDAP transport, bind, search, controls, and result data.
- `internal/changeset` owns preview tokens and is the only permitted caller of
  LDAP mutation functions.
- `internal/jobs` owns background-operation state, progress, and cancellation.
- `internal/secrets`, `internal/credentials`, and `internal/trust` isolate
  credential and certificate decisions.
- `internal/schema`, `internal/ldif`, `internal/aci`, and `internal/compare`
  already represent capability-oriented modules.
- `test/architecture` mechanically protects mutation, secret, bridge, and LDAP
  result boundaries.
- `frontend/src/app/session.ts` intentionally holds shell/session state rather
  than caching directory entries.
- `useEntryViewModel` and `useDitTreeViewModel` are the first feature ViewModels.

The principal structural issue is that `internal/bridge` currently acts as the
transport adapter, composition root, application layer, and coordinator for
many operations. This makes Wails methods the easiest place to add behavior and
will cause business workflows to accumulate at the desktop boundary.

## 4. Target responsibilities

### 4.1 React Views

Views render state and forward user intent. They may own strictly visual state,
such as an open popover or the width of a panel. They must not:

- call Wails or `bridge` directly;
- construct LDAP requests;
- decide whether a change is safe to execute;
- contain credential, trust, retry, or referral policy; or
- translate LDAP result codes into domain decisions.

Representative views include `ConnectionsView`, `BrowserView`, `SearchesView`,
`EntryEditor`, `PreviewDialog`, and `ImportExportWizard`.

### 4.2 Feature ViewModels

In React, ViewModels should normally be hooks rather than classes. Each
top-level feature exposes a hook that returns render-ready state and actions:

```ts
interface EntryViewModel {
  state: 'idle' | 'loading' | 'ready' | 'saving' | 'error';
  entry?: EntryDisplayModel;
  draft: EntryDraft;
  result?: OperationResult;
  reload(): Promise<void>;
  updateValue(attribute: string, index: number, value: Uint8Array): void;
  preview(): Promise<void>;
  commit(previewToken: string): Promise<void>;
  cancel(): void;
}
```

ViewModels may depend on a frontend application client interface, not on the
Wails global. They own loading state, stale-response protection, draft state,
display mapping, and coordination between UI actions. They do not own LDAP
policy or server-side validation.

Use feature-local state by default. Keep Zustand for cross-feature shell state:
active connection, tabs, jobs, notices, preferences, and navigation. Do not put
directory entries or search result pages in a global store.

Suggested frontend organization:

```text
frontend/src/
  app/                 # shell-wide composition and state
  bridge/              # Wails transport implementation and DTOs
  features/
    connections/
      components/
      useConnectionsViewModel.ts
      model.ts
    browser/
      components/
      useDitTreeViewModel.ts
      useEntryViewModel.ts
      model.ts
    search/
    changes/
    schema/
    files/
  shell/
  styles/
```

Existing folders can move feature by feature. A directory reorganization is
not a prerequisite for introducing a ViewModel boundary.

### 4.3 Wails transport adapter

`internal/bridge` should become a thin adapter. A bound method should only:

1. decode and strictly validate its transport request;
2. call one application use case;
3. map the output to a stable response DTO; and
4. return or emit the result without rewriting server diagnostics.

The bridge must not contain LDAP workflow rules, instantiate dependencies, or
call LDAP mutation functions. Bound methods should be grouped by capability,
but all remain methods of the single Wails-bound `Bridge` unless Wails binding
constraints justify a later change.

Transport contracts should use explicit request and response types. Do not
expose persistence records or mutable internal types merely because Wails can
serialize them. Event names and payloads are also versioned contracts.

Keep `frontend/src/bridge/client.ts` as the only Wails lookup. Add a TypeScript
`ApplicationClient` interface implemented by that module so ViewModels can use
an in-memory fake in unit tests.

### 4.4 Go application use cases

Introduce `internal/application` as an orchestration layer. Organize it by
feature, not by technical stereotype:

```text
internal/application/
  connections/
  directory/
  search/
  changes/
  schema/
  transfer/            # LDIF, DSML, delimited, spreadsheet
  history/
```

Each exported operation represents a user intention. Initial use cases should
include:

**Queries**

- `ListProfiles`
- `GetConnectionState`
- `ReadRootDSE`
- `ListChildren`
- `ReadEntry`
- `ValidateFilter`
- `SearchEntries`
- `ReadSchema`
- `ListJobs`
- `ReadHistory`

**Commands**

- `SaveProfile`
- `Connect`
- `Disconnect`
- `TestConnection`
- `PreviewChangeSet`
- `CommitChangeSet`
- `DiscardChangeSet`
- `CancelJob`
- `ImportLDIF`
- `ExportEntries`
- `DecideTrust`

A use case receives `context.Context`, a typed input, and its dependencies. It
returns a typed output plus an error only when the application could not
complete the request. LDAP acceptance or rejection remains represented by the
structured LDAP result.

Interfaces belong in the package that consumes them. For example, a directory
query package defines the smallest `DirectoryReader` interface that it needs;
`internal/ldapx` or a connection manager satisfies it. Avoid a single large
repository interface mirroring the whole LDAP protocol.

### 4.5 Domain modules

Preserve and strengthen the existing capability packages:

- `changeset`: change intent, diff, preview, concurrency guards, dispatch;
- `schema`: schema model, parsing, compatibility, and validation;
- `aci`: access-control models and parsing;
- `compare`: directory comparison and reconciliation planning;
- `ldif` and `dsml`: lossless format parsing and rendering;
- `profiles`: connection-profile rules without secrets;
- `credentials`: credential metadata without secret values;
- `jobs`: operation lifecycle and outcomes; and
- `ldapx`: LDAP protocol types and the initial LDAP adapter.

Do not move packages only to match a diagram. Extract a domain type from
`ldapx` only when a use case needs that type without needing the LDAP adapter.
If extraction becomes useful, introduce a focused package such as
`internal/directory`, then migrate one type family at a time.

Domain rules must not import Wails, React transport concepts, filesystem
implementations, platform keychains, or concrete LDAP clients.

### 4.6 Infrastructure adapters

The existing packages form the initial adapters:

| Port or capability | Current implementation |
| --- | --- |
| Directory connection and operations | `internal/ldapx` and `internal/bridge/connections.go` |
| Credential metadata | `internal/credentials` |
| Secret storage | `internal/secrets` |
| Certificate decisions | `internal/trust` |
| Profile persistence | `internal/profiles` |
| History persistence | `internal/history` |
| Logs and redaction | `internal/logging` |

Move connection ownership out of the bridge into a dedicated application
lifecycle component, for example `internal/connections`. It should own sessions,
bind identity, capability snapshots, shutdown, and connection-state events.

Keep adapters replaceable through narrow consumer-owned interfaces. Do not add
interfaces around pure functions or stable value objects solely for symmetry.

### 4.7 Composition root

Create one composition root, preferably `internal/bootstrap`, that opens stores
and constructs adapters, use cases, job registries, and the bridge. `main.go`
should configure logging and Wails, call the bootstrap package, and bind the
result.

After migration:

- `main.go` knows Wails and bootstrap;
- bootstrap knows concrete implementations;
- bridge knows application use-case interfaces;
- use cases know domain types and narrow ports; and
- domain packages know neither Wails nor concrete infrastructure.

## 5. LDAP operation model

### 5.1 Query flow

```text
View action
  -> ViewModel query
  -> typed bridge request
  -> application query use case
  -> DirectoryReader port
  -> LDAP adapter
  -> structured result/page
  -> ViewModel display state
```

Queries must carry cancellation, timeout, paging, server-limit, referral, and
partial-result information. A partial page is successful data with a truncation
condition; it is not converted into either total success or a generic error.

### 5.2 Mutation flow

Every LDAP write follows this sequence:

```text
Draft
  -> normalize and validate locally
  -> read current server state
  -> compute ChangeSet and warnings
  -> issue expiring preview token
  -> render exact preview
  -> explicit confirmation
  -> consume token
  -> recheck versions/assertions
  -> dispatch through changeset
  -> store structured outcome and audit record
  -> invalidate/reload affected view state
```

The following invariants remain non-negotiable:

- only `internal/changeset` may call LDAP mutation functions;
- the bound bridge exposes no direct add, modify, delete, rename, or replay;
- commit accepts a preview token, not a new change payload;
- a consumed or expired token cannot be reused;
- cancellation never implies rollback unless the server confirmed rollback;
- every entry in a bulk operation has its own outcome; and
- irreversible or indeterminate outcomes are explicitly labelled.

Use the LDAP assertion control for optimistic concurrency where supported. Keep
the current entry-version fallback for servers that do not expose the control
or stable change markers. Capability fallback must always be visible to users.

### 5.3 Connection lifecycle

The connection component owns one state machine per profile:

```text
disconnected -> connecting -> connected -> disconnecting -> disconnected
                      |             |
                      v             v
                    failed       interrupted
```

It is responsible for transport, TLS, bind, keepalive, Root DSE discovery,
capability discovery, referral policy, and cleanup. ViewModels receive state
snapshots and events; they never hold an LDAP connection.

## 6. Security and data-handling rules

1. Simple-bind secrets may cross only an encrypted transport unless the user
   explicitly selects and acknowledges a plaintext profile policy.
2. Persist only credential references. Secret bytes remain inside
   `internal/secrets` and the shortest possible bind scope.
3. Never place secrets in profiles, frontend stores, events, results, history,
   logs, clipboard data, crash diagnostics, or test fixtures.
4. Construct filters through parsing/building APIs and apply RFC 4515 escaping;
   never concatenate untrusted values into an LDAP filter.
5. Treat referrals as a trust-boundary transition. Do not automatically send
   credentials to a referred host.
6. Preserve raw byte values for LDAP attributes. Text decoding is a view/editor
   decision and must not destroy the original representation.
7. Preserve the LDAP result code, matched DN, diagnostic message, referrals,
   and controls. Friendly explanations are additive.
8. Discover Root DSE and schema capabilities. Unsupported capabilities produce
   explicit reasons rather than disappearing controls or silent fallbacks.
9. Route all persisted application data through redaction and atomic-write
   policies appropriate to that store.

Relevant protocol references:

- [RFC 4511: LDAPv3 Protocol](https://www.rfc-editor.org/rfc/rfc4511)
- [RFC 4513: LDAP Authentication and TLS](https://www.rfc-editor.org/rfc/rfc4513)
- [RFC 4515: LDAP Filter Strings](https://www.rfc-editor.org/rfc/rfc4515)
- [RFC 4528: LDAP Assertion Control](https://www.rfc-editor.org/rfc/rfc4528)

## 7. Dependency rules to enforce

Add or extend architecture tests so these rules fail in CI:

1. Only `internal/changeset` invokes LDAP mutation functions.
2. Only `internal/secrets` imports platform credential APIs.
3. Only the connection/LDAP adapter imports `github.com/go-ldap/ldap/v3`.
4. `internal/application` does not import Wails packages.
5. Domain packages do not import `internal/bridge`, Wails, or platform adapters.
6. `internal/bridge` calls application use cases instead of LDAP operations.
7. Frontend files outside `frontend/src/bridge` do not access `window.go` or
   Wails runtime globals.
8. React view components do not import the concrete bridge client after their
   feature receives a ViewModel.
9. Every server-touching operation preserves `ldapx.Result` or a richer type
   containing it.
10. The bound surface exposes no method returning a secret-bearing type.

Rules should be introduced alongside the corresponding migration phase. Do not
enable a rule before existing code has a compliant path.

## 8. Testing strategy

### Domain tests

Use table-driven and fuzz tests for DN/filter handling, LDIF fidelity, schema
parsing, diffs, concurrency versions, warning generation, and redaction. Domain
tests require neither Wails nor a server.

### Application tests

Test every use case against small fakes implementing its ports. Cover success,
LDAP rejection, partial results, timeout, cancellation, missing capabilities,
referrals, stale versions, unavailable credential stores, and indeterminate
outcomes.

### Adapter tests

Use disposable OpenLDAP and ApacheDS instances for protocol behavior. Keep
server-specific expectations in adapter/integration tests rather than domain or
ViewModel tests.

### Bridge contract tests

Test strict DTO decoding, stable serialization, event payloads, absence of
unsafe methods, and preservation of structured results. These tests should not
retest use-case behavior.

### Frontend tests

Test ViewModels with an in-memory `ApplicationClient`. Component tests assert
rendering and user interaction using a supplied ViewModel. Retain a small set of
desktop integration tests for Wails binding, event delivery, and window-level
workflows.

## 9. Incremental migration

### Phase 0 — Baseline and vocabulary

Deliverables:

- accept this document as the target architecture;
- inventory bridge methods as query, command, transport-only, or obsolete;
- record the current import graph and direct frontend bridge consumers;
- keep the existing architecture tests green; and
- define common naming: `Input`, `Output`, `Query`, `Command`, `Result`, and
  `ApplicationClient`.

Exit criteria:

- every currently implemented bridge operation has a target use case;
- placeholders returning `not implemented` are identified and do not drive the
  target design; and
- no behavior has changed.

### Phase 1 — Application boundary and composition root

Deliverables:

- add `internal/bootstrap`;
- introduce application services for app info, profiles, and connection state;
- inject these services into `bridge.Bridge`;
- move store creation and connection lifecycle construction out of
  `internal/bridge.New`; and
- retain the existing Wails method signatures initially.

Exit criteria:

- profile and connection bridge methods contain transport mapping only;
- application use cases run in tests with fake dependencies; and
- startup, shutdown, profile, and connection behavior remain unchanged.

### Phase 2 — Directory queries

Deliverables:

- introduce use cases for Root DSE, child listing, entry reads, filter
  validation, and searches;
- define narrow `DirectoryReader` and capability interfaces in consuming
  packages;
- move query orchestration out of the bridge; and
- standardize query outputs carrying structured LDAP results and partial-state
  metadata.

Exit criteria:

- the bridge does not coordinate LDAP reads;
- cancellation and stale-response behavior are tested; and
- OpenLDAP and ApacheDS integration suites pass.

### Phase 3 — Frontend ViewModels

Migrate one vertical feature at a time in this order:

1. connections;
2. directory tree;
3. entry viewer/editor;
4. searches;
5. changes and jobs;
6. schema and file transfer.

For each feature:

- define display and draft models;
- introduce a `use...ViewModel` hook;
- inject `ApplicationClient`;
- remove concrete bridge calls from components;
- keep transient server data feature-local; and
- add ViewModel tests before moving the next feature.

Exit criteria:

- migrated views render supplied state and forward actions;
- the only concrete Wails client remains under `frontend/src/bridge`; and
- Zustand remains limited to shell/session concerns.

### Phase 4 — Command and mutation application layer

Deliverables:

- wrap `changeset.Pipeline` in `PreviewChangeSet`, `CommitChangeSet`, and
  `DiscardChangeSet` use cases;
- move audit, refresh/invalidation, job registration, and outcome coordination
  out of bridge methods;
- make imports, bulk changes, schema changes, configuration changes, replay,
  and reversal produce change sets rather than alternate write paths; and
- add assertion-control support behind a capability-aware concurrency port.

Exit criteria:

- every mutation feature reaches one application command pipeline;
- architecture tests prove that no bridge or feature package bypasses it;
- preview/commit token semantics and stale-entry conflicts are tested; and
- bulk cancellation reports every item as completed, failed, skipped, or
  indeterminate.

### Phase 5 — Adapter and contract hardening

Deliverables:

- isolate `go-ldap` imports to the LDAP adapter;
- make referral and capability policies explicit dependencies;
- introduce explicit bridge DTOs where internal structs currently leak;
- version event payloads and document compatibility expectations; and
- consolidate redaction and atomic persistence policies.

Exit criteria:

- use-case tests require no concrete LDAP, Wails, filesystem, or keychain
  implementation;
- transport DTO changes are detectable by contract tests; and
- secret-scan and architecture suites cover every new adapter.

### Phase 6 — Cleanup

Deliverables:

- remove unused bridge placeholders or implement them through use cases;
- remove superseded helpers and duplicate DTOs;
- update contributor documentation with the dependency rules;
- document the final package map; and
- review naming and package size after the behavior has settled.

Exit criteria:

- no production workflow lives in the bridge or React view components;
- all dependency rules are enforced in CI;
- unit, frontend, architecture, secret-scan, and integration tests pass; and
- a developer can add a query or command by following one documented example.

## 10. Pull-request strategy

Keep changes reviewable and reversible:

1. Add a use case and its tests around existing behavior.
2. Switch one bridge method to the use case.
3. Add the dependency rule that protects the new boundary.
4. Introduce or migrate the corresponding frontend ViewModel.
5. Remove the superseded path only after tests cover the new one.

Do not combine package moves, behavior changes, generated Wails binding changes,
and UI redesign in the same pull request.

## 11. Definition of done for a feature

A feature conforms to this architecture when:

- its View contains presentation behavior only;
- its ViewModel exposes render-ready state and user actions;
- it uses a typed frontend application-client contract;
- its bridge method only maps transport data;
- its Go workflow is an application use case;
- its external dependencies are accessed through narrow ports;
- LDAP results and partial outcomes remain structured;
- mutations use preview and tokenized commit;
- secrets cannot appear in returned or persisted data;
- cancellation and capability degradation are explicit; and
- domain, use-case, bridge, ViewModel, and relevant integration tests pass.

## 12. Explicit non-goals

- No microservices or local HTTP server.
- No replacement of Wails, React, Go, or Zustand as part of this migration.
- No big-bang directory reorganization.
- No global cache of LDAP entries or search results.
- No application-managed credential vault.
- No automatic referral credential forwarding.
- No generic repository abstraction over the entire LDAP protocol.
- No direct mutation method added for convenience.

The intended result is a desktop application whose UI can evolve independently,
whose LDAP behavior remains protocol-faithful, and whose safety properties are
enforced by code structure and tests rather than developer convention.
