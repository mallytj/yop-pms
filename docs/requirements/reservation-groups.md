# Reservation Groups RTM

> **⚠️ PLACEHOLDER — DEFERRED**
>
> Group reservations (allotments, cutoff, group rates, rooming lists, master folio
> routing) are **out of scope** for the current reservation-flow work. Groups are a
> nice-to-have, not a make-or-break feature for shipping the core reservation API.
> Momentum on the single-reservation flow takes priority.
>
> Do **not** implement against this doc. When groups are picked up in a future PR,
> this file will be replaced with a full RTM covering lifecycle, allotments vs
> general inventory, cutoff release, group rate precedence, rooming-list import,
> master folio routing, group cancel cascade, and group-vs-direct booking races.

## Known requirements (placeholders)

- Group lifecycle + state machine (tentative / confirmed / closed / cancelled).
- Allotment as block of N rooms × room_type × date range, written to ledger.
- Cutoff date worker releases unbooked allotment back to general inventory.
- Master folio = a single A/B/C folio dedicated as the group master (per
  `folios.md`); routing rules from member folios deferred.
- Rooming-list bulk import (CSV/JSON), creates one member reservation per entry.
- Group rate plan(s) override standard rate plan rules for member reservations.
- New permission scope (`groups:*`) to be added when authz lands.

## Open questions for the future group PR

- Allotment status: new ledger enum (`allotted`) vs reuse `on_hold`?
  (Reusing `on_hold` collides with single-reservation TTL semantics in ADR-010 —
  likely needs a new enum or a `hold_kind` discriminator column.)
- Group rate vs rate-plan precedence.
- Spillover policy when allotment exhausted (allowed / rejected / per-group flag).
- Group cancel cascade vs members already checked-in.
- Group split / merge.
- Group deposit on master folio vs per member.
