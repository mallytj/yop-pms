# ADR 008: Cursor Pagination

## Status

**Accepted**

All list endpoints use cursor pagination. Cursor is opaque `base64url(json({k: [sort_keys], f: sha256(filter_set), v: 1}))`. `sort` must produce a stable, total tuple (e.g. `created_at, id`); UUIDv7 makes the `id` tiebreaker free. `limit` default 50, max 100. Filter-set fingerprint is compared on continuation; mismatch returns 400 `filter_changed`. Forward-only — no random access.

Alternatives: offset/limit (linear scan + write-shift jitter), explicit `after_id` (exposes sort schema), Redis-stored cursor state (infra for no gain), Relay-style connections (over-engineered for REST).

---

See: `internal/platform/pagination/`, `docs/requirements/reservations.md` (R-RES-CRUD-003, §9)
