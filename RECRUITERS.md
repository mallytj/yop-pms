# Yop PMS — Why you should interview me

I built this. Alone. After hours. It's a production-grade Property Management System — the kind hotels pay thousands a month for. I wrote every line of Go, every SQL migration, every Svelte component, every ADR, every test.

Not a tutorial project. Not a bootcamp final. Not "full-stack blog app #473."

**Here's what it proves about how I think.**

---

## I design for correctness at the database level

No application code can double-book a room. The database enforces it.

```sql
EXCLUDE USING GIST (assigned_room_id WITH =, stay_period WITH &&)
WHERE (deleted_at IS NULL AND assigned_room_id IS NOT NULL)
```

This is one constraint. There are dozens more — CHECK constraints on every text column, UNIQUE on soft-deleted rows via partial indexes, RESTRICT on every foreign key so nothing disappears accidentally.

I wrote a tool that reads these constraints from the live database and generates validation code for both Go structs and TypeScript types. One source of truth. Zero drift. If I add a constraint in a migration, the frontend forms update automatically.

**Employers should care because:** I don't trust application code alone. I push invariants to the database where they cannot be bypassed. That's the difference between "it works on my machine" and "it works in production."

---

## I build systems that survive crashes

When a reservation is created, two things happen in one database transaction:

1. The reservation row is inserted
2. An outbox event is enqueued

If the server crashes between step 1 and step 2, the reservation rolls back. The email is never lost because it was never committed without its trigger. A background worker polls with `SELECT FOR UPDATE SKIP LOCKED`, retries with exponential backoff, and dead-letters after 3 failures.

The idempotency middleware uses Redis and an `Idempotency-Key` header. If the guest's browser retries the POST — because the network timed out, because the user double-clicked, because the payment provider was slow — the mutation runs exactly once. No duplicate charges. No duplicate confirmations.

Holds expire via background sweep. If a worker crashes mid-sweep, the 5-minute visibility timeout means the next poll reclaims the abandoned rows. No manual cleanup. No stale inventory.

**Employers should care because:** I think about what breaks. I don't build happy-path-only software. Every crash scenario has a recovery mechanism.

---

## I document every decision with alternatives considered

There are 22 Architecture Decision Records in `docs/adr/`. Each one answers: "What did we decide, what were the alternatives, and why did we reject them?"

For example: the reservation locking model. Three options considered:
- Separate "room lock" table with TTL — rejected (two state machines that drift)
- Redis distributed lock — rejected (Redis is transient; DB can't observe it)
- Actual architecture chosen: the `hold` status *is* the lock. The reservation state machine governs availability. One mechanism.

Every code header cites its traceability:

```go
// Core Requirements: R-RES-CRUD-005, R-RES-CRUD-006, ADR-015
```

This means when I come back to a file six months later, I know exactly which decisions constrained it and which requirements it satisfies.

**Employers should care because:** Documentation isn't an afterthought — it's part of engineering. I write for the next person who has to understand my code. That next person might be you.

---

## I build for observability from day one

OpenTelemetry traces every HTTP request through every middleware, every database query, every Redis call. Structured JSON logging via `slog` with trace IDs propagated through context. I can tell you exactly which span was slow in any request.

```
GET /v1/reservations/550e8400-e29b-41d4-a716-446655440000
  ├── middleware.RequestLogger (2ms)
  ├── middleware.Idempotency (1ms)
  ├── service.GetReservation (45ms)
  │   ├── store.GetReservationByID (38ms) ← slow query candidate
  │   └── cache.Get (3ms)
  └── json.WriteJSON (1ms)
```

**Employers should care because:** Observability isn't a nice-to-have. It's how you debug production without guessing. I treat it as infrastructure, not an afterthought.

---

## The tech stack tells a deliberate story

| Layer | Choice | Why not the obvious alternative |
|-------|--------|--------------------------------|
| Backend | Go + Chi | Not Node.js (runtime errors). Not Python (GIL, deployment). Go for performance + simplicity |
| Database | PostgreSQL 18 + SQLC | Not an ORM. Raw SQL with type-safe code generation. No hidden N+1 queries |
| Frontend | SvelteKit 5 (Runes) | Not React (boilerplate). Compiled, minimal JS output, native reactivity |
| API contract | Swagger → OpenAPI → TS types | Not hand-written types. One change propagates from Go struct to frontend type automatically |
| Caching | Redis + PostgreSQL NOTIFY | Not TTL-polling. Freshness within milliseconds of any mutation |
| CI/CD | GitHub Actions | `go vet`, `golangci-lint`, `svelte-check`, `govulncheck` per push |
| Container | Docker scratch (~30MB) | Multi-stage build, statically linked, non-root user |

Every choice has a justification. None are "because it's popular."

**Employers should care because:** I evaluate technology against the problem, not against the job market. I can justify my decisions.

---

## What's coming next

- **Room Tetris** — RL agent (tabular Q-learning) for inventory allocation. When a booking would be rejected, the agent searches upgrade chains across the room board. Trained via Python simulator, runs in Go at inference time. Full audit trail.
- **Hey Yop** — voice-driven PMS operations via LLM with structured output.
- **Guest Desire Engine** — Bayesian attribute inference + EMA sentiment scoring from call transcripts and booking history.
- **Public booking engine** — embeddable widget that bypasses OTA commissions.

---

## What this says about me

- I can build a full-stack system from database schema to CSS
- I think about failure modes, not just happy paths
- I document decisions and trace code to requirements
- I choose technology deliberately, not by popularity
- I work on this after my day job because I love building

**If you want a junior who ships code that survives production, understand trade-offs, and can explain why every line exists — I'm your candidate.**

---

*Built with Go 1.25, Chi, PostgreSQL 18, Redis 7, SvelteKit 5, SQLC, Goose, OpenTelemetry. CI via GitHub Actions. ~30MB Docker image. 22 ADRs. 100% self-taught, 0% tutorial code.*
