# Issue tracker: Linear

Issues live in Linear. Use the `linear` CLI v2.0.0 for all operations.

## Workspace

- **Slug**: `yopp` (configured via `LINEAR_API_KEY` env var)

## Conventions

- **Create an issue**: `linear issue create --title "..." --description-file /tmp/body.md --project "<project>" --label "<label>"`. Prefer `--description-file` for markdown bodies to avoid shell escaping issues.
- **Read an issue**: `linear issue view <team_key>-<number>` (e.g. `YOP-42`). Use `--no-comments` to skip comment threads, `--json` for structured output.
- **List issues**: `linear issue list --state unstarted --project "<project>" --label "<label>" --limit 50`. Filter by `--state` (triage, backlog, unstarted, started, completed, canceled), `--project`, `--label`, `--team`.
- **Comment on an issue**: `linear comment add <issueId> --body-file /tmp/comment.md`.
- **Apply / remove labels**: `linear issue update <issueId> --label "<label>"` (adds; to remove, update with remaining labels).
- **Close**: `linear issue update <issueId> --state completed`.

## Lifecycle & auto-tracking

### Branch naming convention

Use Linear's branch naming pattern to auto-link commits and PRs:

```
<username>/<team-key>-<number>-<slug>

e.g.  mally/YOP-50-add-booking-search
      mally/YOP-42-diagnose-slow-queries
```

When branch name includes the issue key (`YOP-NN`), Linear auto-tracks:
- Commit messages reference the issue
- PR creation links to the issue automatically
- PR merge → issue auto-closes (no manual status update needed)

### Issue lifecycle

```
Backlog → Todo → In Progress → In Review → Done
```

Linear states map: Backlog=backlog, Todo=unstarted, In Progress=started, Done=completed.

**Do not manually close implementation issues.** Linear auto-closes them when PR merges — as long as the branch follows the naming convention above.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if this repo treats external PRs as feature requests; `/triage` reads this flag.)_

Linear integrates with GitHub PRs automatically — issues link to PRs via branch name or manual linking.

## When a skill says "publish to the issue tracker"

Create a Linear issue.

## When a skill says "fetch the relevant ticket"

Run `linear issue view <team_key>-<number>`.

## Wayfinding operations

Used by `/wayfinder`.

- **Map**: a single issue labelled `wayfinder:map`.
- **Child ticket**: an issue linked as a Linear sub-issue or referenced in the map body with `Part of YOP-<n>`.
- **Blocking**: Linear's native issue relations (`linear issue update <child> --parent <parent>` for sub-issues; `linear issue relation add <child> blocks <blocker>` for blocking). Fallback: `Blocked by: YOP-<n>` in the child body.
- **Frontier query**: list map's open children, drop any with open blockers or assignee; first in map order wins.
- **Claim**: `linear issue update <n> --assignee self`.
- **Resolve**: comment the answer, close the issue, append context pointer to map.
