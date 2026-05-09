# Authorization RTM

> **⚠️ PLACEHOLDER — DEFERRED TO FUTURE SPRINT/PR**
>
> Authorization is **not** in scope for the current reservation-flow work. This file
> exists so other domain docs can reference an eventual authz contract, but every
> requirement, role, and endpoint listed below is **provisional** and may change
> entirely when the auth PR lands.
>
> Do **not** implement against this doc yet. Treat any `*:permission_name` referenced
> from other RTMs as a stub that will be wired up in the auth sprint.

## 1. Intended Direction (provisional)

- Permissions as `<resource>:<action>` strings.
- Roles as named permission bundles, assigned per user × property.
- Multi-tenancy enforced at DB layer via PostgreSQL RLS (`app.current_property_id`).
- RBAC (functional perms) handled in the application layer; RLS strictly tenant isolation.

Per-domain permission lists live in their own RTMs. There is **no central permission
matrix** in this file by design — each domain doc owns its own perms.

## 2. Open Decisions

All of the following are **TBD** and will be resolved in the auth PR:

- Role inheritance vs flat
- Additive grants only vs grant + explicit deny
- Multi-property session model (single `current_property_id` vs array)
- Cross-property roles (`licence:admin` semantics)
- `system` role / service accounts (RLS bypass scope)
- Permission cache TTL + revocation propagation
- Audit log for permission changes
- API-key / partner integration model (OTA webhooks today live outside RBAC)

## 3. Until Then

Handlers may stub authz checks (e.g. always allow in dev) but **must** leave a clear
hook (`requirePermission("…")` middleware or equivalent) so the auth PR can wire real
checks without rewriting handler bodies.
