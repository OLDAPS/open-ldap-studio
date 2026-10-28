from playwright.sync_api import Page, expect

from .fixtures.app import AppFixture
from .fixtures.data import BASE_DN, PROFILE


def test_search_inputs_preserve_the_raw_ldap_filter_and_request_options(app: AppFixture, page: Page):
    """search inputs preserve the raw LDAP filter and request options"""
    app.open()
    app.navigate("Searches")
    expect(page.get_by_label("Filter", exact=True)).to_have_value("(objectClass=*)")
    ldap_filter = r"(&(objectClass=inetOrgPerson)(cn=Alice\2a Example))"
    page.get_by_label("Search base", exact=True).fill(BASE_DN)
    page.get_by_label("Filter", exact=True).fill(ldap_filter)
    page.get_by_label("Returning attrs", exact=True).fill("uid, cn, mail, +")
    for scope in ("base", "one", "sub"):
        page.get_by_role("combobox", name="Scope", exact=True).select_option(scope)
        expect(page.get_by_role("combobox", name="Scope", exact=True)).to_have_value(scope)
    page.get_by_label("Size limit", exact=True).fill("250")
    page.get_by_label("Time limit", exact=True).fill("15")
    page.get_by_role("combobox", name="Aliases", exact=True).select_option("never")
    page.get_by_role("combobox", name="Referrals", exact=True).select_option("ask")
    expect(page.get_by_label("Filter", exact=True)).to_have_value(ldap_filter)
    expect(page.get_by_label("Search base", exact=True)).to_have_value(BASE_DN)
    expect(page.get_by_label("Returning attrs", exact=True)).to_have_value("uid, cn, mail, +")
    expect(page.get_by_role("button", name="Export LDIF / CSV", exact=True)).to_be_disabled()
    expect(page.get_by_role("button", name="Bulk modify…", exact=True)).to_be_disabled()
    expect(page.get_by_text("no saved searches yet")).to_be_visible()


def test_schema_starts_with_navigation_categories_and_guidance_for_selecting_a_server_element(
    app: AppFixture, page: Page,
):
    """schema starts with navigation categories and guidance for selecting a server element"""
    app.open()
    app.navigate("Schema")
    sidebar = page.get_by_role("complementary", name="Schema", exact=True)
    expect(sidebar).to_contain_text("Object classes")
    expect(sidebar).to_contain_text("Attribute types")
    expect(sidebar).to_contain_text("Matching rules · syntaxes")
    page.get_by_role("searchbox", name="Search the schema").fill("inetOrgPerson")
    expect(page.get_by_role("searchbox", name="Search the schema")).to_have_value("inetOrgPerson")
    expect(
        page.get_by_text("Open a connection and pick an element on the left.", exact=False)
    ).to_be_visible()


def test_ldif_drafts_count_records_and_survive_switching_source_diff_and_report_tabs(
    app: AppFixture, page: Page,
):
    """LDIF drafts count records and survive switching source, diff and report tabs"""
    app.open()
    app.navigate("LDIF & files")
    source = page.get_by_role("textbox", name="LDIF source")
    expect(source).to_be_in_viewport(ratio=1)
    editor_bounds = source.bounding_box()
    assert editor_bounds is not None
    assert editor_bounds["width"] > 300
    assert editor_bounds["height"] > 200
    expect(page.get_by_text("no change records", exact=True)).to_be_visible()
    first = f"dn: cn=first,{BASE_DN}\nchangetype: add\nobjectClass: person\ncn: first\nsn: example\n"
    source.fill(first)
    expect(page.get_by_text("1 change record", exact=True)).to_be_visible()
    draft = f"{first}\ndn: cn=second,{BASE_DN}\nchangetype: delete\n"
    source.fill(draft)
    expect(page.get_by_text("2 change records", exact=True)).to_be_visible()
    page.get_by_role("tab", name="Diff vs. server", exact=True).click()
    expect(page.get_by_role("tabpanel", name="Diff vs. server", exact=True)).to_contain_text(
        "before / after attribute diff for each record"
    )
    page.get_by_role("tab", name="Import report", exact=True).click()
    expect(page.get_by_role("tabpanel", name="Import report", exact=True)).to_contain_text(
        "the last run’s report"
    )
    page.get_by_role("tab", name="Source", exact=True).click()
    expect(source).to_have_value(draft)
    source.clear()
    expect(page.get_by_text("no change records", exact=True)).to_be_visible()


def test_ldif_defaults_to_a_dry_run_and_has_no_execution_target_when_disconnected(
    app: AppFixture, page: Page,
):
    """LDIF defaults to a dry run and has no execution target when disconnected"""
    app.open()
    app.navigate("LDIF & files")
    dry_run = page.get_by_role("checkbox", name="dry run", exact=True)
    continue_on_error = page.get_by_role("checkbox", name="continue on error", exact=True)
    expect(dry_run).to_be_checked()
    expect(continue_on_error).not_to_be_checked()
    continue_on_error.check()
    expect(continue_on_error).to_be_checked()
    dry_run.uncheck()
    expect(dry_run).not_to_be_checked()
    expect(page.get_by_role("button", name="Execute", exact=True)).to_be_disabled()
    result = page.get_by_role("complementary", name="Dry-run result", exact=True)
    expect(result).to_contain_text("0 would succeed")
    expect(result).to_contain_text("Nothing has been dry-run yet.")
    expect(result.get_by_role("button", name="Save report", exact=True)).to_be_disabled()


def test_the_ldif_execution_button_names_the_active_directory(app: AppFixture, page: Page):
    """the LDIF execution button names the active directory"""
    app.open()
    app.connect()
    app.navigate("LDIF & files")
    target = f"{PROFILE['host']}:{PROFILE['port']}"
    expect(page.get_by_role("button", name=f"Execute on {target}", exact=True)).to_be_visible()
    expect(page.get_by_role("complementary", name="Workspace files")).to_contain_text(target)
