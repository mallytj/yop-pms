# ADR 016: Guest-Aware Hold TTLs

## Status

**Accepted**

## Context

The reservation system uses a "Hold-as-Scaffold" pattern where creating a hold immediately locks a physical room via the inventory ledger. While anonymous website holds must expire quickly to release inventory for other guests, internal staff-led bookings (e.g., phone reservations) often require significantly more time to collect guest details, verify payment, or coordinate with other staff. 

A single, short TTL for all holds would frustrate staff and result in lost bookings. A single, long TTL would lead to "inventory leaks" where abandoned website checkouts block rooms for hours.

## Decision

We have chosen to implement a multi-tiered TTL strategy for reservation holds. The hold-expiry background worker will apply different expiration thresholds based on the reservation source and whether a guest identity has been established:

- **Website Source:** Strict short TTL (e.g., 15-30 mins) to match typical payment provider checkout sessions.
- **Internal Source (Anonymous):** Short grace period (e.g., 30-60 mins) to allow for draft creation.
- **Internal Source (with Guest ID):** Significantly longer grace period (e.g., 12-24 hours) as an attached Guest ID indicates higher intent and a need for operational flexibility.

## Consequences

### ✅ Positive (The "Wins")

- **Operational Flexibility:** Staff can safely lock rooms during phone calls without fear of immediate auto-cancellation.
- **Inventory Protection:** Prevents stale website checkouts from blocking inventory for extended periods.
- **Data Integrity:** The "Hold-as-Scaffold" pattern remains robust by providing a safety net for all sources.

### ⚠️ Negative (The "Costs")

- **Worker Complexity:** The hold-expiry worker logic must now join against property settings and check multiple conditions (source, guest presence) rather than a simple timestamp check.

## Alternatives Considered

- **Universal Source TTL:** Rejected. A TTL long enough for staff would be too risky for the public website; a TTL short enough for the website would be unusable for staff workflows.
- **Manual Staff Release Only:** Rejected. Risks permanent inventory blocks if staff forget to cancel draft or abandoned internal holds.

## References

- [ADR-013: Locking & Availability Strategy](013-locking-availability-strategy.md)
- [Reservation API Requirements](../requirements/reservations.md) (R-RES-INTEG-007)
