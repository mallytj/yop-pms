# Feature Job Spec: Live Tape Chart

**Status:** Draft (Stage 4 — to-spec) **Depends on:** Core: bookings engine
(CRUD, state machine, availability), SSE hub, room inventory ledger **Role
specs:**
`docs/role-specs/live-tape-chart/{receptionist,manager,housekeeping,maintenance}.md`
**Glossary:** `docs/CONTEXT.md` → Domain Terms → Tape Chart

---

## Problem

There is no visual overview of rooms × dates. Staff can't see which rooms are
free, when, for how long — not in one place. The backend API already supports
reservations, inventory, and availability queries, but there's no grid UI. Every
major PMS (Cloudbeds, Mews, Opera) has a tape chart for a reason: it's the
fastest way to understand and manipulate room inventory.

The tape chart must feel immediate: drag a block, it moves. A co-worker books a
room across the lobby, the grid updates before your next blink.

## Solution

A real-time visual grid: **room rows × date columns** with draggable blocks.
SvelteKit 5 frontend, Go/Chi backend, SSE for live sync. Four role-specific
views sharing the same grid component, differing in overlay layer and
interaction set.

### Grid component (shared)

- CSS Grid layout: one row per room, one column per day
- TanStack Virtual for horizontal + vertical virtualisation (100+ rooms × 365+
  days)
- `@thisux/sveltednD` for drag-and-drop (create/move/resize)
- `@tanstack/hotkeys` for keyboard shortcuts
- SSE subscription for real-time block updates from other clients
- **Today column** highlighted, advances at property-local midnight

---

## Requirements by Role

### Receptionist (`Role: Reception`)

| What                                                                                          | Why                                                             |
| --------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| Room rows × date columns, scroll past & future                                                | Phone guest, need to see availability quickly across date range |
| Reservation blocks = `reservation_item` rows, grouped by accent colour per parent reservation | Visual grouping of multi-room bookings; one item = one row      |
| Create drag on empty cell → `status=hold` block → side panel for guest details → confirm      | Speed: drag a room, fill details at leisure                     |
| Drag-and-snap: resize (shorten/lengthen), move (change room/dates)                            | Guest changes mind mid-call, adjust without starting over       |
| Conflict detection: red highlight + tooltip during drag, server-enforced on save              | Never double-book; visual feedback before server round-trip     |
| Hover card on block: price, dates, guest name, status, code                                   | Quick glance without opening panel                              |
| Click block → side panel: full details, per-night rates, guest info, folio summary            | Deep info when needed                                           |
| Right-click / keyboard shortcuts: check-in, check-out, cancel, view                           | Power user speed                                                |
| Header bar: arrivals today / departures / stayovers / vacant (live via SSE)                   | "How's the day looking?" at a glance                            |
| Search bar (`/`) + room type filter strips                                                    | Find a specific booking or room quickly                         |
| Multi-room drag: drag selection over multiple rooms → batch hold creation                     | Family booking, company block                                   |
| **NOT shown:** housekeeping status (dirty/clean/inspected) — that's Housekeeping view         | The receptionist needs occupancy, not linen state               |

### Manager (`Role: Manager`)

| What                                                           | Why                               |
| -------------------------------------------------------------- | --------------------------------- |
| Same grid + occupancy heatmap (colour intensity = % occupancy) | Revenue pulse at a glance         |
| Filter by room type, date range, occupancy %                   | Slice data to spot trends         |
| Export/print current view                                      | Morning meeting, reporting        |
| Toggle rate overlay: show nightly rate in cells                | Rate strategy, discount decisions |

### Housekeeping (`Role: Housekeeping`)

| What                                                                     | Why                                  |
| ------------------------------------------------------------------------ | ------------------------------------ |
| Same grid layout, cell status: `clean → dirty → in_progress → inspected` | Staff assignment, status tracking    |
| Room-row colour driven by `housekeeping_status`, not occupancy           | Cleanliness is the primary dimension |
| Click cell to cycle status                                               | Quick update while walking corridor  |
| Assignment view: assign housekeeper to a set of rooms for the day        | Work distribution                    |
| Maintenance blocks shown as striped overlay                              | Avoid assigning a room mid-repair    |

### Maintenance (`Role: Maintenance`)

| What                                                                                | Why                                                |
| ----------------------------------------------------------------------------------- | -------------------------------------------------- | ---------- | --------------- | ------------------- |
| Same grid layout                                                                    | Shared visibility across roles                     |
| Create, move, resize maintenance blocks (striped, distinct from reservation blocks) | Schedule deep-clean, repair, inspection            |
| Block types: `deep_clean                                                            | repair                                             | inspection | out_of_service` | Categorise the work |
| **Maintenance blocks are exclusive** — no reservation can overlap                   | Someone should not be in a room during maintenance |
| Past blocks: read-only (log of completed work)                                      | Record keeping                                     |

---

## State Machine

**Reservation item status flow** (existing, already implemented):

```
hold ──→ confirmed ──→ checked_in ──→ checked_out ──→ archived
  │                                                        ↑
  └────── cancelled ←──────────────────────────────────────┘
```

- `hold` → dashed/outline block. TTL-bound (`website_hold_ttl_seconds`,
  `internal_hold_ttl_seconds`).
- `confirmed` → solid block.
- `checked_in` → block locked (move triggers room-move flow §2.3, resize
  disabled for past nights).
- Drag on any status respects the state machine transitions.

**Maintenance blocks** — no state machine. `active` (in period) or inactive.
Binary.

---

## Edge Cases

| Edge case                          | Behaviour                                                                                                                                                                                                              |
| ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Hold expiry race**               | Clerk holds a room, starts filling details. Before they confirm, SSE pushes an update — room now occupied. Grid: hold block shows conflict overlay. Save: server rejects. Clerk notified.                              |
| **Concurrent drag conflict**       | Two clerks drag the same cell. First save wins. Losing client gets SSE update → block snaps into winning position with conflict animation.                                                                             |
| **Drag-to-conflict**               | Mid-drag over occupied cell → red highlight + tooltip "Occupied: [reservation code]". Drop in dead slot → snap-back to original.                                                                                       |
| **Checked-in item drag**           | Moving a checked-in reservation block → triggers room-move flow (§2.3). DNM-level check applied.                                                                                                                       |
| **Partial drag**                   | Drag block edge left → shorten stay. Single operation on the item. No split.                                                                                                                                           |
| **SSE reconnect (event replay)**   | SSE events carry no sequential IDs. Client reconnects after network drop with stale grid. Add `id:` field to SSE events, client sends `Last-Event-Id` on reconnect for replay.                                         |
| **SSE update during active drag**  | SSE reconciliation suppresses for blocks currently being dragged — don't overwrite optimistic position under cursor. Resume on drag end.                                                                               |
| **SSE merge**                      | Incoming block update (create, move, resize, delete, status change) → grid must reconcile against local array. If local has an unsaved hold block and SSE creates a confirmed block overlapping it → conflict overlay. |
| **Drag partial-conflict recovery** | Move operation spans N nights, but night 2 of the new range has a conflict → full revert. Atomic: all-or-nothing for any block move/resize. No partial moves.                                                          |
| **Grid scale**                     | 100+ rooms × 365+ days. Virtual scroll both axes. Only visible cells render. CSS `contain: layout paint` on cells.                                                                                                     |

---

## Routes

### Pages (`/web/src/routes/`)

| Route                   | View                             | Role         |
| ----------------------- | -------------------------------- | ------------ |
| `/`                     | Dashboard / overview             | All          |
| `/planner/reservations` | Reception tape chart (main view) | Reception    |
| `/housekeeping`         | Housekeeping tape chart          | Housekeeping |
| `/planner/rates`        | Manager occupancy view           | Manager      |
| `/planner/maintenance`  | Maintenance tape chart           | Maintenance  |

### API — Tape chart endpoint

Unified endpoint returning both reservation blocks and maintenance blocks in a
single response. Avoids separate fetch-and-merge logic on the client.

| Method | Path                                               | Purpose                                                |
| ------ | -------------------------------------------------- | ------------------------------------------------------ |
| `GET`  | `/v1/tape-chart?from=DATE&to=DATE&include=shallow` | Fetch all blocks for grid (reservations + maintenance) |

**Full response** (`?include=full`, default):

```json
{
  "from": "2026-07-13",
  "to": "2026-07-27",
  "room_types": [
    {
      "id": "uuid",
      "name": "Deluxe King",
      "rooms": [
        {
          "id": "uuid",
          "name": "101",
          "reservations": [
            {
              /* YOP-51 fat payload */
            }
          ],
          "maintenance_blocks": [
            { "id": "uuid", "reason": "str", "start": "date", "end": "date" }
          ]
        }
      ]
    }
  ]
}
```

**Shallow response** (`?include=shallow`):

```json
{
  "from": "2026-07-13",
  "to": "2026-07-27",
  "reservations": [
    {
      /* YOP-51 fat payload */
    }
  ],
  "maintenance_blocks": [
    { "id": "uuid", "reason": "str", "start": "date", "end": "date" }
  ]
}
```

**Date-range max 90 days** — reject requests beyond the window.

**No pagination** — PoLP role filtering + date range sufficient. Cursor-based
pagination can be added backward-compat later.

**Internal Go package:** `internal/planner/` (same pattern as
`internal/booking/` with `/v1/reservations`).

### Maintenance CRUD

| Method   | Path                                       | Purpose                                 |
| -------- | ------------------------------------------ | --------------------------------------- |
| `GET`    | `/v1/maintenance/blocks?from=DATE&to=DATE` | Fetch maintenance blocks for date range |
| `POST`   | `/v1/maintenance/blocks`                   | Create maintenance block                |
| `PUT`    | `/v1/maintenance/blocks/{id}`              | Update maintenance block                |
| `DELETE` | `/v1/maintenance/blocks/{id}`              | Delete maintenance block                |

### Existing relevant endpoints

| Method  | Path                          | Purpose                                                   |
| ------- | ----------------------------- | --------------------------------------------------------- |
| `GET`   | `/v1/tape-chart`              | Tape chart grid data (unified reservations + maintenance) |
| `GET`   | `/v1/reservations/{id}`       | Full reservation detail (side panel)                      |
| `POST`  | `/v1/reservations`            | Create reservation (hold → confirm)                       |
| `PATCH` | `/v1/reservations/items/{id}` | Move or resize a block                                    |
| `GET`   | `/v1/sse`                     | SSE subscription (realtime grid updates)                  |

### Workers

| Worker                              | Purpose                                                                                                                                     | Existing?                            |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| **Hold expiry sweeper**             | Cancel `status=hold` reservations past TTL. Push SSE event on expiry → grid removes dashed block.                                           | Yes (extend)                         |
| **Per-property timezone awareness** | Grid "today" column depends on property-local midnight. MVP: single timezone, read from `property.locale`. Multi-TZ deferred to future ADR. | New (config read)                    |
| **Inventory update broadcaster**    | When any block changes (create/move/resize/status), push event to SSE hub. Grid subscribers reconcile.                                      | New (thin wrapper over existing hub) |

### Migrations

| What                                                                                    | Status                                                    |
| --------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| `maintenance_blocks` table                                                              | Already exists (`migrations/00002_inventory_pricing.sql`) |
| `room_inventory_ledger` grid index                                                      | Already exists (`idx_inv_ledger_grid_view`)               |
| Composite index `(property_id, room_type_id, calendar_date)` on `room_inventory_ledger` | New — required for tape chart date-range queries at scale |
| No other schema changes needed for MVP                                                  | ✅                                                        |

---

## Architecture Notes

### Data flow

```
[Browser: SvelteKit grid]
        │ ▲
        │ │ Drag operations → optimistic UI
        │ │ SSE events → reconcile local state
        ▼ │
[Go API: /v1/tape-chart/*]
        │
        ▼
[PostgreSQL: room_inventory_ledger + reservations + maintenance_blocks]
        │
        ▼
[SSE Hub: broadcast inventory changes]
        │
        ▼
[All connected browsers receive update]
```

### Drag → save protocol

1. **On drag start**: Quick availability check
   (`GET /v1/tape-chart/availability`)
2. **On drag move**: Highlight conflict in real time (client-side, no server
   call)
3. **On drag end (drop)**: If no conflict → optimistic local update.
   `PATCH /v1/reservations/items/{id}` (existing endpoint handles move + resize)
4. **Server validates**: state machine, availability, conflict check
5. **On success**: Server persists, SSE broadcast to all clients
6. **On failure**: Server returns 409 Conflict. Client reverts optimistic state,
   shows error toast

---

## Pre-implementation Checklist

- [ ] Spike prototype validated CSS Grid + TanStack Virtual + drag-and-drop
- [ ] Svelte 5 Runes compatible with `@thisux/sveltednD` (class-based drag
      state)
- [ ] SSE subscription and event serialisation for inventory changes
- [ ] `GET /v1/tape-chart` query — joins `reservation_items` +
      `maintenance_blocks` into unified type
- [ ] Move/resize validation — enforces state machine, availability, maintenance
      overlap (including partial-conflict full revert)
- [ ] SSE event types: `block_created`, `block_moved`, `block_resized`,
      `block_status_changed`, `block_deleted`
- [ ] SSE event IDs (`id:` field) + client replay on `Last-Event-Id`
- [ ] SSE reconciliation suppressed on blocks with active drag; resumed on drag
      end
- [ ] Multi-room select + batch drag (Shift+click range, Ctrl+click toggle)
- [ ] Date-range max 90-day on `GET /v1/tape-chart`
- [ ] 5s Redis TTL on availability check counts, invalidate on inventory write
- [ ] Composite index on
      `room_inventory_ledger(property_id, room_type_id, calendar_date)`
- [ ] Static headers: left-pin room column + top-pin date row
