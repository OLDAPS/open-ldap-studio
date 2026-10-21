"""Browse, inspect and refresh real LDAP data through the application's UI."""

from base64 import b64decode
import re

from playwright.sync_api import Page, expect

from e2e.live.app import LiveApp
from e2e.live.runtime import ADMIN_DN, BASE_DN, LDAPService


def test_browse_pages_and_open_an_entry_from_the_last_page(app: LiveApp, page: Page):
    tree = app.browse()
    expect(tree.get_by_role("button", name="ou=people", exact=True)).to_have_count(0)
    tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
    for ou in ("people", "groups", "services", "archive"):
        expect(tree.get_by_role("button", name=f"ou={ou}", exact=True)).to_be_visible()
    tree.get_by_role("button", name="ou=people", exact=True).dblclick()
    expect(tree.get_by_role("button", name="cn=jrivera", exact=True)).to_be_visible()

    # Filtering makes the explicit paging action visible in the virtual list.
    field = page.get_by_role("searchbox", name="Filter the directory tree")
    field.fill("no-person-matches-this")
    for loaded in (100, 200):
        more = tree.get_by_role("button", name=f"▾ fetch more · {loaded} loaded", exact=True)
        expect(more).to_be_visible()
        more.click()
    expect(tree.get_by_role("button", name=re.compile("fetch more"))).to_have_count(0)

    # Sample page boundaries against an independent LDAP client. Exhaustive
    # protocol paging belongs to the Go integration tests.
    dns = re.findall(r"^dn: (.+)$", app.ldap.search(f"ou=people,{BASE_DN}", "one"), re.M)
    assert len(dns) == 205
    for index in (0, 99, 100, 199, 204):
        rdn = dns[index].split(",", 1)[0]
        field.fill(rdn.upper())
        expect(tree.get_by_role("button", name=rdn, exact=True)).to_have_count(1)
    last = tree.get_by_role("button", name=dns[-1].split(",", 1)[0], exact=True)
    last.click()
    expect(page.get_by_text(f"dn: {dns[-1]}", exact=True)).to_be_visible()
    field.clear()
    expect(tree.get_by_role("button", name="cn=jrivera", exact=True)).to_be_visible()


def test_browsing_reports_the_server_size_limit_instead_of_hiding_truncation(
    app: LiveApp, page: Page,
):
    tree = app.people(page_size=10, size_limit=7)
    page.get_by_role("searchbox", name="Filter the directory tree").fill("no-person-matches-this")
    expect(tree.get_by_role(
        "button", name=re.compile("7 loaded · server stopped at its limit"),
    )).to_be_visible()
    expect(tree.get_by_role("button", name=re.compile("fetch more"))).to_have_count(0)


def test_inspect_text_binary_multi_value_and_operational_attributes_with_raw_ldif(
    app: LiveApp, page: Page, ldap_service: LDAPService,
):
    tree = app.read_person("cn=tvasquez")
    expect(page.get_by_text("Tomás Vásquez", exact=True)).to_be_visible()
    expect(page.get_by_text("Trabaja desde Madrid — acentos: áéíóú ñ ¿?", exact=True)).to_be_visible()
    photo = page.get_by_role("img", name="Entry photo", exact=True)
    expect(photo).to_be_visible()
    expect(photo).to_have_js_property("complete", True)
    assert photo.evaluate("image => image.naturalWidth > 0")

    expect(page.get_by_text("modifyTimestamp", exact=True)).to_have_count(0)
    page.get_by_role("checkbox", name="Operational attributes", exact=True).check()
    for attribute in ("modifyTimestamp", "createTimestamp", "entryUUID", "modifiersName"):
        expect(page.get_by_text(attribute, exact=True)).to_be_visible()
    inspector = page.get_by_role("complementary", name="Entry info", exact=True)
    expect(inspector).to_contain_text(ADMIN_DN)
    expect(inspector.get_by_text(re.compile(r"^\d{14}Z$"))).to_have_count(2)
    page.get_by_role("checkbox", name="Operational attributes", exact=True).uncheck()
    expect(page.get_by_text("modifyTimestamp", exact=True)).to_have_count(0)

    dn = f"cn=tvasquez,ou=people,{BASE_DN}"
    raw = ldap_service.search(dn, attributes=("jpegPhoto",))
    encoded_photo = re.search(r"^jpegPhoto:: (.+)$", raw, re.M)
    assert encoded_photo is not None
    assert b64decode(encoded_photo[1]).startswith(b"\xff\xd8\xff")
    page.get_by_role("tab", name="LDIF view", exact=True).click()
    ldif = page.get_by_role("tabpanel", name="LDIF view", exact=True)
    expect(ldif.get_by_text(f"dn: {dn}", exact=True)).to_be_visible()
    expect(ldif.get_by_text(f"jpegPhoto:: {encoded_photo[1]}", exact=True)).to_be_visible()
    page.get_by_role("tab", name="Object class", exact=True).click()
    expect(page.get_by_role("tabpanel", name="Object class").get_by_text(
        "inetOrgPerson", exact=True,
    )).to_be_visible()

    page.get_by_role("tab", name="Attributes", exact=True).click()
    tree.get_by_role("button", name="cn=aschmidt", exact=True).click()
    mail = page.get_by_role("button").filter(has=page.get_by_text("mail", exact=True))
    expect(mail).to_contain_text("a.schmidt@example.org")
    expect(mail).not_to_contain_text("security@example.org")
    mail.get_by_role("button", name="+1 more", exact=False).click()
    for address in ("a.schmidt@example.org", "anna.schmidt@example.org",
                    "schmidt@example.org", "security@example.org"):
        expect(mail.get_by_text(address, exact=True)).to_be_visible()

    # Switching and closing documents must retain the other entry's contents.
    tab = page.get_by_role("tab", name="cn=tvasquez Close cn=tvasquez", exact=True)
    tab.focus()
    tab.press("Enter")
    expect(page.get_by_text("Tomás Vásquez", exact=True)).to_be_visible()
    page.get_by_role("button", name="Close cn=tvasquez", exact=True).click()
    expect(page.get_by_text("Anna Schmidt", exact=True)).to_be_visible()


def test_entry_refresh_observes_an_independent_ldap_write(
    app: LiveApp, page: Page, ldap_service: LDAPService, ldap_branch: str,
):
    tree = app.browse()
    for label in (BASE_DN, "ou=archive", ldap_branch.split(",", 1)[0]):
        tree.get_by_role("button", name=label, exact=True).dblclick()
    tree.get_by_role("button", name="cn=subject", exact=True).click()
    expect(page.get_by_text("Original LDAP value", exact=True)).to_be_visible()
    dn = f"cn=subject,{ldap_branch}"
    changed = "Live LDAP refresh from a separate client"
    ldap_service.modify(f"dn: {dn}\nchangetype: modify\nreplace: description\ndescription: {changed}\n")
    expect(page.get_by_text(changed, exact=True)).to_have_count(0)
    page.get_by_role("button", name="Refresh", exact=True).filter(
        has_text=re.compile("^Refresh$"),
    ).click()
    expect(page.get_by_text(changed, exact=True)).to_be_visible()
