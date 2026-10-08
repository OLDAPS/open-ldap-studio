# Contract: On-Disk Formats

**Date**: 2026-08-31 | **Plan**: [../plan.md](../plan.md)

Every persisted artefact and its migration contract. FR-108 requires a migration path for any
change to these, and FR-013 requires profiles to be safe to commit to version control.

**Universal rules**

1. Every file carries `"schemaVersion": <int>` as its first key, from the first release. Retrofitting
   a version field onto unversioned files is not possible, which is why it ships on day one.
2. A file whose `schemaVersion` exceeds what this build knows is **not** opened and not rewritten —
   it reports that a newer version wrote it.
3. No file in this contract may contain secret material. Enforced by the CI scan behind SC-008.

**Location**: platform config dir (`$XDG_CONFIG_HOME/open-ldap-studio`,
`~/Library/Application Support/OpenLDAPStudio`, `%APPDATA%\OpenLDAPStudio`); caches in the platform
cache dir, so purging caches never touches configuration.

---

## `profiles.json` — connection profiles and folders

```jsonc
{
  "schemaVersion": 1,
  "folders": [{ "id": "…", "name": "Production", "parentId": null }],
  "profiles": [{
    "id": "…", "name": "corp-prod", "folderId": "…",
    "host": "ldap.example.com", "port": 636, "encryption": "ldaps",
    "tls": { "verifyCertificate": true, "verifyHostname": true, "clientCertRef": "" },
    "credentialId": "cred-7f3a",                   // → credentials.json → secretRef → OS store
    "readOnly": false, "tags": ["production"],
    "timeouts": { "connectMs": 10000, "readMs": 30000 },
    "limits": { "sizeLimit": 0, "timeLimit": 0, "pageSize": 1000 },
    "aliases": "never", "referrals": "ask"
  }]
}
```

`credentialId` points at `credentials.json`, which holds the `secretRef` — the lookup key in the
platform credential service. The file is designed to be committed to a repository; a reviewer can
verify that by inspection, which is the point.

## `credentials.json` — credential definitions (no secrets)

Second-pass addition; see research R10 and data-model §1.1.

```jsonc
{
  "schemaVersion": 1,
  "credentials": [{
    "id": "cred-7f3a", "name": "admin · corp", "type": "simple",
    "bindDN": "cn=admin,dc=example,dc=com",
    "secretRef": "open-ldap-studio/cred-7f3a",   // a REFERENCE, resolved via the OS store
    "createdAt": "2026-06-28T09:00:00Z", "lastUsedAt": "2026-08-31T09:41:00Z"
  }]
}
```

**This file has no vault.** There is no encrypted blob, no master-password salt, no lock state, and
no export format that carries a secret — all prohibited by Constitution II. `secretRef` is a lookup
key and nothing else, which is why this file is as safe to commit as `profiles.json`.

## `trust.json` — certificate trust store

```jsonc
{
  "schemaVersion": 1,
  "decisions": [{
    "host": "ldap.example.com", "port": 636,
    "fingerprintSha256": "…",          // the identity of the decision
    "chainPem": "-----BEGIN CERTIFICATE-----…",
    "acceptedAt": "2026-08-31T10:00:00Z",
    "reason": "self-signed certificate in chain"
  }]
}
```

Session-scoped decisions are held in memory and never written here (data-model §1).

## `history/<profileId>.jsonl` — operation history

Append-only, one JSON object per line, so a crash truncates at most the last record.

```jsonc
{"schemaVersion":1,"id":"…","ts":"…","serverIdentity":"ldap.example.com:636",
 "bindDN":"cn=admin,…","kind":"modify","request":"dn: …\nchangetype: modify\n…",
 "result":{"code":0,"matchedDN":"","diagnosticMessage":""},
 "beforeStateRef":"history/blobs/…","reversible":true}
```

Redaction is applied **before** the line is written (FR-093). `beforeStateRef` points at a blob;
its absence sets `reversible: false`, which is what `ReversalAvailable` reports.

`history/searches/<profileId>.jsonl` follows the same shape for search records (FR-090).

## `projects/<id>/project.json` — schema project

```jsonc
{
  "schemaVersion": 1, "id": "…", "name": "corp-schema",
  "source": { "kind": "fromServer", "profileId": "…", "importedAt": "…" },
  "elements": [{
    "kind": "objectClass", "oid": "2.16.840.1.113730.3.2.2",
    "names": ["inetOrgPerson"], "superiors": ["organizationalPerson"],
    "classType": "structural", "must": ["cn","sn"], "may": ["mail","uid"],
    "obsolete": false,
    "rawDefinition": "( 2.16.840.1.113730.3.2.2 NAME 'inetOrgPerson' …)"
  }]
}
```

`rawDefinition` is retained verbatim alongside the parsed form. Export re-emits it unchanged for
any element the user did not edit — the same raw-is-truth rule as ACIs and filters.

Also exportable as `.schema` files and as LDIF for a server commit (FR-070).

## Workspace files

| File | Contents |
|------|----------|
| `searches.json` | Saved searches — filter stored as a raw string, no profile binding (FR-035) |
| `bookmarks.json` | Per-profile DN bookmarks |
| `templates.json` | Entry templates |
| `preferences.json` | Panes `3a` and `5a`–`5h`, including the syntax→value-editor mapping (`5c`) and the keymap (`5g`). Update checks default to `never` (FR-016) |
| `workspace/*.ldif` | User LDIF documents in the editor's file tree (`1e`) |

## Cache (separate root, purgeable)

`cache/<profileId>/` — schema snapshot, tree structure hints, search history index. Purgeable in a
single action including data orphaned by a deleted profile (FR-100). Never contains secrets, and
never the only copy of anything.

---

## Interchange formats (not application state)

Written and read by import/export, and governed by their standards rather than by this contract:
**LDIF** (RFC 2849, byte-fidelity per SC-007), **DSML v2**, **CSV**, **JSON**, **XLSX**, **ODS**.
The exported profile bundle uses the `profiles.json` shape above.

---

## Contract tests

| # | Assertion | Serves |
|---|-----------|--------|
| F1 | Every written file has `schemaVersion` as its first key | FR-108 |
| F2 | A file with a higher `schemaVersion` is refused, not rewritten | FR-108 |
| F3 | A scan of every written file finds no secret material, over a session exercising all stories | SC-008 |
| F4 | A profile bundle round-trips export → import with no field loss | FR-013 |
| F5 | History redaction happens at write time — the raw file never contains a password value | FR-093 |
| F6 | `PurgeCache` removes orphaned profile caches too | FR-100 |
| F7 | `credentials.json` contains no secret material and no vault structure | Constitution II, R10 |
| F8 | Logs rotate at 10 MB × 3 and redact before writing | R17, FR-093 |
| F9 | A rejects `.ldif` from a failed run re-executes cleanly | flow 7e, FR-056 |
