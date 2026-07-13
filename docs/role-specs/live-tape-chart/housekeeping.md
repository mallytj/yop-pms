# Live Tape Chart — Housekeeping

## Problem

Housekeeping staff need a fast, low-noise way to see which rooms need attention and move rooms through readiness states. They do not manage bookings, prices, or guest accounts; the interface should avoid exposing data they do not need.

## Solution

Provide a separate `/housekeeping` route with a matrix-style action console focused on room readiness. The default view shows operational status and suggested priority, with one-tap next actions for normal cleaning flow. Guest details stay hidden by default under the Principle of Least Privilege, with an optional `housekeeping:see_details` permission for hotels that want housekeepers to see names or richer prep context.

## Who

Housekeeping staff clean rooms, refresh stayovers, report room problems, and keep rooms ready for arrivals. They work from an operational queue, usually on mobile/tablet or a shared back-office device, and need to answer: "What room should I do next?" and "What state is this room in now?"

They are not reservation agents. They should not need to understand the full tape chart, pricing, guest folios, or booking lifecycle to complete their work.

## Goals

- Know which rooms need cleaning or linen work now.
- Move each room through the next valid readiness state with one tap.
- Prioritise rooms by arrival pressure without treating the priority list as a hard rule.
- See maintenance/out-of-service rooms that affect today's work.
- Protect guest privacy by default while allowing property-configured detail access where operationally needed.

## Tasks

### Daily (every shift)

| Task | How |
| --- | --- |
| See rooms needing attention | Open `/housekeeping`; matrix sorted by suggested operational urgency. |
| Start cleaning a room | Tap the row action: `dirty` → `in_progress`. |
| Mark a room clean | Tap the row action: `in_progress` → `clean`. |
| Mark a room inspected | If property uses inspection and user has permission, tap `clean` → `inspected`. |
| Complete linen-change work | Tap `linen_change` → `clean`. |
| Spot urgent rooms | Use row badges: `Due 14:00`, `Arrival today`, `Stayover linen`, `Blocked`. |
| See maintenance impact | Maintenance/out-of-service rooms stay visible with blocking reason. |

### Frequent (weekly+)

| Task | How |
| --- | --- |
| Clean vacant dirty rooms early | Dirty vacant rooms with no arrival today/tomorrow stay visible below arrival-pressure rooms. |
| Work around stayover needs | Stayover linen-change rooms remain in queue even if vacant rooms exist; staff can choose based on local judgement. |
| Inspect rooms for larger hotels | Property enables `inspected` step; supervisors with `housekeeping:inspect` mark rooms inspected. |
| View limited room details | Inline row expand shows additional fields if user has `housekeeping:see_details`. |

### Rare

| Task | How |
| --- | --- |
| Flag a room problem | If permitted, mark room out of service / create maintenance block from row action or overflow action. |
| Review blocked room | Open inline maintenance detail for out-of-service room. |
| Bulk inspect rooms | Optionally mark multiple clean rooms inspected; only for harmless transitions. |

## Matrix Shape

The housekeeping route is not a date tape chart. It is a room-readiness matrix.

Suggested columns:

| Column | Purpose |
| --- | --- |
| Room | Room number/name and room type. |
| Current occupancy | Vacant / occupied / departing / arriving. |
| Housekeeping status | `dirty`, `in_progress`, `clean`, `inspected`, `linen_change`, `out_of_service`. |
| Suggested priority | Operational ordering and badges, not a hard rule. |
| Maintenance state | Active maintenance/decommission block if any. |
| Prep flags | Safe operational flags such as cot, accessibility setup, extra bed, linen-change due. |
| Action | One-tap next valid transition. |

## Status Transitions

Normal transitions:

```text
post-checkout → dirty
dirty → in_progress
in_progress → clean
clean → inspected       (only if inspection enabled + permitted)
linen_change → clean
any active status → out_of_service + maintenance block/problem report
```

Inspection is property-configured. Bigger chains may require supervisor inspection; smaller hotels can skip `inspected` and treat `clean` as ready.

No free-form status editing in normal flow. The row action shows the next valid action only.

## Suggested Priority

Priority is a recommended sort, not a hard business rule. Staff can still work rooms in the order that fits real-world operations.

Default suggested order:

1. Dirty departing room with arrival today, earliest check-in first.
2. Dirty vacant room with arrival tomorrow.
3. Dirty vacant room with no arrival today or tomorrow.
4. Linen-change stayover due today.
5. In-progress rooms.
6. Clean rooms awaiting optional inspection.
7. Out-of-service / maintenance rooms visible, outside cleaning queue.

Reason: hotels often clean rooms early, but stayover linen changes may still need attention while guests are out of room.

## Data They Need

### To choose the next room

- Room number/name.
- Room type.
- Current occupancy state: vacant, occupied, departing, arriving.
- Arrival pressure: today/tomorrow/none and expected check-in time if known.
- Current housekeeping status.
- Maintenance/out-of-service indicator.
- Linen-change due indicator.

### To prepare the room

Default least-privilege fields:

- Occupancy count.
- Accessibility setup flag.
- Cot / extra bed flag.
- Pet-friendly setup flag, if property supports pets.
- VIP / manager-note indicator without note content unless permitted.
- Linen-change due.

With `housekeeping:see_details`:

- Guest name.
- Arrival/departure time.
- Visible prep notes.
- Special requirements relevant to room setup.

### To complete work

- Next valid status action.
- Whether inspection is required before the room is sellable/ready.
- Whether maintenance blocks prevent readiness.
- Error if another staff member changed the room state first.

## Pain Points

| Pain point | Fix |
| --- | --- |
| Housekeepers do not need full receptionist tape chart complexity. | Separate `/housekeeping` route with readiness matrix. |
| Popups/drawers slow down repetitive state changes. | One-tap row action for next valid transition. |
| Guest privacy risk from exposing full reservation details. | Hide guest PII by default; use `housekeeping:see_details` for property-approved detail access. |
| Priority is useful but never perfectly matches real operations. | Treat priority as suggested sort with filters/badges, not a locked workflow. |
| Bigger hotels require supervisor inspection; small hotels may not. | Make `inspected` step property-configured. |
| Maintenance blocks can surprise housekeeping. | Keep out-of-service rooms visible with reason/status. |

## Alerts

Housekeeping portal shows actionable operational alerts only:

- Arrival room not clean by configurable cutoff.
- Room still `dirty` after checkout with same-day arrival.
- Room left `in_progress` too long.
- Maintenance block affecting today/tomorrow arrival.
- Room marked `out_of_service`.

No chat/inbox in v1. No noisy SSE popups. Matrix rows update live and urgent rows rise or receive badges.

Future: connect these alerts to a notification service that notifies the appropriate staff.

## Permissions

- `housekeeping:read` — access housekeeping route and safe room readiness data.
- `housekeeping:update_status` — move rooms through normal status transitions.
- `housekeeping:inspect` — mark `clean` rooms as `inspected` where inspection is enabled.
- `housekeeping:see_details` — see guest names and fuller prep context.
- `maintenance:create` — flag a room out-of-service / create maintenance block, if property allows housekeeping to do this.

Permission model follows the Principle of Least Privilege. Derived safe fields should be preferred over raw guest notes.

## Constraints

- Housekeeping route is separate from receptionist tape chart, not a view toggle.
- Housekeeping cannot create, modify, cancel, check in, or check out reservations.
- Housekeeping cannot see rates or revenue data.
- Guest names and detailed notes hidden unless `housekeeping:see_details` is granted.
- Normal status changes are one-tap and modal-free.
- `inspected` status is optional and property-configured.
- Every room status change and maintenance-affecting action must be auditable.
- Audit logging should cover the whole booking/tape-chart process, not only reservations.

## Requirements (Draft)

- **R-HK-TAPE-001**: System provides a separate `/housekeeping` route for housekeeping staff, distinct from the receptionist tape chart.
- **R-HK-TAPE-002**: Housekeeping route renders a room-readiness matrix, not a rooms-by-dates booking chart.
- **R-HK-TAPE-003**: Matrix shows room, occupancy pressure, housekeeping status, suggested priority, maintenance state, prep flags, and next action.
- **R-HK-TAPE-004**: Matrix sorts by suggested operational urgency while allowing staff to work out of order.
- **R-HK-TAPE-005**: Default priority order places same-day arrival dirty rooms first, tomorrow arrivals second, dirty vacant rooms with no near arrival third, then stayover linen changes, in-progress rooms, inspection queue, and maintenance/out-of-service rooms.
- **R-HK-TAPE-006**: Normal room status transitions are one-tap row actions with no drawer or modal required.
- **R-HK-TAPE-007**: Supported transitions include `dirty → in_progress`, `in_progress → clean`, optional `clean → inspected`, and `linen_change → clean`.
- **R-HK-TAPE-008**: `inspected` is property-configured; properties may skip inspection and treat `clean` as ready.
- **R-HK-TAPE-009**: Guest PII is hidden by default from housekeeping users.
- **R-HK-TAPE-010**: Users with `housekeeping:see_details` can expand a row inline to see guest name, arrival/departure time, visible prep notes, and relevant special requirements.
- **R-HK-TAPE-011**: Housekeeping route exposes safe prep flags by default, including accessibility setup, cot/extra bed, pet setup if supported, VIP/manager-note indicator, and linen-change due.
- **R-HK-TAPE-012**: Housekeeping users cannot create, modify, cancel, check in, or check out reservations from the housekeeping route.
- **R-HK-TAPE-013**: Housekeeping users cannot see rates or revenue data.
- **R-HK-TAPE-014**: Route access requires `housekeeping:read`.
- **R-HK-TAPE-015**: Room status updates require `housekeeping:update_status`.
- **R-HK-TAPE-016**: Marking a room inspected requires `housekeeping:inspect`.
- **R-HK-TAPE-017**: Flagging a room out-of-service or creating a maintenance block requires `maintenance:create`.
- **R-HK-TAPE-018**: Maintenance/out-of-service rooms remain visible in the matrix with a blocking reason/status.
- **R-HK-TAPE-019**: Portal shows actionable alerts for late dirty rooms, same-day arrival risk, long in-progress duration, maintenance blocking arrivals, and out-of-service rooms.
- **R-HK-TAPE-020**: Alert delivery is in-portal only for v1; future notification service integration is out of scope for this role spec.
- **R-HK-TAPE-021**: Every housekeeping status transition and maintenance-affecting action writes an audit record.
- **R-HK-TAPE-022**: Audit coverage for booking, housekeeping, maintenance, and tape-chart mutations must be handled as a whole-process concern in the Feature Job Spec.
- **R-HK-TAPE-023**: Matrix updates live when relevant room readiness, maintenance, or arrival pressure changes.

## Next step

Feed into Live Tape Chart Feature Job Spec → planner/API contract → to-tickets → Linear issues.
