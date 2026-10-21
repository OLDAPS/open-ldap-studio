"""The real LDAP session remains available throughout the desktop workspace."""

import pytest
from playwright.sync_api import Page, expect

from .app import LiveApp
from .runtime import ADMIN_DN


@pytest.mark.parametrize("screen,sidebar", [
    ("Connections", "Connections"),
    ("DIT browser", "Directory tree"),
    ("Searches", "Searches"),
    ("Schema", "Schema"),
    ("LDIF & files", "Workspace files"),
    ("Preferences", "Preferences"),
])
def test_perspective_preserves_the_real_ldap_connection(
    app: LiveApp, page: Page, screen: str, sidebar: str,
):
    app.open()
    profile = app.connect()
    app.navigate(screen)
    expect(page.get_by_role("complementary", name=sidebar, exact=True)).to_be_visible()
    expect(page.get_by_role("status")).to_contain_text(f"bound: {ADMIN_DN}")
    expect(page.get_by_role("menubar", name="Application menu")).to_be_visible()
    expect(page.get_by_role("region", name="Progress and logs")).to_be_in_viewport(ratio=1)
    assert app.rpc("WhoAmI", profile["id"])["identity"] == f"dn:{ADMIN_DN}"
    app.screenshot(screen)
    app.navigate("Connections")
    page.get_by_role("button", name=app.profile_name, exact=True).click()
    page.get_by_role("button", name="Disconnect", exact=True).click()
    expect(page.get_by_role("status")).to_contain_text("not connected")
