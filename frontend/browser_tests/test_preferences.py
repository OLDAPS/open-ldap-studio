import pytest
from playwright.sync_api import Page, expect

from .fixtures.app import AppFixture

PANES = [
    {"name": "Appearance", "control": "Interface font size", "value": "14 px"},
    {"name": "Browser & tree", "control": "Entries per page", "value": "100"},
    {"name": "Entry editor", "control": "Schema check", "value": "typing"},
    {"name": "Value editors", "control": "Unknown syntax", "value": "text"},
    {"name": "LDIF & text editors", "control": "Wrap lines at", "value": "78"},
    {"name": "Connections & timeouts", "control": "Connect timeout", "value": "10 s"},
    {"name": "Credentials & security", "control": "Password prompt", "value": "session"},
    {"name": "Keyboard shortcuts", "control": "Search commands", "value": ""},
    {"name": "Updates & about", "control": "Check for updates", "value": "never"},
]


@pytest.mark.parametrize("pane", PANES, ids=[pane["name"] for pane in PANES])
def test_pane_displays_its_controls_and_defaults(app: AppFixture, page: Page, pane: dict):
    """{pane[name]}: displays its controls and defaults"""
    app.open()
    app.navigate("Preferences")
    page.get_by_role("complementary", name="Preferences", exact=True).get_by_role(
        "button", name=pane["name"], exact=True
    ).click()
    expect(page.get_by_text(f"Preferences › {pane['name']}", exact=True)).to_be_visible()
    expect(page.get_by_label(pane["control"], exact=True)).to_have_value(pane["value"])
    expect(page.get_by_role("complementary", name="Live preview", exact=True)).to_be_visible()


def test_settings_search_is_case_insensitive_supports_no_matches_and_can_be_cleared(
    app: AppFixture, page: Page,
):
    """settings search is case insensitive, supports no matches and can be cleared"""
    app.open()
    app.navigate("Preferences")
    sidebar = page.get_by_role("complementary", name="Preferences", exact=True)
    settings_filter = page.get_by_role("searchbox", name="Search settings")
    settings_filter.fill("SECURITY")
    expect(sidebar.get_by_role("button")).to_have_count(1)
    sidebar.get_by_role("button", name="Credentials & security", exact=True).click()
    expect(page.get_by_label("Password prompt", exact=True)).to_have_value("session")
    settings_filter.fill("no-such-preference")
    expect(sidebar.get_by_text("no matching settings")).to_be_visible()
    settings_filter.clear()
    expect(sidebar.get_by_role("button")).to_have_count(len(PANES))


def test_theme_and_density_update_immediately_and_persist_while_navigating(app: AppFixture, page: Page):
    """theme and density update immediately and persist while navigating"""
    app.open()
    app.navigate("Preferences")
    root = page.locator("html")
    for label, theme in [("Light", "light"), ("High contrast", "high-contrast"), ("Dark", "dark")]:
        page.get_by_role("button", name=label, exact=True).click()
        expect(root).to_have_attribute("data-theme", theme)
    page.get_by_role("button", name="Comfortable", exact=True).click()
    expect(root).to_have_attribute("data-density", "comfortable")
    app.navigate("Searches")
    expect(root).to_have_attribute("data-density", "comfortable")
    expect(root).to_have_attribute("data-theme", "dark")
    app.navigate("Preferences")
    page.get_by_role("button", name="Compact", exact=True).click()
    expect(root).to_have_attribute("data-density", "compact")


def test_follow_system_respects_the_browser_color_preference(app: AppFixture, page: Page):
    """follow system respects the browser color preference"""
    page.emulate_media(color_scheme="light")
    app.open()
    app.navigate("Preferences")
    page.get_by_role("button", name="Follow system", exact=True).click()
    expect(page.locator("html")).to_have_attribute("data-theme", "light")


def test_browser_defaults_require_an_explicit_fetch_and_ldif_retains_line_ending_choices(
    app: AppFixture, page: Page,
):
    """browser defaults require an explicit fetch and LDIF retains line-ending choices"""
    app.open()
    app.navigate("Preferences")
    sidebar = page.get_by_role("complementary", name="Preferences", exact=True)
    sidebar.get_by_role("button", name="Browser & tree", exact=True).click()
    expect(page.get_by_role("checkbox", name="Fetch on scroll", exact=True)).not_to_be_checked()
    expect(page.get_by_label("Size limit", exact=True)).to_have_value("1000")
    sidebar.get_by_role("button", name="LDIF & text editors", exact=True).click()
    expect(page.get_by_label("Fold base64 at", exact=True)).to_have_value("76")
    expect(page.get_by_label("Line ending", exact=True)).to_have_value("lf")
    page.get_by_label("Line ending", exact=True).select_option("crlf")
    expect(page.get_by_label("Line ending", exact=True)).to_have_value("crlf")


def test_keyboard_command_search_shows_bindings_and_an_empty_result(app: AppFixture, page: Page):
    """keyboard command search shows bindings and an empty result"""
    app.open()
    app.navigate("Preferences")
    page.get_by_role("complementary", name="Preferences", exact=True).get_by_role(
        "button", name="Keyboard shortcuts", exact=True
    ).click()
    expect(page.get_by_text("New connection", exact=True)).to_be_visible()
    expect(page.get_by_text("Ctrl+N", exact=True)).to_be_visible()
    command_filter = page.get_by_role("searchbox", name="Search commands")
    command_filter.fill("REFRESH")
    expect(page.get_by_text("Refresh entry", exact=True)).to_be_visible()
    expect(page.get_by_text("F5", exact=True)).to_be_visible()
    expect(page.get_by_text("New connection", exact=True)).to_have_count(0)
    command_filter.fill("not-a-command")
    expect(page.get_by_text("No command matches that search.")).to_be_visible()
    command_filter.clear()
    expect(page.get_by_text("New connection", exact=True)).to_be_visible()


def test_update_checks_default_to_off_and_the_about_pane_identifies_a_development_build(
    app: AppFixture, page: Page,
):
    """update checks default to off and the about pane identifies a development build"""
    app.open()
    app.navigate("Preferences")
    page.get_by_role("complementary", name="Preferences", exact=True).get_by_role(
        "button", name="Updates & about", exact=True
    ).click()
    expect(page.get_by_label("Check for updates", exact=True)).to_have_value("never")
    expect(page.get_by_label("Proxy", exact=True)).to_have_value("system")
    expect(page.get_by_text("development build", exact=True)).to_be_visible()
