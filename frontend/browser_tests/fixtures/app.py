"""Page helpers for the production frontend and its synthetic backend."""

import json
import re
from copy import deepcopy
from pathlib import Path

from playwright.sync_api import Error, Page, expect

from .bridge import INSTALL_BRIDGE
from .data import PROFILE, default_seed


class AppFixture:
    def __init__(self, page: Page, output_path: str) -> None:
        self.page = page
        self.output_path = Path(output_path)
        self.screenshots: list[tuple[str, bytes]] = []
        self.bridge_calls: list[dict] | None = None

    def open(self, seed: dict | None = None) -> None:
        data = {**default_seed(), **deepcopy(seed or {})}
        self.page.add_init_script(f"({INSTALL_BRIDGE})({json.dumps(data)})")
        self.page.goto("/")
        expect(self.page.get_by_role("navigation", name="Perspectives")).to_be_visible()
        self.wait_for_call("GetCommands")
        # Startup must finish before scenarios inject connection or job events.
        self.wait_for_call("ConnectionStates")

    def wait_for_call(self, method: str) -> None:
        self.page.wait_for_function(
            "method => window.__testBridge.calls.some(call => call.method === method)",
            arg=method,
            timeout=5000,
        )

    def navigate(self, name: str) -> None:
        button = self.page.get_by_role("navigation", name="Perspectives").get_by_role(
            "button", name=name, exact=True
        )
        button.click()
        expect(button).to_have_attribute("aria-current", "page")

    def connect(self) -> None:
        self.page.get_by_role("complementary", name="Connections", exact=True).get_by_role(
            "button", name=PROFILE["name"], exact=True
        ).click()
        self.page.get_by_role("button", name="Connect", exact=True).click()
        expect(self.page.get_by_role("button", name="Disconnect", exact=True)).to_be_visible()

    def browse(self) -> None:
        self.connect()
        self.page.get_by_role("button", name="Browse", exact=True).click()
        expect(self.page.get_by_role("complementary", name="Directory tree")).to_be_visible()

    def calls(self, method: str) -> list[dict]:
        return self.page.evaluate(
            "method => window.__testBridge.calls.filter(call => call.method === method)", method
        )

    def fail_next(self, method: str, message: str) -> None:
        self.page.evaluate(
            """({ method, message }) => {
                (window.__testBridge.failures[method] ??= []).push(message);
            }""",
            {"method": method, "message": message},
        )

    def respond_next(self, method: str, response: object) -> None:
        self.page.evaluate(
            """({ method, response }) => {
                (window.__testBridge.responses[method] ??= []).push(response);
            }""",
            {"method": method, "response": response},
        )

    def emit(self, event: str, payload: object) -> None:
        self.page.evaluate(
            "({ event, payload }) => window.__testBridge.emit(event, payload)",
            {"event": event, "payload": payload},
        )

    def screenshot(self, name: str) -> None:
        self.output_path.mkdir(parents=True, exist_ok=True)
        filename = re.sub(r"[^a-z0-9]+", "-", name.lower()).strip("-") + ".png"
        screenshot = self.page.screenshot(path=self.output_path / filename, full_page=True)
        self.screenshots.append((name, screenshot))

    def capture_calls(self) -> None:
        try:
            self.bridge_calls = self.page.evaluate("() => window.__testBridge?.calls ?? []")
        except Error:
            self.bridge_calls = []
        self.output_path.mkdir(parents=True, exist_ok=True)
        (self.output_path / "bridge-calls.json").write_text(
            json.dumps(self.bridge_calls, indent=2, ensure_ascii=False) + "\n", encoding="utf-8"
        )
