"""Own the selected browser suite's environment once, including with xdist."""

import shutil
import subprocess
from pathlib import Path

import pytest

SERVER = pytest.StashKey[object]()


def pytest_addoption(parser: pytest.Parser) -> None:
    parser.addoption(
        "--suite", choices=("e2e", "ipc", "frontend"), default="e2e",
        help="Select critical user journeys, Wails IPC integration, or synthetic frontend checks",
    )
    parser.addoption(
        "--frontend-only", action="store_true", default=False,
        help="Alias for --suite frontend",
    )


def pytest_configure(config: pytest.Config) -> None:
    if config.getoption("--frontend-only"):
        if config.getoption("--suite") not in ("e2e", "frontend"):
            raise pytest.UsageError("--frontend-only cannot be combined with --suite ipc")
        config.option.suite = "frontend"
    # Respect explicit pytest paths and node IDs. A suite changes only the
    # default collection path, so -k, -m and --collect-only work normally.
    if config.args_source == pytest.Config.ArgsSource.TESTPATHS:
        config.args = {
            "e2e": ["e2e/live/critical"],
            "ipc": ["e2e/live/ipc"],
            "frontend": ["browser_tests"],
        }[config.getoption("--suite")]


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
    if config.getoption("--suite") == "frontend":
        from browser_tests.fixtures.server import PreviewServer
        server = PreviewServer()
    else:
        from e2e.live.runtime import LiveSuite
        server = LiveSuite(output_dir)
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
    if pytestconfig.getoption("--suite") == "frontend":
        pytest.fail("Live tests require --suite e2e or --suite ipc")
    if hasattr(pytestconfig, "workerinput"):
        return pytestconfig.workerinput["live_suite"]
    return pytestconfig.stash[SERVER].worker_data()


@pytest.fixture(scope="session")
def browser_context_args(browser_context_args: dict) -> dict:
    return {**browser_context_args, "viewport": {"width": 1440, "height": 1000}}


@pytest.fixture(scope="session", autouse=True)
def delete_output_dir() -> None:
    """The controller clears artifacts before building the shared application.

    Override Playwright's fixture so workers cannot delete that build or
    artifacts from other tests.
    """
