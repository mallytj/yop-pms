Status: in-progress

## Scope

Remaining service methods: actions (cancel, checkin, checkout, mark no-show, reactivate), mutations (stay period, room assignment, room type, rate plan, add item), and rates (booked rates, adjustments, overrides).

## Branch

`feat/reservation/business-logic`

## Files

| File | Action |
| :--- | :----- |
| `internal/booking/actions.go` | create |
| `internal/booking/mutations.go` | create |
| `internal/booking/rates.go` | create |

## Dependencies

Blocked by: service-core
Blocks: handlers-router
