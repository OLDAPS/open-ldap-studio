# Live E2E tests

The default E2E suite runs Chromium → the real Wails websocket dispatcher → the
application's Go bridge → real OpenLDAP. It uses the existing
`server/docker-compose.yml` and committed LDIF seed. There are no injected
bindings, simulated events, intercepted requests or LDAP mocks in this suite.

## Setup

Run on Linux with Docker, Docker Compose **2.24.4+**, the Go version declared in
`go.mod`, Node.js 22+, and Python 3.10+. Install the native Wails dependencies
(GTK 3 and WebKitGTK 4.1) and a display server. On Ubuntu:

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
make test-e2e-headless                # headless Chromium, four workers locally
xvfb-run -a make test-e2e-headless    # no desktop display required
```

The equivalent commands in `frontend/` are `npm run test:e2e` and
`npm run test:e2e:headless`. Both run `e2e/live/`. CI uses two workers.
Individual tests and worker counts can be selected with normal pytest arguments:

```sh
cd frontend
npm run test:e2e:headless -- -k tls
npm run test:e2e:headless -- -n 0 e2e/live/test_changesets.py
python3 -m pytest --collect-only       # no Docker, build or display needed
```

The runner builds the production frontend and the real application once with
Wails' `dev` build tag, then serves those assets through Wails. Every test gets
its own native application process, dynamic HTTP port, connection store, trust
store and logs. The application uses its real session credential fallback with
an unavailable D-Bus session, avoiding writes into the desktop's keyring.

## Docker lifecycle and isolation

The runner applies `server/docker-compose.e2e.yml` over the existing compose
file. A unique `ols-e2e-*` project removes fixed container names, publishes
LDAP/LDAPS on dynamically assigned loopback ports, and owns separate volumes.
It waits for the health check and verifies every seeded entry with real LDAP
searches on three consecutive probes. Parallel workers share only this LDAP
service; write tests use uniquely named branches and remove them afterward.

Containers and test volumes are removed after the run, including a failed
build or test. The development fixture and its data are independent. An
uncatchable process kill can leave a test project; inspect `docker compose ls`
and remove that particular project with the two compose files if needed.

## Coverage

| Area | Live checks |
| --- | --- |
| Connections | Create/save, real admin/person/service binds, invalid passwords/DNs and LDAP result 49, TCP refusal, cancellation without persistence, page reload, connect/disconnect/reconnect, delete and forget secrets, anonymous identity |
| Encryption | Real StartTLS and LDAPS handshakes, self-signed certificate refusal, session/permanent certificate acceptance, verified bind, revocation, explicit verification opt-out with status badges and directory reads |
| Directory | Root DSE/naming contexts, seeded branches, explicit 100/100/5 paging through 205 people, custom page sizes, real server size-limit truncation, filtering and entry documents |
| Entry values | JPEG bytes and rendering, UTF-8, multiple mail values, object classes, operational attributes and server timestamps, LDIF, refresh after an independent LDAP modification |
| Writes | Preview/discard without mutations, single-use commit tokens, real add/modify/rename/move/delete, concurrency conflicts, read-only/production guards, server schema rejection, bulk job events and per-entry outcomes |
| Workspace | All six perspectives keep the actual LDAP session and identity while navigating; disconnect remains effective afterward |

Connections and directory tests operate visible UI controls. The editor's write
dialogs are not connected to the UI yet, so `test_changesets.py` connects through
the UI and invokes the actual bound `Preview`/`Commit` methods over Wails IPC.
It verifies effects independently with `ldapsearch` in the Docker container.
Certificate acceptance and revocation also use the actual bound trust methods
because the certificate dialog is not wired into the shell yet.
Search execution, schema editing and import/export remain unimplemented and
are not claimed as successful E2E workflows.

## Reports and CI

The `browser` CI job installs native dependencies and Chromium, then runs the
suite under Xvfb against Docker. It uploads `frontend/playwright-report/` and
`frontend/test-results/` as the `playwright-report` artifact, including the HTML
report, LDAP logs, application build log, native application logs and failure
screenshots/videos/traces. Local runs have no retries; CI retries twice.

```sh
python3 -m playwright show-trace frontend/test-results/<test>/trace.zip
```

Wails 2.15.0 has a development reload-overlay error in `/wails/ipc.js` caused by
its Svelte constructor API. The suite records that exact upstream stack in
`wails-runtime-errors.json`; all other uncaught browser errors fail the test.
The real IPC connection, application and LDAP responses are still exercised.

The older synthetic UI scenarios remain available as `make test-browser` and
`make test-browser-headless`; see [frontend-only checks](frontend-browser-testing.md).
