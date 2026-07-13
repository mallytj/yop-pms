# Live Tape Chart — Receptionist

## Who

Front desk staff taking reservations via phone calls, walk-ins, emails. Their job is to get the right guest into the right room on the right dates, fast, without mistakes. Guest is usually on the phone, waiting to hear back on email or standing right in front of them.

## Goals

- **Create a reservation quickly**: guest on the phone, every second counts. Drag → hold → guest details at leisure → confirm.
- **Create multiple reservations quickly**: family booking, company block, multiple emails. Drag multiple rooms → bucket into reservations in the modal.
- **Find a room quickly**: matching guest needs (walk-in shower, dates, room types, etc). Search bar + room type filters + rate matrix.
- **Never double-book**: server-enforced. No optimistic render. SSE pushes real-time block updates. Mid-drag conflict = red highlight + tooltip.
- **See daily reservation numbers**: arrivals, departures, stayovers, vacant — header strip, live via SSE. Hover shows how many remain.
- **View reservations**: hover card shows prices, length, guest status, special requests. Sibling blocks highlight on same reservation.
- **Quick in-depth view**: click block → side panel with full details, per-night rates, guest info, folio summary.
- **Shorten/lengthen existing reservations**: drag block edges. Snaps back + tooltip on conflict. Past nights locked post-checkin.
- **Check guests in/out with minimal friction**: right-click, keybind, hover button, or click block. Multiple paths.

## Tasks

### Daily (every shift)

| Task | How |
|------|-----|
| Take phone reservations | Search bar (/) → check availability on chart → drag room × dates → hold created. Guest details in side panel. Confirm when ready. |
| Walk-in bookings | Button (w) or drag today column. Straight to checked_in. Room must be assigned. |
| Look up existing reservation | Search bar (/) → type guest name → chart scrolls + highlights block. |
| Modify reservation dates | Drag block edges. Resize. Conflict = snap back + tooltip. |
| Cancel reservations | Right-click or x → cancel modal with reason (audit). Undo toast. No "are you sure?" |
| See daily numbers | Header strip: arrivals / departures / stayovers / vacant. Live via SSE. |
| Reassign bookings | Drag block to different room row. DNM badge warns. Conflict = snap back. |
| Check guests in | Select block → i or right-click or hover button. Green left border on checked_in. Unassigned room = blocked. |
| Check guests out | Select block → o or right-click or hover button. Outstanding folio balance = blocked. Greyed terminal style. |
| No-show marking | Overdue booked block (amber tint) → right-click → "Mark No-Show." Future ledger released. |
| Overstay handling | Pulsing checked_in block past departure → right-click → "Extend Stay" or "Force Checkout." Pulse stops after 15 min. |

### Frequent (weekly+)

| Task | How |
|------|-----|
| Multi-room bookings (same guest) | Drag multiple room rows → modal buckets them into one reservation. Default: all same guest. |
| Multi-guest company bookings | Drag multiple room rows → modal: split into separate reservations, optionally link to named group. |
| Maintenance / decommissioning | Right-click empty cell or room header → "Mark as Maintenance" or "Decommission." Light red diagonal stripes. Non-interactive. |
| Rate matrix overview | Toggle to room type view → heatmap of rates + availability per date. |

### Future (spec'd, not built v1)

- **Waitlist**: guest on fully-booked dates, callback on cancellation
- **Group management**: cascade cancel, group labels, post-hoc editing
- **Print/export**: arrival lists, housekeeping sheets
- **Overbooking**: manager-approved deliberate overbook for high-demand dates

## Data They Need

### To check availability

- Dates (check-in, check-out) — drag on chart or date picker
- Room type (double, twin, suite, single) — filter pills at top
- Number of guests (adults, children)
- Available rooms: green-tinted empty cells, or room type rollup counts
- Rate: hover on date cell shows base rate for that type+date. Rate matrix toggle shows heatmap.

### To create the booking

- Guest name and contact (phone, email) — search existing (pg_trgm) or create new inline
- Special requests (high floor, quiet room, cot, accessibility) — in side panel. Badges on block for high-signal (♿, cot, DNM)
- Rate confirmation: standard or override — in modal
- Payment method: prepaid, pay at check-out, invoice — stubbed

### To modify or cancel

- Existing reservation details: dates, room, guest, rate, status — in hover card or side panel
- Cancellation policy: free cancel window, fee if late — stubbed
- Availability for new dates: drag edge, conflict cells light up red

### To view overall status

- Occupancy stats: arrivals, departures, stayovers, vacant — header strip
- Overdue arrivals: amber tint on booked blocks past check-in time
- Overstays: pulsing blocks (15 min time-limited)

## Pain Points (and fixes)

| Pain point | Fix |
|------------|-----|
| "Is room 12 free on the 15th?" | Quick room jump search bar + cell-level availability tinting |
| "Guest wants 3 nights but we only have 2" | Visual gap highlighting on drag + toast with alternative room type suggestions |
| "What's the rate for a double on a Friday?" | Hover on date cell shows rate. Rate matrix toggle shows heatmap |
| "Guest called to extend but I don't know if the room is free" | Drag right edge — snap back + tooltip if blocked. Visual gap if partial |
| Two receptionists booking same room | No optimistic render. Loading state. SSE pushes new blocks in real-time. Mid-drag conflict = red + tooltip |

### Block Visual States

| State | Visual |
|-------|--------|
| hold (unconfirmed) | Lower opacity + diagonal line pattern |
| confirmed | Solid block, blue left border |
| checked_in | Solid block, green left border |
| checked_out | Greyed out, locked |
| cancelled | Removed from chart (undo toast) |
| no_show | Greyed out, terminal |
| overstay | Pulsing animation, 15 min limit |
| maintenance / decommissioned | Light red diagonal stripes, non-interactive |
| DNM flag set | Lock badge on block |
| Accessibility required | ♿ badge |
| Cot required | Cot badge |
| Sibling of hovered block | Border highlight |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| g d | Jump to date |
| / | Focus search bar |
| n | New booking |
| w | Walk-in |
| ↑↓←→ / h j k l | Navigate between blocks |
| i | Check-in selected |
| o | Check-out selected |
| x | Cancel selected (modal with reason) |
| e | Edit selected |
| Enter | Open reservation |

## Constraints

- Cannot book a room that's already occupied for those dates (DB-enforced, no optimistic render)
- DNM flag blocks reassignment unless `reservations:override_dnm`
- Check-in requires assigned room
- Check-out blocked by outstanding folio balance
- Past nights immutable post-checkin
- All destructive actions: undo toast, no confirmation dialog (except cancel which requires reason)
- Real-time sync via SSE: blocks appear on other receptionists' screens within ms

## Architecture Notes (for implementation)

- SSE via NOTIFY reservation_changes (ADR-010, ADR-017)
- Availability via ledger (ADR-013)
- Hold-as-scaffold: drag creates hold with auto-pinned ledger rows
- State machine rollup (ADR-015) drives block borders
- `stay_period_envelope` (ADR-020) for date-range query efficiency
- `?include=` query param (ADR-022) for response depth: hover card = shallow, side panel = deep
- Rate matrix queries `pricing.daily_price_grid` aggregated by room type + date

## Requirements (Draft)

### Chart Rendering

- **R-TAPE-001**: Tape chart renders rooms on Y axis, dates on X axis, default view centered on today showing next 14 days, infinitely scrollable in both horizontal directions.
- **R-TAPE-002**: Each reservation renders as a block spanning its room row and date range, with visual state encoded per the Block Visual States table.
- **R-TAPE-003**: Empty cells show subtle green tint when available, no tint when occupied. Cells containing maintenance/decommission blocks show light red diagonal stripes and are non-interactive.
- **R-TAPE-004**: Header strip shows today's occupancy stats (arrivals, departures, stayovers, vacant rooms) updated live via SSE. Hover shows remaining counts.
- **R-TAPE-005**: Date-skip control (g d or UI input) scrolls chart to target date, anchoring it as the new leftmost column.

### Block Interactions

- **R-TAPE-006**: Hovering a block shows a detail card with: guest name, reservation code, dates, room type, total price, LOS, special request badges, and status. Sibling blocks (same reservation) show a border highlight.
- **R-TAPE-007**: Dragging a block's left or right edge resizes the stay period. Backend validates availability on release. On 409 conflict: block snaps back to original size, tooltip shows conflicting dates. Red highlight on conflicting cells.
- **R-TAPE-008**: Dragging a block vertically to a different room row triggers room reassignment. Backend validates no conflict + room type match. On 409: snaps back + tooltip. DNM flag blocks reassignment unless `reservations:override_dnm`.
- **R-TAPE-009**: Right-clicking a block shows context menu: Check In, Check Out, Cancel, Edit Details, Mark No-Show (context-dependent by current state).
- **R-TAPE-010**: Clicking a block opens a side panel with full reservation details: guest info, per-night rate breakdown, items, folio summary, metadata, special requests.
- **R-TAPE-011**: Destructive actions (cancel, no-show, checkout) show an undo toast instead of a confirmation dialog. Cancel additionally requires a reason in a modal for audit purposes. Undo available for configurable timeout (e.g., 10 seconds).
- **R-TAPE-012**: Past nights on checked-in blocks are visually locked — left edge cannot be dragged earlier than today.
- **R-TAPE-013**: Blocks show badges for high-signal metadata only: DNM (lock), accessibility (♿), cot, and overstay warning. Other special requests visible in hover card only.

### Reservation Creation Flow

- **R-TAPE-014**: Dragging across a date range on a room row initiates a new booking. On release, a modal opens showing selected item(s).
- **R-TAPE-015**: Dragging across multiple room rows (shift-click or multi-select) selects multiple items. Modal shows all selected, default-bucketed into one reservation.
- **R-TAPE-016**: Modal supports per-item bucketing: items can be split across multiple reservations within the same modal. Each reservation gets its own guest assignment. Unassigned items go into a new reservation via [+ Add reservation].
- **R-TAPE-017**: Default: all dragged items land in one reservation with one guest. "Same guest" toggle off → each item becomes its own reservation with independent guest fields. Items can be drag-rearranged between reservation buckets within the modal.
- **R-TAPE-018**: Creation submits as a hold (not confirmed). Chart shows the block immediately with hold styling (lower opacity + diagonal pattern). Room is locked via ledger. Guest details added later at receptionist's pace.
- **R-TAPE-019**: No optimistic render for creation: dragged cells show loading state until API returns 201. Block renders only on success.
- **R-TAPE-020**: On availability conflict during drag, conflicting cells highlight red inline. On release, toast shows alternative suggestions (different room types or adjacent dates) from the availability endpoint.
- **R-TAPE-021**: Walk-in button (w) pre-fills today's date, requires room assignment, creates reservation in checked_in state atomically. Bypasses hold phase.

### Check-in / Check-out

- **R-TAPE-022**: Check-in available on confirmed blocks with room assigned. Actionable via: right-click, i keybind, hover button, or side panel. Block left border turns green on success.
- **R-TAPE-023**: Check-in blocked with inline error if room is unassigned: "Assign room first."
- **R-TAPE-024**: Check-out available on checked_in blocks. Actionable via: right-click, o keybind, hover button, or side panel. Block transitions to greyed-out terminal style on success.
- **R-TAPE-025**: Check-out blocked with inline error if folio has outstanding balance: "Settle balance first."
- **R-TAPE-026**: booked blocks past `lower(stay_period)` + `no_show_grace_minutes` show amber tint. Right-click → "Mark No-Show" with optional fee (stubbed). Block transitions to greyed terminal.
- **R-TAPE-027**: checked_in blocks past `upper(stay_period)` + `late_checkout_grace_minutes` pulse for 15 minutes. Right-click → "Extend Stay" (opens date picker) or "Force Checkout." Pulse stops after 15 min or on resolution.

### Real-Time Updates (SSE)

- **R-TAPE-028**: Chart subscribes to SSE via NOTIFY reservation_changes. New/changed blocks appear on all connected clients within ms of commit.
- **R-TAPE-029**: SSE-pushed blocks appear with a subtle pulse animation for a few seconds to alert the receptionist of changes made by others.
- **R-TAPE-030**: If a conflicting block lands via SSE while the receptionist is mid-drag, the drag target highlights red and a tooltip shows: "Room just booked — choose another."
- **R-TAPE-031**: Header strip stats update in real-time via SSE as reservations are created, checked in, checked out, or cancelled.

### Rate Matrix

- **R-TAPE-032**: Rate matrix toggle switches Y axis from individual rooms to room types aggregated. Each cell shows base rate + availability count (e.g., "£120, 3/8 left").
- **R-TAPE-033**: Rate matrix cells use colour gradient: green (plenty available) → amber (filling up) → red (sold out / nearly full).
- **R-TAPE-034**: Hovering a date cell in room view (non-matrix) shows the base rate for that room type on that date. Tapping on tablet.
- **R-TAPE-035**: Block hover card shows total reservation price. Side panel shows per-night breakdown with rate plan and any adjustments.

### Filtering

- **R-TAPE-036**: Room type filter pills at top of chart toggle visibility of room rows by type. Multi-select (show Doubles + Twins, hide Suites).
- **R-TAPE-037**: Room type rollup view collapses individual room rows into aggregated type rows showing occupancy count per type per date. Expandable back to individual rooms.
- **R-TAPE-038**: Quick room jump search autocompletes room numbers and scrolls the chart to that row.

### Keyboard Shortcuts

- **R-TAPE-039**: All shortcuts use leader-key (chained) pattern for ergonomics. Global shortcuts: g d (jump to date), / (search bar), n (new booking), w (walk-in).
- **R-TAPE-040**: Block-level shortcuts when a block is selected/focused: i (check-in), o (check-out), x (cancel with reason modal), e (edit), Enter (open reservation).
- **R-TAPE-041**: Arrow keys (↑↓←→) and vim-style (h j k l) navigate between blocks. Selection wraps within visible viewport.
- **R-TAPE-042**: Shortcuts shown in a help overlay accessible via ? key.

### Maintenance & Decommission

- **R-TAPE-043**: Right-click empty cell or room header → "Mark as Maintenance" or "Decommission." Creates maintenance ledger rows for selected date range.
- **R-TAPE-044**: Maintenance and decommissioned cells render identically: light red diagonal stripe pattern, non-interactive. Cannot be dragged onto.
- **R-TAPE-045**: Decommissioned rooms hidden from chart by default. Toggle in room type filter to show/hide.

### Error Handling & Constraints

- **R-TAPE-046**: All mutations send `If-Match: <version>` header. 412 response triggers inline error with option to refresh block data.
- **R-TAPE-047**: Idempotency key automatically generated per mutation. 409 `IDEMPOTENCY_IN_PROGRESS` shows "Please wait, request in progress" — auto-retries once.
- **R-TAPE-048**: Backend 409 Conflict responses include `conflicting_dates` array. Chart uses this to highlight conflicting cells red.
- **R-TAPE-049**: Backend 403 Forbidden responses show inline error with the missing permission. No retry — escalate to manager path suggested.

### Client State & Performance

- **R-TAPE-050**: Chart fetches reservation data for visible date range only. Infinite scroll lazy-loads adjacent date ranges as user scrolls, with skeleton blocks during load.
- **R-TAPE-051**: Pagination uses `?include=shallow` for chart blocks (guest name, dates, status, price total) and `?include=deep` for side panel (all fields per ADR-022).
- **R-TAPE-052**: Tablet-friendly: all interactive elements (block edges, buttons, context menu items) have minimum 44px touch targets. Drag handles have enlarged hit areas on touch devices.
- **R-TAPE-053**: Chart renders using virtual scrolling for both axes (rooms + dates) to support properties with 100+ rooms without performance degradation.

## Next step

Feed into Live Tape Chart Feature Job Spec → to-tickets → Linear issues.
