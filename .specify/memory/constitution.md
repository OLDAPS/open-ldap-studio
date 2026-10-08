<!--
SYNC IMPACT REPORT
Version change: 1.2.0 → 1.3.0
Bump rationale: MINOR. All GNU Pass provisions are struck from Principle II and the Security
section, leaving the platform credential interface as the sole mandated integration point. This
removes a stated allowance (the opt-in direct pass provider) rather than a requirement: no
obligation changes, and no implementation is invalidated, so it does not meet the MAJOR bar.
It exceeds PATCH because what is permitted changes. NOTE: the versioning policy in Governance
describes MINOR as guidance "added or materially expanded" and has no explicit bucket for
material narrowing; that gap is worth closing in a future amendment.

Modified principles:
  - II. Credential and Transport Security (NON-NEGOTIABLE) — GNU Pass paragraph removed; the
    rationale no longer cites a bridged pass store as an example; title unchanged
  - I, III, IV, V — unchanged

Added sections: none

Removed sections: none (one paragraph removed from Principle II; one Security bullet generalized)

Deferred TODOs:
  - TODO(TECH_STACK): Desktop framework, language, and LDAP client library are not yet
    selected. Record the decision in the first /speckit-plan and amend Section
    "Security and Data Protection Standards" with the binding stack constraints, including the
    concrete credential-store binding for that stack (MINOR bump).

Deferred discussions (not governance, tracked here so the thread is not lost):
  - GNU Pass / password-store support was deliberately removed from the normative text on
    2026-08-24 pending a separate discussion. It is neither required nor prohibited today: a
    user whose password store is exposed through a Secret Service bridge is already served by
    the mandated interface, since that is a property of the bridge and not of any application
    code. Revisit if a direct pass provider is wanted.
-->

# Open LDAP Studio Constitution

## Core Principles

### I. Safety-First Directory Mutations (NON-NEGOTIABLE)

A directory is production infrastructure; an accidental write can lock out an organization.
Every operation that modifies the directory (add, modify, modrdn, delete, password reset,
bulk import) MUST present a preview of the exact LDAP operations to be sent — target DN,
attribute-level before/after diff, and total affected entry count — and MUST require explicit
user confirmation before dispatch. Bulk and recursive operations MUST support a dry-run that
executes the full validation path without sending a write. Delete of a non-leaf entry MUST
state the subtree size and require a distinct confirmation from single-entry delete. No code
path may auto-commit a mutation as a side effect of navigation, search, or refresh.

**Rationale**: Read operations are recoverable; writes to a live directory frequently are not.
The cost of one extra confirmation is trivial against the cost of a destroyed OU.

### II. Credential and Transport Security (NON-NEGOTIABLE)

**Secret storage.** Bind credentials MUST be delegated to the platform secret service and MUST
NOT be stored, encrypted or otherwise, in application-owned files. The mandated integration point
is the platform's native credential interface — the freedesktop Secret Service D-Bus API on
Linux and BSD, Keychain Services on macOS, Credential Manager on Windows — reached through an
in-process library binding. The application MUST NOT shell out to a password-manager CLI on the
default path: subprocess invocation drags PATH resolution, output parsing, and prompt routing
inside the security boundary, and is the common cause of a GUI session receiving no unlock
prompt at all.

Whatever the backend: connection profiles MUST persist only a reference to the stored secret,
MUST remain free of secret material, and MUST be safe to commit to version control. Storing
secrets in plaintext, in an application-managed keyfile or embedded vault, or behind an
in-application master password is prohibited without exception.

**Unlock model.** The application MUST start without prompting for any password and MUST NOT
gate its own launch behind an authentication screen. Unlock is deferred and system-owned: the
first operation that actually needs a secret — a bind, or saving a credential — triggers the
system credential prompt, exactly as Chrome defers to the desktop keyring prompt rather than
authenticating at startup. That prompt MUST be raised by the platform agent (keychain dialog,
wallet prompt, pinentry), never by an in-application password field: the application MUST never
receive, display, or process the master passphrase. Unlock lifetime and caching policy belong to
the agent; the application MUST NOT extend, re-prompt around, or cache secrets beyond the agent's
TTL, and MUST hold retrieved secrets in memory only, discarding them on disconnect and on exit.
A locked or unavailable agent MUST surface as a recoverable, clearly explained error — never as a
silent failure, and never as a prompt for the secret by other means.

**Transport.** TLS certificate and hostname verification MUST be enabled by default; disabling
it MUST be an explicit, per-profile, clearly labeled opt-in that is surfaced in the UI whenever
that profile is connected. StartTLS or LDAPS MUST be offered for every profile. Logs and
exported diagnostics MUST redact userPassword, bind credentials, and any attribute configured as
sensitive.

**Rationale**: A developer tool that leaks directory admin credentials is a privilege-escalation
vector into every system that trusts the directory. Delegating both storage and unlock to the
system means the application holds no long-lived secret of its own, inherits the user's existing
policy and audit surface, and cannot become the weakest link through home-grown crypto or a
master-password implementation. Binding to the platform interface rather than to one password
manager is what makes that portable: GNOME Keyring, KWallet, and KeePassXC all satisfy it
through the same code path.

### III. Protocol Fidelity Over Abstraction

The application MUST NOT hide the LDAP protocol from the developer using it. Raw access is
mandatory at every layer: users MUST be able to view and edit distinguished names as text,
write RFC 4515 filter strings directly, inspect operational attributes, and see the exact
LDAP result code and diagnostic message returned by the server on both success and failure.
Import and export MUST use RFC 2849 LDIF as the interchange format. Convenience abstractions
(schema-aware editors, templates, tree browsers) are additive and MUST always expose an escape
hatch to the underlying operation. The application MUST NOT silently rewrite, normalize, or
reorder user-authored filters, DNs, or attribute values before transmission.

**Rationale**: The audience is developers debugging directory behavior. An abstraction that
obscures what was actually sent to the server makes the tool useless for its primary purpose.

### IV. Non-Blocking, Transparent Desktop UX

All network I/O MUST execute off the UI thread; the interface MUST NOT freeze during a bind,
search, or write. Every operation that contacts a server MUST be cancellable by the user and
MUST enforce a configurable timeout. Searches MUST use paged results and MUST surface the
server's size and time limits rather than silently truncating. The application MUST display
connection state, the active bind DN, and the target server for every action so the user always
knows which directory they are about to modify. Errors MUST be shown verbatim alongside any
plain-language interpretation — never replaced by one.

**Rationale**: Directory operations run against remote servers of unknown latency; a frozen
window with no cancel and no server identity is both unusable and dangerous.

### V. Test-First Against Real Directories (NON-NEGOTIABLE)

Tests MUST be written and MUST fail before implementation begins. Every feature that speaks
LDAP MUST have integration tests executed against a real directory server running in a
container (OpenLDAP as the baseline), not against a mock of the client library. Mocks are
permitted only for unit-level tests of pure logic that never touches the protocol. Required
integration coverage: bind flows (simple, SASL, StartTLS, LDAPS, failure paths), search with
paging and referrals, each mutation type including rollback and error handling, and LDIF
round-trip fidelity. A protocol-facing change without a failing-then-passing integration test
MUST NOT be merged.

**Rationale**: LDAP server behavior — result codes, schema enforcement, referral handling —
diverges from specification in practice. Only a real server proves the client is correct.

## Security and Data Protection Standards

- The application MUST function fully offline with respect to third parties: no directory data,
  DNs, attribute values, connection profiles, or queries may be transmitted to any service other
  than the LDAP servers the user explicitly configured.
- Telemetry, crash reporting, and update checks MUST be opt-in and MUST NOT include directory
  content or connection details.
- Dependencies MUST be pinned with a committed lockfile, and a dependency vulnerability scan
  MUST run in CI on every pull request. Known high-severity advisories block merge.
- Cached directory data persisted to disk MUST be scoped per connection profile and MUST be
  purgeable from the UI in a single action.
- The application MUST support the standard LDAP authentication mechanisms expected by
  enterprise directories: simple bind, SASL EXTERNAL with client certificates, and GSSAPI.
- Secret-manager access MUST sit behind a single internal provider interface so the platform
  credential store and any backend added later are interchangeable and independently testable.
  No call site outside that provider may read or write a secret.
- Installing or configuring an external password manager MUST NOT be a precondition for first
  run. Where no credential store is reachable, the application MUST remain usable: profiles
  without a stored secret MUST still connect by prompting for the bind credential per session,
  held in memory only and never written to disk.
- The credential-store binding MUST be a maintained dependency. Selecting an unmaintained or
  archived library for this path is a merge blocker, and an existing binding that becomes
  unmaintained MUST be tracked for replacement.
- TODO(TECH_STACK): Desktop framework, implementation language, and LDAP client library are
  not yet selected. The first `/speckit-plan` MUST record this decision, after which this
  section is amended with the binding stack, the concrete credential-store library for that
  stack, and the supported platform matrix.

## Development Workflow and Quality Gates

- All changes land through pull request. Every PR description MUST state which principles the
  change touches and MUST justify any deviation; unjustified deviation blocks merge.
- CI MUST run, and MUST pass, before merge: unit tests, container-backed LDAP integration tests,
  linting, and the dependency vulnerability scan.
- Any change to a mutation path, a credential path, or TLS configuration requires review by a
  second maintainer. Self-approval is not permitted on these paths.
- Changes touching the secret provider MUST include tests proving that no secret reaches disk,
  logs, or crash output, that a locked or absent agent yields a recoverable error, and that the
  default path performs no subprocess invocation.
- User-facing strings for errors MUST preserve the server's original result code and diagnostic
  message; reviewers MUST reject changes that discard them.
- Complexity MUST be justified in the PR: a simpler implementation that satisfies the spec wins
  by default.
- Releases MUST follow semantic versioning, and any change to connection-profile or cache
  on-disk format MUST ship with a migration path.

## Governance

This constitution supersedes all other development practices, conventions, and preferences in
this project. Where a style guide, tooling default, or prior code pattern conflicts with a
principle here, the principle governs.

Amendments MUST be proposed as a pull request that modifies this document, states the rationale,
and specifies the version bump under the policy below. An amendment that adds a constraint to
existing code MUST include a migration plan for the code that does not yet comply. Amendments
are ratified by maintainer approval on that pull request.

Versioning policy for this document:

- **MAJOR**: A principle is removed or redefined in a way that invalidates previously compliant
  work.
- **MINOR**: A new principle or section is added, or existing guidance is materially expanded.
- **PATCH**: Clarification, wording, or typo fixes that do not change what is required.

Compliance review: every pull request is a compliance checkpoint, verified by the reviewer
against the principles above. Maintainers MUST additionally review this document at each minor
release for provisions that have drifted from practice, and either amend the document or correct
the practice. Runtime development guidance for agents and contributors lives in `CLAUDE.md`,
which MUST NOT contradict this constitution.

**Version**: 1.3.0 | **Ratified**: 2026-08-24 | **Last Amended**: 2026-08-24
