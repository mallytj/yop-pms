# Architecture Decision Records (ADRs)

This directory contains architecture decision records for Yop PMS. Each ADR documents a major decision, its context, consequences, and alternatives considered.

## Format

Each ADR follows this structure:

- **Status** — Accepted, Proposed, Deprecated, Superseded
- **Context** — Why we needed to make a decision
- **Decision** — What we decided to do
- **Consequences** — Positive and negative impacts
- **Alternatives** — Options we considered
- **References** — Links to related code, docs, and external resources

## ADRs

### Foundation & Architecture

| #                                | Title              | Status   |
| -------------------------------- | ------------------ | -------- |
| [001](001-monorepo.md)           | Monorepo           | Accepted |
| [002](002-techstack.md)          | Tech Stack         | Accepted |
| [003](003-schema_first_api.md)   | Schema-First API   | Accepted |
| [004](004-core_db_principles.md) | Core DB Principles | Accepted |

### Platform Layer & Infrastructure

| #                                          | Title                        | Status   |
| ------------------------------------------ | ---------------------------- | -------- |
| [005](005-error-handling-strategy.md)      | Error Handling Strategy      | Accepted |
| [006](006-structured-logging-approach.md)  | Structured Logging           | Accepted |
| [007](007-idempotency-key-enforcement.md)  | Idempotency Key Enforcement  | Accepted |
| [008](008-redis-caching-layer.md)          | Redis Caching Layer          | Accepted |
| [009](009-opentelemetry-observability.md)  | OpenTelemetry Observability  | Accepted |
| [010](010-reactive-cache-invalidation.md)  | Reactive Cache Invalidation  | Accepted |
| [011](011-check-constraint-consistency.md) | Check Constraint Consistency | Accepted |
| [012](012-transactional-outbox-worker.md)  | Transactional Outbox Worker  | Accepted |
| [014](014-cursor-pagination.md)            | Cursor Pagination            | Accepted |

### Reservations Domain

| #                                           | Title                            | Status   |
| ------------------------------------------- | -------------------------------- | -------- |
| [013](013-locking-availability-strategy.md) | Locking & Availability Strategy  | Accepted |
| [015](015-state-machine-rollup.md)          | Reservation State Machine Rollup | Accepted |
| [016](016-guest-aware-hold-ttl.md)          | Guest-Aware Hold TTLs            | Accepted |
| [018](018-stay-period-time-semantics.md)    | `stay_period` Time Semantics     | Proposed |
| [019](019-payment-authorization-model.md)   | Payment Authorization for Holds  | Proposed |
| [020](020-reservation-envelope.md)          | Reservation Envelope Column      | Proposed |

## How to Create a New ADR

1. **Identify the decision** — What are we deciding? Why now?
2. **Write the ADR** — Follow the template below
3. **Get feedback** — Code review + architecture discussion
4. **Accept or reject** — Update Status field
5. **Reference in code** — Link ADRs from related code via comments

### ADR Template

```markdown
# ADR NNN: [Short Decision Title]

## Status

**[Accepted | Proposed | Deprecated | Superseded by ADR-NNN]**

## Context

**What is the problem we are solving?**

Describe the forces at play, constraints, and the pain point that triggered this decision.

## Decision

**What are we doing about it?**

State the decision clearly. Use strong language like "We will use..." or "We have chosen to..."

Include implementation details: specific libraries, patterns, or protocols that are vital.

## Consequences

**What is the aftermath of this decision?**

### ✅ Positive (The "Wins")

- **Benefit 1:** How this improves the system (performance, security, developer experience)
- **Benefit 2:** Next benefit

### ⚠️ Negative (The "Costs")

- **Trade-off 1:** What did we give up?
- **Trade-off 2:** New complexity being introduced

## Alternatives Considered

- **Alternative A:** Why we rejected this
- **Alternative B:** Why this wasn't a fit

## References

- [Link to issue/ticket]
- [Link to related ADR]
- [Link to external documentation]
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

## Querying ADRs

Find ADRs by topic:

### Foundation

- [ADR-001](001-monorepo.md) — Monorepo architecture
- [ADR-002](002-techstack.md) — Tech stack selection

### API & Database

- [ADR-003](003-schema_first_api.md) — Schema-first API design
- [ADR-004](004-core_db_principles.md) — Database conventions (financials, timestamps, uniqueness)

### Error Handling

- [ADR-005](005-error-handling-strategy.md) — Centralized APIError with SQLSTATE mapping

### Observability & Logging

- [ADR-006](006-structured-logging-approach.md) — Structured JSON logging with slog
- [ADR-009](009-opentelemetry-observability.md) — Distributed tracing with OpenTelemetry

### Data & Caching

- [ADR-007](007-idempotency-key-enforcement.md) — Idempotency via Redis + Idempotency-Key header
- [ADR-008](008-redis-caching-layer.md) — Simple prefix-namespaced Redis cache client
- [ADR-010](010-reactive-cache-invalidation.md) — PostgreSQL LISTEN/NOTIFY drives cache invalidation; TTLs are safety net
- [ADR-011](011-check-constraint-consistency.md) — DB check constraints synced to backend YAML and frontend TypeScript
- [ADR-014](014-cursor-pagination.md) — Cursor pagination convention for list endpoints

### Background Work

- [ADR-012](012-transactional-outbox-worker.md) — Transactional outbox pattern for emails, audit, deferred work

### Reservations Domain

- [ADR-013](013-locking-availability-strategy.md) — Hold-as-lock + auto-pin + ledger-as-truth
- [ADR-015](015-state-machine-rollup.md) — Item state changes drive reservation status via deterministic rollup
- [ADR-016](016-guest-aware-hold-ttl.md) — Multi-tiered hold expiry based on source and guest presence
- [ADR-018](018-stay-period-time-semantics.md) — `stay_period` TSTZRANGE bounds carry property check-in/out times
- [ADR-019](019-payment-authorization-model.md) — Card auth at hold time for website source
- [ADR-020](020-reservation-envelope.md) — Materialised `stay_period_envelope` column on reservations

### Frontend / Transport

- [ADR-017](017-realtime-frontend-sse.md) — SSE push for real-time frontend updates

## Principles

1. **Decisions are permanent records** — Don't edit ADRs; create new ones if you change your mind
2. **Include tradeoffs** — Every decision has pros and cons; be explicit
3. **Link to code** — Show where the decision is implemented
4. **Be specific** — "We use Redis" vs "We use Redis with 24h TTL for availability cache keys"
5. **Explain why** — Context is as important as the decision itself

## Further Reading

- [Lightweight ADRs by Michael Nygard](http://thinkrelevant.net/blog/2011/11/15/documenting-architecture-decisions/)
- [ADRs on GitHub](https://github.com/joelparkerhenderson/architecture_decision_record)
- [Architectural Thinking](https://www.thoughtworks.com/en-us/insights/article/architectural-thinking)
