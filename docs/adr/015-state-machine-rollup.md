# ADR 016: Reservation State Machine Rollup Rule

## Status

**Accepted**

## Context

A reservation may consist of multiple `reservation_items` (multi-room booking, family across two adjoining rooms, group sub-booking). Each item has its own lifecycle:

- A booked item can be checked in independently of its siblings (one guest arrives Friday evening, another Saturday morning).
- An item can be marked `no_show` while another in the same reservation is `checked_in`.
- An item can be cancelled while siblings remain active.

The reservation as a whole also has a status (the front desk wants to know "is this booking checked in?" without inspecting every item). This produces two state machines that must stay coherent:

- **Reservation status** — `hold`, `confirmed`, `checked_in`, `checked_out`, `cancelled`, `archived`
- **Reservation item status** — `booked`, `checked_in`, `checked_out`, `no_show`, `cancelled`, `archived`

Three approaches were considered:

1. Single state machine on the reservation; ignore per-item status.
2. Two independent state machines, no automatic coupling.
3. Two state machines with a deterministic rollup rule (item changes → reservation derived).

The first loses the ability to track partial check-ins or mixed outcomes. The second leaves the reservation status semantically ambiguous (what does a reservation in `confirmed` mean if one item is `checked_in` and another is `cancelled`?).

The schema already supports option 3: both enums exist, each table has its own `status` column. The decision is how the rollup is computed and triggered.

## Decision

The reservation status is **derived from its items** by a deterministic rollup rule, applied transactionally after every item state change.

### Rollup rule

Given the multiset of item statuses for a reservation, the reservation status is set as follows (first match wins):

| Condition on items                              | Reservation status |
| ----------------------------------------------- | ------------------ |
| All items in `cancelled` (or `archived`-from-cancel) | `cancelled`     |
| All items in terminal states AND ≥1 `checked_out` | `checked_out`    |
| ≥1 item `checked_in` AND no items in `booked`    | `checked_in`      |
| Otherwise                                        | _unchanged_        |

Terminal states for items: `checked_out`, `no_show`, `cancelled`, `archived`.

### Example traces

**Single-item reservation, normal flow:**
| Action | Item status | Reservation status |
| --- | --- | --- |
| Create (hold) | `booked` | `hold` |
| Confirm (payment / staff) | `booked` | `confirmed` |
| Check in | `checked_in` | `checked_in` |
| Check out | `checked_out` | `checked_out` |

**Two-item reservation, partial check-in:**
| Action | Item A | Item B | Reservation |
| --- | --- | --- | --- |
| Create | `booked` | `booked` | `confirmed` |
| Check in A | `checked_in` | `booked` | `confirmed` (rule "no items in booked" fails) |
| Check in B | `checked_in` | `checked_in` | `checked_in` (rule applies) |
| Check out A | `checked_out` | `checked_in` | `checked_in` (no terminal-all yet) |
| Check out B | `checked_out` | `checked_out` | `checked_out` |

**Mixed outcome:**
| Action | Item A | Item B | Reservation |
| --- | --- | --- | --- |
| Create | `booked` | `booked` | `confirmed` |
| Check in A | `checked_in` | `booked` | `confirmed` |
| B is no-show after grace | `checked_in` | `no_show` | `checked_in` |
| Check out A | `checked_out` | `no_show` | `checked_out` |

### Trigger mechanism

The rollup runs in two places:

1. **Application layer**, in the same transaction as any item-level state change. The handler updates the item, computes the new reservation status from all sibling items (single query: `SELECT status FROM reservation_items WHERE reservation_id = $1 AND deleted_at IS NULL`), and writes the new reservation status if it changed.

2. **Database trigger** as a safety net for direct SQL writes (migrations, admin tools, manual fixes). `AFTER UPDATE OF status ON reservation_items` fires a stored function applying the same rule.

The rule is implemented in **one place** (a SQL function called from both contexts) to prevent application/trigger drift.

### Status invariants enforced by the rule

- A reservation can never be `checked_in` while any item is still `booked`.
- A reservation cannot be `checked_out` until every item has reached a terminal state.
- A reservation in `cancelled` always has all items terminal in cancel-equivalent states.

### Manual override (rare)

In genuinely weird situations (data correction, OTA mishap), staff with `reservations:archive` permission may force a reservation status via an admin endpoint. The override is audited; the next item-level change re-runs the rollup and may overwrite the manual value. This is intentional: the rollup is the source of truth, manual overrides are temporary nudges.

### What doesn't roll up

The hold→confirmed transition is **not** an item-level concern; it is driven by payment outcome (website) or staff action (internal) on the reservation directly. Rollup never moves a reservation from `hold` to `confirmed`. This is because items always start `booked` regardless of reservation state, and `hold`/`confirmed` is a property of the booking commitment, not of any individual room.

Similarly, `archived` is set by the archival worker (R-RES-INTEG-008) on the reservation; items roll up to `archived` after the worker writes them. No rollup-driven archival.

## Consequences

### ✅ Positive

- **Single source of truth** — Item statuses are authoritative; reservation status is derived. No drift possible if both call the same function.
- **Multi-item reservations behave intuitively** — Partial check-in/check-out, mixed cancel/show, all express naturally.
- **Database-enforced safety net** — Trigger catches direct SQL writes that bypass the application layer.
- **Deterministic and testable** — Rule is a pure function of item-status multiset → reservation status; trivial to property-test.

### ⚠️ Negative

- **Two writes per item transition** — Application updates item, then updates reservation. The trigger version is a single round trip but the application path makes two writes. Acceptable: both go in one transaction, latency is one network hop.
- **Rule must be implemented in PL/pgSQL and Go** — Or in PL/pgSQL only and called from Go. We chose the latter (single source) but PL/pgSQL is harder to unit-test in isolation. Go-side wrapper provides a thin adapter for tests.
- **`hold`/`confirmed` exemption is special-case** — Reservation-level state that does not roll up from items is a rule footnote that reviewers must remember.
- **Manual overrides are clobberable** — A staff override is silently undone by the next item change. Documented; surfaced in the admin UI as a warning.

## Alternatives Considered

- **Reservation status only, drop item status** — Rejected. Loses ability to track per-room outcomes; multi-room reservations become opaque.
- **Two independent state machines** — Rejected. Reservation status becomes meaningless ("confirmed" might mean anything from "all booked" to "half cancelled half checked-in"). Front desk dashboards become unworkable.
- **Roll up via materialised view refreshed periodically** — Rejected. Stale state during the refresh window causes user-visible inconsistency. Synchronous rollup is cheap.
- **Computed column / generated column on reservations** — Rejected. Postgres generated columns cannot reference other tables (the items). Workaround would be a function-based view, which loses indexing benefits.
- **Event sourcing on items, project to reservation read model** — Rejected. Massive overshoot for a 5-property PMS. Re-evaluate if the audit / replay needs grow significantly.

## References

- `docs/requirements/reservations.md` — Section 7.3 rollup rule, sections 7.1 + 7.2 state machines
- `migrations/00001_initial_schemas_enums_functions_extensions.sql` — Enum definitions
- `internal/booking/state_machine.go` — Rollup function (TBD)
- `internal/store/queries/reservations.sql` — `RollupReservationStatus` query (TBD)
- ADR-013: Locking and Availability Strategy (state transitions interact with ledger writes)
