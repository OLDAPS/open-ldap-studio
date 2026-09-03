# A directory to develop against

A seeded OpenLDAP instance, so the app has something real to browse.

```sh
cd server
docker compose up -d          # wait for healthy, ~20s on a cold start
docker compose down           # stop, keep the data
docker compose down -v        # stop and discard the data, next up re-seeds
```

## Connect from the app

Open the Connections perspective, press **+**, and fill in the wizard:

| Wizard field | Value |
|---|---|
| Connection name | `local-dev` |
| Hostname | `localhost` |
| Port | `1389` |
| Encryption | *No encryption*, or *StartTLS* |
| Bind DN | `cn=admin,dc=example,dc=org` |
| Password | `adminpassword` |
| Base DN | `dc=example,dc=org` |

Ports are unprivileged (1389 / 1636 rather than 389 / 636) so none of this
needs root.

### Other identities

| Bind DN | Password | For |
|---|---|---|
| `cn=admin,dc=example,dc=org` | `adminpassword` | full read/write |
| `cn=configadmin,cn=config` | `configpassword` | browsing `cn=config` |
| `cn=readonly-svc,ou=services,dc=example,dc=org` | `readonly` | a non-admin bind |
| `cn=jrivera,ou=people,dc=example,dc=org` | `password` | any seeded person |

Every generated person uses the password `password`.

### TLS

StartTLS on 1389 and LDAPS on 1636 both work. The certificate is **self-signed
on purpose** — an untrusted chain is what the app's certificate-trust dialog
exists to handle, and a fixture that never triggers it cannot exercise it. So
expect the app to challenge you the first time, which is the behaviour under
test.

The certificate covers `localhost`, `127.0.0.1` and `ldap.example.org`. To
inspect or trust it out of band:

```sh
docker compose cp certs:/certs/ca.crt ./ca.crt   # while the certs container exists
# or
openssl s_client -connect localhost:1636 -showcerts </dev/null
```

## What is in the directory

216 entries under `dc=example,dc=org`:

```
dc=example,dc=org
├── ou=people      205 entries — 5 named, 200 generated
├── ou=groups        4 groupOfNames, with DN-valued members
├── ou=services      2 bind accounts
└── ou=archive       empty, on purpose
```

The fixture is deliberately awkward as well as tidy, so each screen has
something real to show:

| In the data | The screen it serves |
|---|---|
| 205 children under `ou=people` | the tree's explicit "fetch next 100 of N…" node — paging is reachable, not theoretical |
| `jpegPhoto` on all five named people | entry-info photo panel, image value editor |
| `userPassword` as `{SSHA}` | password value editor, which reports the scheme |
| `manager`, `member` | DN picker, and reference chasing |
| `posixAccount` alongside `inetOrgPerson` | object-class tab's structural / auxiliary split |
| `cn=tvasquez` — `Tomás Vásquez`, accented description | UTF-8 in the tree label, grid and LDIF export |
| `cn=lchen` description leads with a space | LDIF export **must** base64 it; if a round-trip loses that space, this entry shows it |
| `cn=jrivera` description > 120 chars | value truncation, and the editor that shows the whole thing |
| `cn=aschmidt` has 4 `mail` values | the "+N more" fold, which defaults to 3 |
| `ou=archive` is empty | a safe target for copy / move / subtree delete |
| `cn=config` is browsable | configuration screens |

Re-seeding happens only on a fresh volume, so edits made from the app persist
across `down` / `up`. Use `down -v` to get back to the fixture as shipped.

## Changing the fixture

`seed/*.ldif` is generated and committed, so `docker compose up` needs nothing
but Docker. To change it, edit the generator and re-run:

```sh
python3 make-seed.py          # needs Pillow, for the photos
docker compose down -v && docker compose up -d
```

`BULK_USERS` at the top of `make-seed.py` controls how many filler people are
produced — raise it if you want to page further.

## ApacheDS

OpenLDAP is the default because `test/containers/openldap.go` already runs it,
so the app is developed against the same server its integration tests assert
on, and because it seeds straight from LDIF.

ApacheDS is available for the access-control screens, which need X.500
`prescriptiveACI` — something OpenLDAP does not implement:

```sh
docker compose --profile apacheds up -d     # port 10389
```

Bind as `uid=admin,ou=system` / `secret`. It starts with an empty
`dc=example,dc=com` partition.

**It is not seeded, and it cannot hold this fixture as-is.** Loading
`seed/*.ldif` into it was tried and these are the results:

| | |
|---|---|
| Base DN | its built-in partition is `dc=example,dc=**com**`, not `.org` — the LDIF needs rewriting or a new partition configured |
| `posixAccount` | the `nis` schema ships **disabled**; enable it first, or the five named people are rejected |
| `jpegPhoto` | rejected as `INVALID_ATTRIBUTE_SYNTAX` even with a valid JFIF image, so the photo fixtures are lost |
| Seeding | there is no LDIF-directory contract like slapd's, so seeding means an `ldapadd` sidecar after start-up |
| TLS | not configured in this image, so the certificate-trust screen has nothing to trigger it |

With the `nis` schema enabled and `jpegPhoto` dropped, 211 of 215 entries load.
Treat ApacheDS as a second server for the ACI editor and the two-connection
compare — not as a copy of the fixture.

## Note on the image

The compose file pins `bitnamilegacy/openldap:2.6.10`, not `bitnami/openldap`.
Bitnami moved its free catalogue to the `bitnamilegacy` namespace in 2025 and
`docker.io/bitnami/openldap` no longer resolves. The legacy images are frozen
but complete, and they keep the environment-variable contract the Go test
harness is written against.

Nothing here depends on Bitnami as such — what it depends on is an OpenLDAP
image that seeds from an LDIF directory, and `bitnamilegacy` is the one that
still does.

Two images the Go test harness names are now unpullable and will fail the same
way:

- `test/containers/openldap.go` → `bitnami/openldap:latest`
- `test/containers/apacheds.go` → `marcelocg/apacheds:latest` (gone entirely;
  `openmicroscopy/apacheds` is what this compose file uses instead)
