---
name: "Execution Task"
about: Single shippable unit. One branch, one concern, merge <3 days.
title: "[EXEC]: "
labels: execution
assignees: ""
---

## Scope

One thing. If you need "and" — split.

## Branch

`<prefix>/<domain>/<unit>`

## Why no design doc?

Not every task needs a PLAN.md, ADR, or RTM. If the scope is well-understood, extracts existing patterns, or has no external dependencies — state why full ceremony would be overhead.

## Deliverables

- [ ]

## Dependencies

Blocked by: #
Blocks: #

## Done when

- [ ] make audit clean
- [ ] Tests pass
- [ ] Branch lives <3 days before PR
- [ ] PR <=400 lines changed (exclude generated code, migrations)
