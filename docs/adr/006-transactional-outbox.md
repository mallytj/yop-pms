# ADR 006: Transactional Outbox

## Status

**Accepted**

Handlers enqueue async work (emails, webhooks, provider calls) by inserting a row into `internal.outbox_events` in the same transaction as the domain mutation. A poll loop claims rows with `FOR UPDATE SKIP LOCKED`, sets `process_at = now() + 5min` (visibility timeout for crash recovery), and retries on transient failure with `min(2^n, 1800)s` backoff. Exhausted rows emit `pg_notify('outbox_dead_lettered')`. Outbox is async delivery only — durable audit lives in `auth.audit_logs` (see ADR-008).

Alternatives: inline goroutines (lost on crash), Redis queue (transient state; rollback race), RabbitMQ/NATS (infra overhead for 5-property PMS), `LISTEN/NOTIFY` as delivery (no persistence).

---

See: `internal/platform/worker/`, `migrations/00006_outbox_worker.sql`, `docs/guides/platform-layer.md#outbox-worker`
