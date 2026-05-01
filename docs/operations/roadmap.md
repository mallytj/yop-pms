# Project Roadmap

## ✅ PR 1: Foundation & Skeleton

**Goal:** Start fresh, clean slate the whole project for future scalability

- [x] Project directory restructuring
- [x] Database migration overhaul (clean slate)
- [x] Update architecture (install `air`, update dockerisation, Makefiles etc)
- [x] Initialise the Chi router and base middleware
- [x] Create first route `/heathz`
- [x] Lay down foundations for contract based API architecture
- [x] Graceful shutdown handler for `SIGTERM` and `SIGINT`

## PR 2: Infrastructure (Current)

**Goal:** Implement the infrastructure for the rest of the project that ensure the system is reliable, traceable and safe

### **1. Observability Layer**

- [x] **OpenTelemetry Integration:** Initalise tracker provider in `main.go`
- [x] **Middleware:** Add `otelchi` for HTTP tracing and `otelpgx` for database
      spans
- [x] **Structured Logging:** Continue on the `logger *slog.Logger` from `main.go`

### **2. Reliability & Safety**

- [x] **Idempotency Middleware:** Redis check for `Idempotency-Key` on
      `POST/PATCH` to ensure no double transactions where necessary
- [x] **Global Errors:** Standardised `ApiErrors` using `platform/apierrors` in
      Go
- [x] **Constraint Consistency:** Read `CHECK` constraints from migrations,
      convert into `constraints.yml` then distribute to backend and frontend

### **3. Background Worker**

- [x] **Outbox Schema:** Create `admin.outbox_events` table with indexes for
      `SKIP LOCKED`
- [x] **Worker Engine:** Background Goroutine with a polling loop and
      `outbox_events` listeners
- [x] **Task Handlers:** Initial system notifications handler (e.g log/slack)
- [x] **Dead Letter Stategy:** Only run a task 5 number of times, if it fails
      set as a `dead_outbox_event` (will be handled in another area)

### **4. Reactive Cache & Events**

- [x] **Event Listener:** Create `platform/events` package to subscribe to (to
      start with) PostgreSQL `LISTEN/NOTIFY` events
- [x] **Cache Package:** Create `platform/cache` package to handle the caching
      logic with Go and Redis
- [x] **Cache Invalidator:** Create a cache invalidator to handle reactivity

## PR 3: Booking Engine (Future)

### 1. Reservation Backend

- [ ] Create `docs/rtm/reservations.md` for the reservation requirements
- [ ] Create `docs/flows/reservation.md` for sequence diagrams of the reservation
      flows both green and red paths
  - The `SKIP LOCKED` or `SELECT FOR UPDATE` strategy for handling concurrent
    reservation attempts on the same room/date
  - Link to Websocket or similar for real time frontend in future
  -
- [ ] Create `internal/booking/[handler/service].go` for the reservation domain
      logic
- [ ] Build swagger contracts for the routes
- [ ] Add to `docs/guide` and `README.md`

- [ ] **Real-time Frontend:** Websocket integration for the planner
- [ ] **Well-Documented Flows:** All complex flows must be documented in [docs/flows](./docs/flows)
