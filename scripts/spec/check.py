"""Offline consistency checks for a Spec Kit feature directory.

    python3 scripts/spec/check.py [--feature specs/001-open-ldap-studio] [--notes]

Exit status 1 if any error is found. Notes are things worth knowing that do not
fail the check. Nothing here touches the network or writes a file.

Standard library only.
"""

from __future__ import annotations

import argparse
import glob
import os
import re
import sys
from dataclasses import dataclass

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import specmodel  # noqa: E402

DOCS = [
    "spec.md",
    "plan.md",
    "research.md",
    "data-model.md",
    "quickstart.md",
    "tasks.md",
    "contracts/*.md",
    "checklists/*.md",
]

# FR-017, FR-006 – FR-009, FR-053–FR-056, SC-004 and so on.
REF_RE = re.compile(r"\b(FR|SC)-(\d{3})(?:\s*[–—-]\s*(?:(?:FR|SC)-)?(\d{3}))?")
US_RE = re.compile(r"\bUS(\d{1,2})\b")
TASK_RE = re.compile(r"^- \[([ xX])\] T(\d{3})\b")
MAX_RANGE = 60  # a longer "range" is prose such as "FR-010 – 100 ms", not a range


@dataclass
class Finding:
    level: str  # "error" | "note"
    where: str
    msg: str

    def __str__(self) -> str:
        return f"{self.level:5}  {self.where:24} {self.msg}"


@dataclass
class Feature:
    root: str
    docs: dict[str, str]

    @property
    def spec_text(self) -> str:
        return self.docs.get("spec.md", "")


def load(root: str) -> Feature:
    docs: dict[str, str] = {}
    for pattern in DOCS:
        for path in sorted(glob.glob(os.path.join(root, pattern))):
            with open(path, encoding="utf-8") as fh:
                docs[os.path.relpath(path, root)] = fh.read()
    return Feature(root, docs)


def expand_refs(line: str, kind: str) -> list[tuple[int, bool]]:
    """Ids of ``kind`` ("FR" or "SC") cited on a line, each with whether it is
    the end of a range. Ranges are expanded; implausible ranges are dropped."""
    out: list[tuple[int, bool]] = []
    for m in REF_RE.finditer(line):
        if m.group(1) != kind:
            continue
        a = int(m.group(2))
        b = int(m.group(3)) if m.group(3) else None
        if b is not None and a < b <= a + MAX_RANGE:
            out.extend((n, n == b) for n in range(a, b + 1))
        else:
            out.append((a, False))
    return out


def check_spec(spec: specmodel.Spec, text: str, add) -> None:
    if specmodel.render(spec) != text:
        add("error", "spec.md", "the model does not reproduce the file byte for byte; the parser is out of date")

    nums = [c.key for c in spec.stories]
    if not nums:
        add("error", "spec.md", "no user stories found")
    elif nums != list(range(1, len(nums) + 1)):
        add("error", "spec.md", f"user stories are not numbered 1..{len(nums)} in order: {nums}")
    for c in spec.stories:
        if not c.meta.get("title"):
            add("error", f"spec.md US{c.key}", "story has no title")

    for kind, chunks, label in (("fr", spec.frs, "FR"), ("sc", spec.scs, "SC")):
        ids = [c.key for c in chunks]
        dup = sorted({i for i in ids if ids.count(i) > 1})
        if dup:
            add("error", "spec.md", f"duplicate {label} ids: {', '.join(f'{label}-{d:03d}' for d in dup)}")
        if ids and sorted(set(ids)) != list(range(1, max(ids) + 1)):
            missing = sorted(set(range(1, max(ids) + 1)) - set(ids))
            add("error", "spec.md", f"{label} ids have gaps: missing {', '.join(f'{label}-{m:03d}' for m in missing)}")
        if not ids:
            add("error", "spec.md", f"no {label} found")

    for c in spec.frs:
        if not c.meta.get("section"):
            add("error", f"spec.md FR-{c.key:03d}", "requirement is not under a ### topic heading")

    if not spec.edges:
        add("error", "spec.md", "no edge cases found")
    for c in spec.edges:
        if not c.meta.get("group"):
            add("error", f"spec.md edge {c.key}", "edge case is not under a group heading")


def check_refs(feature: Feature, spec: specmodel.Spec, add) -> None:
    fr_defined = {c.key for c in spec.frs}
    sc_defined = {c.key for c in spec.scs}
    story_nums = {c.key for c in spec.stories}
    for name, text in feature.docs.items():
        for lineno, line in enumerate(text.splitlines(), 1):
            for kind, defined, label in (("FR", fr_defined, "FR"), ("SC", sc_defined, "SC")):
                for n, _ in expand_refs(line, kind):
                    if n not in defined:
                        add("error", f"{name}:{lineno}", f"cites {label}-{n:03d}, which spec.md does not define")
            if name != "spec.md":
                for m in US_RE.finditer(line):
                    if int(m.group(1)) not in story_nums:
                        add("error", f"{name}:{lineno}", f"cites US{m.group(1)}, but spec.md has no such story")


def check_tasks(tasks_text: str, story_nums: set[int], add) -> list[int]:
    ids: list[int] = []
    for lineno, line in enumerate(tasks_text.splitlines(), 1):
        m = TASK_RE.match(line)
        if not m:
            continue
        ids.append(int(m.group(2)))
    dup = sorted({i for i in ids if ids.count(i) > 1})
    if dup:
        add("error", "tasks.md", f"duplicate task ids: {', '.join(f'T{d:03d}' for d in dup)}")
    if ids and sorted(set(ids)) != list(range(1, max(ids) + 1)):
        missing = sorted(set(range(1, max(ids) + 1)) - set(ids))
        add("error", "tasks.md", f"task ids have gaps: missing {', '.join(f'T{m:03d}' for m in missing)}")
    if not ids:
        add("error", "tasks.md", "no tasks found")
    return ids


def screens_map(screens_text: str) -> dict[int, set[int]]:
    """FR number -> the stories contracts/screens.md places it in. Tables are
    read by header name because they do not all have the same columns."""
    out: dict[int, set[int]] = {}
    header: list[str] | None = None
    for line in screens_text.splitlines():
        if not line.startswith("|"):
            header = None
            continue
        cells = [c.strip() for c in re.split(r"(?<!\\)\|", line.strip().strip("|"))]
        if cells and cells[0] in ("ID", "Element"):
            header = [c.lower() for c in cells]
            continue
        if header is None or set("".join(cells)) <= set("-: "):
            continue
        row = dict(zip(header, cells))
        stories = {int(s) for s in re.findall(r"US(\d+)", row.get("story", ""))}
        reqs = row.get("requirements") or row.get("requirement") or ""
        for n, _ in expand_refs(reqs, "FR"):
            out.setdefault(n, set()).update(stories)
    return out


def run(root: str) -> tuple[list[Finding], dict]:
    findings: list[Finding] = []

    def add(level: str, where: str, msg: str) -> None:
        findings.append(Finding(level, where, msg))

    feature = load(root)
    if "spec.md" not in feature.docs:
        add("error", root, "spec.md not found")
        return findings, {}
    spec = specmodel.parse(feature.spec_text)
    check_spec(spec, feature.spec_text, add)
    check_refs(feature, spec, add)
    task_ids = check_tasks(feature.docs.get("tasks.md", ""), {c.key for c in spec.stories}, add)

    fr_ids = sorted(c.key for c in spec.frs)
    mapped = {n for n, s in screens_map(feature.docs.get("contracts/screens.md", "")).items() if s}
    unmapped = [n for n in fr_ids if n not in mapped]
    if unmapped:
        add(
            "note",
            "contracts/screens.md",
            f"{len(unmapped)} of {len(fr_ids)} requirements are not tied to a story by any screen: "
            + ", ".join(f"FR-{n:03d}" for n in unmapped),
        )

    stats = {
        "stories": len(spec.stories),
        "requirements": len(fr_ids),
        "success criteria": len(spec.scs),
        "edge cases": len(spec.edges),
        "tasks": len(task_ids),
        "documents": len(feature.docs),
    }
    return findings, stats


def main(argv: list[str] | None = None) -> int:
    here = os.path.dirname(os.path.abspath(__file__))
    default = os.path.join(here, "..", "..", "specs", "001-open-ldap-studio")
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--feature", default=default, help="feature directory (default: specs/001-open-ldap-studio)")
    ap.add_argument("--notes", action="store_true", help="also print notes")
    args = ap.parse_args(argv)

    findings, stats = run(os.path.normpath(args.feature))
    errors = [f for f in findings if f.level == "error"]
    notes = [f for f in findings if f.level == "note"]
    for f in errors:
        print(f)
    if args.notes:
        for f in notes:
            print(f)
    summary = ", ".join(f"{v} {k}" for k, v in stats.items())
    print(f"spec check: {len(errors)} error(s), {len(notes)} note(s) [{summary}]")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
