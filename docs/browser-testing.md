# Critical E2E tests

The default suite runs Chromium → the real Wails websocket dispatcher → the
Go bridge → a disposable OpenLDAP service. Its 13 cases cover shipped
connection and directory workflows through visible controls.

## Test layers

| Suite | Location | Purpose |
| --- | --- | --- |
| Critical E2E | `frontend/e2e/live/critical/` | Complete user journeys with real application and LDAP state |
| Wails IPC integration | `frontend/e2e/live/ipc/` | Changesets and certificate trust whose dialogs are not connected to the shell |
| Synthetic frontend checks | `frontend/browser_tests/` | Deterministic UI states using an injected bridge; these are not E2E tests |
| Component and unit tests | `frontend/tests/` | UI details, formatting, validation and pure logic |
| Go integration tests | `internal/bridge/`, `test/containers/` | LDAP behavior and exhaustive protocol cases |

CI runs the critical E2E and IPC suites together, sharing one application build
and LDAP service. Synthetic checks remain available separately; see
[frontend browser checks](frontend-browser-testing.md).

## Setup

Run on Linux with Docker, Docker Compose **2.24.4+**, the Go version declared in
`go.mod`, Node.js 22+, and Python 3.10+. Install GTK 3, WebKitGTK 4.1 and a display
server. On Ubuntu:

```sh
sudo apt-get install libgtk-3-dev libwebkit2gtk-4.1-dev xvfb xauth
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install -r frontend/requirements-e2e.txt
python3 -m playwright install --with-deps chromium
npm --prefix frontend ci
```

The Docker daemon must be running and accessible to your user. Missing Docker,
a failed seed, an unavailable display or a failed build fails the run.

## Run

With the Python environment active, from the repository root:

```sh
make test-e2e                         # visible Chromium, one test at a time
make test-e2e-headless                # critical journeys, four workers locally
make test-ipc-headless                # Wails IPC integration
xvfb-run -a make test-e2e-headless     # no desktop display required
```

Equivalent commands in `frontend/` are `npm run test:e2e`,
`npm run test:e2e:headless`, `npm run test:ipc` and
`npm run test:ipc:headless`. CI uses two workers. All runs have no automatic
retries: an intermittent failure must be investigated.

Normal pytest arguments select individual files, tests and worker counts:

```sh
cd frontend
npm run test:e2e:headless -- -k tls
npm run test:ipc:headless -- -n 0 e2e/live/ipc/test_changesets.py
npm run test:e2e:headless -- e2e/live/critical e2e/live/ipc  # both live suites
python3 -m pytest --collect-only                          # critical journeys
python3 -m pytest --suite ipc --collect-only
python3 -m pytest --suite frontend --collect-only
```

Collection does not start Docker, build the application or require a display.
`--suite` chooses the default test directory and environment; explicit pytest
paths and node IDs are preserved. `--frontend-only` remains an alias for
`--suite frontend`.

## Critical journeys

| Journey | Observable result |
| --- | --- |
| Connect, reload, navigate, disconnect and reconnect | LDAP identity and an open entry survive workspace navigation; reconnect reads a different entry successfully |
| Check network and recover from rejected credentials | The server supplies the base DN and LDAP result 49; corrected credentials bind; cancelling saves nothing |
| Connect with a saved bad password | A real connection job fails, exposes its diagnostic and never falls back to anonymous |
| Connect to an unreachable server | The network diagnostic is visible and no profile is saved |
| Save profile and restart the native application | Options survive a process restart; session fallback credentials expire and no password reaches application files or logs |
| Cancel then confirm deletion | Cancelling keeps LDAP usable; confirmation closes the session, removes the profile and forgets its secret |
| Connect anonymously | Real directory data can be read without storing a credential |
| StartTLS and LDAPS | Both reject the self-signed certificate by default; explicit verification opt-out permits a real bind/read and remains visible in the status bar |
| Browse a large directory | Naming contexts and branches load; explicit pages advance through 205 people and an entry from the last page opens |
| Browse a limited result | The server's size limit is reported as truncation |
| Inspect entry data | UTF-8, JPEG, multi-value attributes, operational metadata, raw LDIF and document switching show real data |
| Refresh an entry | A change made by an independent LDAP client becomes visible only after refresh |

The two TLS transports are separate cases. Pagination checks counters and
samples at page boundaries; Go integration tests cover exhaustive paging.
JPEG bytes in LDIF are checked against an independent `ldapsearch`, rather than
another read through the same application API.

IPC integration retains real add/modify/rename/move/delete, preview/discard,
single-use tokens, stale-write rejection, read-only/production guards, schema
rejection, bulk job outcomes, and certificate acceptance/revocation. Writes
are verified independently with LDAP tools inside the test container.

## Adding coverage

Add an E2E case when a regression would block a shipped connection or directory
workflow, lose data, expose credentials or defeat an essential protection.
Drive the workflow through visible controls and assert its user-visible result
or independently observed LDAP effect.

Keep formatting variants, exhaustive input matrices, static shell content,
preference panes and disabled placeholder controls in component/unit or
synthetic frontend tests. An isolated screenshot is a diagnostic, not a
success condition. Do not repeat the connection setup in a new E2E case solely
to check a label that an existing journey already covers.

Direct calls to unwired write/trust methods belong in the IPC suite. Once their
UI workflows are available, add user journeys that exercise preview,
confirmation, cancellation and the resulting directory state. Search execution,
schema editing and import/export are not implemented user journeys yet.

## Environment and isolation

The controller builds the production frontend and application once with Wails'
`dev` build tag. Every test receives a fresh browser context, native application
process, dynamic HTTP port, profile store, trust store and logs. The real session
credential fallback uses an unavailable D-Bus session, avoiding the desktop's
keyring. The restart test restarts that process with the same profile directory.

The runner overlays `server/docker-compose.e2e.yml` onto the development compose
file using a unique `ols-e2e-*` project. LDAP/LDAPS use dynamic loopback ports and
separate volumes. Startup verifies the full committed seed on three consecutive
LDAP probes. Workers share only this LDAP service; mutations use unique branches
with fixture cleanup, including when branch population fails.

The runner removes its containers and volumes after success or failure.
An uncatchable process kill can leave a test project; inspect `docker compose ls`
and remove that particular project with the two compose files if needed.

## Reports and CI

The `browser` CI job runs both live suites under Xvfb and uploads
`frontend/playwright-report/` and `frontend/test-results/` as the
`playwright-report` artifact. Reports include LDAP/build/native logs and failure
screenshots, videos and traces. Successful critical journeys do not take extra
screenshots. A custom pytest `--output` also controls the application build
directory.

```sh
python3 -m playwright show-trace frontend/test-results/<test>/trace.zip
```

Wails 2.15.0 has a development reload-overlay error in `/wails/ipc.js` caused by
its Svelte constructor API. The fixture records that exact upstream stack in
`wails-runtime-errors.json`; all other uncaught browser errors fail the test
and are retained in `browser-errors.json`.
