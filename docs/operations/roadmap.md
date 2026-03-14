# Project Roadmap

## ✅ PR 1: Foundation & Skeleton (Current)
**Goal:** Start fresh, clean slate the whole project for future scalability
- [ ] Project directory restructuring
- [ ] Database migration overhaul (clean slate)
- [ ] Update architecture (install `air`, update dockerisation, Makefiles etc)
- [ ] Initialise the Chi router and base middleware
- [ ] Create first route `/heathz`
- [ ] Lay down foundations for contract based API architecture
- [ ] Graceful shutdown handler for `SIGTERM`

## PR 2: Infrastructure (Next)
**Goal:** Implement the infrastructure for the rest of the project that ensure the system is reliable, traceable and safe

**1. Observability Layer**
- [ ] **OpenTelemetry Integration:** Initalise tracker provider in `main.go`
- [ ] **Middleware:** Add `otelchi` for HTTP tracing and `otelpgx` for database spans
- [ ] **Structured Logging:** Continue on the `logger *slog.Logger` from `main.go`

**2. Reliability & Safety**
- [ ] **Idempotency Middleware:** Redis check for `Idempotency-Key` on `POST/PATCH` to ensure no double transactions where necessary
- [ ] **Global Errors:**  Standardised `ApiErrors` using `platform/apierrors` in Go and a centralised `config/errors.json` naming convention for consistency
- [ ] **Constraint Consistency:** Read `CHECK` constraints from migrations, convert into `constraints.yml` then distribute to backend and frontend

**3. Background Worker**
- [ ] **Outbox Schema:** Create `admin.outbox_events` table with indexes for `SKIP LOCKED`
- [ ] **Worker Engine:** Background Goroutine with a polling loop and `outbox_events` listeners
- [ ] **Task Handlers:** Initial system notifications handler (e.g log/slack)
- [ ] **Dead Letter Stategy:** Only run a task 5 number of times, if it fails set as a `dead_outbox_event` (will be handled in another area)

**4. Reactive Cache & Events**
- [ ] **Event Listener:** Create `platform/events` package to subscribe to (to start with) PostgreSQL `LISTEN/NOTIFY` events
- [ ] **Cache Package:** Create `platform/cache` package to handle the caching logic with Go and Redis
- [ ] **Cache Invalidator:** Create a cache invalidator to handle reactivity

## PR 3: Booking Engine (Future)
- [ ] **Real-time Frontend:** Websocket integration for the planner
- [ ] **Well-Documented Flows:** All complex flows must be documented in [docs/flows](./docs/flows)