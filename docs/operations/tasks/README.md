# Task Tracking

Backlog for upcoming execution tasks. One directory per domain, one file per
task. A task becomes a GitHub Issue when execution starts.

## Convention

```
docs/operations/tasks/<domain>/
├── <NN>-<short-name>.md     ← individual task
└── README.md                ← optional domain overview
```

Each task file uses the
[Execution Task](../../../.github/ISSUE_TEMPLATE/execution-task.md) template.
Status is tracked in the file header:

```markdown
Status: backlog | in-progress | review | done
```

The file lives in the repo forever — committed alongside design docs as a
historical record of what was planned and when. No need to delete after
completion.

## Workflow

### Phase 1: Design

Open a **Technical Design** issue that produces:
- ADR(s) documenting architectural decisions
- RTM (requirements traceability matrix) in `docs/requirements/`
- Sequence diagrams in `docs/flows/`
- Implementation PLAN.md in the domain package

Merge docs straight to `main` — zero risk, independent of code.

### Phase 2: Task breakdown

Split the PLAN.md into one execution task per shippable unit. Write a task file
for each in `docs/operations/tasks/<domain>/`. Each file uses the
[execution template](../../../.github/ISSUE_TEMPLATE/execution-task.md).

### Phase 3: Execute one task at a time

For each task file (in numbered order):

1. Create a GitHub Issue from `../../../.github/ISSUE_TEMPLATE/execution-task.md`,
   paste the content from the task file
2. Create branch: `feat/<domain>/<unit>`
3. Code, test, open PR
4. `make audit` must pass before merge
5. Merge to `main`, close issue
6. Move to the next task file — repeat until all tasks in the domain are done

```
Task file (backlog) → GitHub Issue → branch → PR → merge → done → next task
```

**Rules:**
- No branch opens until the previous one merges (tasks are serial)
- No branch lives >3 days — if it will, split the task
- No `t.Skip()` in new tests
- PR ≤400 lines (exclude generated code and migrations)

---

## Worked example: planner-api

### Step 1 — Design issue

Title: `[DESIGN] Planner API`

Produces and merges to main:
- `docs/adr/023-planner-data-model.md`
- `docs/flows/planner.md`
- `docs/requirements/planner.md`
- `internal/planner/PLAN.md`

### Step 2 — Task files

Created in `docs/operations/tasks/planner-api/`:

```
01-schema.md        →  migrations + SQLC queries
                        feat/planner/schema

02-types-sm.md      →  types, errors, state machine (pure Go)
                        feat/planner/types-sm

03-service-core.md  →  service layer (Create, Confirm, Availability)
                        feat/planner/service-core

04-handlers-router.md →  handlers + router + wire in cmd/server/api.go
                          feat/planner/handlers-router

05-workers.md       →  background sweep workers (if needed)
                        feat/planner/workers
```

### Step 3 — Execute one at a time

```
Task: planner-api/01-schema
  → GitHub Issue #15 "[EXEC] Planner schema"
  → Branch: feat/planner/schema
  → Files: migrations/00009_planner.sql, store/queries/planner.sql
  → make audit, PR, merge
  → Close #15

Task: planner-api/02-types-sm
  → GitHub Issue #16 "[EXEC] Planner types + state machine"
  → Branch: feat/planner/types-sm
  → Files: types.go, errors.go, state_machine.go
  → make audit, PR, merge
  → Close #16

...repeat for 03, 04, 05
```

Each branch is small (1–5 files, <400 LOC), open <3 days, and merges
independently. Main is always shippable.

### Post-mortem comparison

The `feat/reservation-api` branch tried to do all of this in one branch:

```
DESIGN + schema + types + service + handlers + middleware + workers + SSE
= 1 branch, 6+ weeks, 6 rebases, ~12k LOC
```

With the task breakdown it would have been 9 execution tasks, each 0.5–3 days:

| # | Task | Files | Est. |
| :--- | :--- | :--- | :--- |
| 1 | Schema | migrations + sqlc | 1d |
| 2 | Platform additions | db/tx.go, helpers | 0.5d |
| 3 | Types + state machine | types, errors, state_machine | 1d |
| 4 | Service core | Create + Confirm + availability | 2d |
| 5 | Business logic | actions, mutations, rates | 3d |
| 6 | Middleware | auth, ifmatch, permission | 0.5d |
| 7 | Handlers + router | handlers_*, router, wire | 2d |
| 8 | Workers | workers.go + wire | 1d |
| 9 | SSE | hub.go + wire | 1d |

---

## Current domains

- `planner-api/` — planner backend (coming after reservation)
- `seeding/` — seed data for dev and tests (can plan now)
- `planner-ui/` — planner frontend (after planner backend)
