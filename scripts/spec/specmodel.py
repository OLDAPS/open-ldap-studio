"""A lossless model of a Spec Kit ``spec.md``.

The file is read as an ordered list of chunks. Most of it stays raw text; the
parts other tools care about (user stories, functional requirements, success
criteria and edge cases) are typed chunks that remember their id. Rendering is
the concatenation of the chunks, so ``render(parse(text)) == text`` for any
text, which is the property every other tool here relies on: a tool can change
one chunk and be certain nothing else moved.

Standard library only.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field

STORY_RE = re.compile(r"^### User Story (\d+) - (.+?) \(Priority: (P\d)\)\s*$")
FR_RE = re.compile(r"^- \*\*FR-(\d{3})\*\*: ")
SC_RE = re.compile(r"^- \*\*SC-(\d{3})\*\*: ")
EDGE_RE = re.compile(r"^- What happens ")
GROUP_RE = re.compile(r"^\*\*(.+)\*\*\s*$")
H2_RE = re.compile(r"^## (.+?)\s*$")
H3_RE = re.compile(r"^### (.+?)\s*$")


@dataclass
class Chunk:
    """One piece of the file. ``text`` includes its own line endings."""

    kind: str  # "raw" | "story" | "fr" | "sc" | "edge"
    text: str
    key: int | None = None  # story number, FR number, SC number, edge ordinal (1-based)
    meta: dict = field(default_factory=dict)


@dataclass
class Spec:
    chunks: list[Chunk]

    def of(self, kind: str) -> list[Chunk]:
        return [c for c in self.chunks if c.kind == kind]

    @property
    def stories(self) -> list[Chunk]:
        return self.of("story")

    @property
    def frs(self) -> list[Chunk]:
        return self.of("fr")

    @property
    def scs(self) -> list[Chunk]:
        return self.of("sc")

    @property
    def edges(self) -> list[Chunk]:
        return self.of("edge")


def parse(text: str) -> Spec:
    chunks: list[Chunk] = []
    raw: list[str] = []
    h2 = h3 = group = ""
    edge_n = 0
    story: Chunk | None = None

    def flush_raw() -> None:
        if raw:
            chunks.append(Chunk("raw", "".join(raw)))
            raw.clear()

    for line in text.splitlines(keepends=True):
        stripped = line.rstrip("\r\n")

        m = STORY_RE.match(stripped)
        if m:
            flush_raw()
            story = Chunk(
                "story",
                line,
                int(m.group(1)),
                {"title": m.group(2), "priority": m.group(3)},
            )
            chunks.append(story)
            h3 = stripped[4:]
            continue

        if story is not None:
            # A story runs until the next heading that is not a story.
            if H2_RE.match(stripped) or H3_RE.match(stripped):
                story = None
            else:
                story.text += line
                continue

        h2m, h3m = H2_RE.match(stripped), H3_RE.match(stripped)
        if h2m:
            h2, h3, group = h2m.group(1), "", ""
        elif h3m:
            h3, group = h3m.group(1), ""
        elif h3 == "Edge Cases" and GROUP_RE.match(stripped):
            group = GROUP_RE.match(stripped).group(1)

        fr, sc = FR_RE.match(stripped), SC_RE.match(stripped)
        if fr:
            flush_raw()
            chunks.append(Chunk("fr", line, int(fr.group(1)), {"section": h3}))
        elif sc:
            flush_raw()
            chunks.append(Chunk("sc", line, int(sc.group(1)), {"section": h3}))
        elif h3 == "Edge Cases" and EDGE_RE.match(stripped):
            flush_raw()
            edge_n += 1
            chunks.append(Chunk("edge", line, edge_n, {"group": group}))
        else:
            raw.append(line)

    flush_raw()
    return Spec(chunks)


def render(spec: Spec) -> str:
    return "".join(c.text for c in spec.chunks)
