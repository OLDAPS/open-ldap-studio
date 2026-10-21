from playwright.sync_api import Page, expect

from .fixtures.app import AppFixture
from .fixtures.data import CONNECTED, PROFILE

JOB = {
    "jobId": "job-browser-test",
    "kind": "Directory export",
    "mode": "dryRun",
    "total": 10,
    "profileId": PROFILE["id"],
}


def test_a_background_job_shows_progress_and_a_successful_completion(app: AppFixture, page: Page):
    """a background job shows progress and a successful completion"""
    app.open()
    panel = page.get_by_role("region", name="Progress and logs")
    app.emit("job:started", JOB)
    expect(panel.get_by_role("cell", name="Directory export dry run")).to_be_visible()
    expect(
        page.get_by_role("status").get_by_role("button", name="1 running", exact=True)
    ).to_be_visible()
    app.emit(
        "job:progress",
        {"jobId": JOB["jobId"], "done": 5, "total": 10, "message": "5 entries processed"},
    )
    expect(panel.get_by_role("progressbar", name="Directory export progress")).to_have_attribute(
        "aria-valuenow", "50"
    )
    expect(panel.get_by_text("5 entries processed", exact=True)).to_be_visible()
    app.emit(
        "job:finished", {"jobId": JOB["jobId"], "state": "succeeded", "summary": "Export complete"}
    )
    expect(panel.get_by_text("succeeded", exact=True)).to_be_visible()
    expect(panel.get_by_role("button", name="Cancel", exact=True)).to_have_count(0)
    expect(
        page.get_by_role("status").get_by_role("button", name="1 running", exact=True)
    ).to_have_count(0)


def test_a_running_job_can_be_cancelled_from_the_progress_panel(app: AppFixture, page: Page):
    """a running job can be cancelled from the progress panel"""
    app.open()
    app.emit("job:started", JOB)
    panel = page.get_by_role("region", name="Progress and logs")
    panel.get_by_role("button", name="Cancel", exact=True).click()
    expect(panel.get_by_text("cancelled", exact=True)).to_be_visible()
    assert app.calls("Cancel")[0]["args"] == [JOB["jobId"]]


def test_a_failed_background_job_is_recorded_with_its_diagnostic_in_errors(app: AppFixture, page: Page):
    """a failed background job is recorded with its diagnostic in Errors"""
    app.open()
    app.emit("job:started", JOB)
    app.emit(
        "job:finished",
        {
            "jobId": JOB["jobId"],
            "state": "failed",
            "summary": "Directory export failed",
            "error": "LDAP result 4: sizeLimitExceeded",
        },
    )
    panel = page.get_by_role("region", name="Progress and logs")
    panel.get_by_role("tab", name="Errors 1", exact=True).click()
    expect(panel.get_by_text("Directory export failed", exact=True)).to_be_visible()
    expect(panel.get_by_text("LDAP result 4: sizeLimitExceeded", exact=True)).to_be_visible()


def test_capability_and_credential_store_notices_retain_their_reasons_in_console(
    app: AppFixture, page: Page,
):
    """capability and credential-store notices retain their reasons in Console"""
    app.open()
    capability = {
        "profileId": PROFILE["id"],
        "capability": "schema",
        "reason": "The server does not publish a subschema subentry",
    }
    app.emit("capability:unavailable", capability)
    app.emit("capability:unavailable", capability)
    app.emit("credential:storeUnavailable", {"reason": "No Secret Service in this browser test"})
    panel = page.get_by_role("region", name="Progress and logs")
    panel.get_by_role("tab", name="Console", exact=True).click()
    expect(panel.get_by_text(capability["reason"], exact=True)).to_have_count(1)
    expect(panel).to_contain_text(
        "No platform credential store is available; secrets are held for this session only."
    )
    expect(panel).to_contain_text("No Secret Service in this browser test")


def test_connection_loss_is_visible_in_status_and_errors_then_reconnect_updates_the_badges(
    app: AppFixture, page: Page,
):
    """connection loss is visible in status and Errors, then reconnect updates the badges"""
    app.open({"connections": [CONNECTED]})
    app.emit("conn:state", {**CONNECTED, "state": "lost"})
    app.emit("conn:lost", {"profileId": PROFILE["id"], "reason": "Test network interrupted"})
    expect(page.get_by_role("status")).to_contain_text("connection lost")
    panel = page.get_by_role("region", name="Progress and logs")
    panel.get_by_role("tab", name="Errors 1", exact=True).click()
    expect(panel).to_contain_text(f"Connection lost: {PROFILE['id']}")
    expect(panel).to_contain_text("Test network interrupted")
    app.emit("conn:state", {**CONNECTED, "writesRequireConfirmation": True})
    expect(page.get_by_role("status")).to_contain_text("reconnected")
    expect(page.get_by_role("status")).not_to_contain_text("connection lost")
