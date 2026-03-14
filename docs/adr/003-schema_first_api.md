# ADR 003: Schema-First API Development

## Status
**Accepted**

## Context
In a monorepo with a Go backend and a TypeScript (SvelteKit) frontend, there is a high risk of "Contract Drift." If a developer changes a field name in a Go struct but forgets to update the frontend's fetch logic, the application will crash at runtime. 

Manual synchronization of types across two different languages is error-prone and slows down development velocity.

## Decision
We will adopt a **Schema-First** (or Design-First) approach using **OpenAPI 3.0/Swagger** as the single source of truth.

1.  **Definition:** The API is defined via Swagger comments in the Go source code.
2.  **Generation:** Every time the API changes, we run `make swag` to generate a `swagger.json` file.
3.  **Synchronization:** We use `openapi-typescript` to convert that JSON spec into a native TypeScript definition file (`api.d.ts`) inside the `web/` directory.

## Consequences

### + Positive (The "Wins")
* **End-to-End Type Safety:** If a backend field is renamed, the SvelteKit project will fail to compile until the frontend code is updated to match.
* **Auto-Documentation:** We get a live Swagger UI for free at `/swagger/index.html`.
* **Reduced Human Error:** Developers no longer guess what an API returns; the IDE provides autocomplete based on the generated types.

### - Negative/Neutral (The "Costs")
* **Build Step Dependency:** The frontend developers must run the generation script (`make gen`) whenever the backend API structure changes. This is at the repo level by prepush scripts.
* **Tooling Overhead:** Requires maintaining `swag`, `swagger2openapi`, and `openapi-typescript` in the `setup.sh` and `Makefile`.

## References
* Related to [ADR-002: Tech Stack](./002-techstack.md)