"""Critical connection journeys, driven through visible application controls."""

import json
import re
import socket

import pytest
from playwright.sync_api import Page, expect

from e2e.live.app import LiveApp
from e2e.live.runtime import ADMIN_DN, ADMIN_PASSWORD, BASE_DN


def test_connect_reload_navigate_disconnect_and_reconnect(app: LiveApp, page: Page):
    profile = app.connect()
    expect(page.get_by_role("status")).to_contain_text(f"bound: {ADMIN_DN}")
    expect(page.get_by_role("status")).to_contain_text("cleartext")

    # A page reload must restore the session and the saved profile.
    page.reload()
    app.select_profile()
    expect(page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible()
    assert app.rpc("WhoAmI", profile["id"])["identity"] == f"dn:{ADMIN_DN}"
    tree = app.open_browser()
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    tree.get_by_role("button", name="ou=people", exact=True).dblclick()
    tree.get_by_role("button", name="cn=jrivera", exact=True).click()
    expect(page.get_by_text("Jordan Rivera", exact=True)).to_be_visible()

    # One session crosses every workspace, retaining its open entry document.
    for screen in ("Connections", "Searches", "Schema", "LDIF & files", "Preferences", "DIT browser"):
        app.navigate(screen)
        expect(page.get_by_role("status")).to_contain_text(f"bound: {ADMIN_DN}")
    expect(page.get_by_text("Jordan Rivera", exact=True)).to_be_visible()
    app.disconnect(profile)
    app.connect_saved(profile)

    # A fresh directory read proves reconnect restored a usable LDAP session.
    tree = app.open_browser()
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    tree.get_by_role("button", name="ou=people", exact=True).dblclick()
    tree.get_by_role("button", name="cn=lchen", exact=True).click()
    expect(page.get_by_text("Li Chen", exact=True)).to_be_visible()


def test_check_network_recover_from_a_rejected_bind_and_cancel_without_saving(
    app: LiveApp, page: Page,
):
    app.network()
    page.get_by_role("button", name="Check network parameter", exact=True).click()
    expect(page.get_by_text("✓ reachable", exact=True)).to_be_visible()
    page.get_by_role("button", name="3 · Browser options", exact=True).click()
    expect(page.get_by_label("Base DN", exact=True)).to_have_value(BASE_DN)

    app.authentication(password="incorrect-password")
    page.get_by_role("button", name="Check authentication", exact=True).click()
    expect(page.get_by_text(re.compile("✗ result 49"))).to_contain_text("invalidCredentials")
    expect(page.get_by_role("status")).to_contain_text("not connected")
    page.get_by_label("Password", exact=True).fill(ADMIN_PASSWORD)
    page.get_by_role("button", name="Check authentication", exact=True).click()
    expect(page.get_by_text(f"✓ bound as {ADMIN_DN}", exact=True).first).to_be_visible()

    page.get_by_role("button", name="Cancel", exact=True).click()
    expect(page.get_by_text("no connections yet — use + to add one")).to_be_visible()
    assert app.rpc("ListProfiles") == []
    assert app.rpc("ConnectionStates") in (None, [])
    assert app.native.profiles() == []


def test_saved_bad_credentials_fail_without_anonymous_fallback(app: LiveApp, page: Page):
    profile = app.save(password="incorrect-password")
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_text("Last attempt", exact=True)).to_be_visible()
    expect(page.get_by_role("status")).to_contain_text("not connected")
    expect(page.get_by_role("button", name="Browse", exact=True)).to_have_count(0)
    state = app.rpc("ConnectionState", profile["id"])
    assert state["state"] == "disconnected"
    assert "49" in state["message"] or "invalidCredentials" in state["message"]


def test_unreachable_server_reports_the_network_failure_without_saving(app: LiveApp, page: Page):
    # A bound but non-listening socket deterministically refuses TCP connects.
    with socket.socket() as reserved:
        reserved.bind(("127.0.0.1", 0))
        app.network(port=reserved.getsockname()[1])
        page.get_by_role("button", name="Check network parameter", exact=True).click()
        expect(page.get_by_role("alert").filter(has_text="Unreachable")).to_contain_text(
            "connection refused",
        )
    assert app.rpc("ListProfiles") == []
    assert app.native.profiles() == []


def test_profile_options_survive_a_native_restart_without_persisting_the_password(
    app: LiveApp, page: Page,
):
    profile = app.connect(page_size=25, size_limit=300, read_only=True, production=True)
    assert profile["limits"]["pageSize"] == 25
    assert profile["limits"]["sizeLimit"] == 300
    assert profile["readOnly"] is True
    assert "production" in profile["tags"]
    assert profile["baseDn"] == BASE_DN
    assert app.rpc("ProfileHasSecret", profile["id"]) is True
    expect(page.get_by_role("status")).to_contain_text("read-only")
    expect(page.get_by_role("status")).to_contain_text("production")
    assert len(app.native.profiles()) == 1
    assert ADMIN_PASSWORD not in json.dumps(app.native.profiles())

    app.restart()
    app.select_profile()
    assert app.rpc("GetProfile", profile["id"]) == profile
    expect(page.get_by_role("button", name="Connect", exact=True)).to_be_visible()
    expect(page.get_by_role("status")).to_contain_text("not connected")
    # This environment uses the real in-memory credential fallback. Its secret
    # must disappear on process exit, while the connection profile survives.
    assert app.rpc("ProfileHasSecret", profile["id"]) is False
    for path in app.native.data_dir.rglob("*"):
        if path.is_file():
            assert ADMIN_PASSWORD.encode() not in path.read_bytes(), str(path)
    assert ADMIN_PASSWORD not in app.native.log_path.read_text(encoding="utf-8")


def test_delete_connected_profile_can_be_cancelled_then_forgets_the_secret(app: LiveApp, page: Page):
    profile = app.connect()
    page.once("dialog", lambda dialog: dialog.dismiss())
    page.get_by_role("button", name="Delete", exact=True).click()
    expect(page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible()
    assert app.rpc("WhoAmI", profile["id"])["identity"] == f"dn:{ADMIN_DN}"
    assert app.rpc("ProfileHasSecret", profile["id"]) is True

    page.once("dialog", lambda dialog: dialog.accept())
    page.get_by_role("button", name="Delete", exact=True).click()
    expect(page.get_by_text("no connections yet — use + to add one")).to_be_visible()
    assert app.rpc("ListProfiles") == []
    assert app.rpc("ProfileHasSecret", profile["id"]) is False
    assert app.rpc("ConnectionState", profile["id"])["state"] == "disconnected"
    assert app.native.profiles() == []


def test_anonymous_connection_reads_the_directory_without_storing_a_secret(app: LiveApp, page: Page):
    app.read_person(anonymous=True)
    expect(page.get_by_role("status")).to_contain_text("bound: anonymous")
    expect(page.get_by_text("Jordan Rivera", exact=True)).to_be_visible()
    profile = app.rpc("ListProfiles")[0]
    assert app.rpc("ProfileHasSecret", profile["id"]) is False
    reply = app.rpc("WhoAmI", profile["id"])
    assert reply["identity"] == ""
    assert reply["result"]["code"] == 0


@pytest.mark.parametrize("encryption", ["startTLS", "ldaps"])
def test_tls_refuses_an_untrusted_certificate_until_verification_is_explicitly_disabled(
    app: LiveApp, page: Page, encryption: str,
):
    app.network(encryption=encryption)
    verify = page.get_by_role("checkbox", name="Verify chain", exact=True)
    expect(verify).to_be_checked()
    check = page.get_by_role("button", name="Check network parameter", exact=True)
    check.click()
    expect(page.get_by_role("alert").filter(has_text="Unreachable")).to_contain_text("certificate")
    assert app.rpc("ListProfiles") == []
    assert app.rpc("ListTrustDecisions") in (None, [])

    verify.uncheck()
    check.click()
    expect(page.get_by_text("✓ reachable · encrypted (unverified)", exact=True)).to_be_visible()
    app.authentication()
    profile = app.finish()
    app.connect_saved(profile)
    status = page.get_by_role("status")
    expect(status).to_contain_text("TLS unverified")
    expect(status).not_to_contain_text("cleartext")
    assert app.rpc("WhoAmI", profile["id"])["identity"] == f"dn:{ADMIN_DN}"
    tree = app.open_browser()
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    tree.get_by_role("button", name="ou=people", exact=True).dblclick()
    tree.get_by_role("button", name="cn=jrivera", exact=True).click()
    expect(page.get_by_text("Jordan Rivera", exact=True)).to_be_visible()
