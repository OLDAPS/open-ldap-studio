"""Real writes through Wails IPC, verified with an independent LDAP client.

Entry editing dialogs are not wired into the UI yet. These tests use the
application's actual bound Preview/Commit API after connecting via the UI.
"""

from base64 import b64decode, b64encode

import pytest
from playwright.sync_api import Error, Page, expect

from e2e.live.app import LiveApp
from e2e.live.runtime import LDAPService


def attribute(name: str, *values: str) -> dict:
    return {"type": name, "values": [b64encode(value.encode()).decode() for value in values]}


def modification(profile: dict, dn: str, value: str) -> dict:
    return {"profileId": profile["id"], "kind": "modify", "ops": [
        {"dn": dn, "type": "replaceAttr", "attribute": attribute("description", value)},
    ]}


def test_preview_is_read_only_and_discarded_tokens_cannot_write(
    app: LiveApp, ldap_service: LDAPService, ldap_branch: str,
):
    profile = app.connect()
    dn = f"cn=subject,{ldap_branch}"
    preview = app.rpc("Preview", modification(profile, dn, "Should not be written"), False)
    assert preview["changeSet"]["affectedCount"] == 1
    operation = preview["changeSet"]["ops"][0]
    assert b64decode(operation["before"][0]) == b"Original LDAP value"
    assert b64decode(operation["after"][0]) == b"Should not be written"
    assert "description: Original LDAP value" in ldap_service.search(dn, attributes=("description",))
    token = preview["token"]["token"]
    app.rpc("Discard", token)
    with pytest.raises(Error, match="no such preview token"):
        app.rpc("Commit", token)
    assert "description: Original LDAP value" in ldap_service.search(dn, attributes=("description",))


def test_commit_modifies_real_ldap_and_consumes_the_preview_token(
    app: LiveApp, ldap_service: LDAPService, ldap_branch: str,
):
    profile = app.connect()
    dn = f"cn=subject,{ldap_branch}"
    preview = app.rpc("Preview", modification(profile, dn, "Committed over Wails IPC"), False)
    token = preview["token"]["token"]
    assert app.rpc("Commit", token)["code"] == 0
    assert "description: Committed over Wails IPC" in ldap_service.search(dn, attributes=("description",))
    with pytest.raises(Error, match="already been committed"):
        app.rpc("Commit", token)


def test_add_rename_move_and_delete_entry_in_real_ldap(
    app: LiveApp, ldap_service: LDAPService, ldap_branch: str,
):
    profile = app.connect()
    dn = f"cn=created,{ldap_branch}"
    attributes = [attribute("objectClass", "inetOrgPerson"), attribute("cn", "created"),
                  attribute("sn", "Created"), attribute("description", "Round trip")]

    def commit(kind: str, operation: dict) -> None:
        preview = app.rpc("Preview", {"profileId": profile["id"], "kind": kind, "ops": [operation]}, False)
        assert app.rpc("Commit", preview["token"]["token"])["code"] == 0

    commit("add", {"dn": dn, "type": "addEntry", "attributes": attributes})
    assert "cn: created" in ldap_service.search(dn, attributes=("cn",))
    commit("modRDN", {"dn": dn, "type": "rename", "newRdn": "cn=renamed", "keepOldRdn": False})
    dn = f"cn=renamed,{ldap_branch}"
    assert "cn: renamed" in ldap_service.search(dn, attributes=("cn",))
    target = f"ou=destination,{ldap_branch}"
    ldap_service.modify(f"dn: {target}\nchangetype: add\nobjectClass: organizationalUnit\nou: destination\n")
    commit("move", {"dn": dn, "type": "rename", "newRdn": "cn=renamed", "newSuperior": target,
                    "keepOldRdn": False})
    dn = f"cn=renamed,{target}"
    assert "cn: renamed" in ldap_service.search(dn, attributes=("cn",))
    commit("delete", {"dn": dn, "type": "deleteEntry"})
    assert "cn=renamed" not in ldap_service.search(target, scope="one")


def test_a_concurrent_ldap_write_invalidates_the_preview(
    app: LiveApp, ldap_service: LDAPService, ldap_branch: str,
):
    profile = app.connect()
    dn = f"cn=subject,{ldap_branch}"
    preview = app.rpc("Preview", modification(profile, dn, "Stale change"), False)
    ldap_service.modify(
        f"dn: {dn}\nchangetype: modify\nreplace: description\ndescription: Changed by another client\n",
    )
    with pytest.raises(Error, match="changed on the server since the preview"):
        app.rpc("Commit", preview["token"]["token"])
    assert "description: Changed by another client" in ldap_service.search(dn, attributes=("description",))


@pytest.mark.parametrize("options,error", [
    ({"read_only": True}, "open read-only"),
    ({"production": True}, "needs its extra confirmation"),
])
def test_write_guards_refuse_real_ldap_changes_below_the_ui(
    app: LiveApp, ldap_service: LDAPService, ldap_branch: str, options: dict, error: str,
):
    profile = app.connect(**options)
    dn = f"cn=subject,{ldap_branch}"
    with pytest.raises(Error, match=error):
        app.rpc("Preview", modification(profile, dn, "Forbidden write"), False)
    assert "description: Original LDAP value" in ldap_service.search(dn, attributes=("description",))


def test_production_confirmation_allows_the_confirmed_real_write(
    app: LiveApp, ldap_service: LDAPService, ldap_branch: str,
):
    profile = app.connect(production=True)
    dn = f"cn=subject,{ldap_branch}"
    preview = app.rpc("Preview", modification(profile, dn, "Confirmed production write"), True)
    assert preview["production"] is True
    assert app.rpc("Commit", preview["token"]["token"])["code"] == 0
    assert "description: Confirmed production write" in ldap_service.search(dn, attributes=("description",))


def test_slapd_rejects_a_schema_invalid_entry_without_creating_it(
    app: LiveApp, ldap_service: LDAPService, ldap_branch: str,
):
    profile = app.connect()
    dn = f"cn=invalid,{ldap_branch}"
    payload = {"profileId": profile["id"], "kind": "add", "ops": [
        {"dn": dn, "type": "addEntry", "attributes": [
            attribute("objectClass", "inetOrgPerson"), attribute("cn", "invalid"),
        ]},
    ]}
    preview = app.rpc("Preview", payload, False)
    with pytest.raises(Error, match="(?i)objectClassViolation|Object Class Violation|65"):
        app.rpc("Commit", preview["token"]["token"])
    assert "cn=invalid" not in ldap_service.search(ldap_branch, scope="one")


def test_multi_entry_commit_emits_real_job_progress_and_outcomes(
    app: LiveApp, page: Page, ldap_service: LDAPService, ldap_branch: str,
):
    profile = app.connect()
    payload = {"profileId": profile["id"], "kind": "bulkModify", "ops": [
        {"dn": dn, "type": "replaceAttr", "attribute": attribute("description", "Bulk commit")}
        for dn in (ldap_branch, f"cn=subject,{ldap_branch}")
    ]}
    preview = app.rpc("Preview", payload, False)
    job_id = app.rpc("CommitAsJob", preview["token"]["token"])
    panel = page.get_by_role("region", name="Progress and logs")
    panel.get_by_role("tab", name="Progress", exact=True).click()
    expect(panel.get_by_role("row").filter(has_text="commit")).to_contain_text("succeeded")
    assert app.rpc("JobState", job_id)["state"] == "succeeded"
    outcomes = app.rpc("JobOutcomes", job_id)
    assert len(outcomes) == 2
    assert all(outcome["status"] == "succeeded" for outcome in outcomes)
    for dn in (ldap_branch, f"cn=subject,{ldap_branch}"):
        assert "description: Bulk commit" in ldap_service.search(dn, attributes=("description",))
