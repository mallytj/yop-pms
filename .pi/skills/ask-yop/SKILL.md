---
name: ask-yop
description:
  Ask which skill or flow fits your situation. Yop-specific router over the full
  agentic engineering workflow.
disable-model-invocation: true
---

# Ask Yop

You don't remember every skill, so ask.

A **flow** is a path through the skills. The Yop workflow has **6 stages** from
idea to shipped feature, with two on-ramps merging onto it. Everything else is
standalone, or a vocabulary layer that runs underneath.

## The main flow: idea → ship (6 stages)

The route most work travels. You have an idea and want it built.

### Stage 1: Role Job Spec (30-60m per role)

Start from the Linear project draft — it defines which roles this feature affects.
Those roles determine what to research and write specs for.

Capture domain knowledge by role BEFORE any feature planning. Ask: "what does
this role need from the system?"

- If no knowledge exists for the role, run `/research` first to gather domain
  context
- Run `/grill-with-docs` per role to interview against the domain model
- Use the Role Job Spec format: Who, Goals, Tasks, Data, Pain Points,
  Constraints, Draft Requirements
- Write to `docs/role-specs/<feature>/<role>.md` so to-spec can pick them up
  across sessions
- **Gate:** `/run-audit full "Role Job Spec: <Role>"` — advisory agents review
  draft for blind spots (GDPR, scale, ops, UX)
- Feed into Stage 4 (to-spec) — a Feature Job Spec draws from multiple Role Job
  Specs in `docs/role-specs/<feature>/`

### Stage 2: Research (Pocock)

Investigate existing solutions. KISS: prefer actively published, trusted,
maintained libraries over handwritten code — don't reinvent what's already
battle-tested.

- `/research` delegates reading legwork to a background agent
- Produces a cited Markdown file in the repo
- Feeds into Stage 3 (Domain Modeling)

### Stage 3: Domain Modeling (Pocock)

Sharpen shared vocabulary. Update `CONTEXT.md`. Create ADRs if needed.

- `/domain-modeling` — challenge fuzzy terms, resolve overloaded words, record
  hard-to-reverse decisions
- `/codebase-design` — deep-module vocabulary for designing module shapes

### Stage 4: to-spec — Feature Job Spec (2-4h)

Turn role specs + research + domain model into a buildable plan. **Yop override:
uses Job Spec format, not user stories.**

- `/to-spec` — Problem, Solution, role-grouped requirements (`**What**: Why`),
  State Machine (optional), Edge Cases, Routes (API + Pages), Workers
  (optional), Migrations (optional)
- No user stories ("As a... I want..."). Solo-dev Problem/Solution format.
- **Gate:** `/run-audit full "Feature Job Spec: <Feature>"` — advisory review
  before to-tickets
- Published to Linear with `ready-for-agent` label.

### Stage 5: to-tickets (30-60m)

Break the feature spec into VERTICAL tracer-bullet tickets with blocking edges.

- `/to-tickets` — each ticket is a narrow but COMPLETE path through every layer
- Blocking edges create a dependency graph. Frontier tickets can run in
  parallel.
- Published to Linear with `ready-for-agent` label.

### Stage 6: implement (1-3d per ticket)

Build one ticket at a time. Unblocked frontier tickets can run in parallel via
separate Git worktrees — one worktree per ticket, clear context between them.
**Yop override: E2E-driven TDD with playwright-capture.**

1. Read ticket + spec
2. Draft E2E happy-path test (Playwright) — agent self-validates
3. TDD build (unit + integration at agreed seams)
4. `/playwright-capture` → `.qa-review/` (agent decides if UI changed; camera,
   not judge)
5. `/code-review` — two-axis review (Standards + Spec)
6. **Human gate:** `/hunk-grill` — human reviews unstaged changes, agent
   addresses in fix loop
7. Commit in clean, logical chunks after approval, then human pushes to PR

**Parallel execution:** tickets with all blockers done can run simultaneously in
separate Git worktrees (`/using-git-worktrees`). Clear context between tickets.

## On-ramps

A starting situation that generates work, then merges onto the main flow.

- **Bugs and requests piling up** → `/triage`. Moves issues through triage
  roles, produces agent-ready issues for `/implement`. Only for issues you
  didn't create — tickets from `/to-tickets` are already agent-ready.

- **Something's broken** → `/diagnosing-bugs`. For hard bugs — resists a first
  glance, intermittent flakes, regressions. Refuses to theorise until it has a
  tight feedback loop. Post-mortem hands off to
  `/improve-codebase-architecture`.

- **A huge, foggy effort — too big for one session** → `/wayfinder`. Charts a
  shared map of decision tickets on Linear, resolves one at a time until the way
  is clear. Produces decisions, not deliverables. Hands off to Stage 4
  (`/to-spec`).

## Codebase health

Not feature work — upkeep.

- `/improve-codebase-architecture` — surfaces deepening opportunities. Generates
  an idea for `/grill-with-docs`.
- `/codebase-design` — deep-module vocabulary for designing module shapes.

## Vocabulary underneath

Two model-invoked references that run _beneath_ the other skills.

- `/domain-modeling` — sharpen domain language, challenge fuzzy terms, record
  ADRs. Keeps `CONTEXT.md` clean.
- `/codebase-design` — deep-module vocabulary (module, interface, depth, seam,
  adapter, leverage, locality).

## Crossing sessions

- `/handoff` — compact conversation into a markdown file. Open a new session and
  reference that file.
- Use `/handoff` to branch into `/prototype` sessions, then hand back findings.
- `/compact` (built-in) — stay in same conversation, summarizing earlier turns.
  Use at intentional breaks between phases.

## Standalone

Off the main flow entirely.

- `/grill-me` — relentless interview for when you have no codebase. Stateless.
- `/grill-with-docs` — same interview but stateful: retains learnings in
  `CONTEXT.md` and ADRs.
- `/prototype` — throwaway program to answer one design question. Keep the
  answer, delete the code.
- `/research` — background agent investigates against primary sources. Produces
  a cited file.
- `/teach` — learn a concept over multiple sessions, using current directory as
  stateful workspace.
- `/run-audit` — Yop-specific. Run role-based audit agents (CTO, Boutique
  Director, Compliancy, UX) against a feature.
- `/writing-great-skills` — reference for writing and editing skills well.

## Context hygiene

Keep Stages 1–4 in one unbroken context window — don't compact until after
`/to-tickets`. Each `/implement` starts fresh, working from the ticket. The
limit is the smart zone (~120k tokens).

## Precondition

Tracker already configured for Yop (Linear). See `docs/agents/`.
labels, single-context docs).
