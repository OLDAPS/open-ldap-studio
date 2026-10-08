# Spec-driven development in Open LDAP Studio

This project is built spec-first with [Spec Kit](https://github.com/github/spec-kit).
This page explains what that means here, what has been done so far, and where the
written record and the reality have drifted apart.

- [What it is, and why it fits this project](#what-it-is-and-why-it-fits-this-project)
- [The artifact chain](#the-artifact-chain)
- [What we have done so far](#what-we-have-done-so-far)
- [How the pieces relate](#how-the-pieces-relate)
- [Working with it](#working-with-it)
- [Known gaps](#known-gaps)

## What it is, and why it fits this project

In spec-driven development (SDD) the specification is the thing you write and
review first, and code is derived from it. Nothing is built that the spec does
not ask for, and a behaviour change starts as a spec change. Spec Kit supplies
the structure: a fixed chain of documents, each produced from the one before, and
a set of agent commands (skills) that generate and check them.

It suits this project for a specific reason. Open LDAP Studio edits production
directories, where an unconfirmed write can lock an organisation out. The
[constitution](../.specify/memory/constitution.md) turns that into five rules,
three of them marked non-negotiable:

| | Principle |
|---|---|
| I | Safety-first directory mutations: every write is previewed and confirmed |
| II | Credential and transport security: secrets live in the OS credential store, never in our files |
| III | Protocol fidelity over abstraction: raw DNs, filters and server diagnostics are always reachable |
| IV | Non-blocking, transparent UX: cancellable, time-limited, always showing which server you are about to change |
| V | Test-first against real directories: protocol-facing code needs a failing-then-passing test on a real server in a container |

The plan is checked against those rules before any design is accepted, so the
rules shape the architecture instead of being checked at review time. For example,
"no write without a preview" became a structural fact: the bridge exposes no write
method that does not take a confirmed preview token.

## The artifact chain

Each step consumes the one before it. All of it is in the repository.

| Step | Artifact | Location | Skill |
|---|---|---|---|
| 1. Principles | Constitution, versioned | [`.specify/memory/constitution.md`](../.specify/memory/constitution.md) | `/speckit-constitution` |
| 2. What and why | Feature specification: user stories, requirements, success criteria | [`specs/001-open-ldap-studio/spec.md`](../specs/001-open-ldap-studio/spec.md) | `/speckit-specify` |
| 3. Close the gaps | Clarifying questions encoded back into the spec; a quality checklist | [`checklists/requirements.md`](../specs/001-open-ldap-studio/checklists/requirements.md) | `/speckit-clarify`, `/speckit-checklist` |
| 4. How | Implementation plan, checked against the constitution | [`plan.md`](../specs/001-open-ldap-studio/plan.md) | `/speckit-plan` |
| 4a. | Research decisions (R1–R21), data model, interface contracts, quickstart | [`research.md`](../specs/001-open-ldap-studio/research.md), [`data-model.md`](../specs/001-open-ldap-studio/data-model.md), [`contracts/`](../specs/001-open-ldap-studio/contracts), [`quickstart.md`](../specs/001-open-ldap-studio/quickstart.md) | `/speckit-plan` |
| 5. Work items | Dependency-ordered tasks `T001`–`T339` | [`tasks.md`](../specs/001-open-ldap-studio/tasks.md) | `/speckit-tasks` |
| 6. Tracking | One GitHub issue per task, grouped into stories and epics | the [issue tracker](https://github.com/OLDAPS/open-ldap-studio/issues) | `/speckit-taskstoissues` |
| 7. Build | Code and tests | the repository | `/speckit-implement` |
| 8. Check | Cross-artifact consistency, and unbuilt work appended as new tasks | | `/speckit-analyze`, `/speckit-converge` |

The skills live in [`.claude/skills/`](../.claude/skills), the templates they fill
in under [`.specify/templates/`](../.specify/templates), and
[`.specify/workflows/speckit/workflow.yml`](../.specify/workflows/speckit/workflow.yml)
describes the full specify, plan, tasks, implement cycle with review gates.

The contracts are numbered so tests can name them: `C1`–`C14` (the bridge API and
package boundaries), `E1`–`E5` (events), `F1`–`F9` (file formats) and `X1`–`X8` (error
model). The task list says these tests are written before the code they constrain.

## What we have done so far

This section is taken from the artifacts and from git. Where it says what the
artifacts show, it does not claim to know which command produced them.

**Before the first commit.** The repository's git history starts on 2026-09-03,
but the specification work is older and went untracked until the pull request
that added this page. The artifacts date it:

| Date | What the artifacts show |
|---|---|
| 2026-08-24 | Spec Kit 1.0.1 initialised with the Claude integration. Constitution ratified, and by the end of the day amended up to 1.3.0, which removed all GNU Pass provisions from Principle II so that the platform credential interface is the only mandated integration point |
| 2026-08-31 | Specification created: 13 user stories, 108 functional requirements, 18 success criteria, 23 edge cases each with a required behaviour, a glossary, and four named exclusions |
| 2026-08-31 | The quality checklist records two validation iterations. The first pass was judged shallow on review, and the spec was rewritten rather than patched: parity with Apache Directory Studio had been asserted rather than delivered, edge cases had no expected behaviour, and one success criterion was circular |
| 2026-08-31 | Plan with five constitution gates, a risk register and delivery milestones; a second design pass over every wireframe screen found deviations three times larger than first recorded and seven capabilities the design promised that the spec did not require; research decisions R1–R21; data model; five contract documents; quickstart |
| 2026-08-31 | 339 tasks in 16 phases, one phase per user story plus setup, foundations and release |

**From the first commit onwards**, git shows:

- **2026-09-03:** initial commit, then `docs/MVP.md`.
- **Tasks became issues.** The tracker holds one issue per task, grouped as
  epic, user story, story, task, with `area/*` labels. Issue bodies carry the
  `T###` id and the relevant spec text.
- **The MVP line was redrawn.** The tracker treated all thirteen user stories as
  the product. [`docs/MVP.md`](MVP.md) re-cuts it around a read-only first
  release (connect, browse, search) and reorders the milestones to M0–M10, moving
  secrets and release automation earlier and writes, fluency, safety net and
  specialist work after the MVP line. It lists the places where the older
  `MVP.md` and the tracker still disagree.
- **Architecture was written down** in [`docs/architecture.md`](architecture.md)
  (the committed version dates from 2026-09-10) and the codebase was reorganised
  towards it (PR #470, merged 2026-09-23).
- **Foundation work started:** the Wails shell, the LDAP layer (`ldapx`), the
  change-set pipeline, the secrets providers, the jobs registry and the frontend
  shell exist in `internal/` and `frontend/`, and the UI was moved to shadcn
  components (PR #471).
- **The delivery pipeline was built** (from 2026-10-08, in a stack of pull
  requests starting with #468): release
  automation and conventional-commit enforcement, whole-tree lint and
  vulnerability gates, container-backed functional tests, git hooks, native
  installers with an install-and-run check, a licence and a README. These map to
  milestone M0 and part of M5 in `MVP.md`.

## How the pieces relate

Four things describe the project, and each is the authority on one question:

| Question | Authority |
|---|---|
| What rules must every change obey? | the constitution |
| What is the product, and what counts as done? | `spec.md` (requirements `FR-001`–`FR-108`, success criteria `SC-001`–`SC-018`) |
| How is it built, and in what order within a story? | `plan.md`, `contracts/`, `tasks.md` |
| What is done, and what do we ship first? | the **issue tracker** for state, [`docs/MVP.md`](MVP.md) for order and the MVP line |

When two disagree, the earlier row wins on intent and the later row is what
gets corrected. The one exception is progress: the tracker is the record, not
the checkboxes in `tasks.md` (see below).

## Working with it

**First time in a fresh clone.** Spec Kit finds the current feature through
`.specify/feature.json`, a per-checkout pointer that is deliberately not committed.
Until it exists, `/speckit-plan`, `/speckit-tasks`, `/speckit-implement` and the
other feature-aware commands stop with "Feature directory not found". Create it:

```bash
printf '{\n  "feature_directory": "specs/001-open-ldap-studio"\n}\n' > .specify/feature.json
```

or set `SPECIFY_FEATURE_DIRECTORY=specs/001-open-ldap-studio` in the environment.
Git ignores the file, so it will not show up in `git status`.

**Changing behaviour.** Edit the spec first, then re-run the plan and task steps
so the change flows down, then update the tracker. A pull request that changes
what the product does without touching the spec is the thing this process exists
to prevent.

**Starting a new feature.** Run `/speckit-specify` with a description. It creates
the next numbered directory under `specs/` and a spec to review before planning.

**Before implementing a task.** Run `/speckit-analyze` to look for contradictions
between spec, plan and tasks. After a stretch of work, `/speckit-converge`
compares the codebase with the spec and appends whatever is still unbuilt to
`tasks.md`.

**Amending the constitution.** Use `/speckit-constitution`. Bump the version by the
policy in the document (major for removing or redefining a principle, minor for
adding or materially expanding one, patch for wording), and put the amendment in
its own pull request. Every pull request is a compliance checkpoint against it.

**Tests come first.** Principle V and the task list both require the contract
tests to exist, and fail, before the code they constrain.

## Known gaps

These are known and deliberately written down rather than hidden.

- **`tasks.md` is not kept up to date.** All 339 checkboxes are unticked although
  code exists. Read progress from the tracker, not from the file.
- **The spec is still marked `Draft`.**
- **The constitution has an open TODO.** `TODO(TECH_STACK)` records that the
  technology stack is chosen but not yet written into the constitution. Task
  T339 ([#467](https://github.com/OLDAPS/open-ldap-studio/issues/467)) closes it
  with a minor bump to 1.4.0, in its own pull request.
- **The constitution refers to a `CLAUDE.md`** for agent and contributor guidance
  that must not contradict it. None exists yet.
- **The issue bodies embed spec text between `spec-sync` markers, but nothing in
  this repository generates or refreshes it.** The text can go stale.
- **Milestone names differ.** `plan.md` numbers its milestones M0–M6, the tracker
  and `MVP.md` use M0–M10. `MVP.md` is current.
- **The wireframe is external.** The screens the plan builds on live in a Claude
  Design project, not in this repository; `contracts/screens.md` is the in-repo
  mapping.
- **Some delivery work has no task.** The integration job, the `--version`
  smoke test, the installers, the git hooks and the licence were done without
  `T###` ids. They should be added to `tasks.md` and the tracker.
