# scripts/spec

Tools for the specification in `specs/`. Python standard library only: nothing to
install. They are run as `python3 <file>`.

| File | What it does |
|---|---|
| `specmodel.py` | Reads `spec.md` as an ordered list of chunks (raw text, user stories, functional requirements, success criteria, edge cases) and writes it back. `render(parse(text)) == text` for any text, which is what lets later tools change one chunk and be sure nothing else moved. |
| `check.py` | Offline consistency checks over the feature directory: story, requirement and success-criterion ids are unique and contiguous; every `FR-`, `SC-` and `US` id cited in any document exists (ranges such as `FR-006 – FR-009` are expanded); task ids are unique and contiguous. Prints notes for things worth knowing, such as requirements no screen ties to a story. |
| `tests/` | `unittest` suites with a small synthetic spec, including the negative cases, plus a round trip of the real `spec.md`. |

```bash
make spec-check     # python3 scripts/spec/check.py --notes
make test-spec      # python3 -m unittest discover -s scripts/spec/tests
```

CI runs both in the `Spec consistency` job. See `docs/spec-driven-development.md`
for how the spec, the plan, the tasks and the tracker fit together.
