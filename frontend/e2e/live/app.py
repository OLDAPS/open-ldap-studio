"""Visible UI actions and read-only inspection of the real Wails application."""

import json
import re
from pathlib import Path

from playwright.sync_api import Locator, Page, expect

from .runtime import ADMIN_DN, ADMIN_PASSWORD, BASE_DN, LDAPService, NativeApp


class LiveApp:
    def __init__(self, page: Page, native: NativeApp, ldap: LDAPService, output_dir: Path) -> None:
        self.page = page
        self.native = native
        self.ldap = ldap
        self.output_path = output_dir
        self.screenshots: list[tuple[str, bytes]] = []
        self.bridge_calls: list[dict] | None = None
        self.profile_name = "Local LDAP E2E"

    def open(self) -> None:
        self.page.goto(self.native.url)
        expect(self.page.get_by_role("navigation", name="Perspectives")).to_be_visible()
        # This calls the real Go method over Wails' websocket. Startup checks
        # also prove no synthetic fixture has been installed in this page.
        info = self.rpc("GetAppInfo")
        assert info["platform"] == "linux"
        assert self.page.evaluate("() => !('__testBridge' in window)")
        expect(self.page.get_by_text("no connections yet — use + to add one")).to_be_visible()

    def rpc(self, method: str, *args: object) -> object:
        """Inspect the real backend via its generated Wails binding."""
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

    def network(self, encryption: str = "none", verify: bool = True, port: int | None = None) -> None:
        self.page.get_by_role("button", name="New connection", exact=True).last.click()
        self.page.get_by_label("Connection name", exact=True).fill(self.profile_name)
        self.page.get_by_label("Hostname", exact=True).fill("127.0.0.1")
        self.page.get_by_label("Encryption", exact=True).select_option(encryption)
        actual_port = port or self.ldap.ports["ldaps" if encryption == "ldaps" else "ldap"]
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
                self.page.get_by_role("checkbox").nth(0).check()
            if production:
                self.page.get_by_role("checkbox").nth(1).check()
        self.page.get_by_role("button", name="Finish", exact=True).click()
        expect(self.page.get_by_role("button", name="Connect", exact=True)).to_be_visible()
        profiles = self.rpc("ListProfiles")
        assert len(profiles) == 1
        return self.rpc("GetProfile", profiles[0]["id"])

    def connect(self, **options) -> dict:
        profile = self.save(**options)
        self.page.get_by_role("button", name="Connect", exact=True).click()
        expect(self.page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible(timeout=15000)
        assert self.rpc("ConnectionState", profile["id"])["state"] == "connected"
        return profile

    def browse(self, **options) -> Locator:
        self.connect(**options)
        self.page.get_by_role("button", name="Browse", exact=True).click()
        tree = self.page.get_by_role("complementary", name="Directory tree", exact=True)
        expect(tree.get_by_role("button", name=BASE_DN, exact=True)).to_be_visible()
        return tree

    def people(self, **options) -> Locator:
        tree = self.browse(**options)
        tree.get_by_role("button", name=BASE_DN, exact=True).dblclick()
        tree.get_by_role("button", name="ou=people", exact=True).dblclick()
        expect(tree.get_by_role("button", name="cn=jrivera", exact=True)).to_be_visible()
        return tree

    def read_person(self, rdn: str = "cn=jrivera", **options) -> Locator:
        tree = self.people(**options)
        tree.get_by_role("button", name=rdn, exact=True).click()
        expect(self.page.get_by_role("img", name="Entry photo", exact=True)).to_be_visible()
        return tree

    def screenshot(self, name: str) -> None:
        self.output_path.mkdir(parents=True, exist_ok=True)
        filename = re.sub(r"[^a-z0-9]+", "-", name.lower()).strip("-") + ".png"
        screenshot = self.page.screenshot(path=self.output_path / filename, full_page=True)
        self.screenshots.append((name, screenshot))

    def capture_calls(self) -> None:
        # Shared report hook compatibility. Never wrap the bridge or record
        # calls: the real native process log is already in the trace directory.
        self.output_path.mkdir(parents=True, exist_ok=True)
        (self.output_path / "profiles.json").write_text(
            json.dumps(self.native.profiles(), indent=2) + "\n", encoding="utf-8",
        )
