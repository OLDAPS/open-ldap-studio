import re

from playwright.sync_api import Locator, Page, expect

from .fixtures.app import AppFixture
from .fixtures.data import (
    ALICE_DN,
    BASE_DN,
    BOB_DN,
    ENTRIES,
    PEOPLE_DN,
    PROFILE,
    ROOT_DSE,
    SUCCESS,
    attribute,
    page_of,
)


def expand_people(app: AppFixture) -> Locator:
    app.browse()
    tree = app.page.get_by_role("complementary", name="Directory tree")
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    tree.get_by_role("button", name="ou=people", exact=True).dblclick()
    expect(tree.get_by_role("button", name="uid=alice", exact=True)).to_be_visible()
    return tree


def read_alice(app: AppFixture) -> Locator:
    tree = expand_people(app)
    tree.get_by_role("button", name="uid=alice", exact=True).click()
    expect(app.page.get_by_text("Alice Example", exact=True)).to_be_visible()
    return tree


def test_naming_contexts_load_first_and_children_are_fetched_only_when_expanded(
    app: AppFixture, page: Page,
):
    """naming contexts load first and children are fetched only when expanded"""
    app.open()
    app.browse()
    tree = page.get_by_role("complementary", name="Directory tree")
    expect(tree.get_by_role("button", name=BASE_DN, exact=True)).to_be_visible()
    assert len(app.calls("ListChildren")) == 0
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    expect(tree.get_by_role("button", name="ou=people", exact=True)).to_be_visible()
    expect(tree.get_by_role("button", name="uid=alice", exact=True)).to_have_count(0)
    assert app.calls("ListChildren")[0]["args"][1] == BASE_DN


def test_the_next_page_is_an_explicit_action_and_appends_entries_once(app: AppFixture, page: Page):
    """the next page is an explicit action and appends entries once"""
    app.open()
    tree = expand_people(app)
    expect(tree.get_by_role("button", name="uid=carol", exact=True)).to_have_count(0)
    tree.get_by_role("button", name="▾ fetch more · 2 loaded", exact=True).click()
    expect(tree.get_by_role("button", name="uid=carol", exact=True)).to_be_visible()
    expect(tree.get_by_role("button", name="uid=alice", exact=True)).to_have_count(1)
    expect(tree.get_by_role("button", name=re.compile("fetch more"))).to_have_count(0)
    assert app.calls("ListChildren")[-1]["args"] == [
        PROFILE["id"],
        PEOPLE_DN,
        {"size": 0, "cookie": "page-2", "includeOperational": False},
    ]
    expect(page.get_by_role("status")).to_contain_text(PROFILE["host"])


def test_collapsing_and_reopening_a_branch_reads_its_current_children(app: AppFixture):
    """collapsing and reopening a branch reads its current children"""
    app.open()
    tree = expand_people(app)
    people = tree.get_by_role("button", name="ou=people", exact=True)
    people.dblclick()
    expect(tree.get_by_role("button", name="uid=alice", exact=True)).to_have_count(0)
    app.respond_next("ListChildren", page_of([BOB_DN]))
    people.dblclick()
    expect(tree.get_by_role("button", name="uid=bob", exact=True)).to_be_visible()
    expect(tree.get_by_role("button", name="uid=alice", exact=True)).to_have_count(0)
    assert len(app.calls("ListChildren")) == 3


def test_filtering_the_loaded_tree_is_local_and_clearing_it_restores_rows(app: AppFixture, page: Page):
    """filtering the loaded tree is local and clearing it restores rows"""
    app.open()
    tree = expand_people(app)
    before = len(app.calls("ListChildren"))
    tree_filter = page.get_by_role("searchbox", name="Filter the directory tree")
    tree_filter.fill("ALICE")
    expect(tree.get_by_role("button", name="uid=alice", exact=True)).to_be_visible()
    expect(tree.get_by_role("button", name="uid=bob", exact=True)).to_have_count(0)
    tree_filter.clear()
    expect(tree.get_by_role("button", name="uid=bob", exact=True)).to_be_visible()
    assert len(app.calls("ListChildren")) == before


def test_entry_views_decode_text_preserve_attribute_options_and_describe_binary_values(
    app: AppFixture, page: Page,
):
    """entry views decode text, preserve attribute options and describe binary values"""
    app.open()
    read_alice(app)
    expect(page.get_by_text("Engenheira — diretório LDAP", exact=True)).to_be_visible()
    expect(page.get_by_text("description;lang-fr", exact=True)).to_be_visible()
    expect(page.get_by_text("Bonjour Alice", exact=True)).to_be_visible()
    expect(page.get_by_text("binary · 5 B", exact=True)).to_be_visible()
    expect(page.get_by_role("complementary", name="Entry info")).to_contain_text(
        "top, person, inetOrgPerson"
    )
    expect(page.get_by_role("button", name="Save (⌘S)", exact=True)).to_be_disabled()


def test_multi_valued_attributes_expand_all_values_and_fold_back_after_three(
    app: AppFixture, page: Page,
):
    """multi-valued attributes expand all values and fold back after three"""
    app.open()
    read_alice(app)
    expect(page.get_by_text("alice.other@example.org", exact=True)).to_have_count(0)
    page.get_by_role("button", name="+2 more", exact=True).click()
    expect(page.get_by_text("alice.other@example.org", exact=True)).to_be_visible()
    expect(page.get_by_text("alice.last@example.org", exact=True)).to_be_visible()
    page.get_by_role("button", name="show fewer", exact=True).click()
    expect(page.get_by_text("alice.last@example.org", exact=True)).to_have_count(0)
    expect(page.get_by_role("button", name="+2 more", exact=True)).to_be_visible()


def test_operational_attributes_are_requested_explicitly_and_update_entry_facts(
    app: AppFixture, page: Page,
):
    """operational attributes are requested explicitly and update entry facts"""
    app.open()
    read_alice(app)
    expect(page.get_by_text("modifyTimestamp", exact=True)).to_have_count(0)
    page.get_by_role("checkbox", name="Operational attributes").check()
    expect(page.get_by_text("modifyTimestamp", exact=True)).to_be_visible()
    expect(page.get_by_role("complementary", name="Entry info")).to_contain_text("20261016123000Z")
    assert app.calls("ReadEntry")[-1]["args"][2]["IncludeOperational"] is True
    page.get_by_role("checkbox", name="Operational attributes").uncheck()
    expect(page.get_by_text("modifyTimestamp", exact=True)).to_have_count(0)
    assert app.calls("ReadEntry")[-1]["args"][2]["IncludeOperational"] is False


def test_ldif_and_object_class_tabs_show_the_selected_entry_faithfully(app: AppFixture, page: Page):
    """LDIF and object-class tabs show the selected entry faithfully"""
    app.open()
    read_alice(app)
    page.get_by_role("tab", name="LDIF view", exact=True).click()
    ldif = page.get_by_role("tabpanel", name="LDIF view", exact=True)
    expect(ldif).to_contain_text(f"dn: {ALICE_DN}")
    expect(ldif).to_contain_text("cn: Alice Example")
    expect(ldif).to_contain_text("description;lang-fr: Bonjour Alice")
    expect(ldif).to_contain_text("jpegPhoto:: AAECAwQ=")
    page.get_by_role("tab", name="Table editor", exact=True).click()
    expect(page.get_by_role("tabpanel", name="Table editor", exact=True)).to_contain_text(
        "one row per child entry"
    )
    page.get_by_role("tab", name="Object class", exact=True).click()
    expect(
        page.get_by_role("tabpanel", name="Object class", exact=True).get_by_text(
            "inetOrgPerson", exact=True
        )
    ).to_be_visible()
    page.get_by_role("tab", name="Attributes", exact=True).click()
    expect(page.get_by_text("Alice Example", exact=True)).to_be_visible()


def test_document_tabs_avoid_duplicates_activate_by_keyboard_and_close(app: AppFixture, page: Page):
    """document tabs avoid duplicates, activate by keyboard and close"""
    app.open()
    tree = read_alice(app)
    tree.get_by_role("button", name="uid=alice", exact=True).click()
    expect(page.get_by_role("tab", name="uid=alice Close uid=alice", exact=True)).to_have_count(1)
    tree.get_by_role("button", name="uid=bob", exact=True).click()
    expect(page.get_by_text("Bob Example", exact=True)).to_be_visible()
    alice = page.get_by_role("tab").filter(
        has=page.get_by_role("button", name="Close uid=alice", exact=True)
    )
    alice.focus()
    page.keyboard.press("Enter")
    expect(alice).to_have_attribute("aria-selected", "true")
    expect(page.get_by_text("Alice Example", exact=True)).to_be_visible()
    page.get_by_role("button", name="Close uid=alice", exact=True).click()
    expect(page.get_by_role("button", name="Close uid=alice", exact=True)).to_have_count(0)
    expect(page.get_by_text("Bob Example", exact=True)).to_be_visible()


def test_entry_refresh_fetches_changed_server_data(app: AppFixture, page: Page):
    """entry refresh fetches changed server data"""
    app.open()
    read_alice(app)
    updated = {**ENTRIES[ALICE_DN], "attributes": [attribute("cn", ["Alice Updated"])]}
    app.respond_next("ReadEntry", {"entry": updated, "result": SUCCESS})
    page.locator(".main").get_by_role("button", name="Refresh", exact=True).click()
    expect(page.get_by_text("Alice Updated", exact=True)).to_be_visible()
    expect(page.get_by_text("Alice Example", exact=True)).to_have_count(0)


def test_an_entry_read_error_remains_visible_until_a_successful_retry(app: AppFixture, page: Page):
    """an entry read error remains visible until a successful retry"""
    app.open()
    tree = expand_people(app)
    # Let the container read finish before failing the next entry read.
    expect(page.get_by_text("organizationalUnit", exact=True).first).to_be_visible()
    app.fail_next("ReadEntry", "LDAP result 50: insufficientAccessRights")
    tree.get_by_role("button", name="uid=alice", exact=True).click()
    expect(page.get_by_role("alert")).to_contain_text("Could not read the entry")
    expect(page.get_by_role("alert")).to_contain_text("LDAP result 50: insufficientAccessRights")
    page.locator(".main").get_by_role("button", name="Refresh", exact=True).click()
    expect(page.get_by_text("Alice Example", exact=True)).to_be_visible()
    expect(page.get_by_role("alert")).to_have_count(0)


def test_a_child_fetch_error_is_shown_on_the_failed_tree_branch(app: AppFixture, page: Page):
    """a child fetch error is shown on the failed tree branch"""
    app.open()
    app.browse()
    app.fail_next("ListChildren", "LDAP result 51: busy")
    tree = page.get_by_role("complementary", name="Directory tree")
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    expect(tree.get_by_title("LDAP result 51: busy", exact=True)).to_contain_text("failed")


def test_server_truncation_is_stated_alongside_the_number_of_children_loaded(app: AppFixture):
    """server truncation is stated alongside the number of children loaded"""
    app.open()
    app.browse()
    app.respond_next("ListChildren", page_of([PEOPLE_DN], {"truncatedByServer": True, "serverLimit": 1}))
    tree = app.page.get_by_role("complementary", name="Directory tree")
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    expect(
        tree.get_by_role("button", name="▾ 1 loaded · server stopped at its limit (1)", exact=True)
    ).to_be_visible()


def test_the_configured_base_dn_is_used_when_root_dse_has_no_naming_contexts(
    app: AppFixture, page: Page,
):
    """the configured base DN is used when root DSE has no naming contexts"""
    app.open({"rootDSE": {**ROOT_DSE, "namingContexts": []}})
    app.browse()
    expect(
        page.get_by_role("complementary", name="Directory tree").get_by_role(
            "button", name=BASE_DN, exact=True
        )
    ).to_be_visible()
    assert app.calls("GetProfile")[0]["args"] == [PROFILE["id"]]


def test_missing_naming_contexts_and_a_missing_base_dn_produce_an_actionable_error(
    app: AppFixture, page: Page,
):
    """missing naming contexts and a missing base DN produce an actionable error"""
    app.open(
        {"profiles": [{**PROFILE, "baseDn": ""}], "rootDSE": {**ROOT_DSE, "namingContexts": []}}
    )
    app.browse()
    expect(page.get_by_role("alert")).to_contain_text(
        "The server published no naming contexts and this connection has no base DN set."
    )


def test_a_root_dse_failure_reports_the_backend_diagnostic(app: AppFixture, page: Page):
    """a root DSE failure reports the backend diagnostic"""
    app.open()
    app.fail_next("RootDSE", "Root DSE denied by test directory")
    app.browse()
    expect(page.get_by_role("alert")).to_contain_text("Root DSE denied by test directory")
