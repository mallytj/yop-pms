# ADR Format

ADRs live in `docs/adr/` and use sequential numbering: `NNNN-title.md`. Number matches the highest existing + 1.

## Template

```md
# ADR NNN: Title

## Status

**Accepted** | **Proposed** | **Deprecated** | **Superseded by ADR-NNN**

One paragraph stating the decision and why.

Alternatives considered (one line each, with rejection reason).

---

See: references to code, migrations, other ADRs
```

## When to offer an ADR

All three of these must be true:

1. **Hard to reverse** — the cost of changing your mind later is meaningful
2. **Surprising without context** — a future reader will look at the code and wonder "why on earth did they do it this way?"
3. **The result of a real trade-off** — there were genuine alternatives and you picked one for specific reasons

If a decision is easy to reverse, skip it. If it's not surprising, nobody will wonder. If there was no real alternative, there's nothing to record.

## Lifecycle

- **Proposed** — open PR for discussion
- **Accepted** — consensus reached, merge
- **Deprecated** — no longer in force but historical
- **Superseded by ADR-NNN** — replaced by a newer decision

## Pruning

If a decision is deferred (not rejected, just out of scope):

1. Move the ADR to `docs/pruned/NNNN-title.md`
2. Add YAML frontmatter: `status: pruned`, `pruned_date`, `pruned_by`, `reason`
3. Add entry in `docs/adr/README.md` Pruned table
