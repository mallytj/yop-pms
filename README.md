# Yop PMS

Yop PMS is a modern, high-performance Property Management System designed for reliability and low operational cost. It is built with a type-safe stack from the database to the frontend to minimize runtime errors and improve developer velocity.

## ✨ Features

*   **Type-Safe Stack:** End-to-end type safety from the database schema to the frontend framework.
*   **High Performance:** Built with Go and SvelteKit for a responsive user experience and efficient resource usage.
*   **Multi-Tenancy by Design:** Core architecture supports managing multiple properties with strict data isolation using Row-Level Security.
*   **Schema-First API:** Uses OpenAPI to generate API contracts, ensuring the backend and frontend are always synchronized.

## 🚀 Tech Stack
***Please note that some of these have not yet been implemented, this is an overview of the project***

The project uses a carefully selected stack to balance performance, reliability, and developer experience.

*   **Frontend:** **SvelteKit (with Runes)** for a reactive, minimal-boilerplate UI.
*   **Backend:** **Go** for its performance and concurrency model.
*   **Database:** **PostgreSQL** for robust, ACID-compliant data storage and **Redis** for caching. Managed via `sqlc` for type-safety, migrated by `goose` for go compatibility.
*   **API Specification:** **OpenAPI (Swagger)** for schema-first API development.
*   **Worker:** Transactional Outbox pattern for async tasks (Emails, Webhooks).
*   **Observability:** **OpenTelemetry** (OTel) for distributed tracing.
*   **Containerization:** **Docker** for containerization and orchestration.
*   **CI/CD:** **GitHub Actions** for CI/CD.


For more details, see [ADR-002: Technology Stack Selection](./docs/adr/002-techstack.md).

## 🏗️ Architecture

Yop PMS is built as a **Monorepo** to simplify development and ensure consistency across different parts of the application (backend, frontend, infrastructure).


Key architectural decisions include:

1.  **Monorepo:** A single repository houses all code, documentation, and infrastructure to ensure atomic commits and simplified onboarding. See [ADR-001: Monorepo](./docs/adr/001-monorepo.md).
2.  **Schema-First API:** The API contract is the single source of truth, defined with Swagger comments in the Go backend. This prevents "contract drift" between the frontend and backend. See [ADR-003: Schema-First API Development](./docs/adr/003-schema_first_api.md).
3.  **Core Database Principles:** The database schema is designed for high integrity, multi-tenancy, and performance, using features like UUIDv7 for primary keys and Row-Level Security for data isolation. See [ADR-004: Core Database Principles](./docs/adr/004-core_db_principles.md).

## Getting Started
1. Run `make setup` to install tools and create your `.env`.
2. Run `make dev` to start the backend, frontend, and database.
3. Visit `localhost:8080/swagger/index.html` for API docs.

## Development

When you change the API structure in the Go code, you must regenerate the API specification and the TypeScript types:

```bash
make gen
```

This command updates the `swagger.json`, converts it to `openapi.json`, and generates the frontend `api.d.ts` file automatically. This is also enforced by pre-push git hooks.

## Useful Links
* [Architectural Design Records](./docs/adr/)
* [Property Management ERD](./docs/database/yop-pms-erd.md)
* [Database Conventions](./docs/database/conventions.md)
* [Project Roadmap](./docs/operations/roadmap.md)