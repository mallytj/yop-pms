# Live Tape Chart — Manager

## Problem

Managers oversee daily operations: they take bookings, manage rooms, and monitor revenue. They need both the receptionist's booking workflow (for quick check-ins and reservations) and an oversight view that reveals occupancy and revenue patterns at a glance. They should not need separate dashboards to answer "how full are we tonight?" or "which room types are underperforming?"

## Solution

Two modes within the tape chart: **Reception** (identical to receptionist booking grid with PII visible) and **Management** (room-type-grouped oversight with rate heatmap, occupancy percentages, and optional maintenance/status overlays). All toggles live under a settings tab to keep the grid clean. The Management view defaults to aggregated per room type — expandable to individual room rows.

## Who

Hotel managers, general managers, assistant managers, and revenue managers. They wear two hats: reservation agent (answering calls, checking guests in/out) and operations overseer (monitoring occupancy, pricing, and room status across the property). They need to spot at a glance: "Is the hotel full this weekend?" "Which nights have gaps?" "Where is revenue strongest?"

## Goals

- Switch seamlessly between making bookings (Reception mode) and monitoring property state (Management mode)
- See entire property: all room types, all rooms, all reservations, all rates
- Spot occupancy gaps and high-rate nights with a color-coded heatmap
- Know at a glance which rooms are under maintenance and what status each room is in
- Drill from aggregated room-type view to individual rooms when investigating gaps

## Tasks

### Daily (every shift)

- Take walk-in and phone bookings (Reception mode)
- Check occupancy for tonight and next few days (Management mode)
- Spot rooms with issues (maintenance blocks, dirty status lingering)
- Adjust rates or override room assignments

### Frequent (weekly)

- Revenue review: which room types are filling? Which are lagging?
- Schedule maintenance based on occupancy gaps visible on the grid
- Toggle overlay settings per need (turn off maintenance when doing revenue review)

### Rare

- Property-level setting changes (default views, rate visibility)
- Night auditor tasks (separate auditor view, handled elsewhere)

## Data They Need

### To make bookings (Reception mode)

- Same as receptionist: full reservation grid, drag-to-create, edit/cancel
- Guest PII visible at a glance (full name, not just initial)

### To monitor property (Management mode)

- All room types grouped, with occupancy % and revenue per room type
- Rate heatmap: color gradient on reservation blocks by nightly rate
- Room status overlay (dirty/clean/out-of-order/occupied/vacant)
- Maintenance block overlay (red diagonal stripes, per YOP-54)
- Expandable individual rooms within each room type group

### To configure

- Settings tab with toggles: maintenance overlay, room status overlay, rate heatmap
- Per-property rate visibility: always-show vs hover-only

## Pain Points

- Currently no way to see both booking grid and revenue snapshot in one screen
- Have to switch between PMS booking screen and separate reporting tools
- Can't see at a glance which room types are under-booked — have to count manually
- Maintenance status isn't visible while taking bookings; creates double bookings

## Constraints

- Management view defaults to aggregated per room type (not flat 200-room grid)
- Reception mode = receptionist view + PII visible. No additional controls unless required
- Aggregated stats (revenue dashboards, ADR, RevPAR) live on a separate dashboard page — not here
- Night auditor has its own view, not mingled with manager
- All overlay toggles live in settings tab, not floating on the grid
- Must not hide rooms (PoLP is for limited roles; manager sees all)

## Requirements (Draft)

### Reception mode
- R-MGR-001: Provide reception mode identical to receptionist tape chart grid
- R-MGR-002: Display full guest name on reservation blocks (receptionist shows initial only)
- R-MGR-003: Full PII + reservation notes + rate breakdown available on reservation hover

### Management mode
- R-MGR-004: Group rooms by room type on the management grid
- R-MGR-005: Default to aggregated room-type view showing occupancy count per day, not individual rooms
- R-MGR-006: Allow toggling between aggregated (room type) and individual room view within each group
- R-MGR-007: Show occupancy percentage per room type in the group header
- R-MGR-008: Show revenue per room type in the group header (total for visible date range)
- R-MGR-009: Apply rate heatmap color gradient to reservation blocks (warm = high rate, cool = low rate)
- R-MGR-010: Show maintenance blocks as red diagonal-striped overlays (per YOP-54 spec)
- R-MGR-011: Support per-property rate visibility toggle: always-show vs hover-only on reservation blocks

### Settings
- R-MGR-012: Provide settings tab accessible from management mode
- R-MGR-013: Allow toggling maintenance block overlay visibility from settings
- R-MGR-014: Allow toggling room status overlay visibility from settings (dirty/clean/out-of-order)
- R-MGR-015: Allow toggling rate heatmap visibility from settings

### Reservation manipulation
- R-MGR-016: Support creating, modifying, and cancelling reservations via drag and click on the grid
- R-MGR-017: Allow modifying any room status (dirty/clean/out-of-order/occupied/vacant)

### Out of scope
- R-MGR-018: Aggregated revenue dashboards (ADR, RevPAR, occupancy trends) — separate dashboard page
- R-MGR-019: Night auditor workflow — separate view
- R-MGR-020: Rate management/editing on the tape chart — rate data API handled separately

## Next step

Feed into Live Tape Chart Feature Job Spec → to-tickets → Linear issues.
