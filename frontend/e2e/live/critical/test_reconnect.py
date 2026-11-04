"""Recover real LDAP sessions through visible connection and browser controls."""

import re
import time

import pytest
from playwright.sync_api import Page, expect

from e2e.live.app import LiveApp
from e2e.live.runtime import ADMIN_DN, BASE_DN, LDAPService


def open_subject(app: LiveApp, page: Page, branch: str, **options) -> dict:
    profile = app.connect(**options)
    tree = app.open_browser()
    for label in (BASE_DN, "ou=archive", branch.split(",", 1)[0]):
        tree.get_by_role("button", name=label, exact=True).dblclick()
    tree.get_by_role("button", name="cn=subject", exact=True).click()
    expect(page.get_by_text("Original LDAP value", exact=True)).to_be_visible()
    return profile


def refresh_entry(page: Page) -> None:
    page.get_by_role("button", name="Refresh", exact=True).filter(
        has_text=re.compile("^Refresh$"),
    ).click()


@pytest.mark.parametrize("encryption", ["none", "startTLS", "ldaps"])
def test_refresh_recovers_an_idle_session_without_a_manual_reconnect(
    app: LiveApp, page: Page, ldap_service: LDAPService, ldap_branch: str,
    ldap_proxy, encryption: str,
):
    proxy = ldap_proxy(encryption)
    profile = open_subject(
        app, page, ldap_branch, encryption=encryption, verify=False, port=proxy.port,
    )
    assert proxy.connections == 1
    status = page.get_by_role("status")
    expect(status).not_to_contain_text("reconnected")
    dn = f"cn=subject,{ldap_branch}"
    changed = f"Read after {encryption} session recovery"
    ldap_service.modify(f"dn: {dn}\nchangetype: modify\nreplace: description\ndescription: {changed}\n")
    expect(page.get_by_text(changed, exact=True)).to_have_count(0)

    proxy.drop()
    refresh_entry(page)

    expect(page.get_by_text(changed, exact=True)).to_be_visible()
    expect(status).to_contain_text("reconnected")
    expect(status).to_contain_text(f"bound: {ADMIN_DN}")
    state = app.rpc("ConnectionState", profile["id"])
    assert state["state"] == "connected"
    assert state["writesRequireConfirmation"] is True
    assert state["boundDn"] == ADMIN_DN
    assert state["encrypted"] is (encryption != "none")
    assert proxy.connections == 2
    if encryption == "none":
        expect(status).to_contain_text("cleartext")
    else:
        expect(status).to_contain_text("TLS unverified")
        expect(status).not_to_contain_text("cleartext")


@pytest.mark.timeout(75)
def test_keep_alive_recovers_the_idle_session_before_the_next_user_read(
    app: LiveApp, page: Page, ldap_service: LDAPService, ldap_branch: str, ldap_proxy,
):
    proxy = ldap_proxy()
    profile = open_subject(app, page, ldap_branch, port=proxy.port)
    dn = f"cn=subject,{ldap_branch}"
    changed = "Read after keep-alive recovery"
    ldap_service.modify(f"dn: {dn}\nchangetype: modify\nreplace: description\ndescription: {changed}\n")
    proxy.drop()

    # No read or manual Connect is dispatched while waiting for the actual
    # thirty-second LDAP idle probe and its real conn:state event.
    status = page.get_by_role("status")
    expect(status).to_contain_text("reconnected", timeout=40000)
    expect(status).to_contain_text(f"bound: {ADMIN_DN}")
    assert proxy.connections == 2
    assert app.rpc("ConnectionState", profile["id"])["writesRequireConfirmation"] is True
    expect(page.get_by_text("Original LDAP value", exact=True)).to_be_visible()
    refresh_entry(page)
    expect(page.get_by_text(changed, exact=True)).to_be_visible()
    assert proxy.connections == 2


def test_rejected_reauthentication_disconnects_without_retry_or_anonymous_fallback(
    app: LiveApp, page: Page, ldap_service: LDAPService, ldap_branch: str, ldap_proxy,
):
    proxy = ldap_proxy()
    dn = f"cn=subject,{ldap_branch}"
    ldap_service.modify(
        f"dn: {dn}\nchangetype: modify\nreplace: userPassword\nuserPassword: first-session-password\n",
    )
    profile = open_subject(
        app, page, ldap_branch, dn=dn, password="first-session-password", port=proxy.port,
    )
    ldap_service.modify(
        f"dn: {dn}\nchangetype: modify\nreplace: userPassword\nuserPassword: rotated-password\n",
    )
    proxy.drop()
    refresh_entry(page)

    error = page.get_by_role("alert").filter(has_text="Could not read the entry")
    expect(error).to_contain_text("invalidCredentials")
    expect(error).to_contain_text("49")
    expect(page.get_by_role("status")).to_contain_text("not connected")
    expect(page.get_by_role("status")).not_to_contain_text("bound: anonymous")
    assert proxy.connections == 2
    assert app.rpc("ConnectionState", profile["id"])["state"] == "disconnected"

    # The rejected session must remain closed when the user requests another
    # read, rather than looping the now-invalid credential or reading anonymously.
    refresh_entry(page)
    expect(error).to_contain_text("that connection is not open")
    assert proxy.connections == 2


@pytest.mark.parametrize("encryption", ["startTLS", "ldaps"])
def test_cancel_a_stalled_tls_connection_then_connect_successfully(
    app: LiveApp, page: Page, ldap_proxy, encryption: str,
):
    proxy = ldap_proxy(encryption)
    proxy.pause()
    profile = app.save(encryption=encryption, verify=False, port=proxy.port)
    page.get_by_role("button", name="Connect", exact=True).click()
    proxy.wait_for_connections(1)
    panel = page.get_by_role("region", name="Progress and logs")
    # Running jobs append a count to the accessible name ("Progress 1").
    panel.get_by_role("tab", name=re.compile(r"^Progress(?:\s*\d+)?$")).click()
    job = panel.get_by_role("row").filter(has_text="connect")
    expect(job).to_contain_text("running")
    started = time.monotonic()
    job.get_by_role("button", name="Cancel", exact=True).click()
    expect(job).to_contain_text("cancelled", timeout=2000)
    assert time.monotonic() - started < 2
    expect(page.get_by_role("status")).to_contain_text("not connected")

    proxy.resume()
    proxy.wait_for_idle()
    assert app.rpc("ConnectionState", profile["id"])["state"] == "disconnected"
    app.connect_saved(profile)
    assert app.rpc("WhoAmI", profile["id"])["identity"] == f"dn:{ADMIN_DN}"
    expect(page.get_by_role("status")).to_contain_text("TLS unverified")
    assert proxy.connections == 2


@pytest.mark.parametrize("encryption", ["startTLS", "ldaps"])
def test_stalled_tls_negotiation_obeys_the_connection_timeout(
    app: LiveApp, page: Page, ldap_proxy, encryption: str,
):
    proxy = ldap_proxy(encryption)
    proxy.pause()
    profile = app.save(encryption=encryption, verify=False, port=proxy.port)
    page.get_by_role("button", name="Connect", exact=True).click()
    proxy.wait_for_connections(1)
    failure = page.get_by_text("Last attempt", exact=True).locator("..")
    expect(failure).to_contain_text(
        re.compile(r"deadline exceeded|i/o timeout"),
        timeout=profile["timeouts"]["connectMs"] + 5000,
    )
    expect(page.get_by_role("status")).to_contain_text("not connected")
    expect(page.get_by_role("button", name="Browse", exact=True)).to_have_count(0)
    state = app.rpc("ConnectionState", profile["id"])
    assert state["state"] == "disconnected"
    assert state["encrypted"] is False
    assert proxy.connections == 1

    proxy.resume()
    proxy.wait_for_idle()
    app.connect_saved(profile)
    assert app.rpc("WhoAmI", profile["id"])["identity"] == f"dn:{ADMIN_DN}"
    assert app.rpc("ConnectionState", profile["id"])["encrypted"] is True
