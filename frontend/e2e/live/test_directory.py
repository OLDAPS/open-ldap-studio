"""Read the committed seed and changes made by an independent LDAP client."""

from base64 import b64decode
import re

import pytest
from playwright.sync_api import Page, expect

from .app import LiveApp
from .runtime import BASE_DN, LDAPService


def test_tree_reads_real_naming_contexts_and_seeded_branches(app: LiveApp, page: Page):
    app.open()
    tree = app.browse()
    expect(tree.get_by_role("button", name="ou=people", exact=True)).to_have_count(0)
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    for ou in ("people", "groups", "services", "archive"):
        expect(tree.get_by_role("button", name=f"ou={ou}", exact=True)).to_be_visible()
    tree.get_by_role("button", name="ou=archive", exact=True).click()
    expect(page.get_by_text("Empty on purpose — a safe target for copy / move / delete", exact=True)).to_be_visible()


def test_explicit_paging_loads_all_205_people_without_duplicates(app: LiveApp, page: Page):
    app.open()
    tree = app.people()
    tree_filter = page.get_by_role("searchbox", name="Filter the directory tree")
    # The virtualised list only mounts visible rows. Filtering retains paging
    # actions, so reach them through the same visible controls a user can use.
    tree_filter.fill("no-person-matches-this")
    more = tree.get_by_role("button", name="▾ fetch more · 100 loaded", exact=True)
    expect(more).to_be_visible()
    more.click()
    more = tree.get_by_role("button", name="▾ fetch more · 200 loaded", exact=True)
    expect(more).to_be_visible()
    more.click()
    expect(tree.get_by_role("button", name=re.compile("fetch more"))).to_have_count(0)
    # Filtering renders every loaded child, avoiding DOM counts of virtual rows.
    dns = re.findall(r"^dn: (.+)$", app.ldap.search(f"ou=people,{BASE_DN}", "one"), re.M)
    assert len(dns) == 205
    for dn in dns:
        rdn = dn.split(",", 1)[0]
        tree_filter.fill(rdn)
        expect(tree.get_by_role("button", name=rdn, exact=True)).to_have_count(1)
    tree_filter.clear()


def test_custom_page_size_reaches_slapd(app: LiveApp, page: Page):
    app.open()
    tree = app.people(page_size=10)
    page.get_by_role("searchbox", name="Filter the directory tree").fill("no-person-matches-this")
    more = tree.get_by_role("button", name="▾ fetch more · 10 loaded", exact=True)
    expect(more).to_be_visible()
    more.click()
    expect(tree.get_by_role("button", name="▾ fetch more · 20 loaded", exact=True)).to_be_visible()


def test_a_real_server_size_limit_is_reported_as_truncation(app: LiveApp, page: Page):
    app.open()
    tree = app.people(page_size=100, size_limit=7)
    page.get_by_role("searchbox", name="Filter the directory tree").fill("no-person-matches-this")
    expect(tree.get_by_role("button", name=re.compile("7 loaded · server stopped at its limit"))).to_be_visible()


def test_filtering_loaded_entries_is_case_insensitive_and_reversible(app: LiveApp, page: Page):
    app.open()
    tree = app.people()
    field = page.get_by_role("searchbox", name="Filter the directory tree")
    field.fill("JRIVERA")
    expect(tree.get_by_role("button", name="cn=jrivera", exact=True)).to_be_visible()
    expect(tree.get_by_role("button", name="cn=lchen", exact=True)).to_have_count(0)
    field.fill("not-a-loaded-entry")
    expect(tree.get_by_role("button", name="cn=jrivera", exact=True)).to_have_count(0)
    field.clear()
    expect(tree.get_by_role("button", name="cn=jrivera", exact=True)).to_be_visible()
    expect(tree.get_by_role("button", name="cn=lchen", exact=True)).to_be_visible()


@pytest.mark.parametrize("rdn,display_name", [
    ("cn=jrivera", "Jordan Rivera"),
    ("cn=tvasquez", "Tomás Vásquez"),
    ("cn=lchen", "Li Chen"),
])
def test_entry_attributes_and_jpeg_are_read_from_the_directory(
    app: LiveApp, page: Page, rdn: str, display_name: str,
):
    app.open()
    app.read_person(rdn)
    expect(page.get_by_text(display_name, exact=True)).to_be_visible()
    expect(page.get_by_text(re.compile(r"binary · JPEG · \d"))).to_be_visible()
    photo = page.get_by_role("img", name="Entry photo", exact=True)
    assert photo.evaluate("image => image.complete && image.naturalWidth > 0")
    expect(page.get_by_role("button", name="Save (⌘S)", exact=True)).to_be_disabled()
    app.screenshot(f"{rdn} read from LDAP")


def test_multi_value_attributes_fold_and_expand_without_losing_server_values(app: LiveApp, page: Page):
    app.open()
    app.read_person("cn=aschmidt")
    mail = page.get_by_role("button").filter(has=page.get_by_text("mail", exact=True))
    expect(mail).to_contain_text("a.schmidt@example.org")
    expect(mail).not_to_contain_text("security@example.org")
    mail.get_by_role("button", name="+1 more", exact=False).click()
    for address in ("a.schmidt@example.org", "anna.schmidt@example.org", "schmidt@example.org", "security@example.org"):
        expect(mail.get_by_text(address, exact=True)).to_be_visible()


def test_operational_attributes_are_requested_from_slapd_on_demand(app: LiveApp, page: Page):
    app.open()
    app.read_person()
    expect(page.get_by_text("modifyTimestamp", exact=True)).to_have_count(0)
    page.get_by_role("checkbox", name="Operational attributes", exact=True).check()
    for attribute in ("modifyTimestamp", "createTimestamp", "entryUUID", "modifiersName"):
        expect(page.get_by_text(attribute, exact=True)).to_be_visible()
    expect(page.get_by_text("Jordan Rivera", exact=True)).to_be_visible()
    expect(page.get_by_role("img", name="Entry photo", exact=True)).to_be_visible()
    inspector = page.get_by_role("complementary", name="Entry info", exact=True)
    expect(inspector).to_contain_text("cn=admin,dc=example,dc=org")
    expect(inspector.get_by_text(re.compile(r"^\d{14}Z$"))).to_have_count(2)
    page.get_by_role("checkbox", name="Operational attributes", exact=True).uncheck()
    expect(page.get_by_text("modifyTimestamp", exact=True)).to_have_count(0)


def test_ldif_shows_real_binary_bytes_and_the_entry_object_classes(app: LiveApp, page: Page):
    app.open()
    app.read_person()
    profile = app.rpc("ListProfiles")[0]
    entry = app.rpc("ReadEntry", profile["id"], f"cn=jrivera,ou=people,{BASE_DN}", {})["entry"]
    photo = next(a for a in entry["attributes"] if a["type"] == "jpegPhoto")["values"][0]
    assert b64decode(photo).startswith(b"\xff\xd8\xff")
    page.get_by_role("tab", name="LDIF view", exact=True).click()
    ldif = page.get_by_role("tabpanel", name="LDIF view", exact=True)
    expect(ldif.get_by_text(f"dn: cn=jrivera,ou=people,{BASE_DN}", exact=True)).to_be_visible()
    expect(ldif.get_by_text(f"jpegPhoto:: {photo}", exact=True)).to_be_visible()
    page.get_by_role("tab", name="Object class", exact=True).click()
    panel = page.get_by_role("tabpanel", name="Object class", exact=True)
    for name in ("top", "person", "organizationalPerson", "inetOrgPerson", "posixAccount"):
        expect(panel.get_by_text(name, exact=True)).to_be_visible()


def test_entry_refresh_observes_an_independent_ldap_write(
    app: LiveApp, page: Page, ldap_service: LDAPService, ldap_branch: str,
):
    app.open()
    tree = app.browse()
    for label in (BASE_DN, "ou=archive", ldap_branch.split(",", 1)[0]):
        tree.get_by_role("button", name=label, exact=True).dblclick()
    tree.get_by_role("button", name="cn=subject", exact=True).click()
    expect(page.get_by_text("Original LDAP value", exact=True)).to_be_visible()
    dn = f"cn=subject,{ldap_branch}"
    changed = "Live LDAP refresh from a separate client"
    ldap_service.modify(f"dn: {dn}\nchangetype: modify\nreplace: description\ndescription: {changed}\n")
    expect(page.get_by_text(changed, exact=True)).to_have_count(0)
    page.get_by_role("button", name="Refresh", exact=True).last.click()
    expect(page.get_by_text(changed, exact=True)).to_be_visible()


def test_a_leading_space_in_an_ldap_value_is_preserved(app: LiveApp, page: Page):
    app.open()
    app.read_person("cn=lchen")
    description = page.get_by_role("button").filter(has=page.get_by_text("description", exact=True))
    assert " leading space — must round-trip as base64" in description.text_content()


def test_entry_documents_switch_with_the_keyboard_and_close(app: LiveApp, page: Page):
    app.open()
    tree = app.read_person()
    tree.get_by_role("button", name="cn=lchen", exact=True).click()
    expect(page.get_by_text("Li Chen", exact=True)).to_be_visible()
    tab = page.get_by_role("tab", name="cn=jrivera Close cn=jrivera", exact=True)
    tab.focus()
    tab.press("Enter")
    expect(page.get_by_text("Jordan Rivera", exact=True)).to_be_visible()
    page.get_by_role("button", name="Close cn=jrivera", exact=True).click()
    expect(page.get_by_text("Li Chen", exact=True)).to_be_visible()
    page.get_by_role("button", name="Close cn=lchen", exact=True).click()
    # Expanding branches opens their entries too; the closed person tabs must
    # disappear even when those branch documents are still open.
    expect(page.get_by_role("tab", name=re.compile("cn=jrivera|cn=lchen"))).to_have_count(0)
