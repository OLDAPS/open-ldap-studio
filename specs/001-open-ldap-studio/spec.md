# Feature Specification: Open LDAP Studio

**Feature Branch**: `001-open-ldap-studio`

**Created**: 2026-08-31

**Status**: Draft

**Input**: User description: "Develop Open LDAP Studio, a desktop app that has all the Apache Directory Studio and more"

## Overview

Open LDAP Studio is a cross-platform desktop workbench for people who operate and debug LDAP directories: directory administrators, identity and platform engineers, and developers whose applications bind against a directory.

The product has two halves. The first is **parity**: everything a practitioner uses Apache Directory Studio for — connection management, tree browsing, entry editing, raw filter search, schema browsing *and offline schema authoring*, LDIF and DSML interchange, the full set of specialised value editors, access-control and administrative-model editing, and cross-server copy. Parity is not asserted in prose; it is enumerated in the **Parity Reference** appendix at the end of this document, where every capability area maps to the requirements that satisfy it, and the four deliberate exclusions are named with reasons.

The second half is **more**: an operating-system credential model that never stores a secret of its own, a mandatory write preview with attribute-level diffs on every mutation, cross-server comparison with generated reconciling LDIF, a durable operation history that can be replayed or reversed, and dry-runnable bulk operations.

This specification defines the complete v1 product. It is deliberately large; the thirteen user stories are ordered so each is independently shippable and independently valuable.

## Glossary

Provided so this document can be read by stakeholders who do not work with directories daily.

| Term | Meaning in this document |
|------|--------------------------|
| **Directory** | A server holding hierarchically organised records, queried over the LDAP protocol. Typically the authoritative source of user accounts and groups for an organisation. |
| **Entry** | One record in the directory — a user, a group, an organisational unit. |
| **DN** (distinguished name) | An entry's unique full-path identifier, e.g. `uid=alice,ou=people,dc=example,dc=com`. |
| **RDN** (relative distinguished name) | The leftmost component of a DN — the entry's name relative to its parent. |
| **Attribute** | A named field on an entry. Attributes may hold several values at once. |
| **Object class** | A named template declaring which attributes an entry must and may have. |
| **Schema** | The server's catalogue of object classes, attribute types, syntaxes, and matching rules — the rules every entry is validated against. |
| **DIT** (directory information tree) | The tree formed by all entries on a server. |
| **Root DSE** | A special entry describing the server itself: what it supports, which naming contexts it holds. |
| **Naming context** | A top-level subtree the server is authoritative for, e.g. `dc=example,dc=com`. |
| **Bind** | Authenticating to a directory server. |
| **Filter** | A query expression selecting entries, written in the standard text syntax, e.g. `(&(objectClass=person)(uid=alice*))`. |
| **LDIF** | The standard plain-text file format for exchanging directory entries and changes. The interchange lingua franca. |
| **Operational attribute** | Server-maintained metadata on an entry, such as creation timestamp, hidden unless explicitly requested. |
| **Control** / **extended operation** | Optional protocol add-ons a server may support — result paging, sorting, password change. |
| **Referral** | A pointer telling the client the data it asked for lives on another server. |
| **Alias** | An entry that points at another entry, like a symbolic link. |
| **ACI** (access control item) | A rule stored in the directory declaring who may read or change what. |
| **Subentry** | An entry holding policy — access control, collective attributes — that applies to a defined region of the tree. |
| **Result code** | The numeric status the server returns for every operation, with an accompanying diagnostic message. |

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connect to a directory and explore it (Priority: P1)

An engineer defines a connection to an LDAP server, supplies a bind identity, connects, and navigates the directory information tree. They expand naming contexts, select an entry, and read its attributes — including operational attributes — with the server, bind DN, and connection state visible at all times. They inspect the root DSE to learn what the server supports, and confirm who they are actually bound as. Connections are organised into folders and stored as files carrying no secret material.

**Why this priority**: Nothing else is reachable without a connection and a browsable tree. On its own this replaces the read-only half of the practitioner's daily work and is a viable standalone release.

**Independent Test**: Define a profile against a containerised directory over plain LDAP, StartTLS, and LDAPS; connect using each supported bind method; browse from the root DSE down three levels; open an entry and confirm every user and operational attribute is displayed. Delivers full read access with no editing capability present.

**Acceptance Scenarios**:

1. **Given** no existing profiles, **When** the user creates a profile with host, port, encryption method, and bind identity and connects, **Then** the application connects, records the profile, and shows the root DSE with its naming contexts, supported controls, supported extended operations, and supported SASL mechanisms.
2. **Given** a connected profile, **When** the user expands a container holding 25,000 children, **Then** entries load incrementally in pages, the tree stays responsive, and both the count loaded so far and the applicable server limit are shown.
3. **Given** a connected profile, **When** the user selects an entry, **Then** all user attributes and, on request, all operational attributes are displayed with their raw values and the entry's exact DN as returned by the server.
4. **Given** a server presenting a certificate that fails validation, **When** the user connects, **Then** the connection is refused, the certificate chain and the specific validation failure are shown, and continuing requires an explicit trust decision — for this session or permanently — that is recorded in a trust store the user can review and revoke from.
5. **Given** a recorded permanent trust decision, **When** the server later presents a different certificate, **Then** the connection is refused again and the change is reported rather than silently accepted.
6. **Given** any connection, **When** the user asks who they are bound as, **Then** the identity the server reports is displayed, distinguished from the identity that was requested.
7. **Given** a search or connect taking longer than expected, **When** the user cancels, **Then** the operation aborts, the interface stays responsive throughout, and the cancellation is reported without corrupting connection state.
8. **Given** a profile saved with a stored credential reference, **When** the application is launched, **Then** it starts with no password prompt of its own, and the system credential prompt appears only when the user first triggers a bind.

---

### User Story 2 - Search with raw filters, saved searches, and bookmarks (Priority: P1)

An engineer writes a filter by hand, chooses a search base, scope, returned attributes, size and time limits, alias and referral behaviour, and any controls, then runs it. Those who prefer to assemble a filter visually can do so and switch to the text form at any point. Results appear in a configurable table that can be sorted — client-side, or by the server where it supports sorting — and exported. Useful searches are saved for reuse; frequently visited entries are bookmarked.

**Why this priority**: Searching with an exact filter is the primary diagnostic act for this audience, and it is what a browser-only tool cannot substitute for.

**Independent Test**: Author a filter with nested boolean and extensible-match components, run it against a containerised directory, verify the result set matches an equivalent command-line search exactly, save it, reopen it, and re-run it unchanged.

**Acceptance Scenarios**:

1. **Given** the search editor, **When** the user types a filter string, **Then** syntax is validated and attribute names and object classes are offered as completions from the connected server's schema, without the typed text being rewritten, reordered, or normalised before transmission.
2. **Given** a syntactically invalid filter, **When** the user runs it, **Then** the search is refused locally with the position and nature of the error, and nothing is sent to the server.
3. **Given** the visual filter builder, **When** the user assembles a filter and switches to the text view, **Then** the generated filter string is shown for editing, and edits made there are preserved when the search is run.
4. **Given** a search exceeding the server's size limit, **When** results return, **Then** the partial result set is shown together with the server's result code and an explicit statement that the server truncated the results.
5. **Given** a result set, **When** the user changes which attributes are shown as columns and sorts by one, **Then** the table updates without re-issuing the search, and the visible set can be exported.
6. **Given** a server advertising server-side sorting, **When** the user requests a server-ordered search, **Then** the sort control is sent and the server's ordering is preserved; **Given** a server that does not, **Then** the option is unavailable and the reason is stated.
7. **Given** a saved search, **When** the user runs it against a different connection, **Then** it executes unchanged against that server and reports that server's results or error.

---

### User Story 3 - Modify entries with a previewed, confirmed write (Priority: P2)

An engineer edits attribute values, adds and removes attributes, creates entries from an object class selection or a template, and deletes entries. Before anything is transmitted the application shows the exact operations it will send — target DN, per-attribute before-and-after values, and affected entry count — and requires confirmation.

**Why this priority**: Writing is the point of the tool for administrators, but it is only responsible to ship once reading and searching are trustworthy. The preview requirement makes this slice larger than a naive editor.

**Independent Test**: Modify, add, and delete entries against a containerised directory; verify for each that the preview matches the operation actually received by the server, that cancelling sends nothing, and that a schema-violating change is rejected with the server's verbatim result code and diagnostic message.

**Acceptance Scenarios**:

1. **Given** an entry open for editing with pending changes, **When** the user commits, **Then** a preview lists each add, replace, and delete with old and new values, and nothing is dispatched until the user confirms.
2. **Given** a preview dialog, **When** the user cancels, **Then** no operation reaches the server and the pending edits remain intact for further work.
3. **Given** a change the server rejects, **When** it is dispatched, **Then** the exact result code and diagnostic message are displayed verbatim alongside any plain-language interpretation, and the entry is refreshed to its actual server state.
4. **Given** an entry with children, **When** the user requests deletion, **Then** the subtree size is stated, recursive deletion is a distinct and separately confirmed choice, and a dry run is offered that runs full validation without writing.
5. **Given** pending edits, **When** the user navigates away, refreshes, or collapses the tree, **Then** nothing is committed as a side effect and the user is asked what to do with the pending edits.
6. **Given** the new-entry wizard, **When** the user picks object classes, **Then** required attributes are enumerated from the schema and the entry cannot be submitted until they are supplied, while an escape hatch allows composing the same entry as raw LDIF.
7. **Given** an attribute carrying a description option such as a binary marker or a language tag, **When** the user edits it, **Then** the option is preserved and separately editable, and values differing only by option are not merged.

---

### User Story 4 - Import and export LDIF and other interchange formats (Priority: P2)

An engineer exports an entry, a result set, or a whole subtree to LDIF, edits LDIF in a dedicated editor with syntax highlighting and validation, and executes an LDIF change file against a server with a preview and a per-record report. Exports are also available as DSML, CSV, JSON, and spreadsheet workbooks.

**Why this priority**: LDIF is the lingua franca for backup, migration, and reproducing a bug report. It also makes every other capability recoverable, since an export can precede any risky change.

**Independent Test**: Export a subtree containing binary values, non-ASCII characters, and folded long lines; re-import into an empty directory; verify a byte-faithful round trip of every attribute value and a per-record success report.

**Acceptance Scenarios**:

1. **Given** a subtree of 50,000 entries, **When** the user exports it to LDIF, **Then** the export streams to disk with progress and a working cancel, producing conformant LDIF with correct base64 encoding and line folding.
2. **Given** an LDIF change file, **When** the user executes it, **Then** the preview summarises operations by type and count, execution reports per-record outcome, and the user chooses in advance whether a failing record halts execution or is skipped and logged.
3. **Given** an LDIF file with a malformed record, **When** it is opened, **Then** the error is reported with its line number and the rest of the file remains usable.
4. **Given** a result set, **When** the user exports to CSV, JSON, or a spreadsheet, **Then** the exported columns match the visible columns and multi-valued and binary attributes are represented losslessly.
5. **Given** a DSML file, **When** the user imports it, **Then** it is subject to the same preview, confirmation, and per-record reporting as LDIF.

---

### User Story 5 - Inspect the schema and edit with schema awareness (Priority: P3)

An engineer browses the connected server's object classes, attribute types, syntaxes, and matching rules; searches the schema; follows references between elements; and sees, while editing an entry, which attributes are required, permitted, single-valued, or read-only, along with each attribute's syntax.

**Why this priority**: Schema knowledge turns editing from guesswork into a checked operation, but editing is usable without it.

**Independent Test**: Browse the schema of a containerised directory, navigate from an object class to a referenced attribute type to its syntax and matching rules, and confirm that entry editing enumerates required and optional attributes consistently with that schema.

**Acceptance Scenarios**:

1. **Given** a connected server, **When** the user opens the schema browser, **Then** object classes, attribute types, syntaxes, and matching rules are listed with OIDs, descriptions, superior elements, and originating definition.
2. **Given** a schema element, **When** the user selects a referenced element, **Then** navigation moves to it and the path taken can be traversed backwards.
3. **Given** a server publishing no readable subschema, **When** the user connects, **Then** the application remains fully usable in a schema-less mode with schema-derived assistance disabled and the reason stated.
4. **Given** two connections, **When** the user compares their schemas, **Then** elements present in only one, and elements present in both with differing definitions, are listed.

---

### User Story 6 - Edit specialised attribute values safely (Priority: P3)

An engineer edits values that are not plain text: passwords with a choice of hash scheme and a verify function, certificates and binary blobs, images, DNs with a tree picker, timestamps, postal addresses, boolean and integer syntaxes, and multi-line text — plus a byte-level editor for anything unrecognised. Every specialised editor exposes the underlying raw value. Where the server supports the standard password-change extended operation, that is offered in preference to writing the password attribute directly.

**Why this priority**: These editors are what make the tool usable on real data, but the raw text path already permits the same work at higher effort.

**Independent Test**: Set a password using each supported hash scheme and verify each against the stored value; upload and download a binary certificate and confirm the bytes are unchanged; edit a DN-syntax attribute through the picker and confirm the transmitted value is the exact DN chosen.

**Acceptance Scenarios**:

1. **Given** a password attribute, **When** the user sets a new value, **Then** a hash scheme is selectable, the plaintext is never written to disk, logs, or diagnostic exports, and a verify action confirms a candidate password against the stored value.
2. **Given** a server advertising the password-change extended operation, **When** the user changes a password, **Then** that operation is offered and, if chosen, used in place of a direct attribute write, with the server's response reported verbatim.
3. **Given** a binary attribute, **When** the user exports and re-imports its value, **Then** the bytes are identical.
4. **Given** a certificate value, **When** the user views it, **Then** subject, issuer, validity window, and fingerprints are decoded while the raw encoded bytes remain viewable and editable.
5. **Given** a value of a syntax the application does not recognise, **When** the user opens it, **Then** a byte-level editor is offered rather than a lossy text conversion.
6. **Given** any specialised editor, **When** the user chooses the raw view, **Then** the underlying value is editable as text or as bytes and takes precedence over the decoded view.

---

### User Story 7 - Move, rename, copy, and duplicate entries across servers (Priority: P3)

An engineer renames an entry, moves a subtree to a new parent, and copies entries or subtrees between two connected servers, choosing collision behaviour and whether operational attributes are carried over.

**Why this priority**: Migration and promotion between environments is routine and painful without direct support, though composable from export and import if absent.

**Independent Test**: Rename an entry and confirm RDN attribute handling matches the chosen option; copy a subtree between two containerised servers and verify entry count, DNs, and attribute values at the destination.

**Acceptance Scenarios**:

1. **Given** a rename, **When** the user commits it, **Then** the preview states old and new DN and whether the old RDN value is retained or deleted, and that choice is honoured.
2. **Given** a cross-server copy, **When** a destination DN already exists, **Then** the user chooses beforehand to skip, overwrite, or merge, and the choice is applied consistently and reported per entry.
3. **Given** a copy that fails partway, **When** it stops, **Then** entries already written are listed, the cause is stated verbatim, and the operation can be resumed or reversed.

---

### User Story 8 - Review, replay, and reverse past operations (Priority: P4)

An engineer opens a per-profile history of every write the application has performed and every search it has run, with timestamp, bind identity, target, the operation as sent, and the server's response. Any write in that history can be exported as LDIF, replayed against a chosen connection, or turned into a reversing LDIF built from the captured before-state.

**Why this priority**: This exceeds the reference tool, whose modification and search logs are read-only text files, and is the strongest answer to "what did I just do to production" — but the product is complete without it.

**Independent Test**: Perform a series of modifications, confirm each appears in the history with its request and response, generate the reversing LDIF, apply it, and verify the directory returns to its original state.

**Acceptance Scenarios**:

1. **Given** a completed write, **When** the user opens the history, **Then** the record shows the operation as transmitted, the server's result code and diagnostic message, the bind DN, and the server identity.
2. **Given** a history record for a modify, **When** the user requests a reversal, **Then** a reversing LDIF is produced from the captured before-state, subject to the same preview and confirmation as any other write.
3. **Given** an operation whose original state was not fully captured, **When** the user requests a reversal, **Then** the application states that it cannot be reversed and why, rather than producing an incomplete reversal.
4. **Given** any history record or diagnostic export, **When** it is written to disk, **Then** bind credentials, password values, and attributes marked sensitive are redacted.

---

### User Story 9 - Compare two directories or subtrees (Priority: P4)

An engineer selects two entries, two subtrees, or the results of one search run against two connections, and sees a structural diff: entries present on only one side, and attribute values that differ, with the option to generate the LDIF that would bring one side into line with the other.

**Why this priority**: Environment drift is common, expensive, and unaddressed by existing tooling; it depends on browsing, searching, and LDIF already working.

**Independent Test**: Seed two containerised directories with a known set of differences, run the comparison, and verify exactly those differences are reported and that the generated reconciling LDIF, when applied, eliminates them.

**Acceptance Scenarios**:

1. **Given** two subtrees, **When** the user compares them, **Then** entries unique to each side and per-attribute value differences are listed, and the user can exclude operational attributes and nominate attributes to ignore.
2. **Given** a comparison result, **When** the user generates reconciling LDIF, **Then** it is offered for review and export and is never applied without the standard preview and confirmation.

---

### User Story 10 - Author access control and administrative policy (Priority: P4)

An engineer views and edits the entries that govern authorisation: access control items, the subtree specifications that scope them, administrative roles, and the subentries carrying them. Each is presented both as a structured editor and as the exact stored text, and the structured view never becomes the only way to express a rule.

**Why this priority**: Access control is where directory mistakes are most damaging, and hand-editing these values as raw strings is the most error-prone task in directory work. It comes after core editing because it builds on the same preview and confirmation path.

**Independent Test**: Against a containerised directory supporting stored access-control rules, create a subentry with a subtree specification and an access-control item through the structured editors, confirm the transmitted value is byte-identical to the equivalent hand-written value, then edit it as raw text and confirm the structured view reflects the change.

**Acceptance Scenarios**:

1. **Given** an access-control item value, **When** the user opens it, **Then** its components are presented in a structured editor and the exact stored string remains viewable and directly editable.
2. **Given** a structured edit, **When** the user commits, **Then** the standard preview shows the resulting value as text before and after, and confirmation is required.
3. **Given** a value the structured editor cannot parse, **When** it is opened, **Then** the raw text is presented with the parse failure explained, and the value is never silently rewritten or discarded.
4. **Given** a subtree specification, **When** the user defines it, **Then** the region it selects can be previewed as a list of matching entries before the specification is saved.
5. **Given** a server that does not support stored access-control rules, **When** the user browses it, **Then** these editors are unavailable and the reason is stated, with no loss of generic entry editing.

---

### User Story 11 - Author schemas offline in a project (Priority: P5)

A schema designer creates a schema project — either empty, from a connected server's schema, or from schema definition files — then creates and edits object classes and attribute types in it, checks the project for structural errors, compares it against another project or a live server, exports it back to file formats, and, where the server permits schema modification, commits the difference to a server under the standard preview and confirmation.

**Why this priority**: Offline schema authoring is a distinct discipline with a distinct audience, and it is the largest capability of the reference tool that is not merely directory browsing. It lands late because it depends on schema reading being solid.

**Independent Test**: Create a project from a containerised server's schema, add an object class referencing a new attribute type, run the project's error check to catch a deliberately dangling reference, export to schema files, re-import into a fresh project, and confirm the definitions are unchanged.

**Acceptance Scenarios**:

1. **Given** no project, **When** the user creates one from a connected server, **Then** the server's object classes, attribute types, syntaxes, and matching rules are imported and editable offline with no further server contact.
2. **Given** a project, **When** the user creates an object class or attribute type, **Then** OID, names, description, superior, syntax, matching rules, usage, and single-valued and obsolete flags are all editable, and the resulting definition text is viewable.
3. **Given** a project containing a reference to an element that does not exist, **When** the user checks the project, **Then** the dangling reference, along with duplicate OIDs and circular superior chains, is reported with the elements involved.
4. **Given** two projects, or a project and a live server, **When** the user compares them, **Then** elements unique to each side and elements defined differently are listed.
5. **Given** a project and a server that permits schema modification, **When** the user commits the difference, **Then** the operations are previewed and confirmed like any other write, and the server's response is reported verbatim.
6. **Given** a server that does not permit schema modification, **When** the user attempts to commit, **Then** the refusal is reported with the server's own message and the project is left unchanged.

---

### User Story 12 - Edit server configuration held as directory entries (Priority: P5)

An engineer whose server exposes its own configuration as directory entries browses and edits that configuration through the same tree, editors, preview, and history as any other part of the directory, with the heightened risk of those entries made explicit.

**Why this priority**: This is how mainstream servers expose runtime configuration, and covering it generically serves every such server rather than one vendor. It is last because it is the narrowest audience and the highest blast radius.

**Independent Test**: Against a containerised server exposing configuration as entries, browse the configuration subtree, change a setting, confirm the preview shows the attribute-level diff and that the change takes effect on the server.

**Acceptance Scenarios**:

1. **Given** a server exposing a configuration naming context, **When** the user connects, **Then** that context is browsable and editable through the standard tree and entry editors.
2. **Given** an edit to a configuration entry, **When** the user commits, **Then** the preview identifies the target as server configuration and warns that the change may alter server behaviour immediately, and confirmation is required.
3. **Given** a configuration change the server rejects, **When** it is dispatched, **Then** the server's result code and diagnostic message are shown verbatim and no partial state is presented as success.

---

### User Story 13 - Run bulk operations with a dry run (Priority: P5)

An engineer applies one change — an attribute set, an attribute removal, a group membership change, a password reset — to every entry matched by a search, previews the full operation set, runs a dry run that validates without writing, then executes with progress, cancellation, and a per-entry report.

**Why this priority**: The highest-leverage and highest-risk capability, so it lands last, on a proven preview, dry-run, and history foundation.

**Independent Test**: Select 500 entries by filter, apply an attribute change, confirm the dry run writes nothing and reports the same per-entry outcomes the real run later produces, then execute and verify every entry.

**Acceptance Scenarios**:

1. **Given** a bulk change over a result set, **When** the user requests a dry run, **Then** the full validation path executes, no write is sent, and per-entry projected outcomes are reported.
2. **Given** a bulk execution in progress, **When** the user cancels, **Then** it stops at the next entry boundary and reports precisely which entries were changed and which were not.
3. **Given** a bulk execution with failures, **When** it completes, **Then** a per-entry report distinguishes succeeded, failed with the server's verbatim message, and skipped entries, and the report is exportable.

---

### Edge Cases

Each case states the required behaviour, so each is directly testable.

**Connectivity and transport**

- What happens when the server is unreachable or DNS fails? The attempt fails within the configured timeout, naming the host and the underlying cause; the profile stays usable and retry is one action.
- What happens when the connection drops mid-operation? The operation is reported as failed with unknown server-side outcome — never as success — the entry is refreshed from the server, and the history records the indeterminate result.
- What happens when TLS negotiation fails, or a server offers no StartTLS despite the profile requiring it? The connection is refused; the application never silently downgrades to an unencrypted connection.
- What happens when the server closes an idle connection? Reconnection is transparent for read operations and explicitly confirmed before any write, so no write is dispatched on an assumed-live connection.

**Credentials and authentication**

- What happens when the system credential store is locked, unavailable, or absent? A recoverable, explained error appears; the user may proceed by entering the credential for this session only, held in memory and never written to disk.
- What happens when the bind is rejected, or the account is locked by the directory? The server's result code and diagnostic message are shown verbatim, and the stored secret is never assumed to be at fault or silently re-prompted in a loop.
- What happens when the credential expires mid-session, or a Kerberos ticket is absent or expired? The failing operation reports the authentication failure distinctly from a network failure, and re-authentication is offered without losing unsaved work.

**Certificates**

- What happens with an expired, self-signed, hostname-mismatched, or incompletely chained certificate? The connection is refused, the specific failure and full chain are shown, and proceeding requires an explicit session-only or permanent trust decision.
- What happens when a permanently trusted certificate is later replaced? The connection is refused again and the change is reported; a prior decision never covers a different certificate.

**Server limits and capabilities**

- What happens when the server's administrative size or time limit is reached? Partial results are shown, explicitly labelled as truncated by the server, with the server's result code.
- What happens when the server does not support paged results, sorting, or a requested control? The dependent option is disabled with the reason stated, and the application falls back to unpaged retrieval with the risk of truncation made explicit rather than failing.
- What happens when the server refuses a control it advertises? The refusal is surfaced verbatim and the operation is not retried silently without the control.

**Directory shape and scale**

- What happens with a container of 100,000+ children, an attribute with thousands of values, or a value of many megabytes? Rendering stays incremental and responsive; oversized values are summarised with explicit user action required to load them fully.
- What happens with looping aliases or referral chains, or referrals to unreachable hosts? Traversal depth is bounded, the loop or unreachable target is reported with the DNs involved, and no infinite retry occurs.

**Data fidelity**

- What happens with values that are not valid UTF-8, are empty, or require base64 in LDIF? They round-trip byte-for-byte; the application never substitutes replacement characters in a stored value.
- What happens with attributes absent from the published schema? They are displayed and editable as raw values, with the absence noted rather than the attribute hidden.
- What happens when a duplicate value is submitted, or two values differ only by case or whitespace the server considers equal? The server's rejection is surfaced verbatim; the application does not silently deduplicate the user's input beforehand.

**Concurrency**

- What happens when an entry is modified or deleted on the server between preview and dispatch? The change is detected, dispatch stops, and re-confirmation against the current state is required.
- What happens when the same entry is open in two editors, or one profile is connected twice? Each editor tracks its own pending state, and committing one warns that another editor holds a stale view.

**Mutation hazards**

- What happens when the user deletes the RDN attribute's value, moves an entry beneath its own descendant, or renames onto an existing DN? The condition is detected and explained before dispatch where determinable locally; otherwise the server's rejection is surfaced verbatim.
- What happens when a multi-entry operation is interrupted? Entries already written are listed, and resuming or reversing is offered; partial completion is never reported as success.

**Application state**

- What happens with unsaved edits at exit, or a profile file edited outside the application? Exit prompts for each editor holding unsaved edits; an externally changed profile is reloaded with the change reported, never silently overwritten.
- What happens when a profile references a secret no longer in the credential store, or cached data belongs to a deleted profile? The missing secret produces a recoverable prompt rather than an error dialog dead-end, and orphaned cached data is purgeable in one action.

## Requirements *(mandatory)*

### Connections, Authentication, and Transport

- **FR-001**: System MUST let users create, edit, duplicate, delete, and organise named connection profiles into nested folders, specifying host, port, encryption mode (none, StartTLS, LDAPS), bind method, bind identity, and per-profile timeouts and limits.
- **FR-002**: System MUST support anonymous bind, simple bind, and the SASL mechanisms EXTERNAL with a client certificate, GSSAPI, DIGEST-MD5, and CRAM-MD5, with realm and quality-of-protection settings where the mechanism defines them.
- **FR-003**: System MUST store bind secrets only in the operating system's credential service, MUST persist in profiles nothing but a reference to the stored secret, and MUST NOT write secret material to any application-owned file.
- **FR-004**: System MUST start without prompting for any password of its own and MUST defer the credential prompt to the platform agent, triggered by the first operation that needs a secret.
- **FR-005**: System MUST remain fully usable where no credential store is reachable, prompting for the bind credential once per session and holding it in memory only.
- **FR-006**: System MUST enable TLS certificate and hostname verification by default; disabling it MUST be an explicit per-profile choice displayed whenever that profile is connected.
- **FR-007**: System MUST present the certificate chain and the specific validation failure when verification fails, and MUST offer session-only or permanent trust as distinct choices.
- **FR-008**: System MUST provide a trust store the user can review, in which every recorded trust decision is listed with its certificate and can be revoked; a permanent decision MUST NOT extend to a different certificate later presented by the same host.
- **FR-009**: System MUST NOT downgrade to an unencrypted connection when a profile specifies StartTLS or LDAPS.
- **FR-010**: System MUST display, for every action, the target server, the active bind DN, and the current connection state.
- **FR-011**: System MUST report the identity the server considers the connection bound as, distinguished from the identity requested.
- **FR-012**: System MUST read and display the root DSE, including naming contexts, supported controls, supported extended operations, supported SASL mechanisms, and vendor and version information.
- **FR-013**: System MUST keep profiles free of secret material and safe to commit to version control, and MUST support exporting and importing them.
- **FR-014**: System MUST let users open several connections at once and move between them without disconnecting.
- **FR-015**: System MUST reconnect transparently for reads after an idle disconnect, and MUST require explicit confirmation before dispatching a write on a reconnected session.
- **FR-016**: System MUST transmit directory data to no destination other than the LDAP servers the user configured, and MUST make any telemetry, crash reporting, or update check opt-in and free of directory content and connection details.

### Browsing and Reading

- **FR-017**: Users MUST be able to browse the directory information tree from the root DSE and from any naming context, expanding containers lazily.
- **FR-018**: System MUST page tree expansion and search results, MUST show how many entries have been loaded, and MUST surface the server's size and time limits instead of silently truncating.
- **FR-019**: System MUST display an entry's user attributes and, on request, its operational attributes, with values shown exactly as returned by the server.
- **FR-020**: Users MUST be able to view and edit any distinguished name as text.
- **FR-021**: System MUST let users control alias dereferencing and referral handling per operation, supporting following referrals, ignoring them, or listing them for manual inspection, and MUST bound traversal depth and report loops with the DNs involved.
- **FR-022**: System MUST let users request supported LDAP controls — including paged results, server-side sorting, virtual list views, subentries, and manage-DSA-IT — and MUST report verbatim when a server refuses one.
- **FR-023**: System MUST let users filter the displayed children of a container without issuing a new search.
- **FR-024**: Users MUST be able to bookmark entries per connection and reach them in one action.
- **FR-025**: Users MUST be able to refresh any entry or subtree on demand, and the system MUST NOT commit any pending change as a side effect of navigation, refresh, or expansion.
- **FR-026**: System MUST summarise rather than eagerly render attribute values above a size threshold, loading them fully only on explicit user action.
- **FR-027**: System MUST display attributes absent from the published schema rather than hiding them, noting the absence.

### Search

- **FR-028**: Users MUST be able to write RFC 4515 filter strings directly, and the system MUST NOT rewrite, normalise, or reorder them before transmission.
- **FR-029**: System MUST validate filter syntax locally before dispatch and report the position and nature of any error without contacting the server.
- **FR-030**: System MUST offer schema-derived completion of attribute names and object classes while composing a filter, as an optional aid that never alters typed text.
- **FR-031**: System MUST provide a visual filter builder that generates a filter string, and switching to the text form MUST expose that string for direct editing, with edits made there taking precedence.
- **FR-032**: Users MUST be able to set search base, scope, returned attribute list, size limit, time limit, alias and referral handling, and requested controls per search.
- **FR-033**: System MUST present results in a table whose columns are selectable and sortable, exportable without re-running the search.
- **FR-034**: System MUST support server-side ordering where the server advertises it, and MUST state the reason when the option is unavailable.
- **FR-035**: Users MUST be able to save searches with all their parameters, re-run them, and run them against a different connection.
- **FR-036**: System MUST retain a per-connection history of executed searches that can be re-opened and re-run.
- **FR-037**: System MUST label a truncated result set as truncated by the server and show the server's result code.
- **FR-038**: System MUST fall back to unpaged retrieval, with the truncation risk stated, against a server that does not support paged results, rather than failing the search.

### Editing and Mutations

- **FR-039**: System MUST require, before dispatching any operation that modifies the directory, a preview stating the target DN, an attribute-level before-and-after diff, and the total number of affected entries, and MUST require explicit confirmation.
- **FR-040**: System MUST dispatch nothing when a preview is cancelled, and MUST preserve the pending edits.
- **FR-041**: Users MUST be able to add attributes, add and remove values, replace values, and delete attributes on an existing entry.
- **FR-042**: Users MUST be able to create entries by choosing object classes, by using a saved template, or by authoring raw LDIF, and every guided path MUST expose the raw-operation escape hatch.
- **FR-043**: Users MUST be able to create a new naming context entry where the server permits it.
- **FR-044**: Users MUST be able to save, reuse, and share entry templates that pre-populate object classes and attributes.
- **FR-045**: System MUST preserve attribute description options, including binary markers and language tags, as separately editable, and MUST NOT merge values that differ only by option.
- **FR-046**: System MUST support deleting a single entry and recursively deleting a subtree, MUST state the subtree size for a non-leaf delete, and MUST require a confirmation distinct from that of a single-entry delete.
- **FR-047**: System MUST offer a dry run for every bulk and recursive operation that executes the full validation path and sends no write.
- **FR-048**: Users MUST be able to rename an entry and move a subtree, choosing explicitly whether the old RDN value is retained or deleted.
- **FR-049**: Users MUST be able to copy or move entries and subtrees within one server and between two connected servers, choosing collision behaviour (skip, overwrite, merge) and whether operational attributes are carried across, before execution.
- **FR-050**: System MUST report a partially completed multi-entry operation by listing entries already written and offering to resume or reverse, and MUST NOT report partial completion as success.
- **FR-051**: System MUST display the server's result code and diagnostic message verbatim on both success and failure, alongside — never replaced by — any plain-language interpretation.
- **FR-052**: System MUST detect that an entry changed on the server between preview and dispatch and MUST require re-confirmation against the current state rather than overwriting silently.

### Interchange Formats

- **FR-053**: System MUST import and export RFC 2849 LDIF as the primary interchange format, covering both content records and change records.
- **FR-054**: System MUST preserve values byte-for-byte across an LDIF export and re-import, including binary values, values needing base64, empty values, and folded long lines.
- **FR-055**: System MUST provide an LDIF editor with syntax highlighting, validation reporting the line number of each error, and execution against a chosen connection under the standard preview and confirmation.
- **FR-056**: Users MUST be able to choose, before executing an LDIF file, whether a failing record halts execution or is skipped and logged, and MUST receive a per-record outcome report.
- **FR-057**: System MUST import and export DSML under the same preview, confirmation, and per-record reporting as LDIF.
- **FR-058**: System MUST export entries, subtrees, and search results to CSV, JSON, and spreadsheet workbooks, representing multi-valued and binary attributes losslessly.
- **FR-059**: Users MUST be able to copy a selected entry or result row to the clipboard as a DN, as LDIF, or as delimited text.
- **FR-060**: System MUST stream large imports and exports with visible progress and a working cancel, without loading the whole data set into memory.

### Schema Browsing and Schema-Aware Editing

- **FR-061**: System MUST read and display the connected server's object classes, attribute types, syntaxes, and matching rules with OIDs, descriptions, superiors, and origin.
- **FR-062**: Users MUST be able to search the schema and navigate between referenced elements in both directions.
- **FR-063**: System MUST use the schema, when available, to enumerate required and optional attributes, flag single-valued and read-only attributes, and warn before dispatching a change that conflicts with the schema — as a warning the user may override, never as a silent block.
- **FR-064**: System MUST remain fully usable against a server publishing no readable schema, disabling schema-derived assistance and stating why.
- **FR-065**: Users MUST be able to compare the schemas of two connections and see elements unique to each and elements defined differently in each.
- **FR-066**: System MUST display the raw definition text of any schema element.

### Offline Schema Projects

- **FR-067**: Users MUST be able to create a schema project that is empty, imported from a connected server, or imported from schema definition files, and to work in it with no server contact.
- **FR-068**: Users MUST be able to create and edit object classes and attribute types in a project, setting OID, names, description, superior, syntax, matching rules, usage, and single-valued and obsolete flags, with the resulting definition text viewable.
- **FR-069**: System MUST check a project for structural errors — dangling references, duplicate OIDs, duplicate names, and circular superior chains — and report each with the elements involved.
- **FR-070**: Users MUST be able to compare a project against another project or a live server, and to export a project to schema definition files.
- **FR-071**: Users MUST be able to commit the difference between a project and a server to that server, under the standard preview and confirmation, and the system MUST report a server's refusal to modify its schema verbatim while leaving the project unchanged.

### Value Editors

- **FR-072**: System MUST provide specialised editors for password, binary, certificate, image, DN, generalized-time, boolean, integer, postal-address, object-class, and multi-line text values.
- **FR-073**: System MUST provide a byte-level editor for values whose syntax it does not recognise, and MUST NOT apply a lossy text conversion to them.
- **FR-074**: Every specialised editor MUST expose the underlying raw value for direct text or byte editing, and the raw value MUST take precedence.
- **FR-075**: System MUST let users set a password using a selectable hash scheme and verify a candidate password against a stored value.
- **FR-076**: System MUST offer the standard password-change extended operation where the server advertises it, in preference to a direct attribute write, reporting the server's response verbatim.
- **FR-077**: System MUST never write password plaintext to disk, to logs, or to diagnostic exports.
- **FR-078**: System MUST decode certificate values to subject, issuer, validity window, and fingerprints while keeping the encoded bytes viewable and editable.
- **FR-079**: Users MUST be able to load a value from a file and save a value to a file with no transformation of its bytes.
- **FR-080**: System MUST let users pick a DN through a tree browser and MUST transmit exactly the DN chosen.

### Access Control and Administrative Model

- **FR-081**: System MUST provide a structured editor for access-control item values that also exposes the exact stored string for direct editing.
- **FR-082**: System MUST provide a structured editor for subtree specifications, and MUST let the user preview the set of entries a specification selects before it is saved.
- **FR-083**: System MUST provide an editor for administrative role values and MUST support creating and editing the subentries that carry administrative policy.
- **FR-084**: System MUST present the raw text with the parse failure explained when a structured editor cannot parse a value, and MUST NOT silently rewrite or discard it.
- **FR-085**: System MUST disable these editors, stating why, against a server that does not support stored access-control rules, without any loss of generic entry editing.

### Server Configuration Held as Entries

- **FR-086**: System MUST let users browse and edit a server's configuration naming context through the standard tree and entry editors where the server exposes configuration as directory entries.
- **FR-087**: System MUST identify the target as server configuration in the preview and warn that the change may alter server behaviour immediately.
- **FR-088**: System MUST report a rejected configuration change with the server's verbatim result code and diagnostic message and MUST NOT present partial state as success.

### History, Comparison, and Bulk Operations

- **FR-089**: System MUST record every write it performs in a per-profile history holding timestamp, bind identity, server, the operation as transmitted, and the server's response, and MUST record an interrupted operation as indeterminate rather than as success or failure.
- **FR-090**: System MUST retain a per-profile log of executed searches alongside the write history.
- **FR-091**: Users MUST be able to export any history record as LDIF and replay it against a chosen connection under the standard preview and confirmation.
- **FR-092**: System MUST generate a reversing LDIF from a captured before-state, and MUST state that an operation cannot be reversed rather than producing an incomplete reversal.
- **FR-093**: System MUST redact bind credentials, password values, and attributes configured as sensitive from every log, history record, and diagnostic export.
- **FR-094**: Users MUST be able to compare two entries, two subtrees, or one search run against two connections, with operational attributes excludable and specific attributes ignorable.
- **FR-095**: System MUST generate reconciling LDIF from a comparison result for review and export, and MUST NOT apply it without the standard preview and confirmation.
- **FR-096**: Users MUST be able to apply a single change to every entry in a search result set, with a preview, a dry run, progress, cancellation at entry boundaries, and an exportable per-entry outcome report.
- **FR-097**: System MUST report bulk outcomes per entry, distinguishing succeeded, failed with the server's verbatim message, and skipped.

### Application Behaviour

- **FR-098**: System MUST perform all network input and output off the interface thread; the interface MUST NOT freeze during a bind, search, or write.
- **FR-099**: Every operation that contacts a server MUST be cancellable and MUST honour a configurable timeout.
- **FR-100**: System MUST scope any cached directory data per connection profile and MUST let the user purge it, including data orphaned by a deleted profile, in a single action.
- **FR-101**: System MUST run on Linux, macOS, and Windows with the same capabilities on each.
- **FR-102**: System MUST be operable from the keyboard for all primary flows and MUST honour the operating system's light and dark appearance setting.
- **FR-103**: System MUST warn about unsaved edits before closing an editor, a connection, or the application, identifying each editor holding unsaved work.
- **FR-104**: System MUST reload a profile changed outside the application, reporting the change rather than silently overwriting it.
- **FR-105**: System MUST track pending state per editor and MUST warn on commit when another open editor holds a stale view of the same entry.
- **FR-106**: System MUST distinguish an authentication failure from a network failure in every error it reports.
- **FR-107**: System MUST externalise all user-facing strings so that translation requires no change to behaviour.
- **FR-108**: System MUST ship a migration path whenever the on-disk format of profiles, schema projects, history, or cached data changes.

### Key Entities

- **Connection Profile**: A named, persisted definition of how to reach one directory — address, port, encryption mode, bind method, bind identity, a reference to a stored secret, timeouts, limits, referral and alias defaults. Contains no secret material. Organised within Connection Folders.
- **Connection Folder**: A nestable grouping of connection profiles, used for organisation and for scoping actions across profiles.
- **Trust Decision**: A recorded acceptance of a specific server certificate, session-only or permanent, reviewable and revocable, bound to the exact certificate accepted.
- **Directory Entry**: A node in the tree, identified by its DN, holding user and operational attributes and a position relative to a parent and children.
- **Attribute Value**: A single value of an attribute on an entry, with its raw bytes, the attribute's syntax, any description options such as a binary marker or language tag, and whether it is binary or text.
- **Schema Element**: An object class, attribute type, syntax, or matching rule, with an OID, names, description, superior references, flags, and origin — read from a server or authored in a project.
- **Schema Project**: An offline, editable collection of schema elements with a source (empty, a server, or files), against which structural checks, comparisons, exports, and server commits are run.
- **Search Definition**: Base, scope, filter string, requested attributes, limits, alias and referral handling, and requested controls; saveable, nameable, and runnable against any connection.
- **Bookmark**: A named reference to a DN within one connection.
- **Entry Template**: A reusable starting point for a new entry, holding object classes and pre-populated attribute values.
- **Interchange Document**: An LDIF, DSML, CSV, JSON, or spreadsheet file produced by export or consumed by import, holding entries or change records.
- **Change Set**: The pending modifications assembled in an editor and rendered as a preview — target DN, per-attribute before-and-after values, and affected entry count — before any dispatch.
- **Operation Record**: One completed request in a profile's history — timestamp, bind identity, server, the operation as transmitted, captured before-state where available, and the server's result code and diagnostic message, or an indeterminate marker where the connection was lost.
- **Access Control Item**: A stored authorisation rule, held as its exact text and presented through a structured editor, scoped by a Subtree Specification.
- **Subtree Specification**: A definition of the region of the tree to which a policy applies, previewable as the set of entries it selects.
- **Comparison Result**: The structural difference between two entries, subtrees, projects, or schemas — items unique to each side and per-item differences — from which reconciling LDIF can be generated.
- **Bulk Job**: A single change applied across a result set, with its dry-run projection, progress, and per-entry outcome report.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user who has never opened the application can define a connection, connect over StartTLS, and read an entry's attributes within 3 minutes without consulting documentation.
- **SC-002**: The application window remains responsive to input at all times, including during binds, searches over 100,000 entries, and bulk writes; no operation blocks the interface for more than 100 milliseconds.
- **SC-003**: Every operation that contacts a server can be cancelled and stops within 2 seconds of the request.
- **SC-004**: A directory container with 100,000 children begins displaying entries within 2 seconds and scrolls without stutter.
- **SC-005**: 100% of directory-modifying operations present an attribute-level preview and require confirmation before dispatch; an automated check over every mutation path finds no exception.
- **SC-006**: 100% of server errors surfaced to the user retain the server's original result code and diagnostic message.
- **SC-007**: An LDIF export followed by an import into an empty directory reproduces every entry and every attribute value byte-for-byte, verified over a corpus containing binary, non-UTF-8, empty, option-bearing, and multi-megabyte values.
- **SC-008**: No secret material — bind credentials or password values — appears in any file the application writes, verified by an automated scan of all profiles, caches, logs, history records, and crash output.
- **SC-009**: The application launches to a usable window in under 3 seconds on a mid-range machine and never prompts for a password of its own at startup.
- **SC-010**: Every capability area in the Parity Reference appendix is marked implemented, and each of the four recorded exclusions is either still justified or removed, before v1 is released.
- **SC-011**: A practitioner migrating from the reference tool completes browse, search, edit, export, and schema-authoring tasks without needing a capability the appendix marks implemented but that is in fact missing; zero such gaps are open at release.
- **SC-012**: A user can reverse a completed single-entry modification through the history in under 60 seconds, and the directory returns to its exact prior state.
- **SC-013**: A comparison of two 10,000-entry subtrees completes within 30 seconds and reports every seeded difference with no false positives.
- **SC-014**: A dry run of a bulk change over 500 entries produces per-entry outcomes identical to the subsequent real execution for at least 99% of entries, with any divergence attributable to a concurrent server-side change.
- **SC-015**: In usability testing with practitioners familiar with the reference tool, at least 90% complete browse, search, edit, and export tasks on first attempt without assistance.
- **SC-016**: The application operates fully against a server offering no schema, no paged-results control, and no extended operations, with each unavailable capability explained rather than failing silently.
- **SC-017**: Every edge case listed in this specification has an automated test asserting the stated behaviour; coverage of that list is 100% at release.
- **SC-018**: Every one of the four deliberate exclusions is discoverable in-product: a user attempting the excluded action is told it is out of scope and what to use instead, rather than meeting a dead end.

## Assumptions

- **Audience**: Users understand LDAP concepts — DNs, object classes, filters, LDIF. The product optimises for their fluency rather than teaching the protocol. The Glossary above exists for stakeholders reading this document, not as a statement about the product's users.
- **Parity definition**: "All of Apache Directory Studio" is resolved concretely in the Parity Reference appendix rather than left to interpretation. Every capability area there is in scope for v1 except the four recorded exclusions.
- **Server-agnostic**: v1 targets any RFC 4510-conformant LDAP v3 server, with OpenLDAP as the baseline for integration testing. Capabilities that depend on optional server support — stored access-control rules, schema modification, configuration-as-entries, paging, sorting, extended operations — degrade explicitly with the reason stated, never silently.
- **Vendor neutrality**: No server vendor gets a bespoke surface in v1. Configuration editing, access control, and schema commit are specified against the standard mechanisms, which is what makes them work for more than one server.
- **Single-user desktop**: A local desktop tool with no server component, no shared backend, and no accounts of its own. All authorisation is the directory's; the application never adds a permission model.
- **Sharing**: Profiles, saved searches, templates, and schema projects are shared as files. There is no built-in synchronisation service.
- **Offline with respect to third parties**: The only network destinations are the LDAP servers the user configured.
- **Platforms**: Linux, macOS, and Windows on current 64-bit desktop hardware, each using its native credential service.
- **Localisation**: English-only interface for v1, with strings externalised so translation needs no code change.
- **Accessibility**: Full keyboard operability and OS appearance settings are in scope for v1; screen-reader certification is a follow-on goal.

## Dependencies

- **Directory servers for testing**: Containerised LDAP servers must be available in development and continuous integration, covering a server with stored access-control rules and modifiable schema, and a server without either, so that both the capability and the degradation paths are exercised.
- **Platform credential services**: The product depends on a working system credential service on each platform. Where absent or locked, the session-only credential path (FR-005) is the fallback, and that path must be tested on each platform.
- **Kerberos infrastructure**: GSSAPI authentication depends on a working Kerberos configuration and credential cache on the user's machine; the product consumes it and does not manage it.
- **Platform trust roots**: Certificate validation depends on the operating system's trust store for its root set, supplemented by the application's own trust decisions (FR-008).
- **Interchange standards**: LDIF (RFC 2849), filter syntax (RFC 4515), and the LDAP v3 protocol suite (RFC 4510 and related) are treated as fixed external contracts.
- **No dependency on an existing codebase**: This is a greenfield product; nothing here assumes reuse of the reference tool's code, and the technology stack is undecided and settled at planning time.

## Out of Scope for v1

Four capabilities of the reference tool are deliberately excluded; each is discoverable in-product per SC-018.

- **Managing embedded or local directory server instances** — creating, starting, stopping, and configuring server installations from within the application. This is server lifecycle management, not directory work, and it ties the product to one server implementation. Users manage servers with their own platform tooling.
- **A plugin runtime for user-authored extensions** — third-party value editors and custom views loaded at runtime. It is a large surface area with a security boundary of its own, and it presumes an extension ecosystem that does not exist on day one.
- **A command-line companion binary** — batch execution outside the desktop application. The bulk operations story covers the underlying need from within the product.
- **Vendor-specific administrative consoles** — bespoke surfaces for one server's configuration model, replication topology, or monitoring. The generic configuration-as-entries story covers the standard mechanism instead.

Also out of scope, and not reference-tool capabilities: replication management and monitoring, continuous synchronisation between directories beyond comparison and reconciling LDIF, mobile or web clients, and certificate authority management.

## Parity Reference

The concrete resolution of "all of Apache Directory Studio". Every area is in scope for v1 and traces to the requirements that satisfy it. SC-010 is verified against this table.

| Capability area | Requirements | Status |
|-----------------|-------------|--------|
| Connection management, folders, profile import/export | FR-001, FR-013, FR-014 | In scope |
| Authentication: anonymous, simple, EXTERNAL, GSSAPI, DIGEST-MD5, CRAM-MD5 | FR-002 | In scope |
| Transport: StartTLS, LDAPS, certificate trust management | FR-006 – FR-009 | In scope |
| Root DSE inspection, server capability discovery | FR-012 | In scope |
| DIT browsing, paged expansion, child filtering, bookmarks | FR-017, FR-018, FR-023, FR-024 | In scope |
| Operational attributes, aliases, referrals, LDAP controls | FR-019, FR-021, FR-022 | In scope |
| Entry editor: table view, template view, LDIF view of an entry | FR-041, FR-042, FR-044, FR-074 | In scope |
| New entry wizards, new context entry, entry templates | FR-042 – FR-044 | In scope |
| Search: raw filters, filter builder, saved searches, search history | FR-028 – FR-036 | In scope |
| Result table: column selection, sorting, export | FR-033, FR-034, FR-058, FR-059 | In scope |
| Rename, move, copy, cross-server copy | FR-048, FR-049 | In scope |
| Delete, recursive delete | FR-046 | In scope |
| LDIF editor, LDIF import/export, LDIF execution | FR-053 – FR-056 | In scope |
| DSML, CSV, spreadsheet export and import | FR-057, FR-058 | In scope |
| Schema browser with element navigation | FR-061, FR-062, FR-066 | In scope |
| Schema-aware entry editing | FR-063 | In scope |
| Schema Editor: offline projects, element authoring, checks, comparison, export, server commit | FR-067 – FR-071 | In scope |
| Value editors: password, binary/hex, certificate, image, DN, time, boolean, integer, postal address, object class, text | FR-072 – FR-080 | In scope |
| ACI item editor | FR-081, FR-084 | In scope |
| Subtree specification editor | FR-082 | In scope |
| Administrative role editor, subentry handling | FR-083 | In scope |
| Modification logs, search logs | FR-089, FR-090 | In scope, extended by FR-091, FR-092 |
| Batch operations across a result set | FR-096, FR-097 | In scope, extended by dry run (FR-047) |
| Managing embedded server instances | — | **Excluded**, see Out of Scope |
| Plugin runtime for third-party extensions | — | **Excluded**, see Out of Scope |
| Vendor-specific configuration consoles | FR-086 – FR-088 cover the generic mechanism | **Excluded** as a bespoke surface |

Capabilities beyond the reference tool: OS credential storage with no application-owned secret (FR-003 – FR-005), mandatory preview on every mutation (FR-039), dry run (FR-047), reversible history (FR-091, FR-092), cross-server comparison with reconciling LDIF (FR-094, FR-095), password-change extended operation (FR-076), attribute description options (FR-045), server-side sorting and virtual list views (FR-022, FR-034), and stale-editor detection (FR-105).
