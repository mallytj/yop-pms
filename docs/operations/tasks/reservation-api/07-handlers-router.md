Status: in-progress

## Scope

All HTTP handlers (CRUD, lifecycle, rates, misc), router with middleware wiring, and integration into cmd/server/api.go.

## Branch

`feat/reservation/handlers-router`

## Files

| File | Action |
| :--- | :----- |
| `internal/booking/handlers_crud.go` | create |
| `internal/booking/handlers_lifecycle.go` | create |
| `internal/booking/handlers_rates.go` | create |
| `internal/booking/handlers_misc.go` | create |
| `internal/booking/router.go` | create |
| `cmd/server/api.go` | modify |

## Dependencies

Blocked by: business-logic, middleware
Blocks: workers, sse
