# Frontend-only browser checks

These are the original synthetic frontend scenarios, retained for checking UI
states that the backend cannot currently produce. The default E2E commands use
the real application and LDAP service; see [Live E2E tests](browser-testing.md).

Python's Playwright API and pytest test the application's production frontend
in Chrome using its bundled Chromium browser. There are two ways to run the
same 66 scenarios: a visible browser and a headless browser for local checks.

## Setup

Install Python 3.10 or later and Node.js 22 or later. From the repository root:

```bash
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install -r frontend/requirements-e2e.txt
cd frontend
npm ci
python3 -m playwright install chromium
```

On Windows, use `.venv\Scripts\Activate.ps1` in PowerShell to activate the virtual
environment. Keep it active while running the browser tests. Python dependencies
are pinned in `frontend/requirements-e2e.txt`.

On a supported Linux distribution, `python3 -m playwright install --with-deps chromium`
also installs the browser's required system libraries.

## Run

From `frontend/`:

| Command | Behavior |
| --- | --- |
| `npm run test:browser` | Opens a visible Chrome window, runs one scenario at a time and slows actions by 350 ms so you can follow the navigation |
| `npm run test:browser:headless` | Runs the same scenarios headlessly in parallel |

The equivalent repository-root commands are `make test-browser` and
`make test-browser-headless`.

Both commands use `frontend/e2e/run.py` to invoke pytest. The headless run uses
four workers locally and two in CI; the visible run uses one browser at a time.
`frontend/pytest.ini` configures Chromium, the base URL and failure artifacts;
`frontend/conftest.py` sets the 1440 × 1000 viewport. Individual scenarios can be
selected with pytest arguments:

```bash
npm run test:browser:headless -- -k anonymous
python3 -m pytest --suite frontend browser_tests/test_connections.py
python3 -m pytest --suite frontend --collect-only
```

Each scenario opens a fresh page and closes it when it finishes. The visible
run saves videos and traces for successful scenarios as well. The six screen
smoke tests attach screenshots in both modes.

The shared pytest setup builds the frontend and starts its preview server on
**http://127.0.0.1:4173** automatically. This is separate from Vite development
on port 5173 and Wails on port 34115. An existing server on 4173
causes a failure; stop it before running tests so every run builds and tests the
current source. The setup stops the
server it started after the run, including when scenarios fail.

## Implemented scenarios

Each test has a fresh browser context and test directory state. Scenarios use
visible controls and accessible names, wait for outcomes, and check uncaught
browser exceptions. There are no fixed sleeps or screenshot baselines.

| Screen or shared area | Scenarios | Test file |
| --- | --- | --- |
| Six main perspectives | Open Connections, DIT browser, Searches, Schema, LDIF & files and Preferences; verify their content, sidebar, menu, document strip, logs and status; attach screenshots; check navigation/status at the desktop minimum of 1024 × 640 | `frontend/browser_tests/test_shell.py` |
| Connections | Empty workspace; connect → browse → disconnect; failed connect and retry; profile refresh errors; deletion accepted/cancelled | `frontend/browser_tests/test_connections.py` |
| Connection wizard: network | Finish/check enablement; encryption and default port; custom numeric port; network check; check invalidation after edits; unreachable-server diagnostic | `frontend/browser_tests/test_connections.py` |
| Connection wizard: authentication | Simple/anonymous selection; masked password; transport-dependent cleartext warnings; rejected bind code, diagnostic and interpretation | `frontend/browser_tests/test_connections.py` |
| Connection wizard: browser/edit options | Complete all four pages; base DN discovery; limits, aliases, referrals, read-only and production options; password sent separately; anonymous save; failed save/retry; cancellation | `frontend/browser_tests/test_connections.py` |
| Directory tree | Lazy children; explicit next page and cookie; refetch after collapse; local case-insensitive filtering; server truncation; failed branch; naming-context fallback and errors | `frontend/browser_tests/test_browser.py` |
| Entry views and info | UTF-8 text; language options; binary description; multi-value folding; operational attributes/timestamps; LDIF fidelity; all entry tabs; document selection by keyboard and close; refresh; read error/retry | `frontend/browser_tests/test_browser.py` |
| Searches | Raw filter and request fields; all scopes; aliases/referrals; empty results and disabled result actions | `frontend/browser_tests/test_workspaces.py` |
| Schema | Empty-state guidance; element categories; schema search input | `frontend/browser_tests/test_workspaces.py` |
| LDIF & files | Record counting; draft retained across Source/Diff/Report tabs; default dry run; continue-on-error checkbox; disconnected execution disabled; named target | `frontend/browser_tests/test_workspaces.py` |
| Nine preference panes | Appearance, Browser & tree, Entry editor, Value editors, LDIF & text editors, Connections & timeouts, Credentials & security, Keyboard shortcuts, Updates & about | `frontend/browser_tests/test_preferences.py` |
| Preference interactions | Settings search/no results; live themes/density and navigation; system appearance; explicit tree fetch default; line endings; command filtering/bindings; update checks off | `frontend/browser_tests/test_preferences.py` |
| Shell safety and menus | Connection-based menu enablement; read-only delete disabled; production, cleartext, anonymous and TLS badges; navigation without a backend; panel selection/collapse | `frontend/browser_tests/test_shell.py` |
| Progress, errors and console | Job progress/completion; cancellation; failure diagnostics; capability notice deduplication; credential-store fallback notice; connection loss/reconnect | `frontend/browser_tests/test_events.py` |

## Backend boundary

The production application calls Go through generated Wails bindings. A regular
browser has no native bridge. Tests inject `window.go.bridge.Bridge` and the
runtime event interface **before** the app loads, using
the init script in `frontend/browser_tests/fixtures/bridge.py`. This small script executes
in the browser's JavaScript runtime, with all data and scenario control supplied
by Python. The real React components, state, layouts,
generated binding calls and event subscriptions run normally. No production
mock mode is added to the app.

The fixture supplies synthetic profiles, paged directory entries and command
metadata. Tests can queue a failed call or response and emit the same events
the desktop backend emits. Calls are recorded so tests can check paging,
read options and the separation of profiles from passwords. Only methods needed
by these screens are implemented; missing methods fail through the normal
bridge availability check.

These tests verify browser behavior with deterministic backend data. They do
not verify a real LDAP connection, server protocol semantics, the native window
or an operating-system credential store. The existing Go integration tests
(`make test-integration`, requiring Docker) cover the live LDAP boundary.

## Scenarios waiting for feature wiring

The source tree contains more dialogs and editor components than users can
currently open from the six perspectives. The following scenarios should become
browser tests when those workflows are connected. Placeholder text or inert
buttons are not treated as successful completion of these operations.

| Workflow | Acceptance scenario to add |
| --- | --- |
| Search lifecycle and filter builder | Build/validate a filter, run it, page results, open a result, save/rerun a search, export selected results; retain raw filter text and report server limits |
| Entry writes and value editors | Create an entry, add an attribute with options, edit typed/binary values, preview the diff, cancel or commit, verify reread and modification log; guard pending edits |
| Rename, copy, move and delete | Set RDN/conflict policies, preview changes, confirm execution, refresh affected branches, and recover from cancellation or partial completion |
| Trust, credentials and referrals | Open a certificate challenge, inspect the chain, reject/trust once/trust permanently; prompt for credentials and select referral connections |
| Server schema and offline projects | Load/select schema, inspect hierarchy/usage/source, create/import a project, lint/edit it, block invalid exports, preview/apply a valid change |
| ACI and configuration | Select supported server models, edit structured/source representations, preview and commit; explain unavailable capabilities |
| Import/export and bulk operations | Choose direction/format/options, validate and dry-run, execute a cancellable job, inspect outcome counts and export rejected LDIF |
| History | Filter/search records, inspect redacted details, preview a reversible record's revert, and reject irreversible operations |
| Preference persistence and command actions | Apply/export/reset preferences across application restarts; rebind shortcuts and detect conflicts; execute menu/keyboard actions once handlers are registered |

## Reports and CI

The `browser` CI job runs the critical E2E and Wails IPC suites documented in
[Live E2E tests](browser-testing.md). These frontend-only checks are available
locally with the commands above.
Pytest validates markers strictly in both modes.

Open `frontend/playwright-report/index.html` directly in a browser to inspect
the pytest HTML report. Failure traces, screenshots
and videos are in `frontend/test-results/`; failing tests also attach the bridge
call log as JSON. Both directories are ignored by Git. The preview server's build
and startup log is in `frontend/playwright-report/preview-server.log`. Open an individual
trace with `python3 -m playwright show-trace <path-to-trace.zip>`.
