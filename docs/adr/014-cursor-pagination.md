# ADR 014: Cursor Pagination for List Endpoints

## Status

**Accepted**

## Context

The reservation list endpoint, and most other list endpoints in the PMS, must remain performant and consistent as the dataset grows. Two pagination styles are common:

1. **Offset/limit** — `?offset=2000&limit=50`. Simple to implement, simple for clients (jump to any page), familiar from SQL.
2. **Cursor** — `?cursor=<opaque>&limit=50`. Cursor encodes a position in a stable sort key. Pages move forward (and optionally backward) one window at a time.

Offset has two well-known failure modes at scale and under writes:

- **Performance** — `OFFSET 50000 LIMIT 50` requires the database to scan 50,050 rows and discard the first 50,000. Latency grows linearly with offset.
- **Inconsistency under writes** — Inserts/deletes between page fetches shift every row. A reservation cancelled mid-pagination causes one entry to be skipped on the next page; a new booking causes one to be repeated. The user sees a jittery list.

Even at modest dataset sizes (a single hotel will accumulate ~10k reservations per year, multi-property tenants more), these failure modes appear quickly. The reservation list is read frequently (front desk dashboard, arrivals/departures view, search) — predictable performance matters.

## Decision

All list endpoints use **cursor pagination**. The cursor is an opaque, base64-encoded JSON object containing the sort key value(s) of the last row returned, plus a hash of the filter set to detect filter mismatch on continuation requests.

### Cursor format

```
base64url(json({
  "k": [<sort_key_1>, <sort_key_2>, ...],   // sort key tuple (descending precedence)
  "f": "<sha256(filter_set)>",              // filter fingerprint
  "v": 1                                    // schema version
}))
```

Example cursor for `sort=-created_at`:
```
base64url(json({"k":["2026-05-07T10:23:11.123Z"],"f":"a3f9...","v":1}))
```

### Request shape

```
GET /api/v1/reservations?
  property_id=...&
  status[]=confirmed&
  q=smith&
  sort=-created_at&
  cursor=eyJrIj…&
  limit=50
```

- `cursor` — opaque; clients never construct or interpret it
- `limit` — max 100, default 50
- `sort` — explicit sort key; server-allowed values per endpoint (default `-created_at`)
- Filters — additive; cursor is invalidated if filters change

### Response shape

```json
{
  "data": [...],
  "page": {
    "next_cursor": "eyJrIj…",
    "has_more": true
  }
}
```

`next_cursor` is null when there are no more rows. There is no `prev_cursor` in the foundation phase; reverse pagination is added per-endpoint when needed (UI-driven; most list views scroll forward only).

### Filter fingerprint

The cursor embeds `sha256(canonical_json(filter_set))`. On continuation, the server recomputes the fingerprint from the current request and compares. Mismatch returns 400 with `error: "filter_changed", hint: "drop the cursor when filters change"`. This prevents "torn" pagination where a client changes filters but reuses an old cursor.

### Sort key requirements

Every paginatable list must sort by a tuple that is **stable and total**:

- Stable: the value does not change for an existing row (use `created_at` not `updated_at`; use `id` as tiebreaker for created_at collisions)
- Total: no two rows share the same tuple (UUIDv7 ids guarantee this)

Standard tiebreaker: append `id` (UUIDv7) as the last sort key. Cursor encodes both values.

### Default sort

Each endpoint declares its default sort. For reservations: `-created_at, -id`. For the calendar view: `lower(stay_period) asc, id asc`. Sort options are explicit per endpoint; not all sorts on all endpoints.

### No deep page jumping

There is no "go to page 47" affordance. UIs that require it (rare) must paginate forward and cache. This is an explicit trade-off: forward-only navigation in exchange for predictable performance.

## Consequences

### ✅ Positive

- **Constant-time pagination** — `WHERE created_at < $1 ORDER BY created_at DESC LIMIT 50` is index-bound regardless of dataset size.
- **Stable under writes** — A row inserted between page fetches does not shift existing rows; clients see a consistent slice of the time-ordered stream.
- **Filter-change detection** — Embedded fingerprint catches the easy mistake of mixing cursors across filter sets.
- **One convention across the API** — Future list endpoints inherit this pattern; clients learn it once.
- **UUIDv7 makes tiebreak free** — Ids already encode insertion order; no additional column needed.

### ⚠️ Negative

- **No random access** — Clients cannot "jump to page 5". Forward iteration only (with reverse added per-endpoint when justified). UIs that present a page picker must change.
- **Opaque cursors** — Clients cannot inspect or hand-construct cursors. Debugging requires server-side decoding (provide a dev-only `/internal/decode-cursor` endpoint if useful).
- **Cursor invalidation on filter change** — Clients must know to reset the cursor when filters change; otherwise a 400. Documented; SDK handles automatically.
- **Schema evolution** — Cursor `v` field allows future format changes, but old clients holding old cursors will see 400 after a server upgrade. Acceptable for short-lived cursors (pagination is interactive).

## Alternatives Considered

- **Offset/limit** — Rejected for the reasons above. Acceptable for tiny endpoints (≤200 rows total, e.g. property list); use sparingly and document the limit.
- **Keyset pagination with explicit `after_id` parameter** — Equivalent to cursor but requires clients to know the sort schema. Cursor wraps this complexity.
- **Token-based pagination via Redis** — Server-side cursor state in Redis. Adds infrastructure dependency for no benefit; opaque cursor in URL is stateless.
- **GraphQL Relay-style connections** — Over-engineered for a REST API; the `data + page.next_cursor` shape captures the essential bits.

## References

- `internal/platform/pagination/` — Cursor encode/decode + fingerprint helpers (TBD)
- `docs/requirements/reservations.md` — R-RES-CRUD-003, section 9 list endpoint signature
- ADR-013: Locking and Availability Strategy (cursor stability under concurrent writes)
