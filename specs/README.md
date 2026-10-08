# Specifications

Each directory here is one feature, written with [Spec Kit](https://github.com/github/spec-kit)
before it is built. How the process works, and what this project has done with
it, is in [`docs/spec-driven-development.md`](../docs/spec-driven-development.md).

## [`001-open-ldap-studio`](001-open-ldap-studio): the product

Read in this order:

| File | What it answers |
|---|---|
| [`spec.md`](001-open-ldap-studio/spec.md) | What the product is and what counts as done: 13 user stories, requirements `FR-001`–`FR-108`, success criteria `SC-001`–`SC-018` |
| [`checklists/requirements.md`](001-open-ldap-studio/checklists/requirements.md) | How the spec was validated, and what review found wrong with it |
| [`plan.md`](001-open-ldap-studio/plan.md) | How it is built, checked against the constitution, with a risk register |
| [`research.md`](001-open-ldap-studio/research.md) | The decisions behind the plan (`R1`–`R21`) and the alternatives rejected |
| [`data-model.md`](001-open-ldap-studio/data-model.md) | The entities and the files the application writes |
| [`contracts/`](001-open-ldap-studio/contracts) | The bridge API, events, file formats, error model and screens; the numbered contract tests (`C`, `E`, `F`, `X`) come from here |
| [`quickstart.md`](001-open-ldap-studio/quickstart.md) | How to build, run and validate each user story |
| [`tasks.md`](001-open-ldap-studio/tasks.md) | The work, as tasks `T001`–`T339` in dependency order |

The rules every change is checked against are in the
[constitution](../.specify/memory/constitution.md).

**Progress is tracked on the [issue tracker](https://github.com/OLDAPS/open-ldap-studio/issues),
not in `tasks.md`**, whose checkboxes are not kept up to date. Milestone order and
the MVP line are in [`docs/MVP.md`](../docs/MVP.md).
