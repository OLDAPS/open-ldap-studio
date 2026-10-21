"""User actions shared by critical journeys and Wails IPC integration tests."""

import re

from playwright.sync_api import Locator, Page, expect

from .runtime import ADMIN_DN, ADMIN_PASSWORD, BASE_DN, LDAPService, NativeApp


class LiveApp:
    def __init__(self, page: Page, native: NativeApp, ldap: LDAPService) -> None:
        self.page = page
        self.native = native
        self.ldap = ldap
        self.profile_name = "Local LDAP E2E"

    def open(self) -> None:
        self.page.goto(self.native.url)
        expect(self.page.get_by_role("navigation", name="Perspectives")).to_be_visible()
        assert self.rpc("GetAppInfo")["platform"] == "linux"
        assert self.page.evaluate("() => !('__testBridge' in window)")

    def restart(self) -> None:
        self.native.stop()
        self.native.start()
        self.open()

    def rpc(self, method: str, *args: object) -> object:
        """Call a real binding for inspection or an IPC integration workflow."""
        return self.page.evaluate(
            "async ({method, args}) => await window.go.bridge.Bridge[method](...args)",
            {"method": method, "args": list(args)},
        )

    def navigate(self, name: str) -> None:
        button = self.page.get_by_role("navigation", name="Perspectives").get_by_role(
            "button", name=name, exact=True,
        )
        button.click()
        expect(button).to_have_attribute("aria-current", "page")

    def select_profile(self) -> None:
        self.page.get_by_role("complementary", name="Connections", exact=True).get_by_role(
            "button", name=self.profile_name,
        ).click()

    def network(self, encryption: str = "none", verify: bool = True, port: int | None = None) -> None:
        self.page.get_by_role("complementary", name="Connections", exact=True).get_by_role(
            "button", name="New connection", exact=True,
        ).click()
        self.page.get_by_label("Connection name", exact=True).fill(self.profile_name)
        self.page.get_by_label("Hostname", exact=True).fill("127.0.0.1")
        self.page.get_by_label("Encryption", exact=True).select_option(encryption)
        actual_port = port if port is not None else self.ldap.ports[
            "ldaps" if encryption == "ldaps" else "ldap"
        ]
        self.page.get_by_label("Port", exact=True).fill(str(actual_port))
        if encryption != "none" and not verify:
            self.page.get_by_role("checkbox", name="Verify chain", exact=True).uncheck()

    def authentication(self, dn: str = ADMIN_DN, password: str = ADMIN_PASSWORD,
                       anonymous: bool = False) -> None:
        self.page.get_by_role("button", name="2 · Authentication", exact=True).click()
        if anonymous:
            self.page.get_by_role("button", name="Anonymous", exact=True).click()
        else:
            self.page.get_by_label("Bind DN", exact=True).fill(dn)
            self.page.get_by_label("Password", exact=True).fill(password)

    def finish(self) -> dict:
        self.page.get_by_role("button", name="Finish", exact=True).click()
        expect(self.page.get_by_role("button", name="Connect", exact=True)).to_be_visible()
        profiles = self.rpc("ListProfiles")
        assert len(profiles) == 1
        return self.rpc("GetProfile", profiles[0]["id"])

    def save(self, *, encryption: str = "none", verify: bool = True,
             dn: str = ADMIN_DN, password: str = ADMIN_PASSWORD, anonymous: bool = False,
             page_size: int = 100, size_limit: int = 0, read_only: bool = False,
             production: bool = False) -> dict:
        self.network(encryption, verify)
        self.authentication(dn, password, anonymous)
        self.page.get_by_role("button", name="3 · Browser options", exact=True).click()
        self.page.get_by_label("Base DN", exact=True).fill(BASE_DN)
        self.page.get_by_label("Page size", exact=True).fill(str(page_size))
        self.page.get_by_label("Size limit", exact=True).fill(str(size_limit))
        if read_only or production:
            self.page.get_by_role("button", name="4 · Edit options", exact=True).click()
            if read_only:
                self.page.get_by_role(
                    "checkbox", name=re.compile("^Refused at the changeset boundary"),
                ).check()
            if production:
                self.page.get_by_role(
                    "checkbox", name=re.compile("^The status bar carries the tag"),
                ).check()
        return self.finish()

    def connect_saved(self, profile: dict) -> None:
        self.page.get_by_role("button", name="Connect", exact=True).click()
        expect(self.page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible(timeout=15000)
        assert self.rpc("ConnectionState", profile["id"])["state"] == "connected"

    def connect(self, **options) -> dict:
        profile = self.save(**options)
        self.connect_saved(profile)
        return profile

    def disconnect(self, profile: dict) -> None:
        self.navigate("Connections")
        self.select_profile()
        self.page.get_by_role("button", name="Disconnect", exact=True).click()
        expect(self.page.get_by_role("status")).to_contain_text("not connected")
        assert self.rpc("ConnectionState", profile["id"])["state"] == "disconnected"

    def open_browser(self) -> Locator:
        self.page.get_by_role("button", name="Browse", exact=True).click()
        tree = self.page.get_by_role("complementary", name="Directory tree", exact=True)
        expect(tree.get_by_role("button", name=BASE_DN, exact=True)).to_be_visible()
        return tree

    def browse(self, **options) -> Locator:
        self.connect(**options)
        return self.open_browser()

    def people(self, **options) -> Locator:
        tree = self.browse(**options)
        tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
        tree.get_by_role("button", name="ou=people", exact=True).dblclick()
        expect(tree.get_by_role("button", name="cn=jrivera", exact=True)).to_be_visible()
        return tree

    def read_person(self, rdn: str = "cn=jrivera", **options) -> Locator:
        tree = self.people(**options)
        tree.get_by_role("button", name=rdn, exact=True).click()
        expect(self.page.get_by_text(f"dn: {rdn},ou=people,{BASE_DN}", exact=True)).to_be_visible()
        return tree
