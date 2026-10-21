"""Fresh native Wails process per test; one real LDAP fixture per test run."""

from collections.abc import Generator
import json
from pathlib import Path
from uuid import uuid4

import pytest
from playwright.sync_api import Page

from .app import LiveApp
from .runtime import BASE_DN, LDAPService, NativeApp


@pytest.fixture(scope="session")
def ldap_service(live_environment: dict) -> LDAPService:
    service = LDAPService(live_environment["project"])
    service.ports = live_environment["ports"]
    return service


@pytest.fixture
def ldap_branch(ldap_service: LDAPService) -> Generator[str, None, None]:
    name = f"e2e-{uuid4().hex}"
    dn = f"ou={name},ou=archive,{BASE_DN}"
    ldap_service.modify(
        f"dn: {dn}\nchangetype: add\nobjectClass: organizationalUnit\nou: {name}\n\n"
        f"dn: cn=subject,{dn}\nchangetype: add\nobjectClass: inetOrgPerson\n"
        "cn: subject\nsn: Subject\ndescription: Original LDAP value\n"
    )
    try:
        yield dn
    finally:
        ldap_service.delete_tree(dn)


@pytest.fixture
def native_app(live_environment: dict, tmp_path: Path, output_path: str) -> Generator[NativeApp, None, None]:
    application = NativeApp(Path(live_environment["binary"]), tmp_path, Path(output_path))
    try:
        application.start()
        yield application
    finally:
        application.stop()


@pytest.fixture
def app(page: Page, native_app: NativeApp, ldap_service: LDAPService,
        output_path: str) -> Generator[LiveApp, None, None]:
    errors: list[str] = []
    runtime_errors: list[str] = []

    def page_error(error) -> None:
        # Wails 2.15.0's Svelte reload overlay uses the removed Svelte `new`
        # constructor API. Its failure is outside the app and does not affect
        # IPC; retain it as evidence while still failing on application errors.
        if (error.message == "Cannot read properties of null (reading 'nodes')"
                and "/wails/ipc.js:" in error.stack.splitlines()[1]):
            runtime_errors.append(error.stack)
        else:
            errors.append(error.stack)

    page.on("pageerror", page_error)
    fixture = LiveApp(page, native_app, ldap_service, Path(output_path))
    yield fixture
    if runtime_errors:
        Path(output_path).mkdir(parents=True, exist_ok=True)
        (Path(output_path) / "wails-runtime-errors.json").write_text(
            json.dumps(runtime_errors, indent=2) + "\n", encoding="utf-8",
        )
    assert errors == [], f"Uncaught browser errors: {errors}"
