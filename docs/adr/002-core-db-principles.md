# ADR 002: Core DB Principles

## Status

**Accepted**

Multi-tenant PostgreSQL with: `INTEGER` for all financials (no floats); `TIMESTAMPTZ` exclusively; UUIDv7 primary keys; `RESTRICT` foreign keys; soft delete via `deleted_at` with `WHERE (deleted_at IS NULL)` uniqueness; `property_id` + Row-Level Security on every tenant-isolated table; `version` column for optimistic locking on high-concurrency tables; `CHECK` constraints with names `{table}_{column}_{suffix}`; `CITEXT` for emails/usernames/codes.

Alternatives: UUIDv4 (B-tree fragmentation), serial PKs (info leak + merge pain), float money (rounding bugs), `CASCADE` deletes (history loss), per-row app-level RLS (drift risk).

---

See: `docs/CONTEXT.md` (Conventions), `migrations/`, `cmd/tools/sync-constraints/`
