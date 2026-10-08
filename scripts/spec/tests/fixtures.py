"""A small but complete spec used by the tests, and a helper to lay one out on disk."""

import os
import tempfile

MINI_SPEC = """# Feature Specification: Demo

**Status**: Draft

## Overview

A demonstration.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - First thing (Priority: P1)

Body one.

**Acceptance Scenarios**:

1. **Given** a, **When** b, **Then** c.

---

### User Story 2 - Second thing (Priority: P2)

Body two.

---

### Edge Cases

Each case states the behaviour.

**Group A**

- What happens when x? It does y.

**Group B**

- What happens when z? It does w.

## Requirements *(mandatory)*

### Topic One

- **FR-001**: System MUST a.
- **FR-002**: System MUST b.

### Topic Two

- **FR-003**: System MUST c.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Something measurable.

## Assumptions

- None.
"""

MINI_TASKS = """# Tasks

- [ ] T001 Set up
- [ ] T002 [P] [US1] Build the first thing
- [x] T003 [US2] Build the second thing
"""

MINI_SCREENS = """# Screens

| ID | Screen | Story | Requirements | Bridge | Status |
|----|--------|-------|-------------|--------|--------|
| **1a** | Start | US1 | FR-001, FR-002 | `Open` | **Build** |

| ID | Screen | Story | Requirements | Status |
|----|--------|-------|-------------|--------|
| **2a** | Second | US2 | FR-001 – FR-003 | **Build** |
"""


class Layout:
    """Writes a feature directory to a temporary location.

    Keyword names are the files they stand for; pass ``None`` to leave a file out.
    """

    NAMES = {
        "spec_md": "spec.md",
        "plan_md": "plan.md",
        "research_md": "research.md",
        "tasks_md": "tasks.md",
        "screens_md": "contracts/screens.md",
    }

    def __init__(self, **files):
        unknown = set(files) - set(self.NAMES)
        if unknown:
            raise KeyError(f"Layout does not know {sorted(unknown)}; add it to NAMES")
        self._tmp = tempfile.TemporaryDirectory()
        self.root = self._tmp.name
        content = {"spec.md": MINI_SPEC, "tasks.md": MINI_TASKS, "contracts/screens.md": MINI_SCREENS}
        for key, text in files.items():
            content[self.NAMES[key]] = text
        for name, text in content.items():
            if text is None:
                continue
            path = os.path.join(self.root, name)
            os.makedirs(os.path.dirname(path), exist_ok=True)
            with open(path, "w", encoding="utf-8") as fh:
                fh.write(text)

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        self._tmp.cleanup()
