import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
import specmodel  # noqa: E402
from fixtures import MINI_SPEC  # noqa: E402

REAL = os.path.join(os.path.dirname(__file__), "..", "..", "..", "specs", "001-open-ldap-studio", "spec.md")


class RoundTrip(unittest.TestCase):
    def test_fixture_is_reproduced_exactly(self):
        self.assertEqual(specmodel.render(specmodel.parse(MINI_SPEC)), MINI_SPEC)

    def test_any_text_is_reproduced_exactly(self):
        for text in [
            "",
            "no newline at end",
            "\n\n\n",
            "line one\r\nline two\r\n",
            "### User Story 1 - T (Priority: P1)\nbody",
            "- **FR-001**: a\n- **FR-001**: duplicate\n",
            "### Edge Cases\n**G**\n- What happens x\n",
        ]:
            with self.subTest(text=text):
                self.assertEqual(specmodel.render(specmodel.parse(text)), text)

    @unittest.skipUnless(os.path.exists(REAL), "the real spec is not in this checkout")
    def test_the_real_spec_is_reproduced_exactly(self):
        with open(REAL, encoding="utf-8") as fh:
            text = fh.read()
        self.assertEqual(specmodel.render(specmodel.parse(text)), text)


class Extraction(unittest.TestCase):
    def setUp(self):
        self.spec = specmodel.parse(MINI_SPEC)

    def test_stories(self):
        self.assertEqual([c.key for c in self.spec.stories], [1, 2])
        self.assertEqual(self.spec.stories[0].meta, {"title": "First thing", "priority": "P1"})

    def test_a_story_keeps_its_separator_and_stops_at_edge_cases(self):
        last = self.spec.stories[1].text
        self.assertTrue(last.endswith("---\n\n"), repr(last[-20:]))
        self.assertNotIn("Edge Cases", last)

    def test_requirements_remember_their_topic(self):
        self.assertEqual([(c.key, c.meta["section"]) for c in self.spec.frs],
                         [(1, "Topic One"), (2, "Topic One"), (3, "Topic Two")])

    def test_success_criteria(self):
        self.assertEqual([c.key for c in self.spec.scs], [1])

    def test_edge_cases_remember_their_group(self):
        self.assertEqual([(c.key, c.meta["group"]) for c in self.spec.edges], [(1, "Group A"), (2, "Group B")])

    def test_bullets_outside_their_section_are_not_typed(self):
        spec = specmodel.parse("## Notes\n\n- What happens when? Not an edge case here.\n")
        self.assertEqual(spec.edges, [])


if __name__ == "__main__":
    unittest.main()
