# Booking Implementation Plan

<!--toc:start-->

- [Booking Implementation Plan](#booking-implementation-plan)
  - [File layout](#file-layout)
  - [How files connect](#how-files-connect)
  - [Phase 0: Migrations](#phase-0-migrations)
  - [Phase 1: SQLC Queries](#phase-1-sqlc-queries)
  - [Phase 2: Types + State Machine](#phase-2-types-state-machine)
    - [`types.go` — ~150 LOC](#typesgo-150-loc)
    - [`errors.go` — domain sentinels (drop DomainError)](#errorsgo-domain-sentinels-drop-domainerror)
    - [`state_machine.go` — ~100 LOC](#statemachinego-100-loc)
  - [Phase 3: Service — Transactions + Create + Confirm](#phase-3-service-transactions-create-confirm)
    - [Transaction pattern: `ExecuteTx[T]` (platform helper)](#transaction-pattern-executetxt-platform-helper)
    - [`service.go` — ~250 LOC](#servicego-250-loc)
  - [Phase 4: Availability](#phase-4-availability)
    - [`availability.go` — ~120 LOC](#availabilitygo-120-loc)
  - [Phase 5: Validation via constraints](#phase-5-validation-via-constraints)
  - [Phase 6: Handlers](#phase-6-handlers)
  - [Phase 7: Remaining business logic](#phase-7-remaining-business-logic)
    - [`actions.go` — ~280 LOC](#actionsgo-280-loc)
    - [`mutations.go` — ~240 LOC](#mutationsgo-240-loc)
    - [`rates.go` — ~180 LOC](#ratesgo-180-loc)
  - [Phase 8: Router + Middleware](#phase-8-router-middleware)
    - [Auth (StubAuth) — `internal/platform/middleware/auth.go`](#auth-stubauth-internalplatformmiddlewareauthgo)
    - [Permissions: route-static vs body-conditional](#permissions-route-static-vs-body-conditional)
    - [If-Match — `internal/platform/middleware/ifmatch.go`](#if-match-internalplatformmiddlewareifmatchgo)
    - [`router.go` — ~90 LOC](#routergo-90-loc)
  - [Phase 9: SSE (rough sketch — finalised after research)](#phase-9-sse-rough-sketch-finalised-after-research)
  - [Phase 10: Workers](#phase-10-workers)
    - [Hold expiry](#hold-expiry)
    - [No-show](#no-show)
  - [Phase 11: Caching](#phase-11-caching)
  - [Implementation order](#implementation-order)
  - [Items intentionally NOT in this PR](#items-intentionally-not-in-this-pr)
  - [Risks & follow-ups](#risks-follow-ups)
  <!--toc:end-->

> Sources: `docs/flows/reservations.md` (sequence diagrams),
> `docs/requirements/reservations.md` (RTM), `docs/adr/*`, `AGENTS.md`.

PR scope: full reservation API in **one PR** — CRUD + mutations + actions +
rates + availability + workers + SSE scaffold. Payment auth, folios beyond stub,
OTA inbound, reservation groups, real auth are **deferred** to follow-up PRs.

---

## File layout

```text
internal/booking/
├── types.go                  enums, I/O structs (no DomainError — use apierror)
├── include.go                IncludeFlags, ParseIncludeFlags (unexported type, no functions in types.go)
├── errors.go                 *apierror.APIError sentinels for domain errors
├── state_machine.go          transitions, rollup (ADR-015), ActionIdempotency (§7.4)
├── availability.go           CheckAvailability, AutoPinRoom, ConflictCheck
├── service.go                Service struct + NewService + CreateReservation + ConfirmReservation
│                             (foundational CRUD; lifecycle/actions/mutations/rates split below)
├── actions.go                Cancel, Reactivate, Checkin*, Checkout*, MarkNoShow, CancelItem,
│                             ShortenStay (internal, called by UpdateItemStayPeriod when checked_in)
├── mutations.go              UpdateItemStayPeriod, AssignRoom, UpdateItemRoomType,
│                             UpdateItemRatePlan, AddItem, UpdateMetadata
├── rates.go                  GetBookedRates, OverrideNightlyRate, ApplyRateAdjustments,
│                             ApproveRateAdjustments
├── handlers_crud.go          Create, Get, List, UpdateMetadata, AddItem, UpdateItem,
│                             AssignRoom
├── handlers_lifecycle.go     Confirm, Cancel, Reactivate, CheckinReservation, CheckinItem,
│                             CheckoutReservation, CheckoutItem, MarkNoShow, CancelItem
├── handlers_rates.go         UpdateBookedRates, GetBookedRates, ApproveAdjustments,
│                             AdjustRate
├── handlers_misc.go          Availability, GetFolio (stub), CancellationQuote (stub)
├── router.go                 Routes(svc, ifMatchMW) chi.Router builder
├── workers.go                HoldExpirySweep, NoShowReminder, OverstaySweep, ArchivalSweep
└── *_test.go                 integration + unit tests

internal/platform/sse/
├── hub.go                    SSEHub{clients, mu, Subscribe, Broadcast, OnEvent}
└── hub_test.go

internal/platform/middleware/
├── ifmatch.go                RequireIfMatch — parses, puts version into ctx, mismatch → 412
├── auth.go                   StubAuth — X-Property-ID + X-User-Permissions → ctx
└── permission.go             RequirePermission(perm) middleware (403 on miss)

internal/platform/helpers/
├── db_errors.go              (existing)
├── context.go                NEW — ctx keys + GetPropertyIDFromCtx, GetPermissionsFromCtx,
│                             HasPermission, SetPropertyIDInCtx, SetPermissionsInCtx,
│                             SetIfMatchVersion, GetIfMatchVersion
└── psql_errors.go            NEW — PsqlErrToCustomErr wrapping apierror.MapStoreError

internal/platform/db/
└── tx.go                     NEW — generic ExecuteTx[T] helper with RLS set inside
```

**Rules:**

- One package `booking` — no sub-packages (avoids circular imports).
- Handlers are thin: `json.ReadJSON` → `validation.Struct` → domain checks →
  `s.Method(ctx, input)` → `json.WriteJSON`.
- Service methods talk directly to `*store.Queries` (concrete, SQLC-generated)
  via `ExecuteTx[T]`. No `Querier` interface field.
- State machine functions are pure — unit-testable with zero dependencies.
- File size ceiling: ~300 LOC. `handlers_lifecycle.go` may approach 400.
- Each implementation phase ships a companion `*_ec_test.go` file covering all
  applicable edge cases from `docs/requirements/reservations.md §8`.
  Active tests for implemented methods, `t.Skip()` + blocker note for deferred.
  See `service_ec_test.go` for the pattern.

---

## How files connect

```text
handlers_*.go ← Service struct (service.go) ← *store.Queries (SQLC-generated)
                                           ← *pgxpool.Pool (for transactions)
                                           ← *redis.Client (idempotency MW + cache)
                                           ← *sse.Hub (platform/sse)
                                           ← *slog.Logger

workers.go    ← *pgxpool.Pool + *store.Queries (same Service or thin Workers struct)
platform/sse  ← events.Listener (platform/events) + SSE writer
router.go     ← Handler methods + middleware (auth, idempotency, ifmatch, permission)
```

No circular dependencies. `Service` is a flat struct with methods — no
inheritance or interface-hiding.

---

## Phase 0: Migrations

**Prerequisite before any Go code.** Single migration file:
`migrations/NNN-reservation-update.sql` containing all changes below in order.

| ID  | Change                                                                                                                                                                                                                                                                                                                                                                                                      |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M9! | **DESTRUCTIVE — single tx.** Rewrite every `operations.reservations.stay_period` row to property-time bounds (`default_checkin_time`, `default_checkout_time`). ADR-018. Run **first** so envelope backfill below sees correct bounds. Backup checkpoint in `-- +goose Up`. `-- +goose Down` raises exception (irreversible). **Note:** no data in dev; migration kept to prove the pattern before prod.    |
| M6  | `stay_period_envelope TSTZRANGE NOT NULL` on `operations.reservations` + GIST index on `(property_id, stay_period_envelope)`. ADR-020. Backfill from items **(n/a — no data yet)**.                                                                                                                                                                                                                         |
| M7  | Add `'overstay'` value to `operations.reservation_item_status` enum.                                                                                                                                                                                                                                                                                                                                        |
| M10 | `pricing.booked_daily_rates` — drop existing `UNIQUE (reservation_item_id, calendar_date)`, replace with partial `UNIQUE (...) WHERE deleted_at IS NULL`. Required for soft-delete + re-insert.                                                                                                                                                                                                             |
| M11 | Add `'pending_cancellation'` value to `operations.reservation_status` enum. **No code path emits it** this PR — forward-compat for finance PR.                                                                                                                                                                                                                                                              |
| M12 | `daily_room_capacity INT CHECK (daily_room_capacity > 0)` (nullable = unlimited) on `pricing.daily_price_grid`. Capacity consumption checked by counting overlapping `reservation_items` per rate plan per night — item is the unit of consumption.                                                                                                                                                         |
| M13 | `final_price_pence` calculation trigger: `BEFORE INSERT OR UPDATE ON pricing.booked_daily_rates` computes `final_price_pence = base_price_pence + adjustment_value`. Frontend distributes per-night adjustment values before submission. Verify `adjustment JSONB` column exists (00004); add CHECK constraints on JSONB structure if missing.                                                              |
| M14 | `do_not_move BOOLEAN NOT NULL DEFAULT false` on `operations.reservation_items`.                                                                                                                                                                                                                                                                                                                             |
| M15 | `late_checkout_grace_minutes INT` on `operations.property_settings`. Per-property overstay tolerance.                                                                                                                                                                                                                                                                                                       |
| M16 | `reservation_item_id UUID` column on `inventory.room_inventory_ledger` (verify absent first; FK to reservation_items, `ON DELETE RESTRICT`) + index `idx_inv_ledger_reservation_item ON inventory.room_inventory_ledger (reservation_item_id)`. `checkout_session_id` FK retained (renamed in finance PR per ADR-019).                                                                                      |
| M17 | `expires_at TIMESTAMPTZ` on `operations.reservations`. Populated on INSERT for `hold` status by service layer: pick `property_settings.{source}_hold_ttl_seconds` and apply ADR-016 guest-presence tier (anonymous vs guest-attached). Worker only compares `expires_at < now()`. NULL'd on Confirm transition. Add CHECK constraint: `(status = 'hold' AND expires_at IS NOT NULL) OR (status <> 'hold')`. |
| M18 | Audit log trigger: `AFTER INSERT OR UPDATE OR DELETE ON operations.reservations` + `...ON operations.reservation_items` writes to `auth.audit_logs`. Requires `SET LOCAL app.current_user_id` from auth middleware context. ADR-021.                                                                                                                                                                        |
| M19 | `cancellation_intent JSONB` on `operations.reservations` (NULL default). Written in same tx before status flip on cancel; M18 trigger captures via row delta. Shape: `{ reason_code, fee_pence, waive_fee, fee_override_reason, refund_action, cancelled_by_user_id }`. Finance PR replays from this column for fee reconciliation.                                                                         |
| M4  | Property settings TTL/grace cols: `website_hold_ttl_seconds`, `internal_hold_ttl_seconds`, `reservation_archive_after_days`, `no_show_grace_minutes` — add only those not already present.                                                                                                                                                                                                                  |

**Deferred:**

- **M8** — rename `operations.checkout_sessions` → `payment_authorizations` +
  finance cols. Finance PR (ADR-019).
- **M5** — `operations.ota_inbound_messages` + `operations.ota_action` enum. OTA
  PR.

After migration: `make sqlc` then `make gen-constraints`.

---

## Phase 1: SQLC Queries

SQLC queries split across focused files. `make sqlc` regenerates
`internal/store/*.sql.go`.

**`internal/store/queries/global.sql`**

```text
SetCurrentPropertyID   :exec    SELECT set_config('app.current_property_id', $1, true)
SetCurrentUserID       :exec    SELECT set_config('app.current_user_id', $1, true)
```

Called inside every `ExecuteTx[T]` before user-supplied `fn`. RLS + M18 audit
trigger both depend on these.

**`internal/store/queries/reservation_crud.sql`**

```text
CreateReservation              :one     INSERT INTO operations.reservations
CreateReservationItem          :one     INSERT INTO operations.reservation_items
CreateFolio                    :one     stub Folio A (balance_pence=0) — finance PR fills in
GetReservation                 :one     joined items array
GetReservationItems            :many
ListReservations               :many    cursor pagination (ADR-014)
UpdateReservationMetadata      :one     notes, travel_agent, group_id, primary_guest_id
UpdateReservationItem          :one     fields + version bump
NextReservationSequence        :one     trigger-populated; just validate
```

**`internal/store/queries/reservation_items.sql`**

```text
RollupReservationStatus        :one     calls PL/pgSQL function (ADR-015) — never set status directly
AvailabilityByType             :many    ledger aggregate per room type per date
SelectRoomForAutoPin           :one     FOR UPDATE SKIP LOCKED on RIL, deterministic lowest room number
ConflictCheckOnLedger          :one     booking assertion: SELECT COUNT WHERE room_id + dates overlap
InsertLedgerRow                :exec
BulkInsertLedgerRows           :exec    UNNEST array
UpdateLedgerRowRoom            :exec    R-RES-GROOM-007 update-in-place
DeleteLedgerRowsByItem         :exec
DeleteLedgerRowsByItemFromDate :exec
```

**`internal/store/queries/reservation_rates.sql`**

```text
InsertBookedDailyRate          :exec
BulkInsertBookedDailyRates     :exec
SoftDeleteBookedRatesNotInPeriod :exec  (replaces old FromDate)
OverrideNightlyRate            :exec    per-night base_rate_pence change (R-RES-CRUD-016)
ApplyRateAdjustment            :exec    UPDATE adjustment (trigger computes final_price_pence per M13)
ApproveRateAdjustments         :exec    UPDATE approved + approved_by
CheckRatePlanCapacity          :many    M12 capacity check per rate plan per night
GetBookedRates                 :many    per item
```

**`internal/store/queries/reservation_workers.sql`**

```text
GetFolioBalances               :many    stubbed — returns 0 until finance PR
FindExpiredHolds               :many    FOR UPDATE SKIP LOCKED LIMIT 100
FindOverdueCheckins            :many    for no-show reminder (NOTIFY only, no status change)
FindOverstays                  :many    FOR UPDATE SKIP LOCKED LIMIT 100
FindArchivableReservations     :many    FOR UPDATE SKIP LOCKED LIMIT 500
```

**`internal/store/queries/reservation_outbox.sql`**

```text
InsertOutboxEvent              :exec    async work (emails/webhooks) — NOT audit (M18 trigger handles audit)
NotifyChannel                  :exec    SELECT pg_notify($1, $2)
```

---

## Phase 2: Types + State Machine

### `types.go` — ~150 LOC

```go
package booking

// Enums (match Postgres enum values; synced to config/constraints.g.yml by make gen-constraints)
// @enum website internal ota
type ReservationSource string
const (
    SourceWebsite  ReservationSource = "website"
    SourceInternal ReservationSource = "internal"
    SourceOTA      ReservationSource = "ota"
)

// @enum hold confirmed checked_in checked_out pending_cancellation cancelled archived
type ReservationStatus string
// @enum booked checked_in checked_out no_show overstay cancelled archived
type ItemStatus       string

type RefundAction string
const (
    RefundNone     RefundAction = "none"
    RefundOriginal RefundAction = "original"
    RefundCredit   RefundAction = "credit"
)
type AdjustmentType string
const (
    AdjustmentPercent AdjustmentType = "percentage"
    AdjustmentFixed   AdjustmentType = "fixed"
)

// I/O structs (used by handlers + service layer)
type CreateReservationInput struct {
    Source     ReservationSource `json:"source"`
    IsWalkin   bool              `json:"is_walkin"`                          // walk-in iff Source=internal && IsWalkin
    PropertyID uuid.UUID         `json:"property_id"`
    Notes      string            `json:"notes"`                              // ≤2500 (constraints)
    Items      []CreateItemInput `json:"items" constraints:"operations.reservation_items"`
    Guest      *GuestInlinePayload `json:"guest,omitempty"`                  // may reference existing guest (flows/§1.3)
    GroupID    *uuid.UUID        `json:"group_id,omitempty"`
    TravelAgentID *uuid.UUID     `json:"travel_agent_id,omitempty"`
}
type CreateItemInput struct {
    RoomTypeID     uuid.UUID  `json:"room_type_id"`
    AssignedRoomID *uuid.UUID `json:"assigned_room_id,omitempty"`
    ArrivalDate    string     `json:"arrival_date" example:"2026-06-01"`     // ISO8601 date YYYY-MM-DD; server composes TSTZRANGE
    DepartureDate  string     `json:"departure_date" example:"2026-06-05"`
    RatePlanID     *uuid.UUID `json:"rate_plan_id,omitempty"`
    AdultsCount    int        `json:"adults_count"`
    Children       int        `json:"children_count"`
    Guest          *GuestInlinePayload `json:"guest,omitempty"`              // may reference existing guest (flows/§1.3)
}

// Simple for MVP; will expand (address, nationality, document fields) in guest-profile PR.
type GuestInlinePayload struct { Name, Email, Phone string }

type CancelInput struct {
    ReasonCode        string       `json:"reason_code"`
    FeePence          int          `json:"fee_pence"`          // recorded in audit only this PR
    WaiveFee          bool         `json:"waive_fee"`           // recorded in audit only
    FeeOverrideReason string       `json:"fee_override_reason"`
    RefundAction      RefundAction `json:"refund_action"`
}
type RateAdjustment struct {
    CalendarDate time.Time      `json:"calendar_date"`
    Type         AdjustmentType `json:"type"`
    Value        int            `json:"value"`
    Reason       string         `json:"reason"`
}
type RateAdjustInput struct { Adjustments []RateAdjustment `json:"adjustments"` }

// TODO: add Mutation I/O structs (UpdateItemStayPeriodInput, AssignRoomInput, etc.) —
// the above are examples; full types.go will include all I/O shapes.

// Aux types
type RollupResult struct { Status ReservationStatus; Changed bool }
type DateAvailability struct {
    Date      time.Time `json:"date" example:"2026-06-01"`
    Total     int       `json:"total"`
    Available int       `json:"available"`
    Reason    string    `json:"reason,omitempty"`
}
```

### `errors.go` — domain sentinels (drop DomainError)

```go
// Sentinels include .WithSuggestion() for user-facing messages.
// Handler-level overrides (e.g. detail which fields differ) use .WithMessage() at call site.
var (
    ErrVersionMismatch     = apierror.New("VERSION_MISMATCH",
        "this record was modified by another user; your uncommitted changes may be lost", 412)
    ErrUnassignedItems     = apierror.New("UNASSIGNED_ITEMS",
        "items missing room assignment", 409).
        WithSuggestion("assign a room to each item before confirming")
    ErrTerminal            = apierror.New("TERMINAL_RESERVATION",
        "reservation in terminal state", 409)
    ErrOutstandingBalance  = apierror.New("OUTSTANDING_FOLIO_BALANCE",
        "folio balance must be zero", 409).
        WithSuggestion("settle the outstanding folio balance before checkout")
    ErrHasCheckedInItems   = apierror.New("RESERVATION_HAS_CHECKED_IN_ITEMS",
        "checkout or cancel checked-in items first", 409)
    ErrDoNotMove           = apierror.New("DO_NOT_MOVE",
        "do-not-move flag set", 403).
        WithSuggestion("remove the do-not-move flag or contact a manager")
    ErrRatePlanCapacity    = apierror.New("RATE_PLAN_CAPACITY_EXCEEDED",
        "rate plan daily capacity exceeded", 409).
        WithSuggestion("select a different rate plan or reduce the stay length")
    ErrInvalidTransition   = apierror.New("INVALID_TRANSITION",
        "state transition not allowed", 409)
    ErrIdempotencyConflict = apierror.New("IDEMPOTENCY_IN_PROGRESS",
        "a request with this idempotency key is still in flight", 409).
        WithSuggestion("retry with the same idempotency key to receive the stored result")
    ErrSourceDeferred      = apierror.New("SOURCE_DEFERRED",
        "this source is not yet implemented in the API", http.StatusNotImplemented)
)
```

### `state_machine.go` — ~100 LOC

Pure functions, no imports beyond `booking` itself.

```go
func ValidateReservationTransition(from, to ReservationStatus) error
func ValidateItemTransition(from, to ItemStatus) error

// ADR-015
func RollupReservationStatus(current ReservationStatus, items []ItemStatus) RollupResult

// §7.4 — Action-level idempotency for mutating endpoints.
// Given an endpoint name, the desired target status, and the current resource status,
// returns (noOp=true) if the action was already applied (handler → 200 with current body),
// or (conflict=true) if the current status forbids the transition (handler → 409).
// When both are false the action proceeds normally.
// Implementation: lookup table per §7.4 (endpoint × desiredStatus × currentStatus → outcome).
func ActionIdempotency(endpoint string, desiredStatus, currentStatus any) (noOp bool, conflict bool)
```

---

## Phase 3: Service — Transactions + Create + Confirm

### Transaction pattern: `ExecuteTx[T]` (platform helper)

```go
// internal/platform/db/tx.go
func ExecuteTx[T any](
    pool *pgxpool.Pool,
    base *store.Queries,
    ctx context.Context,
    fn func(*store.Queries) (T, error),
) (T, error) {
    var zero T
    tx, err := pool.Begin(ctx)
    if err != nil { return zero, fmt.Errorf("begin tx: %w", err) }
    defer tx.Rollback(ctx)

    qtx := base.WithTx(tx)

    propertyID := helpers.GetPropertyIDFromCtx(ctx)
    if propertyID == uuid.Nil {
        return zero, apierror.ErrBadRequest.WithMessage("missing property context")
    }
    if err := qtx.SetCurrentPropertyID(ctx, propertyID.String()); err != nil {
        return zero, fmt.Errorf("set rls: %w", err)
    }
    if userID := helpers.GetUserIDFromCtx(ctx); userID != uuid.Nil {
        if err := qtx.SetCurrentUserID(ctx, userID.String()); err != nil {
            return zero, fmt.Errorf("set user id: %w", err)
        }
    }

    result, err := fn(qtx)
    if err != nil {
        return zero, helpers.PsqlErrToCustomErr(err)
    }
    if err := tx.Commit(ctx); err != nil {
        return zero, fmt.Errorf("commit tx: %w", err)
    }
    return result, nil
}
```

RLS context (`app.current_property_id`) is set **inside** every tx via
`qtx.SetCurrentPropertyID`. Pulled from ctx (populated by `StubAuth`
middleware).

### `service.go` — ~250 LOC

```go
type Service struct {
    db  *pgxpool.Pool
    q   *store.Queries
    rdb *redis.Client
    log *slog.Logger
    sse *sse.Hub
}

func NewService(db *pgxpool.Pool, q *store.Queries, rdb *redis.Client, log *slog.Logger, hub *sse.Hub) *Service
```

`CreateReservation(ctx, input)` — dispatches by source:

```text
if input.Source == SourceWebsite { return nil, ErrSourceDeferred }       // booking-engine PR
Validate (struct + domain) → Resolve guest → ExecuteTx[Reservation](db, q, ctx, func(qtx) {
    -- ExecuteTx already called SetCurrentPropertyID + SetCurrentUserID
    Check availability (assigned_room_id path OR auto-pin via SelectRoomForAutoPin)
    Resolve expires_at (hold only):
      ttl_seconds = property_settings.{input.Source}_hold_ttl_seconds
      if guest unattached → apply ADR-016 anonymous-tier override
      expires_at = now() + ttl_seconds
    INSERT reservation (status per source rules below, expires_at if hold) → version=1
    INSERT items + ledger + booked_daily_rates + folio A (stub)
    NotifyChannel('reservation_changes', payload)
    -- audit trail written automatically by M18 trigger (ADR-021)
    return reservation
})
```

Source → status rules:

| `source` + `is_walkin`    | initial reservation status | initial item status          |
| ------------------------- | -------------------------- | ---------------------------- |
| `internal` + walkin=false | `hold`                     | `booked`                     |
| `internal` + walkin=true  | `checked_in`               | `checked_in` (room required) |
| `ota`                     | `confirmed`                | `booked`                     |
| `website`                 | 501 - deferred             | 501 — deferred               |

`ConfirmReservation(ctx, id, version)` — staff confirms a hold:

```text
ExecuteTx → SELECT res FOR UPDATE → assert status=hold → assert guest_id NOT NULL
→ UPDATE status=confirmed, version++ → NOTIFY → return res
```

(Audit trail written automatically by M18 trigger — no explicit outbox insert.)

---

## Phase 4: Availability

### `availability.go` — ~120 LOC

Availability checks run as store queries (SQLC-generated) called directly from
service methods. No separate AvailabilityService — methods sit on the booking
`Service` struct for simplicity.

```go
func (s *Service) CheckAvailability(ctx, propertyID, roomTypeID, startDate, endDate) ([]DateAvailability, error)
func (s *Service) AutoPinRoom(ctx, qtx *store.Queries, roomTypeID, date) (uuid.UUID, error)   // FOR UPDATE SKIP LOCKED
func (s *Service) ConflictCheck(ctx, qtx *store.Queries, roomID, dates []time.Time, excludeItemID *uuid.UUID) error
```

Cache: `cache:availability:{property_id}:{room_type_id}:{date}` TTL 600s
(reactive invalidation primary; TTL = safety net). Invalidated reactively by
`NOTIFY reservation_changes` → events listener (ADR-010). **Individual
reservations are not cached.**

Events listener wiring (in `cmd/server/main.go`):

```go
listener := events.NewListener(pool, log)
listener.On("reservation_changes", hub.OnReservationChange)   // SSE broadcast + cache invalidation
listener.On("staff_alerts", hub.OnStaffAlert)
go listener.Listen(ctx)
```

---

## Phase 5: Validation via constraints

`internal/platform/validation/validate.go` exists. Single function:

```go
func Struct(v any, tableKey string) []error
```

Walks struct via reflection, matches `json` tags to constraint keys in
`config/constraints.g.yml`. For nested struct slices, add
`constraints:"schema.table"` tag on the slice field to recurse.

| Layer                             | Examples                                                                                                                                                       |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `validation.Struct`               | `notes` ≤ 2500 chars, `adults_count` ≥ 1, `source` not empty, `type`/`value`/`reason` within `adjustment` JSONB                                                |
| Domain check (handler or service) | Walk-in requires `assigned_room_id`, website source → 501, past-date check (per op — see below), LOS vs rate plan restrictions, permission-dependent branching |

**Past-date check (R-RES-VALID-002) per operation:**

| Operation                                                     | Check                                                                           |
| ------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| `POST /reservations` (Create)                                 | `lower(envelope) >= today` (single check across all initial items)              |
| `POST /reservations/{id}/items` (AddItem)                     | check the **new item's** `lower(stay_period)` only (existing items may be live) |
| `PATCH /reservations/{id}/items/{item_id}` stay-period update | check the **target item's new** `lower` only                                    |
| Extend (lengthen upper)                                       | skip past-check (upper changes, not lower)                                      |
| Cancel item                                                   | skip past-check                                                                 |

`reservations:retroactive_create` bypasses all. `is_walkin=true` bypasses Create
check only.

`make gen-constraints` reads the live DB and writes both
`config/constraints.g.yml` and `web/src/lib/types/constraints.g.ts`. Also syncs
Postgres enum values into the constraint file so `validation.Struct` can
validate enum fields against allowed DB values. No manual sync step.

---

## Phase 6: Handlers

Files: `handlers_crud.go`, `handlers_lifecycle.go`, `handlers_rates.go`,
`handlers_misc.go`.

Swagger annotations (godoc `@Summary`, `@Param`, `@Success`, `@Failure`) on each
handler method. Run `swag fmt` after writing annotations.

Every handler follows the same template:

```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var input CreateReservationInput
    if err := json.ReadJSON(r, &input); err != nil {
        json.WriteError(w, r, err); return
    }
    if errs := validation.Struct(input, "operations.reservations"); len(errs) > 0 {
        json.WriteError(w, r, apierror.ErrUnprocessable.WithMessage(errs[0].Error())); return
    }
    // domain checks (e.g. walkin needs assigned_room_id; website → 501)
    res, err := h.svc.CreateReservation(r.Context(), input)
    if err != nil { json.WriteError(w, r, err); return }
    json.WriteJSON(w, http.StatusCreated, res)
}
```

For endpoints with no body (GET by id, checkin with no payload) — skip
parse/validate.

**Batch endpoints (`PATCH /reservations/{id}/checkin` and `/checkout`)** return
207 Multi-Status on partial failures:

```json
{
  "results": [
    {
      "item_id": "...",
      "status": "ok",
      "reservation_item": {
        /* updated */
      }
    },
    {
      "item_id": "...",
      "status": "failed",
      "error": {
        "code": "INVALID_TRANSITION",
        "message": "item already checked_out"
      }
    }
  ]
}
```

HTTP 200 if all succeed, 207 if any fail.

All endpoint handlers (~25 methods):

```text
handlers_crud.go       Create, Get, List, UpdateMetadata, AddItem, UpdateItem, AssignRoom
handlers_lifecycle.go  Confirm, Cancel, Reactivate, CheckinReservation, CheckinItem,
                       CheckoutReservation, CheckoutItem, MarkNoShow, CancelItem
handlers_rates.go      UpdateBookedRates, GetBookedRates, ApproveAdjustments, AdjustRate
handlers_misc.go       Availability, GetFolio (stub), CancellationQuote (stub)
```

`source=website` returns 501 `ErrSourceDeferred`. OTA inbound webhook **not
registered** this PR. Reservation-groups routes **not registered** this PR (chi
404).

---

## Phase 7: Remaining business logic

Test files: `actions_ec_test.go`, `mutations_ec_test.go`, `rates_ec_test.go`,
edge cases for Cancel, Checkin, Checkout, Reactivate, UpdateItem, AssignRoom.

### `actions.go` — ~280 LOC

```go
func (s *Service) CheckinReservation(ctx, id, version)        // §3.1 — 207 on partial
func (s *Service) CheckinItem(ctx, itemID, version)            // §3.2
func (s *Service) CheckoutReservation(ctx, id, version)        // §3.3 — folio balance gate (stub 0 this PR)
func (s *Service) CheckoutItem(ctx, itemID, version)
func (s *Service) ShortenStay(ctx, itemID, version, newDep)    // §3.4 — internal; called by UpdateItemStayPeriod when status=checked_in, not a standalone route
func (s *Service) CancelReservation(ctx, id, version, input)   // §4.1/4.2 — see cancellation rules below
func (s *Service) CancelItem(ctx, itemID, version, input)      // §4.6
func (s *Service) MarkNoShow(ctx, itemID, version)             // §4.3
func (s *Service) Reactivate(ctx, id, version)                 // §4.5
```

**Cancellation behavior this PR** (defers fee posting to finance PR):

- From `hold` → `cancelled` directly. Delete future ledger. No folio tx.
  `fee_pence`/`waive_fee` ignored.
- From `confirmed` → `cancelled` directly (no `pending_cancellation`
  intermediate). `fee_pence` + `waive_fee` accepted, recorded in audit log only.
  No folio insert. Comment marks finance PR location.
- From `no_show` → `cancelled` (same path).
- `GET .../cancellation-quote` returns
  `{ fee_pence: null, status: "not_implemented" }`.

### `mutations.go` — ~240 LOC

```go
func (s *Service) UpdateItemStayPeriod(ctx, itemID, version, arrival, departure)    // §2.1
func (s *Service) AssignRoom(ctx, itemID, roomID, version)                          // §2.2/2.3 — DNM check
func (s *Service) UpdateItemRoomType(ctx, itemID, version, newTypeID, retainPrice)  // §2.5
func (s *Service) UpdateItemRatePlan(ctx, itemID, version, newRatePlanID, retainPrice) // §2.6
func (s *Service) AddItem(ctx, reservationID, version, input)                        // §2.7
```

All require version check (returns `ErrVersionMismatch` 412). Post-checkin
mutations additionally require `reservations:post_checkin_mutate` (service-side
check).

### `rates.go` — ~180 LOC

```go
func (s *Service) GetBookedRates(ctx, itemID)
func (s *Service) OverrideNightlyRate(ctx, itemID, version, date, baseRatePence int)              // R-RES-CRUD-016
func (s *Service) ApplyRateAdjustments(ctx, itemID, version, adjustments, staffID, hasAdjustPerm) // pending or approved
func (s *Service) ApproveRateAdjustments(ctx, itemID, version, dates, approverID)                 // mgr action
```

Adjustments stored as
`adjustment JSONB {type, value, reason, approved, approved_by}`.

```go
// percentage (value = signed integer percentage points, e.g. -10 = 10% off)
delta := math.Round(float64(base) * float64(value) / 100.0)
final := base + int(delta)
if final < 0 { final = 0; log warning per R-RES-EDGE-025/026 }

// fixed (value = signed pence)
final := base + value
if final < 0 { final = 0; log warning }
```

`math.Round` is half-away-from-zero. Negative value = discount.

When `approved=false` the night still shows the adjusted figure; reports flag
pending; managers see approval queue.

---

## Phase 8: Router + Middleware

### Auth (StubAuth) — `internal/platform/middleware/auth.go`

```go
func StubAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        pid := r.Header.Get("X-Property-ID")
        if pid == "" { json.WriteError(w, r, apierror.ErrBadRequest.WithMessage("X-Property-ID required")); return }
        id, err := uuid.Parse(pid)
        if err != nil { json.WriteError(w, r, apierror.ErrBadRequest.WithMessage("invalid X-Property-ID")); return }
        perms := strings.Split(r.Header.Get("X-User-Permissions"), ",")
        ctx := helpers.SetPropertyIDInCtx(r.Context(), id)
        ctx = helpers.SetPermissionsInCtx(ctx, perms)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Real JWT-backed auth replaces only this middleware in auth PR — everything
downstream unchanged.

### Permissions: route-static vs body-conditional

| Middleware (`RequirePermission`) | Service-side (after loading state)                    |
| -------------------------------- | ----------------------------------------------------- |
| `reservations:read`              | (applied to all GET routes)                           |
| `reservations:create`            | `reservations:waive_fee` (only when `waive_fee=true`) |
| `reservations:update`            | `reservations:post_checkin_mutate`                    |
| `reservations:update_item`       | `reservations:override_dnm`                           |
| `reservations:cancel`            | `reservations:override_rate_plan_capacity`            |
| `reservations:reactivate`        | `reservations:override_restrictions`                  |
| `reservations:assign_room`       | `reservations:retroactive_create`                     |
| `reservations:add_item`          | `reservations:change_room_type`                       |
| `reservations:confirm`           | `reservations:adjust_rate`                            |
| `reservations:mark_no_show`      | `reservations:fee_override`                           |
| `reservations:rate_override`     |                                                       |

`helpers.HasPermission(ctx, perm)` is the single check used by both.

### If-Match — `internal/platform/middleware/ifmatch.go`

```go
func RequireIfMatch(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPatch && r.Method != http.MethodPost {
            next.ServeHTTP(w, r); return
        }
        raw := r.Header.Get("If-Match")
        if raw == "" { json.WriteError(w, r, apierror.ErrBadRequest.WithMessage("If-Match required")); return }
        v, err := strconv.Atoi(strings.Trim(raw, `"`))
        if err != nil || v < 1 { json.WriteError(w, r, apierror.ErrBadRequest.WithMessage("invalid If-Match")); return }
        ctx := helpers.SetIfMatchVersion(r.Context(), int32(v))
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### `router.go` — ~90 LOC

```go
func Routes(svc *Service, ifMatch func(http.Handler) http.Handler) func(chi.Router) {
    h := &Handler{svc: svc}
    return func(r chi.Router) {
        r.Get("/availability", h.Availability)  // no auth this PR (public, dev-only)

        r.With(mw.RequirePermission("reservations:create")).Post("/", h.Create)
        r.With(mw.RequirePermission("reservations:read")).Get("/", h.List)

        r.Route("/{id}", func(r chi.Router) {
            r.With(mw.RequirePermission("reservations:read")).Get("/", h.Get)
            r.With(ifMatch, mw.RequirePermission("reservations:update")).Patch("/", h.UpdateMetadata)
            r.With(ifMatch, mw.RequirePermission("reservations:confirm")).Post("/confirm", h.Confirm)
            r.With(ifMatch, mw.RequirePermission("reservations:cancel")).Post("/cancel", h.Cancel)
            r.With(ifMatch, mw.RequirePermission("reservations:reactivate")).Post("/reactivate", h.Reactivate)
            r.With(ifMatch).Patch("/checkin", h.CheckinReservation)
            r.With(ifMatch).Patch("/checkout", h.CheckoutReservation)
            r.With(mw.RequirePermission("reservations:read")).Get("/cancellation-quote", h.CancellationQuote)
            r.With(mw.RequirePermission("reservations:read")).Get("/folios/{folio_id}", h.GetFolio)
            r.With(mw.RequirePermission("reservations:add_item")).Post("/items", h.AddItem)
            r.Route("/items/{item_id}", func(r chi.Router) {
                r.With(ifMatch, mw.RequirePermission("reservations:update_item")).Patch("/", h.UpdateItem)
                r.With(ifMatch).Patch("/checkin", h.CheckinItem)
                r.With(ifMatch).Patch("/checkout", h.CheckoutItem)
                r.With(ifMatch, mw.RequirePermission("reservations:assign_room")).Patch("/assign-room", h.AssignRoom)
                r.With(ifMatch, mw.RequirePermission("reservations:mark_no_show")).Patch("/no-show", h.MarkNoShow)
                r.With(ifMatch, mw.RequirePermission("reservations:cancel")).Post("/cancel", h.CancelItem)
                r.With(ifMatch, mw.RequirePermission("reservations:rate_override")).Patch("/booked-rates", h.UpdateBookedRates)
                r.With(mw.RequirePermission("reservations:read")).Get("/booked-rates", h.GetBookedRates)
                r.With(ifMatch).Post("/booked-rates/approve", h.ApproveAdjustments)
                r.With(ifMatch).Post("/adjust-rate", h.AdjustRate)
            })
        })
    }
}
```

Wired in `cmd/server/api.go`:

```go
r.Use(mw.StubAuth)                                  // sets property_id + perms into ctx
r.Get("/api/v1/sse", sseHub.Subscribe)              // outside /v1 (no idempotency)
r.Route("/v1", func(r chi.Router) {
    r.Use(mw.Idempotency(app.rdb))              // covers POST + PATCH at /v1; verify allowlist before merge
    r.Route("/reservations", booking.Routes(bookingSvc, mw.RequireIfMatch))
    // r.Route("/channels", ...)            — OTA PR
    // r.Route("/reservation-groups", ...)  — groups PR
})
```

---

## Phase 9: SSE (rough sketch — finalised after research)

User researches SSE before coding. Phase below is a skeleton; bracketed items
are open. Final implementation may slip to a follow-up PR.

`internal/platform/sse/hub.go` — cross-cutting hub (booking, future folios,
groups all share).

```go
type SSEHub struct {
    clients map[chan SSEMessage]struct{}
    mu      sync.RWMutex
}
type SSEMessage struct { Event string; Data string }

func (h *SSEHub) Subscribe(w http.ResponseWriter, r *http.Request)  // registers client, removes on r.Context().Done()
func (h *SSEHub) Broadcast(msg SSEMessage)              // non-blocking; drops slow clients, skips closed channels
func (h *SSEHub) OnReservationChange(ctx, event) error  // events.Handler for reservation_changes channel
func (h *SSEHub) OnStaffAlert(ctx, event) error         // events.Handler for staff_alerts channel
```

Mounted at top-level `GET /api/v1/sse` (no idempotency, no auth this PR — public
dev endpoint; auth PR locks it down). Heartbeat every 30s.

**SSE research needed before finalizing (deferred):**

- Go stdlib `http.Flusher` + `r.Context().Done()` for disconnect detection
- Buffered vs unbuffered client channels (backpressure strategy)
- Browser `EventSource` reconnect backoff and `Last-Event-ID`
- Reverse-proxy config (nginx `proxy_buffering off`)

Frontend: `EventSource("/api/v1/sse")`, browser auto-reconnects.

---

## Phase 10: Workers

Test file: `workers_ec_test.go` — edge cases for hold expiry, overstay, archival.

`workers.go` — sweep workers run as standalone goroutines started in
`cmd/server/main.go`:

```go
// cmd/server/main.go
workers := &booking.Workers{db: pool, q: queries, log: log}
go workers.HoldExpirySweep(ctx)    // every 30s
go workers.NoShowReminder(ctx)     // configurable interval
go workers.OverstaySweep(ctx)     // every N min
go workers.ArchivalSweep(ctx)     // daily
```

```go
type Workers struct {
    db  *pgxpool.Pool
    q   *store.Queries
    log *slog.Logger
}

func (w *Workers) HoldExpirySweep(ctx)   // every 30s
func (w *Workers) NoShowReminder(ctx)    // configurable; NOTIFY staff_alerts only — no status change
func (w *Workers) OverstaySweep(ctx)     // every N min — status=overstay
func (w *Workers) ArchivalSweep(ctx)     // daily — soft-delete terminal reservations
```

### Hold expiry

```text
SELECT reservations WHERE status=hold AND expires_at < now()
FOR UPDATE SKIP LOCKED LIMIT 100
foreach:
  BEGIN
    UPDATE reservation SET status=cancelled, deleted_at=now()
    UPDATE reservation_items SET status=cancelled
    DELETE ledger rows for reservation_id
    NOTIFY reservation_changes
    -- audit log written by M18 trigger (ADR-021); outbox for async work only
  COMMIT
```

**No** `payment_authorization` writes, **no** `void_auth` outbox event in this
PR (finance PR / ADR-019 owns auth voiding).

### No-show

Reminder-only (§4.4): worker sends NOTIFY
`staff_alerts {type: no_show_overdue, item_ids: [...]}`. Staff acts manually.

---

## Phase 11: Caching

Cached (Redis):

- `cache:availability:{property_id}:{room_type_id}:{date}` — TTL 600s,
  invalidated by `NOTIFY reservation_changes` listener.

**Not cached:** individual reservation reads (by id), folio reads, booked-rates
reads, cancellation-quote, list cursor pages. RTM R-RES-INTEG-001 reinterpreted
as "availability queries" — RTM line to be updated.

---

## Implementation order

```text
1. migrations/NNN-reservation-update.sql (M9!→M6→M7→M10→M11→M12→M13→M14→M15→M16→M17→M18→M19→M4-ttls)
2. SQLC queries split across 5 focused files (see Phase 1) → make sqlc → make gen-constraints
3. platform additions:
     db/tx.go (ExecuteTx[T])
     helpers/context.go, helpers/psql_errors.go
     middleware/auth.go (StubAuth), middleware/permission.go, middleware/ifmatch.go
     sse/hub.go
4. booking/types.go, errors.go, state_machine.go (pure, unit-test first)
5. booking/availability.go
6. booking/service.go (Create + Confirm)
7. booking/handlers_crud.go (Create, Get, List, UpdateMetadata) + router.go + wire in cmd/server/api.go
   → end-to-end smoke test
8. booking/actions.go + handlers_lifecycle.go
9. booking/mutations.go + remaining CRUD handlers (UpdateItem, AssignRoom, AddItem)
10. booking/rates.go + handlers_rates.go
11. booking/workers.go — register in cmd/server/main.go
12. SSE hub registered in cmd/server/main.go + frontend smoke test
13. Full test pass + make audit
14. Booking tests (`internal/booking/*_test.go`):
      - state_machine_test.go: transition matrix (reservation + item), rollup (ADR-015), invalid transitions
      - action_idempotency_test.go: §7.4 lookup table coverage
      - setup_test.go: TestMain spins up PostgreSQL 18 + Redis containers,
        runs goose migrations, seeds property/room_type/rate_plan test data
      - integration_test.go: scenario tests
          - Create hold → confirm → checkin → checkout (happy path)
          - Create hold → cancel (hold cancellation with fee recording)
          - Create confirmed → modify stay period
          - Concurrent update → 412 VersionMismatch
          - Hold expiry sweep (short TTL, wait, verify cancelled)
          - Overstay sweep (short grace, wait, verify status=overstay)
          - GET with missing X-Property-ID → 400
          - Invalid state transitions → 409
```

---

## Items intentionally NOT in this PR

- Payment authorization / capture / void (ADR-019, finance PR; M8 deferred).
- Folio transactions beyond stub Folio A row.
- `pending_cancellation` state transitions (enum value present, no code path).
- Real auth (`StubAuth` only).
- OTA inbound webhook (`source=ota` enum preserved; no endpoint or worker; M5
  deferred).
- Reservation groups routes (deferred to groups PR — 404 from chi).
- Cancellation fee math (`GET .../cancellation-quote` stub returns null).
- Frontend changes (separate concern).

---

## Risks & follow-ups

- **M9! is irreversible.** Dev-data acceptable cost per user. `-- +goose Down`
  raises exception to block accidental rollback.
- `ExecuteTx` pulls `property_id` from ctx — handler must early-return 400 if
  missing. `StubAuth` already enforces this header.
- `RollupReservationStatus` must accept all current non-archived item statuses —
  pull cancelled too so snapshot is deterministic (ADR-015).
- Existing idempotency MW at `internal/platform/middleware/idempotency.go` keys
  on `Idempotency-Key` per ADR-007 — applied at `/v1` group level.
- `pricing.booked_daily_rates.deleted_at` must exist before M10 partial unique.
  Verify in migration.
- M18 audit trigger depends on `app.current_user_id` session variable —
  `ExecuteTx` must call `qtx.SetCurrentUserID(ctx, userID)` before mutations
  (ADR-021).
