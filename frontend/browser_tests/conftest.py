"""Synthetic browser checks have their own bridge and failure reporting."""

from base64 import b64encode
from collections.abc import Generator

import pytest
import pytest_html
from playwright.sync_api import Page

from .fixtures.app import AppFixture


@pytest.fixture
def app(page: Page, output_path: str, pytestconfig: pytest.Config) -> Generator[AppFixture, None, None]:
    if pytestconfig.getoption("--suite") != "frontend":
        pytest.fail("Synthetic browser checks require --suite frontend")
    errors: list[str] = []
    page.on("pageerror", lambda error: errors.append(error.message))
    fixture = AppFixture(page, output_path)
    yield fixture
    if errors:
        fixture.capture_calls()
    assert errors == [], f"Uncaught browser errors: {errors}"


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
