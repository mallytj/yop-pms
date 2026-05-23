# ADR 022: Response Depth via `?include=`

## Status

**Accepted**

All response endpoints accept `?include=items,guest` to embed related resources
the caller needs. Default embeds items only. `?include=none` omits items for
lightweight list views. Guest expansion only runs a `GetGuest` query when the
flag is set.

Alternatives considered: always-full-depth (wasteful for grids), separate
item/guest endpoints (N+1 round trips), GraphQL (overshoot for 10-domain PMS).

---

See: [`internal/booking/types.go`](../../internal/booking/types.go)
(`IncludeFlags`),
[`internal/booking/include.go`](../../internal/booking/include.go)
(`ParseIncludeFlags`),
[`internal/booking/service.go`](../../internal/booking/service.go)
(`CreateReservation`, `ConfirmReservation` — both accept `IncludeFlags`)
