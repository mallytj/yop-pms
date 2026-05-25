---
name: "\U0001F4A1 Execution Task"
about: Single shippable unit of work. One branch, one concern, merge <3 days.
title: "[EXEC]: "
labels: execution
assignees: ""
---

## Linked Design / Epic

_Reference the design issue or ADR this task implements._

- Design issue: #
- ADR(s):
- Roadmap item:

## Scope (ONE thing)

_What exactly does this branch deliver? If you need "and" — split._

- [ ]

## Branch

`<prefix>/<domain>/<unit>`

_Examples:_
- `feat/planner/schema`
- `feat/planner/service-core`
- `feat/seeding/cmd-seed`
- `feat/planner-ui/design-system-routes`

```

```

## Files Changed

_List files this branch will create/modify. If >8 files across >2 dirs, consider splitting._

| File | Action (create/modify/delete) |
| :--- | :---------------------------- |
|      |                               |
|      |                               |
|      |                               |

## Dependencies

**Blocked by:** # (merge this first)
**Blocks:** #

## Definition of Done

- [ ] `make audit` clean (vet, lint, tests, svelte-check)
- [ ] Tests pass (unit + integration for new code)
- [ ] No `t.Skip()` in new tests — all edge cases active
- [ ] Swagger/OpenAPI contracts generated (`make gen`)
- [ ] Branch lives <3 days before opening PR
- [ ] PR ≤400 lines changed (exclude generated code, migrations)
