# ADR 005: Check-Constraint Consistency

## Status

**Accepted**

DB `CHECK` constraints are the single source of truth for input bounds. `cmd/tools/sync-constraints` reads the live schema and emits `config/constraints.g.yml` (backend) and `web/src/lib/types/constraints.g.ts` (frontend). New constraints must be added via migration + `make gen-constraints`. GIST constraints are not synced (developer must read SQL).

Alternatives: hand-maintained YAML/TS (drift inevitable), runtime introspection only (slow at request time), no constraints (validation lives in handlers).

---

See: `cmd/tools/sync-constraints/`, `docs/guides/backend-constraints.md`, `docs/guides/frontend-constraints.md`
