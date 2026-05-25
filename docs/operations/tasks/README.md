# Task Tracking

Backlog for upcoming execution tasks. One directory per domain, one file per
task. A task becomes a GitHub Issue when execution starts.

## Convention

```
docs/operations/tasks/<domain>/
├── <NN>-<short-name>.md     ← individual task
└── README.md                ← optional domain overview
```

Each task file uses the [Execution Task](../../.github/ISSUE_TEMPLATE/execution-task.md)
template. Status is tracked in the file header:

```markdown
Status: backlog | in-progress | review | done
```

The file lives in the repo forever — committed alongside design docs as a
historical record of what was planned and when. No need to delete after
completion.

## Workflow

```
backlog  →  create GitHub Issue from file  →  branch  →  PR  →  merge  →  done
```

1. Write the task file in the repo (commit to main)
2. When ready to start, open a GitHub Issue from `.github/ISSUE_TEMPLATE/execution-task.md`
   — paste the content from the task file
3. Create branch per convention: `feat/<domain>/<unit>`
4. Code, PR, merge
5. Update task file status to `done` (optional — helpful for quarterly reviews)

## Task file minimum

```markdown
Status: backlog

## Scope
(one sentence)

## Branch
feat/<domain>/<unit>

## Files
| File | Action |
| :--- | :----- |

## Dependencies
Blocked by:
Blocks:
```

Fill from `docs/operations/roadmap.md` and domain PLAN.md.

## Current domains

- `planner-api/` — planner backend (coming after reservation)
- `seeding/` — seed data for dev and tests (can plan now)
- `planner-ui/` — planner frontend (after planner backend)
