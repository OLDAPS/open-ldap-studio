"""Connect through the UI to real slapd, including real protocol failures."""

import json
import re
import socket

import pytest
from playwright.sync_api import Page, expect

from .app import LiveApp
from .runtime import ADMIN_DN, ADMIN_PASSWORD, BASE_DN


def test_create_connect_browse_disconnect_and_reconnect(app: LiveApp, page: Page):
    app.open()
    profile = app.connect()
    expect(page.get_by_role("status")).to_contain_text(f"bound: {ADMIN_DN}")
    expect(page.get_by_role("status")).to_contain_text("cleartext")
    # WhoAmI is answered by slapd, independently of the client's state badge.
    reply = app.rpc("WhoAmI", profile["id"])
    assert reply["identity"] == f"dn:{ADMIN_DN}"
    assert reply["result"]["code"] == 0
    page.get_by_role("button", name="Browse", exact=True).click()
    tree = page.get_by_role("complementary", name="Directory tree")
    expect(tree.get_by_role("button", name=BASE_DN, exact=True)).to_be_visible()
    app.navigate("Connections")
    page.get_by_role("button", name=app.profile_name, exact=True).click()
    page.get_by_role("button", name="Disconnect", exact=True).click()
    expect(page.get_by_role("status")).to_contain_text("not connected")
    assert app.rpc("ConnectionState", profile["id"])["state"] == "disconnected"
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible()


def test_network_check_discovers_the_server_base_without_saving(app: LiveApp, page: Page):
    app.open()
    app.network()
    page.get_by_role("button", name="Check network parameter", exact=True).click()
    expect(page.get_by_text("✓ reachable", exact=True)).to_be_visible()
    page.get_by_role("button", name="3 · Browser options", exact=True).click()
    expect(page.get_by_label("Base DN", exact=True)).to_have_value(BASE_DN)
    assert app.rpc("ListProfiles") == []
    assert app.rpc("ConnectionStates") in (None, [])
    assert app.native.profiles() == []
    page.get_by_role("button", name="Cancel", exact=True).click()
    expect(page.get_by_text("no connections yet — use + to add one")).to_be_visible()


@pytest.mark.parametrize("dn,password", [
    (ADMIN_DN, ADMIN_PASSWORD),
    (f"cn=jrivera,ou=people,{BASE_DN}", "password"),
    (f"cn=readonly-svc,ou=services,{BASE_DN}", "readonly"),
])
def test_authentication_checks_real_admin_person_and_service_credentials(
    app: LiveApp, page: Page, dn: str, password: str,
):
    app.open()
    app.network()
    app.authentication(dn, password)
    expect(page.get_by_label("Password", exact=True)).to_have_attribute("type", "password")
    page.get_by_role("button", name="Check authentication", exact=True).click()
    expect(page.get_by_text(f"✓ bound as {dn}", exact=True).first).to_be_visible()
    assert app.rpc("ListProfiles") == []


@pytest.mark.parametrize("dn,password", [
    (ADMIN_DN, "incorrect-password"),
    (f"cn=does-not-exist,{BASE_DN}", ADMIN_PASSWORD),
])
def test_rejected_bind_reports_slapd_result_49_and_can_be_corrected(
    app: LiveApp, page: Page, dn: str, password: str,
):
    app.open()
    app.network()
    app.authentication(dn, password)
    page.get_by_role("button", name="Check authentication", exact=True).click()
    expect(page.get_by_text(re.compile("✗ result 49"))).to_contain_text("invalidCredentials")
    expect(page.get_by_role("status")).to_contain_text("not connected")
    assert app.rpc("ListProfiles") == []
    page.get_by_label("Bind DN", exact=True).fill(ADMIN_DN)
    page.get_by_label("Password", exact=True).fill(ADMIN_PASSWORD)
    page.get_by_role("button", name="Check authentication", exact=True).click()
    expect(page.get_by_text(f"✓ bound as {ADMIN_DN}", exact=True).first).to_be_visible()


def test_a_saved_bad_password_fails_the_real_connect_job_without_anonymous_fallback(
    app: LiveApp, page: Page,
):
    app.open()
    profile = app.save(password="incorrect-password")
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_text("Last attempt", exact=True)).to_be_visible()
    expect(page.get_by_role("status")).to_contain_text("not connected")
    expect(page.get_by_role("button", name="Browse", exact=True)).to_have_count(0)
    state = app.rpc("ConnectionState", profile["id"])
    assert state["state"] == "disconnected"
    assert "49" in state["message"] or "invalidCredentials" in state["message"]


def test_connection_refused_is_reported_without_saving(app: LiveApp, page: Page):
    app.open()
    # A bound but non-listening socket deterministically refuses TCP connects.
    with socket.socket() as reserved:
        reserved.bind(("127.0.0.1", 0))
        app.network(port=reserved.getsockname()[1])
        page.get_by_role("button", name="Check network parameter", exact=True).click()
        expect(page.get_by_role("alert").filter(has_text="Unreachable")).to_contain_text(
            "connection refused",
        )
    assert app.rpc("ListProfiles") == []


def test_saving_options_persists_a_profile_without_the_bind_password(app: LiveApp, page: Page):
    app.open()
    profile = app.connect(page_size=25, size_limit=300, read_only=True, production=True)
    assert profile["limits"]["pageSize"] == 25
    assert profile["limits"]["sizeLimit"] == 300
    assert profile["readOnly"] is True
    assert "production" in profile["tags"]
    assert profile["baseDn"] == BASE_DN
    assert app.rpc("ProfileHasSecret", profile["id"]) is True
    persisted = app.native.profiles()
    assert len(persisted) == 1
    assert persisted[0]["id"] == profile["id"]
    assert ADMIN_PASSWORD not in json.dumps(persisted)
    for path in (app.native.data_dir / "state").rglob("*.log"):
        assert ADMIN_PASSWORD not in path.read_text(encoding="utf-8")
    expect(page.get_by_role("status")).to_contain_text("read-only")
    expect(page.get_by_role("status")).to_contain_text("production")


def test_profile_survives_a_page_reload_and_connects_with_the_stored_secret(
    app: LiveApp, page: Page,
):
    app.open()
    profile = app.save()
    page.reload()
    page.get_by_role("button", name=app.profile_name, exact=True).click()
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible()
    assert app.rpc("ConnectionState", profile["id"])["boundDn"] == ADMIN_DN


def test_deleting_a_connected_profile_closes_ldap_and_forgets_its_secret(app: LiveApp, page: Page):
    app.open()
    profile = app.connect()
    page.once("dialog", lambda dialog: dialog.dismiss())
    page.get_by_role("button", name="Delete", exact=True).click()
    expect(page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible()
    assert app.rpc("ProfileHasSecret", profile["id"]) is True
    page.once("dialog", lambda dialog: dialog.accept())
    page.get_by_role("button", name="Delete", exact=True).click()
    expect(page.get_by_text("no connections yet — use + to add one")).to_be_visible()
    assert app.rpc("ListProfiles") == []
    assert app.rpc("ProfileHasSecret", profile["id"]) is False
    assert app.rpc("ConnectionState", profile["id"])["state"] == "disconnected"
    assert app.native.profiles() == []


def test_anonymous_connection_uses_real_server_permissions(app: LiveApp, page: Page):
    app.open()
    profile = app.connect(anonymous=True)
    expect(page.get_by_role("status")).to_contain_text("bound: anonymous")
    assert app.rpc("ProfileHasSecret", profile["id"]) is False
    reply = app.rpc("WhoAmI", profile["id"])
    assert reply["identity"] == ""
    assert reply["result"]["code"] == 0
    page.get_by_role("button", name="Browse", exact=True).click()
    expect(page.get_by_role("button", name=BASE_DN, exact=True)).to_be_visible()


@pytest.mark.parametrize("encryption", ["startTLS", "ldaps"])
def test_real_tls_handshake_refuses_the_self_signed_certificate(
    app: LiveApp, page: Page, encryption: str,
):
    app.open()
    app.network(encryption=encryption)
    page.get_by_role("button", name="Check network parameter", exact=True).click()
    expect(page.get_by_role("alert").filter(has_text="Unreachable")).to_contain_text(
        "certificate",
    )
    assert app.rpc("ListProfiles") == []
    assert app.rpc("ListTrustDecisions") in (None, [])


@pytest.mark.parametrize("encryption", ["startTLS", "ldaps"])
def test_explicitly_unverified_tls_can_bind_and_read_real_entries(
    app: LiveApp, page: Page, encryption: str,
):
    app.open()
    tree = app.browse(encryption=encryption, verify=False)
    status = page.get_by_role("status")
    expect(status).to_contain_text("TLS unverified")
    expect(status).not_to_contain_text("cleartext")
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    expect(tree.get_by_role("button", name="ou=people", exact=True)).to_be_visible()


@pytest.mark.parametrize("encryption", ["startTLS", "ldaps"])
@pytest.mark.parametrize("scope", ["session", "permanent"])
def test_accepting_the_real_certificate_enables_verified_tls_and_revocation_reblocks_it(
    app: LiveApp, page: Page, encryption: str, scope: str,
):
    app.open()
    # Observe the actual Wails event; nothing is injected into the event stream.
    page.evaluate("""() => {
        window.__ldapTrustChallenges = [];
        window.runtime.EventsOn('trust:challenge', challenge => {
            window.__ldapTrustChallenges.push(challenge);
        });
    }""")
    app.network(encryption=encryption)
    check = page.get_by_role("button", name="Check network parameter", exact=True)
    check.click()
    expect(page.get_by_role("alert").filter(has_text="Unreachable")).to_contain_text("certificate")
    challenge = page.evaluate("() => window.__ldapTrustChallenges[0]")
    assert challenge["host"] == "127.0.0.1"
    assert challenge["port"] == app.ldap.ports["ldaps" if encryption == "ldaps" else "ldap"]
    assert "BEGIN CERTIFICATE" in challenge["chainPem"][0]
    assert challenge["previouslyTrusted"] is False
    # The trust dialog is not wired into the shell yet; make its real decision
    # through the application's bound method, then retry with visible controls.
    app.rpc("DecideTrust", challenge["host"], challenge["port"], challenge["fingerprint"],
            scope, challenge["failureReason"])
    decisions = app.rpc("ListTrustDecisions")
    assert len(decisions) == 1
    assert decisions[0]["fingerprint"] == challenge["fingerprint"]
    assert decisions[0]["scope"] == scope
    trust_path = app.native.data_dir / "config/open-ldap-studio/trust.json"
    assert trust_path.exists() is (scope == "permanent")
    check.click()
    expect(page.get_by_text("✓ reachable · encrypted", exact=True)).to_be_visible()
    app.authentication()
    page.get_by_role("button", name="Check authentication", exact=True).click()
    expect(page.get_by_text(f"✓ bound as {ADMIN_DN}", exact=True).first).to_be_visible()
    page.get_by_role("button", name="Finish", exact=True).click()
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible()
    expect(page.get_by_role("status")).not_to_contain_text("TLS unverified")
    profile = app.rpc("ListProfiles")[0]
    assert app.rpc("ConnectionState", profile["id"])["tlsVerified"] is True
    page.get_by_role("button", name="Disconnect", exact=True).click()
    app.rpc("RevokeTrustDecision", challenge["fingerprint"])
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_text("Last attempt", exact=True)).to_be_visible()
    expect(page.get_by_role("status")).to_contain_text("not connected")
    assert app.rpc("ListTrustDecisions") == []
