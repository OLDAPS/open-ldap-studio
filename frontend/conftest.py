"""Shared setup, browser error checks and reporting for the Python E2E suite."""

import shutil
import subprocess
from base64 import b64encode
from collections.abc import Generator
from pathlib import Path

import pytest
import pytest_html
from playwright.sync_api import Page

from e2e.fixtures.app import AppFixture
from e2e.fixtures.server import PreviewServer

SERVER = pytest.StashKey[object]()


def pytest_addoption(parser: pytest.Parser) -> None:
    parser.addoption(
        "--frontend-only", action="store_true", default=False,
        help="Run the older synthetic frontend scenarios instead of live LDAP E2E tests",
    )


@pytest.hookimpl(optionalhook=True)
def pytest_configure_node(node) -> None:
    server = node.config.stash.get(SERVER, None)
    if server is not None and hasattr(server, "worker_data"):
        node.workerinput["live_suite"] = server.worker_data()


def pytest_sessionstart(session: pytest.Session) -> None:
    # Only the controller owns the Docker service and application build.
    config = session.config
    if config.option.collectonly or hasattr(config, "workerinput"):
        return
    output_dir = Path(config.getoption("--output"))
    if output_dir.exists():
        shutil.rmtree(output_dir)
    if config.getoption("--frontend-only"):
        server = PreviewServer()
    else:
        from e2e.live.runtime import LiveSuite
        server = LiveSuite()
    config.stash[SERVER] = server
    try:
        server.start()
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        try:
            server.stop()
        except (OSError, RuntimeError, subprocess.SubprocessError) as cleanup_error:
            error = RuntimeError(f"{error}\nCleanup also failed: {cleanup_error}")
        log = server.log_path.read_text(encoding="utf-8") if server.log_path.exists() else ""
        pytest.exit(f"Could not start the test environment: {error}\n{log}", returncode=1)


def pytest_sessionfinish(session: pytest.Session) -> None:
    server = session.config.stash.get(SERVER, None)
    if server is not None:
        server.stop()


@pytest.fixture(scope="session")
def live_environment(pytestconfig: pytest.Config) -> dict:
    if pytestconfig.getoption("--frontend-only"):
        pytest.fail("Live tests require the real Wails/LDAP environment; omit --frontend-only")
    if hasattr(pytestconfig, "workerinput"):
        return pytestconfig.workerinput["live_suite"]
    return pytestconfig.stash[SERVER].worker_data()


@pytest.fixture(scope="session")
def browser_context_args(browser_context_args: dict) -> dict:
    return {**browser_context_args, "viewport": {"width": 1440, "height": 1000}}


@pytest.fixture(scope="session", autouse=True)
def delete_output_dir() -> None:
    """The controller clears artifacts once, before workers or retries start.

    Override Playwright's fixture: rerunning a fixture teardown failure can
    recreate session fixtures and otherwise delete earlier attempts' artifacts.
    """


@pytest.fixture
def app(page: Page, output_path: str) -> Generator[AppFixture, None, None]:
    errors: list[str] = []
    page.on("pageerror", lambda error: errors.append(error.message))
    fixture = AppFixture(page, output_path)
    yield fixture
    if errors:
        fixture.capture_calls()
    assert errors == [], "The application should not raise uncaught browser errors"


@pytest.hookimpl(hookwrapper=True)
def pytest_runtest_makereport(item: pytest.Item):
    outcome = yield
    report = outcome.get_result()
    fixture = item.funcargs.get("app")
    if fixture is None:
        return
    extras = getattr(report, "extras", [])
    if report.when == "call":
        for name, screenshot in fixture.screenshots:
            extras.append(pytest_html.extras.png(b64encode(screenshot).decode("ascii"), name=name))
        if report.failed:
            fixture.capture_calls()
    if report.failed and fixture.bridge_calls is not None:
        extras.append(pytest_html.extras.json(fixture.bridge_calls, name="bridge-calls"))
    report.extras = extras
