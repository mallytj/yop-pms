---
status: pruned
pruned_date: 2026-07-13
pruned_by: YOP-16
reason: deferred-to-finance-pr
---

# ADR 019: Payment Authorization Model for Holds

## Status

**Proposed** (contract committed; implementation deferred to a later finance PR)

## Context

The "Hold-as-Scaffold" pattern (ADR-007) immediately locks a physical room via
the inventory ledger when a `hold` reservation is created. For website-source
holds, no payment was previously required at hold time — only at confirm,
via the checkout session.

This created an inventory-spam attack surface: an unauthenticated bot could
fire 1000 holds across the property's calendar and block every room for the
full hold TTL with no commitment. Even without malice, browser users
abandoning checkouts produces real inventory leaks.

Other sources don't share the problem:
- OTA — already paid upstream by the channel.
- Internal — staff trust gates abuse.
- Walk-in — settled at desk via folio.

A payment authorization at hold time turns a website hold into a credible
commitment without yet charging the guest, and bounds the hold lifetime to
the auth lifetime.

## Decision

Website-source holds require a **card authorization** (not capture) at POST
time. The hold is created only if the auth succeeds.

- POST `/reservations` with `source=website` rejects requests without a valid
  payment-method token (provider-specific; e.g. Stripe PaymentMethod ID).
- The gateway is asked to authorize a property-configurable amount (default:
  one night's rate). No funds are captured.
- On hold confirm (payment webhook → `confirmed`), the auth is captured.
- On cancel before confirm, the auth is voided.
- On TTL expiry (R-RES-INTEG-007), the auth is voided by the worker.
- `checkout_session` is renamed `payment_authorization` and carries
  `provider`, `auth_id`, `expires_at`, `captured_at`, `voided_at`.

OTA, internal, and walk-in sources retain their current paths — no auth at
hold.

Implementation lives in a later finance PR. Current swagger work commits the
**contract** (request/response shapes, error codes, table layout) so endpoint
shapes don't shift later.

## Consequences

### ✅ Positive

- Eliminates anonymous-hold inventory spam.
- Auth lifetime caps hold TTL with no extra worker logic.
- Refund flow becomes implementable (capture → refund through provider).
- Cleaner audit: every committed reservation has a captured authorization.

### ⚠️ Negative

- Adds a payment-provider dependency at hold time — degrades gracefully but
  hard to operate without one.
- Increases POST latency by one provider round-trip.
- Requires a provider abstraction layer (Stripe first; others later).

## Alternatives

- **Capture at hold (deposit model)** — rejected: friction on browse-to-book,
  legal complexity around refunds for never-confirmed bookings.
- **No auth — current flow** — rejected: spam vector unaddressed.
- **CAPTCHA / rate limit only** — rejected: doesn't solve abandonment, only
  malice.

## References

- ADR-007 — Locking & availability strategy
- ADR-010 — Guest-aware hold TTLs
- `/docs/requirements/reservations.md` §1, §6 (integration), §10 (OTA)
- `/docs/CONTEXT.md` — Payment authorization
