# ADR 002: Technology Stack Selection

## Status
**Accepted**

## Context
Yop requires a stack that balances three conflicting needs:
1. **High Reliability:** Room bookings must be ACID-compliant to prevent double bookings. (ACID - atomicity, consistency, isolation, durability)
2. **High Performance:** The dashboard must be reactive and fast for staff use.
3. **Low Operational Cost:** Deployment must be viable on low-resource hardware (e.g., a £5/mo VPS).
4. **Type Safety:** The entire pipeline—from the Database (SQLC) to the Frontend (OpenAPI-TS)—is type-safe.
5. **Documentation:** The API must be well-documented to ensure seamless communication between the frontend and backend.
6. **Observability:** The application must be observable to allow for monitoring and debugging.
7. **Security:** The application must be secure to prevent unauthorized access to data.
8. **Scalability:** The application must be scalable to allow for future growth.
9. **CI/CD:** The application must be deployable without risk of going down

## Decision
We have selected a "Type-Safe, Low-Overhead" stack to minimize both resource usage and development friction.

### The Backend: Go (1.25.8)
* **Reasoning:** Go provides near-C performance with a high-level developer experience. Its lightweight goroutines are ideal for handling concurrent booking requests without the memory overhead of Node.js or Java.
* **Database Access:** **SQLC**. We will write raw SQL and generate type-safe Go code. This avoids the "magic" and performance tax of a traditional ORM like GORM.
* **Worker:** **Transactional Outbox pattern** for async tasks (Emails, Webhooks).
* **Router:** **Chi** for its simplicity and performance.
* **Documentation:** **OpenAPI (Swagger)** for schema-first API development.

### The Frontend: SvelteKit 5 (Runes)
* **Reasoning:** Svelte 5's "Runes" provide a signal-based reactivity system that is more performant and easier to debug than Svelte 4 or React. It allows for complex state management (calendars/grids) with minimal boilerplate.
* **Styling:** **Pure CSS**. To minimize build-step complexity and avoid framework-specific technical debt, we will use modern CSS features (Flexbox, Grid, CSS Variables).

### Storage: PostgreSQL & Redis
* **PostgreSQL:** The primary source of truth. Chosen for its robust ACID compliance and advanced features like exclusion constraints (essential for date-overlap prevention).
* **Redis:** Used for transient state, such as caching room availability and managing JWT blacklists if needed.

### Security: Hybrid JWT + Redis
* **Reasoning:** We will use JWTs for stateless authentication to keep the API fast. However, we will use **Redis** to store Refresh Tokens and a "Token Blacklist." This allows us to revoke access immediately (e.g., on logout or password change) without sacrificing the performance of the primary request flow.

### Deployment: Docker & VPS
* **Reasoning:** To keep costs at a minimum, the application will be containerized using Docker and deployed to a single VPS. This avoids the high "cloud tax" of managed platforms like AWS or Vercel.

### Continuous Integration: GitHub Actions
* **Reasoning:** All the project is currently already in GitHub, so it makes sense to use GitHub Actions for CI/CD.
  
### Observability: OpenTelemetry
* **Reasoning:** OpenTelemetry is a vendor-neutral observability framework that provides a comprehensive set of tools for collecting and analyzing telemetry data. It is a lightweight and flexible solution that can be used to monitor the performance of the application.

## Consequences

### + Positive (The "Wins")
* **Extreme Efficiency:** The Go binary and SvelteKit SSR will run comfortably on a machine with 1GB of RAM.
* **Type Safety:** The entire pipeline—from the Database (SQLC) to the Frontend (OpenAPI-TS)—is type-safe.
* **No "Lock-in":** By using Pure CSS and Standard SQL, the core logic is not tied to a specific UI framework or ORM.

### - Negative/Neutral (The "Costs")
* **CSS Verbosity:** Without a utility framework like Tailwind, we must be disciplined with our CSS organization to avoid "spaghetti" styles
* **Language Inconsistency:** Using a pure JS approach would mean that a lot of the code is transferrable. However, we will deal with it through API contracts to communicate and keep type safety consistent and clear.

## Alternatives Considered
* **Node.js/Express:** Rejected due to higher memory consumption and lack of native type-safe concurrency
* **React/Next.js:** Rejected due to the "Virtual DOM" overhead and complex state management patterns (Hooks)
* **SQLite:** Considered for cost, but rejected in favour of PostgreSQL to support advanced concurrency features
* **NoSQL such as MongoDB::** Considered for DX, but rejected in favour of PostgreSQL for advanced relationship management

## References
* [ADR-003: Schema First API](./003-schema_first_api.md) - *For seamless communication between the front and backend*