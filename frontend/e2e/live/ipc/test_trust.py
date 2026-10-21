"""Trust decisions use real Wails bindings until the trust dialog is wired."""

import pytest
from playwright.sync_api import Page, expect

from e2e.live.app import LiveApp
from e2e.live.runtime import ADMIN_DN


@pytest.mark.parametrize("encryption", ["startTLS", "ldaps"])
@pytest.mark.parametrize("scope", ["session", "permanent"])
def test_accepting_the_real_certificate_enables_verified_tls_and_revocation_reblocks_it(
    app: LiveApp, page: Page, encryption: str, scope: str,
):
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
    page.wait_for_function("() => window.__ldapTrustChallenges.length > 0")
    challenge = page.evaluate("() => window.__ldapTrustChallenges[0]")
    assert challenge["host"] == "127.0.0.1"
    assert challenge["port"] == app.ldap.ports["ldaps" if encryption == "ldaps" else "ldap"]
    assert "BEGIN CERTIFICATE" in challenge["chainPem"][0]
    assert challenge["previouslyTrusted"] is False

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
    profile = app.finish()
    app.connect_saved(profile)
    expect(page.get_by_role("status")).not_to_contain_text("TLS unverified")
    assert app.rpc("ConnectionState", profile["id"])["tlsVerified"] is True

    page.get_by_role("button", name="Disconnect", exact=True).click()
    app.rpc("RevokeTrustDecision", challenge["fingerprint"])
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_text("Last attempt", exact=True)).to_be_visible()
    expect(page.get_by_role("status")).to_contain_text("not connected")
    assert app.rpc("ListTrustDecisions") == []
