import json
import re

from playwright.sync_api import Dialog, Page, expect

from .fixtures.app import AppFixture
from .fixtures.data import BASE_DN, PROFILE, SUCCESS


def test_an_empty_workspace_invites_creating_the_first_connection(app: AppFixture, page: Page):
    """an empty workspace invites creating the first connection"""
    app.open({"profiles": []})
    expect(page.get_by_text("no connections yet — use + to add one")).to_be_visible()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    expect(page.get_by_label("Hostname", exact=True)).to_be_visible()
    expect(page.get_by_role("button", name="Finish", exact=True)).to_be_disabled()


def test_a_saved_connection_can_connect_browse_and_disconnect(app: AppFixture, page: Page):
    """a saved connection can connect, browse and disconnect"""
    app.open()
    app.browse()
    expect(page.get_by_role("status")).to_contain_text(f"bound: {PROFILE['bindDn']}")
    expect(
        page.get_by_role("complementary", name="Directory tree").get_by_role(
            "button", name=BASE_DN, exact=True
        )
    ).to_be_visible()
    app.navigate("Connections")
    page.get_by_role("complementary", name="Connections", exact=True).get_by_role(
        "button", name=PROFILE["name"], exact=True
    ).click()
    page.get_by_role("button", name="Disconnect", exact=True).click()
    expect(page.get_by_role("status")).to_contain_text("not connected")
    expect(page.get_by_role("button", name="Connect", exact=True)).to_be_enabled()
    app.navigate("DIT browser")
    expect(page.get_by_text("not connected", exact=True).first).to_be_visible()
    assert len(app.calls("Disconnect")) == 1


def test_connection_failures_show_the_original_diagnostic_and_allow_retry(app: AppFixture, page: Page):
    """connection failures show the original diagnostic and allow retry"""
    app.open()
    page.get_by_role("button", name=PROFILE["name"], exact=True).click()
    app.fail_next("Connect", "LDAP result 49: invalidCredentials — test bind rejected")
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_role("alert")).to_contain_text(
        "LDAP result 49: invalidCredentials — test bind rejected"
    )
    expect(page.get_by_role("status")).to_contain_text("not connected")
    page.get_by_role("button", name="Connect", exact=True).click()
    expect(page.get_by_role("button", name="Browse", exact=True)).to_be_visible()


def test_profile_refresh_reports_a_load_error_and_permits_another_load(app: AppFixture, page: Page):
    """profile refresh reports a load error and permits another load"""
    app.open()
    app.fail_next("ListProfiles", "Profile store is temporarily unavailable")
    refresh = page.get_by_role("complementary", name="Connections", exact=True).get_by_role(
        "button", name="Refresh"
    )
    refresh.click()
    expect(page.get_by_role("alert")).to_contain_text("Profile store is temporarily unavailable")
    app.respond_next("ListProfiles", [{**PROFILE, "name": "Updated example directory"}])
    refresh.click()
    expect(page.get_by_role("button", name="Updated example directory", exact=True)).to_be_visible()
    assert len(app.calls("ListProfiles")) == 3


def test_network_options_set_default_ports_and_preserve_a_manually_entered_port(
    app: AppFixture, page: Page,
):
    """network options set default ports and preserve a manually entered port"""
    app.open()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    port = page.get_by_label("Port", exact=True)
    encryption = page.get_by_label("Encryption", exact=True)
    expect(port).to_have_value("389")
    encryption.select_option("ldaps")
    expect(port).to_have_value("636")
    port.clear()
    port.press_sequentially("10a!389")
    expect(port).to_have_value("10389")
    encryption.select_option("startTLS")
    expect(port).to_have_value("10389")
    expect(page.get_by_role("button", name="Check network parameter")).to_be_disabled()
    page.get_by_label("Hostname", exact=True).fill("ldap.example.test")
    page.get_by_role("button", name="Check network parameter").click()
    expect(page.get_by_text(re.compile("✓ reachable · encrypted"))).to_be_visible()
    assert len(app.calls("SaveProfile")) == 0
    calls = app.calls("TestConnection")
    checked = json.loads(calls[0]["args"][0])
    assert {key: checked[key] for key in ("bindMethod", "port")} == {
        "bindMethod": "anonymous",
        "port": 10389,
    }
    page.get_by_label("Hostname", exact=True).fill("changed.example.test")
    expect(page.get_by_text(re.compile("✓ reachable"))).to_be_hidden()


def test_cleartext_warnings_follow_the_chosen_transport_and_bind_method(app: AppFixture, page: Page):
    """cleartext warnings follow the chosen transport and bind method"""
    app.open()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    page.get_by_label("Encryption", exact=True).select_option("none")
    expect(page.get_by_text("Unencrypted", exact=True)).to_be_visible()
    page.get_by_role("button", name="Next", exact=True).click()
    expect(page.get_by_text("Clear text", exact=True)).to_be_visible()
    page.get_by_role("button", name="Anonymous", exact=True).click()
    expect(page.get_by_label("Password", exact=True)).to_be_hidden()
    expect(page.get_by_text("Clear text", exact=True)).to_be_hidden()
    page.get_by_role("button", name="Simple", exact=True).click()
    page.get_by_role("button", name="Back", exact=True).click()
    page.get_by_label("Encryption", exact=True).select_option("ldaps")
    page.get_by_role("button", name="Next", exact=True).click()
    expect(page.get_by_text("Clear text", exact=True)).to_be_hidden()


def test_the_four_step_wizard_saves_options_and_stores_the_password_separately(
    app: AppFixture, page: Page,
):
    """the four-step wizard saves options and stores the password separately"""
    app.open()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    expect(page.get_by_role("button", name="Back", exact=True)).to_be_disabled()
    page.get_by_label("Connection name", exact=True).fill("Browser-created directory")
    page.get_by_label("Hostname", exact=True).fill("new.example.test")
    page.get_by_label("Encryption", exact=True).select_option("ldaps")
    page.get_by_role("button", name="Next", exact=True).click()
    page.get_by_label("Bind DN", exact=True).fill(PROFILE["bindDn"])
    page.get_by_label("Password", exact=True).fill("test-only-password")
    expect(page.get_by_label("Password", exact=True)).to_have_attribute("type", "password")
    page.get_by_role("button", name="Check authentication").click()
    expect(page.get_by_text(re.compile("✓ bound as")).first).to_be_visible()
    page.get_by_role("button", name="Next", exact=True).click()
    expect(page.get_by_label("Base DN", exact=True)).to_have_value(BASE_DN)
    page.get_by_label("Page size", exact=True).fill("50")
    page.get_by_label("Size limit", exact=True).fill("500")
    page.get_by_label("Time limit (s)", exact=True).fill("20")
    page.get_by_label("Aliases", exact=True).select_option("never")
    page.get_by_label("Referrals", exact=True).select_option("ask")
    page.get_by_role("button", name="Next", exact=True).click()
    expect(page.get_by_role("button", name="Next", exact=True)).to_be_disabled()
    page.get_by_role("checkbox", name=re.compile("Refused at the changeset boundary")).check()
    page.get_by_role("checkbox", name=re.compile("The status bar carries the tag")).check()
    assert len(app.calls("SaveProfile")) == 0
    page.get_by_role("button", name="Finish", exact=True).click()
    expect(
        page.get_by_role("button", name="Browser-created directory read-only", exact=True)
    ).to_be_visible()
    saved = app.calls("SaveProfile")
    assert len(saved) == 1
    saved_profile = json.loads(saved[0]["args"][0])
    expected = {
        "host": "new.example.test",
        "port": 636,
        "encryption": "ldaps",
        "baseDn": BASE_DN,
        "readOnly": True,
        "tags": ["production"],
        "aliases": "never",
        "referrals": "ask",
        "limits": {"pageSize": 50, "sizeLimit": 500, "timeLimit": 20},
    }
    assert {key: saved_profile[key] for key in expected} == expected
    assert "test-only-password" not in saved[0]["args"][0]
    assert "secret" not in saved_profile
    assert app.calls("StoreProfileSecret")[0]["args"] == ["profile-2", "test-only-password"]


def test_an_anonymous_profile_requires_no_bind_dn_or_stored_secret(app: AppFixture, page: Page):
    """an anonymous profile requires no bind DN or stored secret"""
    app.open()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    page.get_by_label("Hostname", exact=True).fill("anonymous.example.test")
    page.get_by_role("button", name="2 · Authentication").click()
    page.get_by_role("button", name="Anonymous", exact=True).click()
    page.get_by_role("button", name="Finish", exact=True).click()
    expect(page.get_by_role("button", name="anonymous.example.test", exact=True)).to_be_visible()
    saved = app.calls("SaveProfile")
    saved_profile = json.loads(saved[0]["args"][0])
    assert {key: saved_profile[key] for key in ("bindMethod", "bindDn")} == {
        "bindMethod": "anonymous",
        "bindDn": "",
    }
    assert len(app.calls("StoreProfileSecret")) == 0


def test_a_rejected_authentication_check_preserves_the_server_code_and_diagnostic(
    app: AppFixture, page: Page,
):
    """a rejected authentication check preserves the server code and diagnostic"""
    app.open()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    page.get_by_label("Hostname", exact=True).fill("ldap.example.test")
    page.get_by_role("button", name="2 · Authentication").click()
    page.get_by_label("Bind DN", exact=True).fill(PROFILE["bindDn"])
    app.respond_next(
        "TestConnection",
        {
            "reachable": True,
            "encrypted": True,
            "tlsVerified": True,
            "bound": False,
            "result": {
                **SUCCESS,
                "code": 49,
                "diagnosticMessage": "test directory: bind rejected",
                "interpretation": "Invalid credentials",
            },
        },
    )
    page.get_by_role("button", name="Check authentication").click()
    expect(
        page.get_by_text(
            "✗ result 49 — test directory: bind rejected (Invalid credentials)", exact=True
        )
    ).to_be_visible()
    assert len(app.calls("SaveProfile")) == 0


def test_an_unreachable_network_check_displays_the_transport_error(app: AppFixture, page: Page):
    """an unreachable network check displays the transport error"""
    app.open()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    page.get_by_label("Hostname", exact=True).fill("offline.example.test")
    app.respond_next(
        "TestConnection",
        {
            "reachable": False,
            "bound": False,
            "encrypted": False,
            "tlsVerified": False,
            "result": SUCCESS,
            "message": "dial tcp: connection refused",
        },
    )
    page.get_by_role("button", name="Check network parameter").click()
    expect(page.get_by_role("alert")).to_contain_text("Unreachable")
    expect(page.get_by_role("alert")).to_contain_text("dial tcp: connection refused")


def test_a_failed_save_keeps_the_wizard_draft_available_for_retry(app: AppFixture, page: Page):
    """a failed save keeps the wizard draft available for retry"""
    app.open()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    page.get_by_label("Hostname", exact=True).fill("retry.example.test")
    page.get_by_role("button", name="2 · Authentication").click()
    page.get_by_role("button", name="Anonymous", exact=True).click()
    app.fail_next("SaveProfile", "Profile store: permission denied")
    page.get_by_role("button", name="Finish", exact=True).click()
    expect(page.get_by_role("alert")).to_contain_text("Profile store: permission denied")
    page.get_by_role("button", name="1 · Network").click()
    expect(page.get_by_label("Hostname", exact=True)).to_have_value("retry.example.test")
    page.get_by_role("button", name="Finish", exact=True).click()
    expect(page.get_by_role("button", name="retry.example.test", exact=True)).to_be_visible()


def test_cancelling_the_wizard_discards_its_draft_without_saving_a_profile(app: AppFixture, page: Page):
    """cancelling the wizard discards its draft without saving a profile"""
    app.open()
    page.get_by_role("button", name="New connection", exact=True).last.click()
    page.get_by_label("Hostname", exact=True).fill("discard.example.test")
    page.get_by_role("button", name="Cancel", exact=True).click()
    assert len(app.calls("SaveProfile")) == 0
    assert len(app.calls("StoreProfileSecret")) == 0
    page.get_by_role("button", name="New connection", exact=True).last.click()
    expect(page.get_by_label("Hostname", exact=True)).to_have_value("")


def test_deleting_a_connection_requires_accepting_the_named_confirmation(app: AppFixture, page: Page):
    """deleting a connection requires accepting the named confirmation"""
    app.open()
    page.get_by_role("button", name=PROFILE["name"], exact=True).click()
    confirmations = []

    def dismiss(dialog: Dialog):
        confirmations.append(dialog.message)
        dialog.dismiss()

    page.once("dialog", dismiss)
    page.get_by_role("button", name="Delete", exact=True).click()
    assert f'"{PROFILE["name"]}"' in confirmations[0]
    assert "stored secret" in confirmations[0]
    assert len(app.calls("DeleteProfile")) == 0
    expect(page.get_by_role("button", name=PROFILE["name"], exact=True)).to_be_visible()
    page.once("dialog", lambda dialog: dialog.accept())
    page.get_by_role("button", name="Delete", exact=True).click()
    expect(page.get_by_text("no connections yet — use + to add one")).to_be_visible()
    assert app.calls("DeleteProfile")[0]["args"] == [PROFILE["id"]]
