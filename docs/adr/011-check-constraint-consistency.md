`markdown

# ADR 011: Implementing consistent constraints from DB to frontend

## Status

**[Accepted]**

## Context

A change in the database's constraints will directly affect the UI/UX for a
user. Ensuring consistent limits and requirements ensures a flawless and
predictable experience limiting bugs which leads to slower time to action

## Decision

Using a `tools/sync-constraints` package. It reads from the Postgres schema to
get all constraints and then converts into a `constraints.g.yml` for the backend
and `constraints.g.ts` for the frontend

## Consequences

An extra step in the development process. No constraints shall be added unless
added through the database.

### ✅ Positive (The "Wins")

- **Consistency From Database -> Frontend:** No matter when the change is made
  any affected forms will automatically update
- **Easy Form Creation:** Instead of jumping back and forth from schema to
  Svelte - there is one source of truth.

### ⚠️ Negative (The "Costs")

- **Have to create the package:** Building the `sync-constraints` package
  may take some time - but worth it at the end of the day
- **Unassigned Constraints may be missed:** If a constraint synced
  i.e GIST constraints - it may be misleading for the developer

## Alternatives Considered

- **Just using values:** Can quickly become inconsistent

## References

- [The Script](../../cmd/tools/sync-constraints/main.go)
- [Backend Usage Documentation](../guides/backend-constraints.md)
- [Frontend Usage Documentation](../guides/frontend-constraints.md)
  `
