Status: not-started

## Scope

SSE hub for real-time frontend updates via PostgreSQL triggers + Go hub +
SvelteKit EventSource. Covers all mutation paths (handlers, workers, admin
tools) since triggers fire automatically.

## Branch

`feat/reservation/sse`

## Files

| File | Action |
| :--- | :----- |
| `migrations/NNN-sse-notify-triggers.sql` | create (single migration, reversible) |
| `internal/platform/realtime/hub.go` | create |
| `internal/platform/realtime/hub_test.go` | create |
| `web/src/lib/realtime/stream.svelte.ts` | create |
| `cmd/server/main.go` | modify (wire Hub + register listener handlers + onReconnect resync) |
| `cmd/server/api.go` | modify (mount `GET /v1/sse` inside `/v1` route group) |

## Dependencies

Blocked by: handlers-router, events-listener
Blocks: —
