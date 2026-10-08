import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
import check  # noqa: E402
from fixtures import MINI_SPEC, MINI_TASKS, Layout  # noqa: E402

FEATURE = os.path.join(os.path.dirname(__file__), "..", "..", "..", "specs", "001-open-ldap-studio")


def errors(layout):
    findings, _ = check.run(layout.root)
    return [f for f in findings if f.level == "error"]


def notes(layout):
    findings, _ = check.run(layout.root)
    return [f for f in findings if f.level == "note"]


class Clean(unittest.TestCase):
    def test_a_consistent_feature_has_no_errors(self):
        with Layout() as lay:
            self.assertEqual(errors(lay), [])

    def test_stats(self):
        with Layout() as lay:
            _, stats = check.run(lay.root)
        self.assertEqual(stats["stories"], 2)
        self.assertEqual(stats["requirements"], 3)
        self.assertEqual(stats["tasks"], 3)

    @unittest.skipUnless(os.path.isdir(FEATURE), "the real feature is not in this checkout")
    def test_the_real_feature_has_no_errors(self):
        findings, _ = check.run(FEATURE)
        self.assertEqual([str(f) for f in findings if f.level == "error"], [])


class SpecRules(unittest.TestCase):
    def test_duplicate_requirement(self):
        with Layout(spec_md=MINI_SPEC.replace("**FR-003**", "**FR-002**")) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("duplicate FR ids: FR-002" in m for m in msgs), msgs)

    def test_gap_in_requirements(self):
        with Layout(spec_md=MINI_SPEC.replace("**FR-002**", "**FR-004**")) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("gaps" in m and "FR-002" in m for m in msgs), msgs)

    def test_gap_in_success_criteria(self):
        with Layout(spec_md=MINI_SPEC.replace("**SC-001**", "**SC-002**")) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("SC ids have gaps" in m for m in msgs), msgs)

    def test_stories_must_be_numbered_in_order(self):
        with Layout(spec_md=MINI_SPEC.replace("User Story 2", "User Story 3")) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("not numbered" in m for m in msgs), msgs)

    def test_requirement_outside_a_topic(self):
        spec = MINI_SPEC.replace("### Topic One\n\n", "").replace("### Topic Two\n\n", "")
        with Layout(spec_md=spec) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("not under a ### topic heading" in m for m in msgs), msgs)

    def test_missing_spec(self):
        with Layout(spec_md=None) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("spec.md not found" in m for m in msgs), msgs)


class References(unittest.TestCase):
    def test_undefined_requirement_is_reported_with_its_line(self):
        with Layout(plan_md="# Plan\n\nSee FR-009 for details.\n") as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("plan.md:3" in m and "FR-009" in m for m in msgs), msgs)

    def test_undefined_success_criterion(self):
        with Layout(research_md="Needs SC-007.\n") as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("research.md:1" in m and "SC-007" in m for m in msgs), msgs)

    def test_a_range_is_expanded_and_every_member_must_exist(self):
        with Layout(plan_md="FR-001 – FR-003 are fine.\nFR-001–FR-005 are not.\n") as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertFalse(any("plan.md:1" in m for m in msgs), msgs)
        self.assertTrue(any("plan.md:2" in m and "FR-004" in m for m in msgs), msgs)
        self.assertTrue(any("plan.md:2" in m and "FR-005" in m for m in msgs), msgs)

    def test_prose_that_looks_like_a_range_is_not_one(self):
        with Layout(plan_md="Within FR-001 - 100 ms of the call.\n") as lay:
            self.assertEqual(errors(lay), [])

    def test_unknown_story_in_a_document(self):
        with Layout(plan_md="This belongs to US7.\n") as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("plan.md:1" in m and "US7" in m for m in msgs), msgs)

    def test_story_ids_inside_other_words_are_ignored(self):
        with Layout(plan_md="US-ASCII and FOCUS8 are not stories.\n") as lay:
            self.assertEqual(errors(lay), [])


class Tasks(unittest.TestCase):
    def test_duplicate_task(self):
        with Layout(tasks_md=MINI_TASKS.replace("T003", "T002")) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("duplicate task ids: T002" in m for m in msgs), msgs)

    def test_gap_in_tasks(self):
        with Layout(tasks_md=MINI_TASKS.replace("T002", "T005")) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("task ids have gaps" in m for m in msgs), msgs)

    def test_task_for_a_missing_story(self):
        with Layout(tasks_md=MINI_TASKS.replace("[US2]", "[US9]")) as lay:
            msgs = [str(e) for e in errors(lay)]
        self.assertTrue(any("tasks.md:5" in m and "US9" in m for m in msgs), msgs)


class Screens(unittest.TestCase):
    def test_tables_with_different_columns_are_both_read(self):
        with Layout() as lay:
            with open(os.path.join(lay.root, "contracts", "screens.md"), encoding="utf-8") as fh:
                text = fh.read()
        mapping = check.screens_map(text)
        self.assertEqual(mapping[1], {1, 2})
        self.assertEqual(mapping[2], {1, 2})
        self.assertEqual(mapping[3], {2})

    def test_requirements_no_screen_places_are_a_note_not_an_error(self):
        screens = "| ID | Screen | Story | Requirements | Status |\n|--|--|--|--|--|\n| **1a** | s | US1 | FR-001 | **Build** |\n"
        with Layout(screens_md=screens) as lay:
            self.assertEqual(errors(lay), [])
            msgs = [str(n) for n in notes(lay)]
        self.assertTrue(any("2 of 3 requirements" in m and "FR-002" in m and "FR-003" in m for m in msgs), msgs)


if __name__ == "__main__":
    unittest.main()
