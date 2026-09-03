#!/usr/bin/env python3
"""
Regenerate the seed LDIF in seed/.

The output is committed, so `docker compose up` needs nothing but Docker. This
script exists so the fixture is reproducible and tunable rather than a wall of
hand-written LDIF nobody dares change: adjust BULK_USERS, re-run, commit.

    python3 make-seed.py

What the fixture is for: every screen in the app should have something real to
show. That means the directory deliberately contains awkward data as well as
tidy data — non-ASCII names, a value that must be base64-encoded on export, a
description past the truncation threshold, multi-valued attributes past the
fold threshold, binary photos, DN-valued references, and enough children under
one container to force paging.
"""

import base64
import hashlib
import io
import os
from pathlib import Path

SEED = Path(__file__).parent / 'seed'
BASE_DN = 'dc=example,dc=org'

# Enough to exceed a 100-entry page twice over, so the tree's explicit
# "fetch next 100 of N…" node is reachable rather than theoretical.
BULK_USERS = 200


def ssha(password: str) -> str:
    """An {SSHA} hash, so the password editor has a real scheme to report."""
    salt = os.urandom(8)
    digest = hashlib.sha1(password.encode() + salt).digest()
    return '{SSHA}' + base64.b64encode(digest + salt).decode()


def photo(rgb: tuple) -> bytes:
    """
    A small JPEG for the entry-info photo panel and the image editor.

    Returned as raw bytes, not base64 text. LDIF distinguishes the two with the
    number of colons — `attr: value` is a literal string, `attr:: value` is
    base64 — and getting that wrong stores the base64 *as* the value, so the
    directory ends up holding a 1 KB ASCII string where a JPEG should be.
    """
    from PIL import Image

    buffer = io.BytesIO()
    Image.new('RGB', (96, 96), rgb).save(buffer, format='JPEG', quality=70)
    return buffer.getvalue()


def fold(line: str, width: int = 76) -> str:
    """
    Fold one LDIF line at an exact column, RFC 2849 style.

    Deliberately not a word wrap. LDIF unfolding strips exactly one space from
    each continuation line, so a wrap that breaks on a space and keeps it would
    put that space back into the value — the fixture would then disagree with
    the parser it exists to test. Folding on a fixed count is lossless because
    unfolding is its exact inverse.

    Everything folded here is ASCII (non-ASCII values are base64 first), so a
    character boundary is a byte boundary and no UTF-8 sequence can be split.
    """
    if len(line) <= width:
        return line
    chunks = [line[:width]]
    rest = line[width:]
    while rest:
        chunks.append(rest[: width - 1])
        rest = rest[width - 1 :]
    return '\n '.join(chunks)


def attr(name: str, value) -> str:
    """
    One LDIF attribute line.

    Bytes are always written base64 with '::'. A str that leads with a space,
    or carries a character outside ASCII, is written base64 too — that is the
    LDIF rule, and getting it wrong here would mean the fixture disagrees with
    the parser it exists to test.
    """
    if isinstance(value, (bytes, bytearray)):
        line = f'{name}:: ' + base64.b64encode(value).decode()
    elif value.startswith((' ', ':', '<')):
        line = f'{name}:: ' + base64.b64encode(value.encode()).decode()
    else:
        try:
            value.encode('ascii')
            line = f'{name}: {value}'
        except UnicodeEncodeError:
            line = f'{name}:: ' + base64.b64encode(value.encode()).decode()
    return fold(line)


def entry(dn: str, *pairs) -> str:
    lines = [f'dn: {dn}']
    for name, value in pairs:
        lines.append(attr(name, value))
    return '\n'.join(lines) + '\n'


def write(name: str, blocks: list) -> None:
    # Trailing blank line included: entries are separated by an empty line, so
    # a file that stops right after its last attribute merges that entry into
    # whatever is appended next. slapd applies these files one at a time and
    # would never notice, but `cat seed/*.ldif | ldapadd` is an obvious thing to
    # do and it must not silently produce one corrupt entry.
    path = SEED / name
    path.write_text('\n'.join(blocks) + '\n')
    print(f'{path.relative_to(Path(__file__).parent)}  ({len(blocks)} entries)')


# --------------------------------------------------------------------------
# 00 — the base and the containers
# --------------------------------------------------------------------------

structure = [
    entry(
        BASE_DN,
        ('objectClass', 'top'),
        ('objectClass', 'dcObject'),
        ('objectClass', 'organization'),
        ('dc', 'example'),
        ('o', 'Example Organisation'),
        ('description', 'Development fixture for Open LDAP Studio'),
    ),
    entry(
        f'ou=people,{BASE_DN}',
        ('objectClass', 'top'),
        ('objectClass', 'organizationalUnit'),
        ('ou', 'people'),
        ('description', 'Staff accounts. Large enough to page.'),
    ),
    entry(
        f'ou=groups,{BASE_DN}',
        ('objectClass', 'top'),
        ('objectClass', 'organizationalUnit'),
        ('ou', 'groups'),
    ),
    entry(
        f'ou=services,{BASE_DN}',
        ('objectClass', 'top'),
        ('objectClass', 'organizationalUnit'),
        ('ou', 'services'),
        ('description', 'Non-human accounts'),
    ),
    # Deliberately empty: a target for copy, move and subtree delete, so those
    # flows can be exercised without damaging anything that matters.
    entry(
        f'ou=archive,{BASE_DN}',
        ('objectClass', 'top'),
        ('objectClass', 'organizationalUnit'),
        ('ou', 'archive'),
        ('description', 'Empty on purpose — a safe target for copy / move / delete'),
    ),
]

# --------------------------------------------------------------------------
# 01 — named people, each carrying something a screen needs
# --------------------------------------------------------------------------

LONG_DESCRIPTION = (
    'Primary contact for the directory migration project, reachable through the '
    'platform on-call rota; this description runs past the 120-character '
    'truncation threshold on purpose so the entry editor has something to '
    'truncate and the value editor has something to open.'
)

PEOPLE = [
    {
        'uid': 'jrivera',
        'sn': 'Rivera',
        'given': 'Jordan',
        'display': 'Jordan Rivera',
        'title': 'Platform Engineer',
        'dept': '4100',
        'employee': '10021',
        'phone': '+44 20 7946 0021',
        'mail': ['j.rivera@example.org', 'jordan.rivera@example.org'],
        'photo': (124, 58, 237),
        'uidn': 10021,
        'extra': [('description', LONG_DESCRIPTION), ('labeledURI', 'https://wiki.example.org/~jrivera')],
    },
    {
        'uid': 'lchen',
        'sn': 'Chen',
        'given': 'Li',
        'display': 'Li Chen',
        'title': 'Directory Administrator',
        'dept': '4100',
        'employee': '10044',
        'phone': '+44 20 7946 0044',
        'mail': ['l.chen@example.org'],
        'photo': (74, 222, 128),
        'uidn': 10044,
        # A value that must be base64-encoded on export because it leads with a
        # space. If a round-trip through the app loses that space, the fixture
        # will show it.
        'extra': [('description', ' leading space — must round-trip as base64')],
    },
    {
        'uid': 'mokafor',
        'sn': 'Okafor',
        'given': 'Mary',
        'display': 'Mary Okafor',
        'title': 'Service Desk Lead',
        'dept': '2200',
        'employee': '10077',
        'phone': '+44 20 7946 0077',
        'mail': ['m.okafor@example.org'],
        'photo': (250, 204, 21),
        'uidn': 10077,
        'extra': [],
    },
    {
        # Non-ASCII throughout: name, surname and description. Exercises UTF-8
        # handling in the tree label, the grid and LDIF export.
        'uid': 'tvasquez',
        'sn': 'Vásquez',
        'given': 'Tomás',
        'display': 'Tomás Vásquez',
        'title': 'Ingeniero de Redes',
        'dept': '3300',
        'employee': '10093',
        'phone': '+34 91 123 4567',
        'mail': ['t.vasquez@example.org'],
        'photo': (248, 113, 113),
        'uidn': 10093,
        'extra': [('description', 'Trabaja desde Madrid — acentos: áéíóú ñ ¿?')],
    },
    {
        'uid': 'aschmidt',
        'sn': 'Schmidt',
        'given': 'Anna',
        'display': 'Anna Schmidt',
        'title': 'Security Engineer',
        'dept': '4100',
        'employee': '10104',
        'phone': '+49 30 123456',
        'mail': [
            # Four values, past the default fold threshold of three, so the
            # editor's "+N more" is reachable.
            'a.schmidt@example.org',
            'anna.schmidt@example.org',
            'schmidt@example.org',
            'security@example.org',
        ],
        'photo': (167, 139, 250),
        'uidn': 10104,
        'extra': [],
    },
]

people = []
for person in PEOPLE:
    pairs = [
        ('objectClass', 'top'),
        ('objectClass', 'person'),
        ('objectClass', 'organizationalPerson'),
        ('objectClass', 'inetOrgPerson'),
        # An auxiliary class, so the object-class tab and the entry-info panel
        # have a structural/auxiliary split to show.
        ('objectClass', 'posixAccount'),
        ('cn', person['uid']),
        ('sn', person['sn']),
        ('givenName', person['given']),
        ('displayName', person['display']),
        ('uid', person['uid']),
        ('title', person['title']),
        ('departmentNumber', person['dept']),
        ('employeeNumber', person['employee']),
        ('telephoneNumber', person['phone']),
        ('o', 'Example Organisation'),
        ('l', 'London'),
    ]
    pairs += [('mail', address) for address in person['mail']]
    pairs += [
        ('uidNumber', str(person['uidn'])),
        ('gidNumber', '5000'),
        ('homeDirectory', f"/home/{person['uid']}"),
        ('loginShell', '/bin/bash'),
        ('userPassword', ssha('password')),
        ('jpegPhoto', photo(person['photo'])),
    ]
    pairs += person['extra']

    # A DN-valued attribute, for the DN picker and for "manager" chasing.
    if person['uid'] != 'jrivera':
        pairs.append(('manager', f'cn=jrivera,ou=people,{BASE_DN}'))

    people.append(entry(f"cn={person['uid']},ou=people,{BASE_DN}", *pairs))

# --------------------------------------------------------------------------
# 02 — groups, with DN-valued members
# --------------------------------------------------------------------------

def group(name: str, description: str, members: list) -> str:
    pairs = [
        ('objectClass', 'top'),
        ('objectClass', 'groupOfNames'),
        ('cn', name),
        ('description', description),
    ]
    pairs += [('member', f'cn={uid},ou=people,{BASE_DN}') for uid in members]
    return entry(f'cn={name},ou=groups,{BASE_DN}', *pairs)


groups = [
    group('platform', 'Platform engineering', ['jrivera', 'lchen', 'aschmidt']),
    group('helpdesk', 'Service desk', ['mokafor']),
    group('network', 'Network engineering', ['tvasquez']),
    # groupOfNames requires at least one member, so an "empty" group holds the
    # conventional placeholder rather than being invalid.
    group('everyone', 'All staff', [p['uid'] for p in PEOPLE]),
]

# --------------------------------------------------------------------------
# 03 — service accounts
# --------------------------------------------------------------------------

services = [
    entry(
        f'cn=readonly-svc,ou=services,{BASE_DN}',
        ('objectClass', 'top'),
        ('objectClass', 'person'),
        ('objectClass', 'organizationalPerson'),
        ('objectClass', 'inetOrgPerson'),
        ('cn', 'readonly-svc'),
        ('sn', 'Read Only Service'),
        ('description', 'Bind account for read-only connections'),
        ('userPassword', ssha('readonly')),
    ),
    entry(
        f'cn=replication,ou=services,{BASE_DN}',
        ('objectClass', 'top'),
        ('objectClass', 'person'),
        ('objectClass', 'organizationalPerson'),
        ('objectClass', 'inetOrgPerson'),
        ('cn', 'replication'),
        ('sn', 'Replication Agent'),
        ('description', 'Used by the replication topology'),
        ('userPassword', ssha('replication')),
    ),
]

# --------------------------------------------------------------------------
# 04 — bulk filler, so paging is real
# --------------------------------------------------------------------------

FIRST = ['Alex', 'Sam', 'Robin', 'Casey', 'Jamie', 'Morgan', 'Riley', 'Quinn', 'Avery', 'Rowan']
LAST = ['Ahmed', 'Baker', 'Costa', 'Dubois', 'Eriksen', 'Fischer', 'Gupta', 'Hansen', 'Iversen', 'Jensen']

bulk = []
for index in range(BULK_USERS):
    given = FIRST[index % len(FIRST)]
    surname = LAST[(index // len(FIRST)) % len(LAST)]
    uid = f'{given[0].lower()}{surname.lower()}{index:03d}'
    bulk.append(
        entry(
            f'cn={uid},ou=people,{BASE_DN}',
            ('objectClass', 'top'),
            ('objectClass', 'person'),
            ('objectClass', 'organizationalPerson'),
            ('objectClass', 'inetOrgPerson'),
            ('cn', uid),
            ('sn', surname),
            ('givenName', given),
            ('displayName', f'{given} {surname}'),
            ('uid', uid),
            ('mail', f'{uid}@example.org'),
            ('title', 'Staff'),
            ('departmentNumber', str(1000 + (index % 9) * 100)),
            ('employeeNumber', str(20000 + index)),
            ('userPassword', ssha('password')),
        )
    )

if __name__ == '__main__':
    SEED.mkdir(exist_ok=True)
    write('00-structure.ldif', structure)
    write('01-people.ldif', people)
    write('02-groups.ldif', groups)
    write('03-services.ldif', services)
    write('04-bulk.ldif', bulk)
    total = len(structure) + len(people) + len(groups) + len(services) + len(bulk)
    print(f'\n{total} entries total')
