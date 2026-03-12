# ADR 004: Core Database Principles

## Status
**Accepted**

## Context
As a Property Management System (PMS), the database is the most critical component of Yop. It must support multi-tenancy, high-concurrency booking, and strict financial auditing. Without a set of enforced standards, the schema will eventually suffer from performance degradation, data "leakage" between properties, and loss of historical audit trails.

## Decision
We will adhere to the following technical requirements (derived from RTM REQ-001 through REQ-015) for all schema designs.

### 1. Data Integrity & Precision
* **Financials (REQ-003):** All monetary values MUST be stored as `INTEGER` representing the smallest currency unit (e.g., pence/cents). No floating-point types.
* **Timestamps (REQ-009):** Use `TIMESTAMPTZ` exclusively to ensure UTC consistency across different time zones.
* **Constraints (REQ-007, REQ-012, REQ-015):** * Every text column must have a length `CHECK` constraint.
    * Booleans must have explicit defaults.
    * Constraint names must follow: `{table}_{column}_{suffix}`.

### 2. Multi-Tenancy & Performance
* **Property Isolation (REQ-005, REQ-014):** Any index or reference dependent on a property MUST include `property_id` to ensure strict tenant isolation and query performance.
* **Indexing (REQ-011):** All Foreign Key columns must have an explicit Index to prevent full table scans on joins.
* **Locking (REQ-010):** High-concurrency tables (e.g., `inventory`, `bookings`) must implement **Optimistic Locking** via a `version` column.

### 3. Identity & Relations
* **Primary Keys (REQ-002):** Use **UUIDv7**. This provides the non-predictability of a UUID with the time-sortable performance of an integer.
* **Historical Integrity (REQ-006, REQ-013):** Foreign keys must use `RESTRICT` instead of `CASCADE` for deletions to preserve historical business data.


### 4. Evolution & Auditing
* **Migrations (REQ-001, REQ-008):** All changes must be SQL-based migrations via `goose`.
* **Audit Trails (REQ-004):** Every core table must include:
    * `created_at`
    * `updated_at`
    * `deleted_at` (for Soft Deletes)

### 5. Practical Guardrails
* **Partial Uniqueness (REQ-016):** Unique constraints must ignore soft-deleted rows using `WHERE (deleted_at IS NULL)`.
* **Identity Casing (REQ-017):** Use `CITEXT` for emails to ensure `User@Email.com` and `user@email.com` are treated as the same entity.
* **Logic Separation (REQ-018):** Business logic (price calculations, state transitions) must reside in the Go application, not in DB triggers.

### 6. Multi-Tenant Security (Row-Level Security)
* **RLS Enforcement (REQ-020):** Every table containing a `property_id` MUST have Row-Level Security enabled.
* **Bypass Prevention (REQ-021):** The application must connect to the database using a non-superuser role. RLS policies will be defined to restrict access based on a session-local variable (e.g. `app.current_property_id`).
* **Logic Isolation (REQ-022):** Go repository methods must set the local transaction context for the property ID before executing any tenant-specific queries.

## Consequences

## Consequences

### + Positive (The "Wins")
* **Low Level Security:** RLS and strict CHECK constraints ensure that if the application code contains a logic error (e.g., forgetting a tenant filter), the database will reject the operation immediately rather than leaking data or corrupting state.
* **Predictable Performance:** Time-sortable UUIDv7s and mandatory property-based indexing ensure the app stays fast as data grows, avoiding the "B-Tree fragmentation" common with standard UUIDs.
* **Bulletproof Audits:** With mandatory audit fields and RESTRICT foreign keys, we can reconstruct the state of any booking or transaction at any point in history.
* **Tenant Isolation:** The architecture is built for multi-property management from day one, with hard-coded isolation at the engine level.

### - Negative/Neutral (The "Costs")
* **Developer Rigor:** Writing migrations takes longer due to mandatory check constraints, explicit naming conventions, and RLS policy definitions.
* **Storage Overhead:** UUIDs, audit fields, and extra indexes occupy more disk space than simple serial integers and flat tables.
* **Transaction Management:** Developers must ensure the "Tenant Handshake" (`SET LOCAL app.current_property_id`) is performed for every database transaction. However, this is solved using helper functions. And it is better that then having exposing other tenants.

## References
* Full Requirements Traceability Matrix: [docs/requirements/rtm.md](../requirements/rtm.md)