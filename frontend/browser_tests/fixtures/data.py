"""Deterministic directory data shared by all browser scenarios."""

from base64 import b64encode
from copy import deepcopy

BASE_DN = "dc=example,dc=org"
PEOPLE_DN = f"ou=people,{BASE_DN}"
ALICE_DN = f"uid=alice,{PEOPLE_DN}"
BOB_DN = f"uid=bob,{PEOPLE_DN}"
CAROL_DN = f"uid=carol,{PEOPLE_DN}"

SUCCESS = {
    "code": 0,
    "matchedDn": "",
    "diagnosticMessage": "",
    "interpretation": "Success",
}

PROFILE = {
    "id": "directory-test",
    "name": "Example directory",
    "host": "ldap.example.test",
    "port": 636,
    "encryption": "ldaps",
    "tls": {"verifyCertificate": True, "verifyHostname": True},
    "bindMethod": "simple",
    "bindDn": f"cn=admin,{BASE_DN}",
    "readOnly": False,
    "tags": [],
    "baseDn": BASE_DN,
    "timeouts": {"connectMs": 10_000, "readMs": 30_000},
    "limits": {"pageSize": 2, "sizeLimit": 1000, "timeLimit": 10},
    "aliases": "find",
    "referrals": "ask",
    "schemaVersion": 1,
}

CONNECTED = {
    "profileId": PROFILE["id"],
    "state": "connected",
    "boundDn": PROFILE["bindDn"],
    "serverIdentity": f"{PROFILE['host']}:{PROFILE['port']}",
    "tlsVerified": True,
    "encrypted": True,
    "readOnly": False,
    "production": False,
    "writesRequireConfirmation": False,
}

ROOT_DSE = {
    "namingContexts": [BASE_DN],
    "supportedControl": ["1.2.840.113556.1.4.319"],
    "supportedExtension": [],
    "supportedSaslMechanisms": ["PLAIN"],
    "supportedLdapVersion": ["3"],
    "subschemaSubentry": "cn=subschema",
    "vendorName": "Browser test directory",
    "raw": {},
}


def attribute(
    name: str,
    values: list[str],
    is_operational: bool = False,
    options: list[str] | None = None,
) -> dict:
    result = {
        "type": name,
        "values": [b64encode(value.encode("utf-8")).decode("ascii") for value in values],
        "isOperational": is_operational,
    }
    if options is not None:
        result["options"] = options
    return result


ENTRIES = {
    BASE_DN: {
        "dn": BASE_DN,
        "hasChildren": "yes",
        "attributes": [attribute("objectClass", ["top", "domain"]), attribute("dc", ["example"])],
    },
    PEOPLE_DN: {
        "dn": PEOPLE_DN,
        "hasChildren": "yes",
        "attributes": [
            attribute("objectClass", ["top", "organizationalUnit"]),
            attribute("ou", ["people"]),
        ],
    },
    ALICE_DN: {
        "dn": ALICE_DN,
        "hasChildren": "no",
        "attributes": [
            attribute("objectClass", ["top", "person", "inetOrgPerson"]),
            attribute("uid", ["alice"]),
            attribute("cn", ["Alice Example"]),
            attribute("sn", ["Example"]),
            attribute(
                "mail",
                [
                    "alice@example.org",
                    "a.example@example.org",
                    "alice.work@example.org",
                    "alice.other@example.org",
                    "alice.last@example.org",
                ],
            ),
            attribute("description", ["Engenheira — diretório LDAP"]),
            attribute("description", ["Bonjour Alice"], options=["lang-fr"]),
            {"type": "jpegPhoto", "values": ["AAECAwQ="], "isOperational": False},
            attribute("modifyTimestamp", ["20261016123000Z"], True),
            attribute("createTimestamp", ["20260101090000Z"], True),
            attribute("modifiersName", [PROFILE["bindDn"]], True),
        ],
    },
    BOB_DN: {
        "dn": BOB_DN,
        "hasChildren": "no",
        "attributes": [
            attribute("objectClass", ["top", "inetOrgPerson"]),
            attribute("uid", ["bob"]),
            attribute("cn", ["Bob Example"]),
        ],
    },
    CAROL_DN: {
        "dn": CAROL_DN,
        "hasChildren": "no",
        "attributes": [
            attribute("objectClass", ["top", "inetOrgPerson"]),
            attribute("uid", ["carol"]),
            attribute("cn", ["Carol Example"]),
        ],
    },
}


def page_of(dns: list[str], extra: dict | None = None) -> dict:
    return deepcopy(
        {
            "entries": [ENTRIES[dn] for dn in dns],
            "loadedCount": len(dns),
            "serverLimit": 0,
            "truncatedByServer": False,
            "result": SUCCESS,
            **(extra or {}),
        }
    )


# This registry exercises menu enablement and shortcut filtering. Application
# features that are not wired to handlers remain unavailable.
COMMANDS = {
    "platform": "linux",
    "menus": ["File", "Edit", "Search", "LDAP", "Schema", "Credentials", "Preferences", "Help"],
    "commands": [
        {
            "id": "connection.new",
            "label": "New connection",
            "menu": "File",
            "order": 1,
            "scope": "global",
            "enablement": "always",
        },
        {
            "id": "entry.refresh",
            "label": "Refresh entry",
            "menu": "LDAP",
            "order": 2,
            "scope": "tree",
            "enablement": "connection.open",
        },
        {
            "id": "entry.delete",
            "label": "Delete entry",
            "menu": "LDAP",
            "order": 3,
            "scope": "tree",
            "enablement": "connection.open && !connection.readOnly",
            "destructive": True,
        },
    ],
    "bindings": {"connection.new": "Ctrl+N", "entry.refresh": "F5", "entry.delete": "Delete"},
    "tree": {
        "File": [{"label": "New connection", "commandId": "connection.new"}],
        "Edit": [],
        "Search": [],
        "LDAP": [
            {"label": "Refresh entry", "commandId": "entry.refresh"},
            {"label": "Delete entry", "commandId": "entry.delete", "destructive": True},
        ],
        "Schema": [],
        "Credentials": [],
        "Preferences": [],
        "Help": [],
    },
}


def default_seed() -> dict:
    return deepcopy(
        {
            "profiles": [PROFILE],
            "connections": [],
            "entries": ENTRIES,
            "rootDSE": ROOT_DSE,
            "children": {
                BASE_DN: [page_of([PEOPLE_DN])],
                PEOPLE_DN: [
                    page_of([ALICE_DN, BOB_DN], {"cookie": "page-2"}),
                    page_of([CAROL_DN], {"loadedCount": 3}),
                ],
            },
            "commands": COMMANDS,
        }
    )
