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

## ✅ PR 2: Infrastructure

**Goal:** Implement the infrastructure for the rest of the project that ensure
the system is reliable, traceable and safe

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

## PR 3: Booking Engine (Future) `feature/booking-engine-core`

### 1. Reservation Backend `feat/reservation-api`

- [ ] Create `docs/rtm/reservations.md` for the reservation requirements
- [ ] Create `docs/adr/013-room-locking-strategy.md` for the architectural
      decision on handling concurrent reservation attempts on the same room/date
- [ ] Create `docs/flows/reservation.md` for sequence diagrams of the reservation
      flows both green and red paths
  - The `SKIP LOCKED` or `SELECT FOR UPDATE` strategy for handling concurrent
    reservation attempts on the same room/date
  - Link to Websocket or similar for real time frontend in future
- [ ] Create `docs/adr/014-real-time-frontend-updates.md` for the architectural
      decision on handling real time updates to the frontend, such as new
      reservations, cancellations or modifications
- [ ] Create `internal/booking/[handler/service].go` for the reservation domain
      logic
- [ ] Build swagger contracts for the routes
- [ ] Add to `docs/guide` and `README.md`

- [ ] Fill the handler and service with the reservation logic, ensuring to
      handle edge cases such as overlapping reservations, cancellations,
      and modifications
- [ ] Write tests for edge cases, i.e for overlapping reservation attempts to
      ensure the locking strategy works as intended

### 2. Planner Backend `feat/planner-api`

- [ ] Create `docs/rtm/planner.md` for the planner requirements
- [ ] Create `docs/flows/planner.md` for sequence diagrams of the planner flows
      both green and red paths
- [ ] Create `internal.planner/[handler/service/types].go` for the planner
      domain logic
- [ ] Create the response and request shape - ensuring to factor in the needs of
      the frontend, such as the infinite scroll and drag and drop functionality
- [ ] Build swagger contracts for the routes
- [ ] Add to `docs/guide` and `README.md`

- [ ] Fill the handler and service with the reservation logic, ensuring to
      handle edge cases
      such as overlapping reservations, cancellations, and modifications
- [ ] Write tests for edge cases, i.e for invalid date ranges, to ensure the
      planner can handle them gracefully

### 3. Seeding `feat/seeding`

- [ ] Create `docs/adr/015-seeding-approach.md` for the architectural decision
      on handling seeding of the database for both local development and E2E
      tests, such as whether to use SQL
      scripts, Go code or a combination of both
- [ ] Create `cmd/seed` and `internal/platform/seeding` package for seeding the
      database with test data, such as rooms, reservations, and rates to support
      the E2E tests and local development

### 4. Planner Frontend `feat/planner-ui`

Overall frontend colour-scheme and design approach must be established through
screenshots and a live design-system route for easy viewing and reference by
the frontend team

_Maybe split into `feat/design-system` and `feat/planner-ui` if it gets too big?_

- [ ] Create `docs/design/README.md` for the design guidelines and principles for
      the frontend, such as the colour scheme, typography, and overall style to
      ensure consistency across the application
- [ ] Create `docs/colours.md` for the colour palette, including hex codes and
      usage examples to ensure a cohesive and visually appealing design
- [ ] Create `docs/design/planner.md` for the specific design of the planner UI,
      such as the layout of the calendar, the style of the reservation blocks, and
      the drag and drop functionality to ensure it meets the requirements and is
      user-friendly
- [ ] Add `docs/design` to `docs/README.md`

- [ ] Create `web/src/routes/design/+page.svelte` — Main design system entry
      point, links to all design pages
- [ ] Create `web/src/routes/design/colours/+page.svelte` — Colour palette
      showcase with hex codes and usage examples
- [ ] Create `web/src/routes/design/typography/+page.svelte` — Font scales,
      weights, line heights with live examples

- [ ] Infinite horizontal scroll of the calendar, with lazy loading of
      reservations as the user scrolls
- [ ] Create `docs/adr/016-tape-chart-approach.md` for the architectural
      decision on handling the planner UI, such as the tape chart approach vs a
      calendar approach also on the Svelte solution to implementing the infinite
      scroll calendar
- [ ] Create `web/src/lib/planner/README.md` for the documentation of the planner
      UI components, such as the calendar, reservation blocks, and drag and drop
      functionality to ensure they are easily understandable and reusable for
      future features
- [ ] Drag and drop functionality for modifying reservations, with real-time
      updates to the backend
- [ ] Clear warnings & errors for invalid use of planner, such as overlapping
      reservations or invalid date ranges
- [ ] Simple 'notion-like' colour scheme - factor over form
- [ ] Ensure all data needed is shown - and no more
- [ ] Multi-select support for creating multiple reservations in future
- [ ] Built for scale - all UI components should be reusable and modular to
      support future features such as the multi-select and real-time updates
- [ ] Tablet friendly - buttons must be large enough to tap on an iPad

### 5. Booking Engine Frontend

- [ ] Multi step reservation form with clear progress indicators
- [ ] Form validation - potentially with Zod?
- [ ] Create `docs/adr/017-form-validation.md` for the decision on handling form
      validation, such as whether to use a library like Zod or to implement custom
      validation logic in the frontend
- [ ] Clear style, only show information necessary for each step to avoid
      overwhelming the user
- [ ] Build for speed of use, allow for import of guest details in future to
      speed up the process for repeat guests
- [ ] Allow for future scalability - for example, adding card details or
      Hey Yop integration for operational efficiency in the future, so the form
      should be built in a way that allows for easy addition of new steps or
      fields without needing to overhaul the whole form

### 6. E2E Tests

Rate management will be added in future. Seeded for now

- [ ] Create E2E tests for the booking engine flows, both for the reservation
      and planner
  - Happy paths
  - Edge cases such as overlapping reservations, invalid date ranges, etc
