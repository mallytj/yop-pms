# Architecture Decision Records (ADRs)

This directory contains architecture decision records for Yop PMS. Each ADR documents a major decision, its context, consequences, and alternatives considered.

## Format

Each ADR follows the short form (per ADR-015):

```
# ADR NNN: Title

## Status

**Accepted** | **Proposed** | **Deprecated** | **Superseded by ADR-NNN**

One paragraph stating the decision and why.

Alternatives considered (one line each, with rejection reason).

---

See: references to code, migrations, other ADRs
```

## Active ADRs

### Foundation & Architecture

| #                                            | Title              | Status   |
| -------------------------------------------- | ------------------ | -------- |
| [001](001-schema-first-api.md)               | Schema-First API   | Accepted |
| [002](002-core-db-principles.md)             | Core DB Principles | Accepted |

### Platform Layer & Infrastructure

| #                                            | Title                        | Status   |
| -------------------------------------------- | ---------------------------- | -------- |
| [003](003-error-handling.md)                 | Error Handling               | Accepted |
| [004](004-idempotency-key.md)                | Idempotency Key              | Accepted |
| [005](005-check-constraint-consistency.md)   | Check-Constraint Consistency | Accepted |
| [006](006-transactional-outbox.md)           | Transactional Outbox         | Accepted |
| [008](008-cursor-pagination.md)              | Cursor Pagination            | Accepted |

### Reservations Domain

| #                                            | Title                            | Status   |
| -------------------------------------------- | -------------------------------- | -------- |
| [007](007-locking-availability.md)           | Locking & Availability           | Accepted |
| [009](009-state-machine-rollup.md)           | State Machine Rollup            | Accepted |
| [010](010-guest-aware-hold-ttl.md)           | Guest-Aware Hold TTLs            | Accepted |
| [012](012-stay-period-semantics.md)          | Stay Period Time Semantics       | Accepted |
| [013](013-reservation-envelope.md)           | Reservation Envelope             | Accepted |
| [014](014-audit-logs-trigger.md)             | Audit Logs via DB Trigger        | Accepted |
| [015](015-response-depth-include.md)         | Response Depth via `?include=`   | Accepted |

### Frontend / Transport

| #                                            | Title                        | Status   |
| -------------------------------------------- | ---------------------------- | -------- |
| [011](011-sse-realtime.md)                   | Real-Time Frontend via SSE   | Accepted |

## Pruned

ADRs deferred to a later milestone. Kept for historical context; not in force.

| #                                                                  | Title                          | Reason                       |
| ------------------------------------------------------------------ | ------------------------------ | ---------------------------- |
| [019](../pruned/019-payment-authorization-model.md)                | Payment Authorization for Holds | Deferred to finance PR        |

## How to Create a New ADR

1. **Identify the decision** — What are we deciding? Why now?
2. **Write the ADR** — Follow the short-form template above
3. **Get feedback** — Code review + architecture discussion
4. **Accept or reject** — Update Status field
5. **Reference in code** — Link ADRs from related code via comments

### ADR Template

```markdown
# ADR NNN: Title

## Status

**Accepted** | **Proposed**

One paragraph stating the decision and why.

Alternatives considered (one line each, with rejection reason).

---

See: references to code, migrations, other ADRs
```

## ADR Workflow

### Proposing

Create a new ADR in `docs/adr/NNNN-title.md` with Status: Proposed. Open a PR for discussion.

### Accepting

Once consensus is reached:

1. Change Status to Accepted
2. Add implementation details if needed
3. Link from related code and docs

### Deprecating

If we decide to use a different approach:

1. Create new ADR with the new decision
2. In old ADR, change Status to Superseded by ADR-XXXX
3. Update references in code to point to new ADR

### Pruning

If a decision is **deferred** to a later milestone (not rejected, just out of scope):

1. Move the ADR file to `docs/pruned/NNNN-title.md`
2. Add YAML frontmatter: `status: pruned`, `pruned_date`, `pruned_by`, `reason`
3. Add an entry in the `## Pruned` table above
4. The file body is preserved as-is for historical context

## Querying ADRs

- 001 Schema-First API
- 002 Core DB Principles
- 003 Error Handling
- 004 Idempotency Key
- 005 Check-Constraint Consistency
- 006 Transactional Outbox
- 007 Locking & Availability
- 008 Cursor Pagination
- 009 State Machine Rollup
- 010 Guest-Aware Hold TTLs
- 011 Real-Time Frontend via SSE
- 012 Stay Period Time Semantics
- 013 Reservation Envelope
- 014 Audit Logs via DB Trigger
- 015 Response Depth via `?include=`

## Principles

1. **Be concise** — Fit the decision in a paragraph. If it needs more, the decision is probably too big or not well understood.
2. **Cover the alternatives** — The decision itself is obvious; the alternatives are what make it worth recording.
3. **Link to code** — Show where the decision is implemented.
4. **Old ADRs stay as-is** — Don't rewrite them. New ADRs use the short form.
