import re

import pytest
from playwright.sync_api import Page, expect

from .fixtures.app import AppFixture
from .fixtures.data import CONNECTED, PROFILE

SCREENS = [
    {
        "name": "Connections",
        "sidebar": "Connections",
        "content": "A connection records where a directory is and how to bind to it.",
    },
    {
        "name": "DIT browser",
        "sidebar": "Directory tree",
        "content": "No entry is loaded. Select one in the tree, or open a search result.",
    },
    {"name": "Searches", "sidebar": "Searches", "content": "no results yet"},
    {
        "name": "Schema",
        "sidebar": "Schema",
        "content": "Open a connection and pick an element on the left.",
    },
    {"name": "LDIF & files", "sidebar": "Workspace files", "content": "no change records"},
    {"name": "Preferences", "sidebar": "Preferences", "content": "Preferences › Appearance"},
]


@pytest.mark.parametrize("screen", SCREENS, ids=[screen["name"] for screen in SCREENS])
def test_screen_opens_with_navigation_documents_logs_and_status(
    app: AppFixture, page: Page, screen: dict,
):
    """{screen[name]}: opens with navigation, documents, logs and status"""
    app.open()
    app.navigate(screen["name"])
    expect(page.get_by_role("complementary", name=screen["sidebar"], exact=True)).to_be_visible()
    expect(page.get_by_text(screen["content"], exact=False)).to_be_visible()
    expect(page.get_by_role("menubar", name="Application menu")).to_be_visible()
    expect(page.get_by_text("No documents open", exact=True)).to_be_visible()
    expect(page.get_by_role("region", name="Progress and logs")).to_be_visible()
    expect(page.get_by_role("region", name="Progress and logs")).to_be_in_viewport(ratio=1)
    expect(page.get_by_role("status")).to_contain_text("not connected")
    app.screenshot(screen["name"])


def test_all_screens_keep_navigation_and_status_visible_at_the_desktop_minimum_of_1024_640(
    app: AppFixture, page: Page,
):
    """all screens keep navigation and status visible at the desktop minimum of 1024 × 640"""
    page.set_viewport_size({"width": 1024, "height": 640})
    app.open()
    for screen in SCREENS:
        app.navigate(screen["name"])
        expect(page.get_by_role("navigation", name="Perspectives")).to_be_in_viewport(ratio=1)
        expect(page.get_by_role("status")).to_be_in_viewport(ratio=1)


def test_logs_can_be_selected_collapsed_and_reopened_without_losing_the_selection(
    app: AppFixture, page: Page,
):
    """logs can be selected, collapsed and reopened without losing the selection"""
    app.open()
    panel = page.get_by_role("region", name="Progress and logs")
    for name, message in [
        ("Progress", "No operations have run in this session."),
        ("Modification Logs", "Committed changes are recorded here as LDIF"),
        ("Search Logs", "Every search is logged with its filter"),
        ("Errors", "No errors in this session."),
        ("Console", "Nothing logged yet."),
    ]:
        panel.get_by_role("tab", name=name, exact=True).click()
        expect(panel.get_by_role("tab", name=name, exact=True)).to_have_attribute(
            "aria-selected", "true"
        )
        expect(panel.get_by_text(message, exact=False)).to_be_visible()
    panel.get_by_role("button", name="Collapse panel").click()
    expect(panel.get_by_text("Nothing logged yet.")).to_be_hidden()
    app.navigate("Schema")
    panel.get_by_role("button", name="Expand panel").click()
    expect(panel.get_by_role("tab", name="Console", exact=True)).to_have_attribute(
        "aria-selected", "true"
    )
    expect(panel.get_by_text("Nothing logged yet.")).to_be_visible()


def test_ldap_menu_enablement_follows_the_connection_state(app: AppFixture, page: Page):
    """LDAP menu enablement follows the connection state"""
    app.open()
    ldap = page.get_by_role("menubar").get_by_role("button", name="LDAP", exact=True)
    ldap.click()
    expect(page.get_by_role("menuitem", name=re.compile("Refresh entry"))).to_be_disabled()
    expect(page.get_by_role("menuitem", name=re.compile("Delete entry"))).to_be_disabled()
    page.keyboard.press("Escape")
    app.connect()
    ldap.click()
    expect(page.get_by_role("menuitem", name=re.compile("Refresh entry"))).to_be_enabled()
    expect(page.get_by_role("menuitem", name=re.compile("Delete entry"))).to_be_enabled()
    page.keyboard.press("Escape")
    expect(page.get_by_role("menu")).to_be_hidden()


def test_read_only_production_and_unverified_tls_remain_visible_across_screens(
    app: AppFixture, page: Page,
):
    """read-only, production and unverified TLS remain visible across screens"""
    app.open(
        {"connections": [{**CONNECTED, "readOnly": True, "production": True, "tlsVerified": False}]}
    )
    status = page.get_by_role("status")
    for screen in SCREENS:
        app.navigate(screen["name"])
        expect(status).to_contain_text(PROFILE["host"])
        expect(status).to_contain_text(f"bound: {PROFILE['bindDn']}")
        expect(status).to_contain_text("read-only")
        expect(status).to_contain_text("production")
        expect(status).to_contain_text("TLS unverified")
    page.get_by_role("menubar").get_by_role("button", name="LDAP", exact=True).click()
    expect(page.get_by_role("menuitem", name=re.compile("Delete entry"))).to_be_disabled()
    expect(page.get_by_role("menuitem", name=re.compile("Refresh entry"))).to_be_enabled()


def test_an_unencrypted_anonymous_connection_is_labelled_in_the_status_bar(app: AppFixture, page: Page):
    """an unencrypted anonymous connection is labelled in the status bar"""
    app.open({"connections": [{**CONNECTED, "encrypted": False, "boundDn": ""}]})
    expect(page.get_by_role("status")).to_contain_text("bound: anonymous")
    expect(page.get_by_role("status")).to_contain_text("cleartext")
    expect(page.get_by_role("status")).not_to_contain_text("TLS unverified")


def test_plain_browser_mode_explains_that_the_desktop_backend_is_unavailable(page: Page, app: AppFixture):
    """plain browser mode explains that the desktop backend is unavailable"""
    page.goto("/")
    expect(page.get_by_text("no backend", exact=True)).to_be_visible()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    page.get_by_label("Hostname", exact=True).fill("ldap.example.test")
    page.get_by_role("button", name="2 · Authentication").click()
    page.get_by_role("button", name="Anonymous", exact=True).click()
    expect(page.get_by_role("alert")).to_contain_text(
        "This interface is running outside the desktop shell"
    )
    expect(page.get_by_role("button", name="Finish", exact=True)).to_be_disabled()
    app.navigate("DIT browser")
    expect(page.get_by_text("no connection selected")).to_be_visible()
