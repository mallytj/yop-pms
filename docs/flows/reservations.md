# Reservation Flows

Sequence diagrams for every significant reservation lifecycle path. Each diagram
maps to one or more edge cases in `docs/requirements/reservations.md §8`.

> **`stay_period` time semantics (ADR-018).** Bounds are TSTZRANGE values
> pinned to property `default_checkin_time` (lower) and
> `default_checkout_time` (upper) — **not** midnight. API request bodies
> accept `arrival_date`/`departure_date` as DATE; server composes the range.
> Same-day turnover protection emerges from the gap between checkout and
> check-in times. `housekeeping_buffer_minutes` is advisory only.

> **Idempotency-Key handling.** All POST/PATCH endpoints share an
> `Idempotency-Key` middleware (R-RES-VALID-004). Concurrent retries with
> the same key receive `409 IDEMPOTENCY_IN_PROGRESS` (R-RES-EDGE-044).
> Diagrams show the Redis SET NX step where it makes the flow easier to
> follow but it is the same middleware in every case.

## Table of Contents

- [1. Creation Flows](#1-creation-flows)
  - [1.1 Website hold → payment success → confirmed](#11-website-hold-payment-success-confirmed)
  - [1.2 Website hold → abandoned → auto-cancelled](#12-website-hold-abandoned-auto-cancelled)
  - [1.3 Internal (staff) hold → confirmed](#13-internal-staff-hold-confirmed)
  - [1.4 Walk-in → checked_in (atomic)](#14-walk-in-checked_in-atomic)
  - [1.5 OTA inbound → async worker → confirmed](#15-ota-inbound-async-worker-confirmed)
  - [1.6 Website booking → stale availability → conflict with suggestions](#16-website-booking-stale-availability-conflict-with-suggestions)
- [2. Mutation Flows](#2-mutation-flows)
  - [2.1 Item date change](#21-item-date-change)
  - [2.2 Room assignment / reassignment](#22-room-assignment--reassignment)
  - [2.3 Post-checkin room move](#23-post-checkin-room-move)
  - [2.4 Metadata update](#24-metadata-update)
  - [2.5 Mid-stay room type change](#25-mid-stay-room-type-change)
  - [2.6 Rate change during stay](#26-rate-change-during-stay)
  - [2.7 Add item to existing reservation](#27-add-item-to-existing-reservation)
- [3. Check-in / Check-out](#3-check-in-check-out)
  - [3.1 Whole-reservation check-in](#31-whole-reservation-check-in)
  - [3.2 Per-item check-in (partial, multi-room)](#32-per-item-check-in-partial-multi-room)
  - [3.3 Whole-reservation check-out](#33-whole-reservation-check-out)
  - [3.4 Early check-out (stay period shortened)](#34-early-check-out-stay-period-shortened)
  - [3.5 Batch item status update (207 Multi-Status)](#35-batch-item-status-update-207-multi-status)
  - [3.6 Overstay detection and resolution](#36-overstay-detection-and-resolution)
- [4. Terminal Flows](#4-terminal-flows)
  - [4.1 Cancellation with fee](#41-cancellation-with-fee)
  - [4.2 Cancellation with waived fee](#42-cancellation-with-waived-fee)
  - [4.3 No-show — manual](#43-no-show--manual)
  - [4.4 No-show — sweep worker](#44-no-show--sweep-worker)
  - [4.5 Reactivation — availability pass](#45-reactivation--availability-pass)
  - [4.6 Item-level cancel (multi-room)](#46-item-level-cancel-multi-room)
  - [4.7 Concurrent cancel + update race](#47-concurrent-cancel--update-race)
- [5. Background Workers](#5-background-workers)
  - [5.1 Hold expiry sweep (R-RES-INTEG-007)](#51-hold-expiry-sweep-r-res-integ-007)
  - [5.2 Archival sweep (R-RES-INTEG-008)](#52-archival-sweep-r-res-integ-008)
- [6. Availability Check](#6-availability-check)
  - [6.1 Happy path](#61-happy-path)
  - [6.2 Conflict path (R-RES-AVAIL-006)](#62-conflict-path-r-res-avail-006)
  - [6.3 Concurrent booking race (R-RES-EDGE-047)](#63-concurrent-booking-race-r-res-edge-047)
- [7. Inventory Conflicts](#7-inventory-conflicts)
  - [7.1 Maintenance block rejected by active reservation (R-RES-EDGE-008)](#71-maintenance-block-rejected-by-active-reservation-r-res-edge-008)

**Participant legend (consistent across all diagrams)**

| Participant       | Represents                               |
| ----------------- | ---------------------------------------- |
| `Staff` / `Guest` | Human actor (browser/tablet)             |
| `API`             | Go HTTP handler                          |
| `DB`              | PostgreSQL (all schemas)                 |
| `Redis`           | Cache + idempotency store                |
| `Worker`          | Background worker (outbox / TTL / sweep) |
| `Ext`             | External system (SMTP, payment provider) |

---

## 1. Creation Flows

### 1.1 Website hold → payment success → confirmed

Guest books via website. Reservation born in `hold`. Payment intent created. On
payment success, reservation transitions to `confirmed`.

```mermaid
sequenceDiagram
    actor Guest
    participant API
    participant DB as PostgreSQL
    participant Redis
    participant Ext as Payment Provider
    participant Worker

    Guest->>API: POST /reservations {source=website, room_type, dates, guest_payload, payment_method_token}
    API->>Redis: SET NX idempotency:{key} (2min TTL, in-progress) — collision returns 409 IDEMPOTENCY_IN_PROGRESS
    API->>Ext: Authorize card (no capture) — ADR-019
    Ext-->>API: auth_id, expires_at
    API->>DB: BEGIN
    API->>DB: SELECT availability (ledger aggregate count)
    API->>DB: SELECT room FOR UPDATE SKIP LOCKED (deterministic auto-pin)
    API->>DB: INSERT guest (if inline, email dedup)
    API->>DB: INSERT reservation (status=hold, source=website)
    API->>DB: INSERT reservation_item (assigned_room_id=NULL)
    API->>DB: INSERT ledger rows (status=sold, auto-pinned room, one row per date)
    API->>DB: INSERT booked_daily_rates (one per night)
    API->>DB: INSERT folio A (balance=0)
    API->>DB: INSERT payment_authorization (provider, auth_id, expires_at, status=authorized)
    API->>DB: INSERT outbox_events (type=audit_log)
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: cache reservation:{id}
    API->>Redis: SET idempotency:{key} completed response (24h TTL)
    API-->>Guest: 201 {reservation envelope, payment_authorization_id}

    Guest->>Ext: Submit payment (capture against existing auth)
    Ext->>API: POST /payments/webhook {auth_id, captured=true}
    API->>Ext: Capture auth (auth_id) — server-confirmed
    API->>DB: BEGIN
    API->>DB: UPDATE reservation SET status=confirmed
    API->>DB: UPDATE payment_authorization SET captured_at=NOW()
    API->>DB: INSERT outbox_events (type=confirmation_email)
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Ext: 200 OK

    Worker->>DB: poll outbox_events
    Worker->>Ext: send confirmation email (SMTP)
    Worker->>DB: UPDATE outbox_events status=completed

    Note over API: On ROLLBACK paths (ledger UNIQUE violation, validation fail), API->>Ext: void(auth_id) before responding. Auth is **authorized** at hold-create and **captured** in the webhook step above. Cancel before capture → void via provider. TTL expiry → void via worker (1.2).
```

---

### 1.2 Website hold → abandoned → auto-cancelled

Guest starts booking but never completes payment. Hold TTL expires. Worker
cancels.

```mermaid
sequenceDiagram
    actor Guest
    participant API
    participant DB as PostgreSQL
    participant Redis
    participant Worker

    Guest->>API: POST /reservations {source=website}
    API->>DB: INSERT reservation (status=hold) + ledger rows + checkout_session
    API-->>Guest: 201 {reservation envelope, payment_authorization_id}

    Note over Guest: Guest abandons — never completes payment

    loop Every 30s (R-RES-WORKER-001)
        Worker->>DB: SELECT reservations WHERE status=hold AND expires_at < NOW() FOR UPDATE SKIP LOCKED LIMIT 100
        DB-->>Worker: [stale hold reservation]
        Worker->>DB: BEGIN
        Worker->>DB: UPDATE reservation SET status=cancelled, deleted_at=NOW()
        Worker->>DB: UPDATE reservation_items SET status=cancelled
        Worker->>DB: DELETE ledger rows for reservation_id
        Worker->>DB: UPDATE payment_authorization SET voided_at=NOW()
        Worker->>DB: INSERT outbox_events (type=void_auth, payload={auth_id, provider})
        Note over Worker: Provider void runs out-of-band via outbox dispatcher (R-RES-INTEG-004) — retried with exponential backoff. Tx commits cancel even if provider void temporarily unavailable.
        Worker->>DB: NOTIFY reservation_changes
        Worker->>DB: COMMIT
        Worker->>Redis: INVALIDATE reservation:{id}
    end

    Note over DB: No folio transaction — hold cancellation per R-RES-VALID-014
```

---

### 1.3 Internal (staff) hold → confirmed

Staff drags across calendar. Reservation created in `hold`, no payment. Staff
reviews and confirms.

> **Hold-as-scaffold pattern.** Because ledger rows are inserted at hold
> creation, the room is immediately blocked from website and OTA bookings. Staff
> can safely take time entering guest details before confirming — no race with
> concurrent bookings.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Staff: Step 1 — lock the room before entering guest details

    Staff->>API: POST /reservations {source=internal, room_type, dates}
    API->>Redis: SET NX idempotency:{key}
    API->>DB: BEGIN
    API->>DB: SELECT availability
    API->>DB: SELECT room FOR UPDATE SKIP LOCKED (deterministic auto-pin)
    API->>DB: INSERT reservation (status=hold, source=internal, guest_id=NULL)
    API->>DB: INSERT reservation_item (assigned_room_id=NULL)
    API->>DB: INSERT ledger rows (status=sold, auto-pinned room)
    API->>DB: INSERT booked_daily_rates
    API->>DB: INSERT folio A
    API->>DB: INSERT outbox_events (type=audit_log)
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: cache reservation:{id}
    API-->>Staff: 201 {reservation_id, status=hold}

    Note over DB: guest_id nullable on hold — NOT NULL enforced at confirm only

    Note over Staff: Step 2 — attach guest (staff takes their time, room is locked)

    Staff->>API: PATCH /reservations/{id} {guest_id | guest_payload}
    API->>DB: BEGIN
    API->>DB: INSERT guest (if guest_payload — inline, email dedup)
    API->>DB: UPDATE reservation SET guest_id=..., version=version+1
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {reservation}

    Note over Staff: Step 3 — confirm when ready

    Staff->>API: POST /reservations/{id}/confirm If-Match: {version}
    API->>API: Check permission reservations:confirm
    API->>DB: BEGIN
    API->>DB: Validate guest_id NOT NULL
    API->>DB: UPDATE reservation SET status=confirmed, version=version+1
    API->>DB: INSERT outbox_events (type=confirmation_email)
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {reservation}

    Note over DB: Hold expires after internal_hold_ttl_seconds if not confirmed (worker cleans up)
```

---

### 1.4 Walk-in → checked_in (atomic)

Guest arrives without prior reservation. Staff creates walk-in directly in
`checked_in`.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis
    participant Worker

    Staff->>API: POST /reservations {source=internal, is_walkin=true, assigned_room_id=101, arrival_date=today, departure_date=tomorrow, rate_plan_id?, guest_payload}
    API->>Redis: SET NX idempotency:{key}
    API->>DB: BEGIN
    API->>DB: Validate lower(stay_period) = today
    API->>DB: Validate assigned_room_id NOT NULL
    API->>DB: INSERT guest (inline, email dedup)
    API->>DB: INSERT reservation (status=checked_in, source=internal)
    API->>DB: INSERT reservation_item (status=checked_in, assigned_room_id=101)
    API->>DB: INSERT ledger rows (status=sold, room=101)
    API->>DB: INSERT booked_daily_rates
    API->>DB: INSERT folio A
    API->>DB: INSERT outbox_events (type=audit_log)
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: cache reservation:{id}
    API-->>Staff: 201 {reservation, status=checked_in}

    Note over API: No hold phase — single atomic transition to checked_in
    Note over API: rate_plan_id resolves via property_settings.walkin_rate_plan_id when omitted (R-RES-CRUD-014). Per-night discount post-create via PATCH .../booked-rates
    Note over DB: IF assigned_room_id has conflicting ledger row → ROLLBACK → 409
```

---

### 1.5 OTA inbound → async worker → confirmed

OTA pushes reservation via signed webhook. Acked in <200ms. Worker maps payload
and creates reservation.

```mermaid
sequenceDiagram
    participant OTA as OTA Channel
    participant API
    participant DB as PostgreSQL
    participant Worker
    participant Redis

    OTA->>API: POST /channels/{channel_id}/webhook {X-Yop-Signature: hmac, payload}
    API->>API: Verify HMAC-SHA256 (constant-time compare)
    API->>DB: SELECT ota_inbound_messages WHERE (channel_id, channel_message_id) — dedup check
    alt Already processed
        API-->>OTA: 200 OK (cached response)
    else New message
        API->>DB: INSERT ota_inbound_messages (status=pending, payload=jsonb)
        API-->>OTA: 202 Accepted {message_ref}
    end

    loop Outbox-style processing
        Worker->>DB: SELECT ota_inbound_messages WHERE status=pending FOR UPDATE SKIP LOCKED
        Worker->>Worker: Map channel payload → canonical reservation shape
        Worker->>DB: BEGIN
        Worker->>DB: SELECT availability
        Worker->>DB: SELECT room FOR UPDATE SKIP LOCKED (deterministic auto-pin)
        Worker->>DB: INSERT guest (inline, email dedup)
        Worker->>DB: INSERT reservation (status=confirmed, source=ota)
        Worker->>DB: INSERT reservation_item
        Worker->>DB: INSERT ledger rows (status=sold)
        Worker->>DB: INSERT booked_daily_rates
        Worker->>DB: INSERT folio A
        Worker->>DB: INSERT outbox_events (type=confirmation_email)
        Worker->>DB: NOTIFY reservation_changes
        Worker->>DB: UPDATE ota_inbound_messages status=processed, response=jsonb
        Worker->>DB: COMMIT
        Worker->>Redis: cache reservation:{id}
    end

    Note over Worker: On failure: retry with exponential backoff, dead-letter after N attempts
    Note over DB: No checkout_session — OTA already paid
```

---

### 1.6 Website booking → stale availability → conflict with suggestions

Guest attempts to book based on a search result from 5 minutes ago. In the
meantime, the last room was sold. API returns 409 with alternative suggestions.

```mermaid
sequenceDiagram
    actor Guest
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Guest: Guest searches Double Room for June 1-3. Availability: 1 left.
    Note over Guest: Guest takes 5 mins to fill form. Meanwhile, another guest books the last room.

    Guest->>API: POST /reservations {room_type=double, dates=June 1-3, ...}
    API->>DB: BEGIN
    API->>DB: SELECT availability (ledger aggregate count)
    DB-->>API: {June 1: 0 left, June 2: 0 left}

    API->>DB: SELECT alternatives (find next available dates or different types)
    DB-->>API: suggestions: ["June 5-7 (Double)", "June 1-3 (Suite)"]

    API->>DB: ROLLBACK
    API-->>Guest: 409 Conflict {code: "CONFLICT", message: "...", suggestions: [...]}

    Note over Guest: UI catches 409. Renders "Sorry, those dates just sold out!"
    Note over Guest: UI displays buttons for: [Book June 5-7] [Switch to Suite]
```

---

## 2. Mutation Flows

### 2.1 Item date change

Stay period updated. Triggers availability re-check, ledger row move,
booked_daily_rates recompute. Single transaction.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Staff->>API: PATCH /reservations/{id}/items/{item_id} {stay_period: new_range} If-Match: {item_version}
    API->>DB: SELECT reservation_item WHERE id=item_id — check version matches
    alt Version mismatch
        API-->>Staff: 412 Precondition Failed {current_version}
    else Version match
        API->>DB: BEGIN
        API->>DB: SELECT availability for new dates (exclude self from ledger check)
        alt Dates unavailable
            API->>DB: ROLLBACK
            API-->>Staff: 409 Conflict {conflicting_dates}
        else Available
            API->>DB: DELETE ledger rows WHERE reservation_id=id (old dates)
            API->>DB: INSERT ledger rows for new stay_period (same pinned room)
            API->>DB: DELETE booked_daily_rates WHERE reservation_item_id=item_id
            API->>DB: INSERT booked_daily_rates for new nights
            API->>DB: UPDATE reservation_item SET stay_period=new, version=version+1
            API->>DB: UPDATE reservation SET version=version+1
            API->>DB: NOTIFY reservation_changes
            API->>DB: COMMIT
            API->>Redis: INVALIDATE reservation:{id}
            API-->>Staff: 200 {updated item}
        end
    end

    Note over API: Post-checkin date PATCH (R-RES-EDGE-002/051): requires reservations:post_checkin_mutate. Extension allowed. Shortening governed by 3.4. Past nights immutable.
```

---

### 2.2 Room assignment / reassignment

Staff assigns or reassigns the specific room for a reservation item. Covers both
initial assignment (auto-pinned ledger slot → explicit room) and changing to a
different room of the same type pre-checkin. Post-checkin room move (guest
physically relocates) is flow 2.3.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Staff->>API: PATCH /reservations/{id}/items/{item_id}/assign-room {room_id: 101}
    API->>DB: BEGIN
    API->>DB: SELECT room — validate room is same type as booked_room_type_id
    API->>DB: SELECT ledger rows WHERE room_id=101 AND dates overlap — conflict check
    alt Room conflict
        API->>DB: ROLLBACK
        API-->>Staff: 409 Conflict {conflicting_reservation}
    else Room free
        API->>DB: UPDATE ledger rows SET room_id=101 WHERE reservation_item_id=item_id
        API->>DB: UPDATE reservation_item SET assigned_room_id=101, version=version+1
        API->>DB: NOTIFY reservation_changes
        API->>DB: COMMIT
        API->>Redis: INVALIDATE reservation:{id}
        API-->>Staff: 200 {updated item}
    end

    Note over DB: EXCLUDE GIST on assigned_room_id catches concurrent assignment race
    Note over DB: Same endpoint for initial assign and reassignment — ledger UPDATE replaces room_id in place
```

---

### 2.3 Post-checkin room move

Guest physically relocates to a different room during their stay. Requires
`reservations:post_checkin_mutate`. Same endpoint as 2.2 but permission-gated.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Staff: Guest currently in room 101. Moving to room 102 mid-stay.

    Staff->>API: PATCH /reservations/{id}/items/{item_id}/assign-room {room_id: 102} If-Match: {item_version}
    API->>API: Check permission reservations:post_checkin_mutate
    alt No permission
        API-->>Staff: 403 Forbidden
    else Permitted
        API->>DB: BEGIN
        API->>DB: SELECT room 102 — validate same type as booked_room_type_id
        API->>DB: SELECT ledger rows WHERE room_id=102 AND dates overlap — conflict check
        alt Room 102 conflict
            API->>DB: ROLLBACK
            API-->>Staff: 409 Conflict {conflicting_reservation}
        else Room 102 free
            API->>DB: UPDATE ledger rows SET room_id=102 WHERE reservation_item_id=item_id
            API->>DB: UPDATE reservation_item SET assigned_room_id=102, version=version+1
            API->>DB: NOTIFY reservation_changes
            API->>DB: COMMIT
            API->>Redis: INVALIDATE reservation:{id}
            API-->>Staff: 200 {updated item}
        end
    end

    Note over Staff: Housekeeping notified separately — physical room move is outside system scope
```

---

### 2.4 Metadata update

Simple fields (notes, travel_agent_id, group_id, primary_guest_id). No availability check. No
ledger change.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Staff->>API: PATCH /reservations/{id} {notes: "...", travel_agent_id: ...} If-Match: {version}
    API->>DB: SELECT reservation — check version + not cancelled
    alt Version mismatch
        API-->>Staff: 412 Precondition Failed
    else
        API->>DB: BEGIN
        API->>DB: UPDATE reservation SET notes=..., travel_agent_id=..., version=version+1
        API->>DB: NOTIFY reservation_changes
        API->>DB: COMMIT
        API->>Redis: INVALIDATE reservation:{id}
        API-->>Staff: 200 {reservation}
    end
```

---

### 2.5 Mid-stay room type change

Guest upgrades or downgrades room type during their stay. Ledger split: past
dates on original room are preserved; future dates move to new type. Requires
`reservations:post_checkin_mutate`.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Staff: Guest booked Double Mon-Fri. Checked in Mon. Upgrades to Suite from Wed.

    Staff->>API: PATCH /reservations/{id}/items/{item_id} {booked_room_type_id: suite_id} If-Match: {item_version}
    API->>API: Check permission reservations:post_checkin_mutate
    API->>DB: BEGIN
    API->>DB: SELECT availability for Suite on remaining nights (today onwards)
    alt Suite unavailable
        API->>DB: ROLLBACK
        API-->>Staff: 409 Conflict {conflicting_dates}
    else Suite available
        API->>DB: SELECT suite room FOR UPDATE SKIP LOCKED (auto-pin future dates only)
        API->>DB: DELETE ledger rows WHERE reservation_item_id=id AND calendar_date >= today
        API->>DB: INSERT ledger rows for new suite room (calendar_date >= today, status=sold)
        API->>DB: DELETE booked_daily_rates WHERE reservation_item_id=id AND calendar_date >= today
        API->>DB: INSERT booked_daily_rates for suite from today (new rate plan lookup)
        API->>DB: UPDATE reservation_item SET booked_room_type_id=suite_id, version=version+1
        API->>DB: NOTIFY reservation_changes
        API->>DB: COMMIT
        API->>Redis: INVALIDATE reservation:{id}
        API-->>Staff: 200 {updated item}
    end

    Note over DB: Past ledger rows (Mon-Tue on Double) preserved — history immutable post-checkin
    Note over Staff: Physical room move handled separately via 2.3 (assign-room) if needed
```

---

### 2.6 Rate change during stay

Staff applies a different rate plan to remaining nights. Past
`booked_daily_rates` rows are untouched (snapshot). Future rows deleted and
recomputed. Pre-checkin variant: full recompute, no permission guard.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Staff: Guest on BAR rate. Staff applies promotional rate from today onwards.

    Staff->>API: PATCH /reservations/{id}/items/{item_id} {rate_plan_id: promo_id} If-Match: {item_version}
    API->>API: Check permission reservations:post_checkin_mutate (if status=checked_in)
    API->>DB: BEGIN
    API->>DB: SELECT price grid for promo_id across remaining nights
    API->>DB: DELETE booked_daily_rates WHERE reservation_item_id=id AND calendar_date >= today
    API->>DB: INSERT booked_daily_rates for remaining nights at new rate
    API->>DB: UPDATE reservation_item SET rate_plan_id=promo_id, version=version+1
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {updated item}

    Note over DB: Past booked_daily_rates rows untouched — snapshot preserved per R-RES-EDGE-038
    Note over DB: Pre-checkin (R-RES-EDGE-039): same flow, all nights recomputed, no permission guard
```

---

### 2.7 Add item to existing reservation

Receptionist adds an extra room to an existing non-terminal reservation.
Common scenario: guest's group expands, or replacement for a previously
cancelled item (per `CONTEXT.md` — cancelled items immutable; new item
appended). Requires `reservations:add_item`.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Staff->>API: POST /reservations/{id}/items {arrival_date, departure_date, booked_room_type_id, rate_plan_id, guest_id?, adults_count, children_count} If-Match: {version}
    API->>API: Check permission reservations:add_item
    API->>DB: SELECT reservation — validate status NOT IN (cancelled, checked_out, archived)
    alt Terminal status
        API-->>Staff: 409 Conflict {code: TERMINAL_RESERVATION}
    else Non-terminal
        API->>DB: BEGIN
        API->>DB: SELECT availability for new item dates
        alt Unavailable
            API->>DB: ROLLBACK
            API-->>Staff: 409 Conflict {conflicting_dates}
        else Available
            API->>DB: SELECT room FOR UPDATE SKIP LOCKED (auto-pin)
            API->>DB: INSERT reservation_item (status=booked)
            API->>DB: INSERT ledger rows (status=sold)
            API->>DB: INSERT booked_daily_rates (one per night)
            API->>DB: Recompute reservation.stay_period_envelope (ADR-020)
            API->>DB: UPDATE reservation SET stay_period_envelope=…, version=version+1
            API->>DB: NOTIFY reservation_changes
            API->>DB: COMMIT
            API->>Redis: INVALIDATE reservation:{id}
            API-->>Staff: 201 {reservation envelope}
        end
    end

    Note over DB: Cancelled items remain cancelled — immutable history. New item gets new id
    Note over API: Per Q11 lean-envelope rule, response = full GET /{id} body
```

---

## 3. Check-in / Check-out

### 3.1 Whole-reservation check-in

All items transition to `checked_in` in one call. Room assignment required on
all items.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Staff->>API: PATCH /reservations/{id}/checkin If-Match: {version}
    API->>DB: SELECT all reservation_items WHERE reservation_id=id
    API->>DB: Validate ALL items have assigned_room_id NOT NULL
    alt Any item unassigned
        API-->>Staff: 409 Conflict {unassigned_item_ids}
    else All assigned
        API->>DB: BEGIN
        API->>DB: UPDATE reservation_items SET status=checked_in (all booked items)
        API->>DB: Run rollup → reservation status=checked_in
        API->>DB: UPDATE reservation SET status=checked_in, version=version+1
        API->>DB: NOTIFY reservation_changes
        API->>DB: COMMIT
        API->>Redis: INVALIDATE reservation:{id}
        API-->>Staff: 200 {reservation, status=checked_in}
    end
```

---

### 3.2 Per-item check-in (partial, multi-room)

One guest of a two-room reservation arrives early. Item checked in
independently.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Staff: Two-room reservation. Guest A arrives. Guest B arrives tomorrow.

    Staff->>API: PATCH /reservations/{id}/items/{item_a}/checkin If-Match: {item_version}
    API->>DB: SELECT item_a — validate assigned_room_id NOT NULL, status=booked
    API->>DB: BEGIN
    API->>DB: UPDATE reservation_item item_a SET status=checked_in, version=version+1
    API->>DB: Run rollup — item_b still booked → reservation stays confirmed
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {item, status=checked_in}

    Note over DB: Reservation stays confirmed until ALL items leave booked state (ADR-016 rollup)

    Staff->>API: PATCH /reservations/{id}/items/{item_b}/checkin If-Match: {item_version}
    API->>DB: UPDATE item_b SET status=checked_in
    API->>DB: Run rollup — no items booked → reservation=checked_in
    API->>DB: UPDATE reservation SET status=checked_in, version=version+1
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {item, status=checked_in}
```

---

### 3.3 Whole-reservation check-out

All items transition to `checked_out`. Reservation rolls up to `checked_out`.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis
    participant Worker

    Staff->>API: PATCH /reservations/{id}/checkout If-Match: {version}
    API->>DB: BEGIN
    API->>DB: UPDATE reservation_items SET status=checked_out (all checked_in items)
    API->>DB: DELETE future ledger rows WHERE date > today (early checkout — R-RES-EDGE-035)
    API->>DB: Run rollup → all items terminal + ≥1 checked_out → reservation=checked_out
    API->>DB: UPDATE reservation SET status=checked_out, version=version+1
    API->>DB: INSERT outbox_events (type=checkout_receipt_email)
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {reservation, status=checked_out}

    Worker->>Ext: send checkout receipt (SMTP)
    Worker->>DB: UPDATE outbox_events status=completed
```

---

### 3.4 Early check-out (stay period shortened)

Guest leaves before scheduled departure. Future ledger rows released.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Staff: Guest booked Mon-Fri, leaves Wednesday.

    Staff->>API: PATCH /reservations/{id}/items/{item_id} {stay_period: Mon-Wed} If-Match: {item_version}
    API->>DB: Validate shortened stay (upper must not exceed original checkout - post-checkin rule)
    API->>DB: BEGIN
    API->>DB: DELETE future ledger rows (calendar_date on or after new_checkout, reservation_id=id)
    API->>DB: DELETE future booked_daily_rates (calendar_date on or after new_checkout)
    API->>DB: UPDATE reservation_item SET stay_period=Mon-Wed, version=version+1
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {item}

    Note over DB: Released Thu/Fri ledger rows immediately available for new bookings
```

---

### 3.5 Batch item status update (207 Multi-Status)

Whole-reservation check-in where some items are in a terminal or incompatible
state. Valid items transition; invalid items reported per-item. Single
transaction for the valid subset.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Staff: 3-item reservation. Item A: booked+assigned. Item B: no_show. Item C: booked+assigned.

    Staff->>API: PATCH /reservations/{id}/checkin If-Match: {version}
    API->>DB: SELECT all reservation_items WHERE reservation_id=id
    API->>API: Validate each item — status=booked AND assigned_room_id NOT NULL

    Note over API: Item B fails validation (no_show is terminal). A and C pass.

    API->>DB: BEGIN
    API->>DB: UPDATE reservation_item item_a SET status=checked_in, version=version+1
    API->>DB: UPDATE reservation_item item_c SET status=checked_in, version=version+1
    API->>DB: Run rollup — item_b terminal (no_show), items a+c checked_in → reservation=checked_in
    API->>DB: UPDATE reservation SET status=checked_in, version=version+1
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}

    API-->>Staff: 207 Multi-Status
    Note over API,Staff: {results: [{id: itemA, status: checked_in}, {id: itemB, error: INVALID_TRANSITION, reason: no_show is terminal}, {id: itemC, status: checked_in}]}
```

---

### 3.6 Overstay detection and resolution

Worker flags items still `checked_in` past
`upper(stay_period) + late_checkout_grace_minutes`. Receptionist resolves
either by extending the stay (PATCH dates) or forcing a checkout. Mid-stay
reservation cancel is forbidden (R-RES-VALID-013) — must check out first.

```mermaid
sequenceDiagram
    participant Worker
    participant DB as PostgreSQL
    participant Redis
    actor Staff
    participant API

    loop Every N minutes (R-RES-WORKER-005)
        Worker->>DB: SELECT reservation_items WHERE status=checked_in AND now() > upper(stay_period) + (property.late_checkout_grace_minutes * interval '1 minute') FOR UPDATE SKIP LOCKED LIMIT 100
        DB-->>Worker: [overdue items]
        loop For each item
            Worker->>DB: BEGIN
            Worker->>DB: UPDATE reservation_item SET status=overstay, version=version+1
            Worker->>DB: NOTIFY reservation_changes
            Worker->>DB: COMMIT
            Worker->>Redis: INVALIDATE reservation:{reservation_id}
        end
    end

    Note over Staff: Overstay surfaces on receptionist dashboard. Two resolution paths:

    alt Extend stay
        Staff->>API: PATCH /reservations/{id}/items/{item_id} {departure_date: new_date} If-Match: {item_version}
        API->>API: Check permission reservations:post_checkin_mutate
        API->>DB: BEGIN
        API->>DB: SELECT availability for extension dates (exclude self)
        alt Available
            API->>DB: INSERT ledger rows for extension nights
            API->>DB: INSERT booked_daily_rates for extension nights
            API->>DB: UPDATE reservation_item SET stay_period=new, status=checked_in, version=version+1
            Note over DB: status returns from `overstay` to `checked_in` — extension is the actor-side resolution path
            API->>DB: Recompute reservation envelope
            API->>DB: NOTIFY reservation_changes
            API->>DB: COMMIT
            API->>Redis: INVALIDATE reservation:{id}
            API-->>Staff: 200 {item, status=checked_in}
        else Unavailable
            API->>DB: ROLLBACK
            API-->>Staff: 409 {conflicting_dates, suggest_room_move: true}
            Note over Staff: Receptionist initiates room move (flow 2.3) for the overstaying guest
        end
    else Force checkout
        Staff->>API: PATCH /reservations/{id}/items/{item_id}/checkout If-Match: {item_version}
        API->>DB: BEGIN
        API->>DB: UPDATE reservation_item SET status=checked_out, version=version+1
        API->>DB: DELETE ledger rows WHERE reservation_item_id=item_id AND calendar_date >= now()::date (defensive — original future is empty post-overstay; covers extension nights if guest was extended then forced out)
        API->>DB: Run rollup
        API->>DB: NOTIFY reservation_changes
        API->>DB: COMMIT
        API->>Redis: INVALIDATE reservation:{id}
        API-->>Staff: 200 {item, status=checked_out}
        Note over Staff: Late-checkout fee posted as manual folio_transaction (finance PR)
    end

    Note over DB: Overstay collision with incoming booking → R-RES-EDGE-058
```

---

## 4. Terminal Flows

### 4.1 Cancellation with fee

Staff cancels reservation. Cancellation fee posted to Folio A.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis
    participant Worker
    participant Ext as SMTP

    Staff->>API: POST /reservations/{id}/cancel {reason_code, fee_pence=5000, waive_fee=false, fee_override_reason?, refund_action=refund_deposit} If-Match: {version}
    API->>DB: SELECT reservation + items — validate status not terminal (409 if checked_out/archived/cancelled — R-RES-EDGE-040) AND no item is checked_in (409 R-RES-VALID-013)
    Note over API,Staff: 409 body for R-RES-VALID-013: {code: "RESERVATION_HAS_CHECKED_IN_ITEMS", checked_in_item_ids: [uuid, ...], remediation: "Check-out or shorten the listed items, then retry cancel."}
    API->>DB: BEGIN
    API->>DB: UPDATE reservation SET status=cancelled, deleted_at=NOW(), version=version+1
    API->>DB: UPDATE reservation_items SET status=cancelled (all non-terminal)
    API->>DB: DELETE ledger rows WHERE reservation_id=id AND calendar_date >= today
    API->>DB: INSERT folio_transaction (folio_a, description=cancellation_fee, amount=5000, status=pending)
    API->>DB: INSERT outbox_events (type=cancellation_email)
    API->>DB: INSERT outbox_events (type=audit_log)
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {reservation, status=cancelled}

    Worker->>Ext: send cancellation email
    Worker->>DB: UPDATE outbox_events completed

    Note over DB: refund_action=refund_deposit handled by payment provider integration (future)
```

---

### 4.2 Cancellation with waived fee

Manager waives the cancellation fee. Requires `reservations:waive_fee`
permission.

```mermaid
sequenceDiagram
    actor Manager
    participant API
    participant DB as PostgreSQL
    participant Redis

    Manager->>API: POST /reservations/{id}/cancel {reason_code=goodwill, fee_pence=5000, waive_fee=true} If-Match: {version}
    API->>API: Check permission reservations:waive_fee
    alt No permission
        API-->>Manager: 403 Forbidden
    else Permitted
        API->>DB: BEGIN
        API->>DB: UPDATE reservation SET status=cancelled, deleted_at=NOW()
        API->>DB: UPDATE reservation_items SET status=cancelled
        API->>DB: DELETE ledger rows
        Note over DB: No folio_transaction — fee waived
        API->>DB: INSERT outbox_events (type=cancellation_email)
        API->>DB: NOTIFY reservation_changes
        API->>DB: COMMIT
        API->>Redis: INVALIDATE reservation:{id}
        API-->>Manager: 200 {reservation, status=cancelled}
    end
```

---

### 4.3 No-show — manual

Staff marks a guest as no-show after check-in time passes.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Staff->>API: PATCH /reservations/{id}/items/{item_id}/no-show If-Match: {item_version}
    API->>API: Check permission reservations:mark_no_show
    API->>DB: Validate lower(stay_period) <= NOW() (can only mark after check-in time)
    API->>DB: BEGIN
    API->>DB: UPDATE reservation_item SET status=no_show, version=version+1
    API->>DB: DELETE future ledger rows for item (dates after today)
    API->>DB: Run rollup — if all items terminal → update reservation status
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {item, status=no_show}
```

---

### 4.4 No-show — sweep worker

Worker automatically marks unchecked-in items after grace period.

```mermaid
sequenceDiagram
    participant Worker
    participant DB as PostgreSQL
    participant Redis

    loop Daily (R-RES-WORKER-003)
        Worker->>DB: SELECT reservation_items WHERE status=booked AND lower(stay_period) + no_show_grace_minutes < NOW() FOR UPDATE SKIP LOCKED
        DB-->>Worker: [overdue booked items]
        loop For each item
            Worker->>DB: BEGIN
            Worker->>DB: UPDATE reservation_item SET status=no_show
            Worker->>DB: DELETE future ledger rows
            Worker->>DB: Run rollup
            Worker->>DB: NOTIFY reservation_changes
            Worker->>DB: COMMIT
            Worker->>Redis: INVALIDATE reservation:{reservation_id}
        end
    end
```

---

### 4.5 Reactivation — availability pass

Cancelled reservation reactivated. Dates still available.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Staff->>API: POST /reservations/{id}/reactivate If-Match: {version}
    API->>DB: SELECT reservation — validate status=cancelled
    API->>DB: BEGIN
    API->>DB: SELECT availability for original dates (full re-check)
    alt Dates available
        API->>DB: UPDATE reservation SET status=confirmed, deleted_at=NULL, version=version+1
        API->>DB: UPDATE reservation_items SET status=booked (all cancelled items)
        API->>DB: INSERT ledger rows (re-pin room, one per date)
        API->>DB: NOTIFY reservation_changes
        API->>DB: COMMIT
        API->>Redis: INVALIDATE reservation:{id}
        API-->>Staff: 200 {reservation, status=confirmed}
    else Dates unavailable
        API->>DB: ROLLBACK
        API-->>Staff: 409 Conflict {conflicting_dates}
    end

    Note over API: lower(stay_period) < today check fires before availability re-check (R-RES-EDGE-041). Returns 409 unless reservations:retroactive_create permission held.
```

---

### 4.6 Item-level cancel (multi-room)

One item in a multi-room reservation cancelled independently. Reservation
rollup unchanged unless all items reach a terminal state.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL
    participant Redis

    Note over Staff: 2-room reservation. Cancel item A only (Guest B staying).

    Staff->>API: POST /reservations/{id}/items/{item_id}/cancel {reason_code, fee_pence} If-Match: {item_version}
    API->>DB: SELECT reservation_item — validate status not terminal
    API->>DB: BEGIN
    API->>DB: UPDATE reservation_item SET status=cancelled, deleted_at=NOW(), version=version+1
    API->>DB: DELETE ledger rows WHERE reservation_item_id=item_id AND calendar_date >= today
    API->>DB: INSERT folio_transaction (cancellation_fee if fee_pence > 0)
    API->>DB: Run rollup — item_b still booked → reservation stays confirmed
    API->>DB: NOTIFY reservation_changes
    API->>DB: COMMIT
    API->>Redis: INVALIDATE reservation:{id}
    API-->>Staff: 200 {item, status=cancelled}

    Note over DB: Reservation status unchanged — rollup only promotes when ALL items reach terminal state
```

---

### 4.7 Concurrent cancel + update race

Two staff members act on the same reservation simultaneously. The second
committer receives 412 Precondition Failed regardless of which operation wins.

```mermaid
sequenceDiagram
    actor Staff1
    actor Staff2
    participant API
    participant DB as PostgreSQL

    Note over DB: Reservation version=5

    Staff1->>API: POST /reservations/{id}/cancel If-Match: version=5
    Staff2->>API: PATCH /reservations/{id} {notes: "VIP"} If-Match: version=5

    Note over API: Both read version=5. Cancel commits first.

    API->>DB: Staff1: BEGIN → UPDATE SET status=cancelled, version=6 → COMMIT
    API-->>Staff1: 200 {status=cancelled}

    API->>DB: Staff2: BEGIN → SELECT version → returns 6 (mismatch)
    API->>DB: Staff2: ROLLBACK
    API-->>Staff2: 412 Precondition Failed {current_version: 6, provided_version: 5}

    Note over Staff2: Reload reservation, observe cancelled status, reconcile accordingly
```

---

## 5. Background Workers

### 5.1 Hold expiry sweep (R-RES-INTEG-007)

Worker cancels stale holds. Per-source TTL from `operations.property_settings`.

```mermaid
sequenceDiagram
    participant Worker
    participant DB as PostgreSQL
    participant Redis

    loop Every 30s
        Worker->>DB: SELECT expired holds by source TTL (FOR UPDATE SKIP LOCKED LIMIT 100)
        Note over Worker: JOIN property_settings for website/internal TTL. Filters holds past their TTL. Multi-tiered: internal holds with a guest_id have longer grace.
        DB-->>Worker: [expired holds]
        loop For each expired hold
            Worker->>DB: BEGIN
            Worker->>DB: UPDATE reservation SET status=cancelled, deleted_at=NOW()
            Worker->>DB: UPDATE reservation_items SET status=cancelled
            Worker->>DB: DELETE ledger rows WHERE reservation_id=id
            Worker->>DB: UPDATE checkout_session SET status=expired (if website source)
            Worker->>DB: NOTIFY reservation_changes
            Worker->>DB: COMMIT
            Worker->>Redis: INVALIDATE reservation:{id}
        end
    end

    Note over Worker: SKIP LOCKED allows multiple worker replicas without contention
```

---

### 5.2 Archival sweep (R-RES-INTEG-008)

Worker archives old terminal reservations. Excludes from default list queries.

```mermaid
sequenceDiagram
    participant Worker
    participant DB as PostgreSQL
    participant Redis

    loop Daily
        Worker->>DB: SELECT terminal reservations past archive threshold (FOR UPDATE SKIP LOCKED LIMIT 500)
        Note over Worker: JOIN property_settings for archive_after_days, status IN (checked_out and cancelled)
        DB-->>Worker: [archivable reservations]
        loop For each reservation
            Worker->>DB: BEGIN
            Worker->>DB: UPDATE reservation SET status=archived
            Worker->>DB: UPDATE reservation_items SET status=archived, deleted_at=NOW()
            Worker->>DB: UPDATE folios SET deleted_at=NOW() (zero-balance only)
            Worker->>DB: NOTIFY reservation_changes
            Worker->>DB: COMMIT
            Worker->>Redis: INVALIDATE reservation:{id}
        end
    end

    Note over DB: Archived reservations excluded from list queries by default
    Note over DB: Opt-in via ?include_archived=true
```

---

## 6. Availability Check

### 6.1 Happy path

Rooms available for requested type and dates.

```mermaid
sequenceDiagram
    actor Client
    participant API
    participant Redis
    participant DB as PostgreSQL

    Client->>API: GET /reservations/availability?property_id=&room_type_id=&start_date=&end_date=
    API->>Redis: GET cache:availability:{property}:{type}:{from}:{to}
    alt Cache hit
        Redis-->>API: cached availability response
        API-->>Client: 200 {available: true, remaining_count: 3}
    else Cache miss
        API->>DB: SELECT count(rooms) WHERE room_type_id=... (total rooms in type)
        API->>DB: SELECT count(ledger rows) WHERE room.type=... AND calendar_date IN dates AND status IN (sold, on_hold, maintenance, decommissioned) GROUP BY calendar_date
        API->>DB: Check LOS restrictions from price grid (min_los, max_los, occupancy)
        API->>DB: Check housekeeping buffer (property_settings.housekeeping_buffer_minutes)
        DB-->>API: {total: 4, blocked_per_date: {2026-06-01: 1, 2026-06-02: 2}}
        API->>Redis: SET cache:availability:... (long TTL - safety net, invalidated via NOTIFY per ADR-010)
        API-->>Client: 200 {available: true, remaining_count: 2, per_date: {...}}
    end
```

---

### 6.2 Conflict path (R-RES-AVAIL-006)

No rooms available. Response includes which dates conflict.

```mermaid
sequenceDiagram
    actor Client
    participant API
    participant DB as PostgreSQL

    Client->>API: GET /reservations/availability?room_type_id=double&start_date=2026-06-01&end_date=2026-06-05
    API->>DB: SELECT aggregate availability per date
    DB-->>API: {2026-06-01: 0 remaining, 2026-06-02: 0 remaining, 2026-06-03: 1 remaining, ...}
    API-->>Client: 200 {available: false, conflicting_dates: [...], per_date: {...}}
    Note over Client: conflicting_dates lists fully blocked dates. Per-date gives total/blocked/remaining per night. Client can suggest alternative dates using per_date data.
```

---

### 6.3 Concurrent booking race (R-RES-EDGE-047)

Two clients see availability simultaneously. One wins; the other is caught by DB
constraint.

```mermaid
sequenceDiagram
    actor Guest1
    actor Guest2
    participant API
    participant DB as PostgreSQL

    Note over DB: 1 room remaining of type Double

    Guest1->>API: POST /reservations (last double, June 1-3)
    Guest2->>API: POST /reservations (last double, June 1-3)

    Note over API: Both pass aggregate availability check (read same snapshot)

    API->>DB: Guest1: BEGIN .. SELECT room FOR UPDATE SKIP LOCKED .. returns 101
    API->>DB: Guest2: BEGIN .. SELECT room FOR UPDATE SKIP LOCKED .. room 101 locked, returns NULL

    API->>DB: Guest2: ROLLBACK
    API-->>Guest2: 409 Conflict {conflicting_dates: ["2026-06-01", "2026-06-02"]}

    API->>DB: Guest1: INSERT ledger rows (room=101) + reservation + items
    API->>DB: Guest1: COMMIT
    API-->>Guest1: 201 {reservation}

    Note over DB: SKIP LOCKED ensures Guest2 doesn't wait for Guest1 to commit before failing.
    Note over DB: IF 2 rooms were available, Guest2 would have skipped 101 and picked 102 ✓
```

---

## 7. Inventory Conflicts

### 7.1 Maintenance block rejected by active reservation (R-RES-EDGE-008)

Staff attempts to place a maintenance block over dates already sold to a guest.
The DB ledger UNIQUE constraint prevents the overlap; the application layer
surfaces the conflict before insert.

```mermaid
sequenceDiagram
    actor Staff
    participant API
    participant DB as PostgreSQL

    Note over DB: Room 101 has reservation RES-000123 for June 1-5 (ledger status=sold).

    Staff->>API: POST /rooms/101/maintenance {block_period: June 3-7, type: deep_clean}
    API->>DB: SELECT ledger rows WHERE room_id=101 AND calendar_date IN (June 3-7) AND status=sold
    DB-->>API: {June 3, June 4, June 5 → sold to RES-000123}
    API-->>Staff: 409 Conflict {conflicting_reservations: ["RES-000123"], conflicting_dates: ["June 3","June 4","June 5"]}

    Note over Staff: Must reassign or cancel the conflicting reservation before placing maintenance block
```
