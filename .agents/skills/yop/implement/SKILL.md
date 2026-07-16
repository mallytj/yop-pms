---
name: implement
description:
  Implement ticket from to-tickets. TDD via tdd skill, colocated .spec.svelte
  tests, code-review, human review before commit.
disable-model-invocation: true
---

# Implement — Yop Override

1. **Read ticket.** Load ticket body + parent Feature Job Spec. Know exact
   behavior this delivers.
2. **TDD.** Use `/tdd` [tdd](../../../../.pi/skills/tdd/SKILL.md) skill. Done
   when full suite green + typecheck clean.
3. **E2E tests.** Only for full features (e.g. entire booking flow). Skip for
   simple ops (layout shell, single endpoint).
4. **Code review.** Self-review diff via `/code-review`. Standards + spec match.
5. **Commit.** After human review, clean logical commits referencing ticket ID.
   `git status` clean.

One worktree per ticket. Fresh context each run.
