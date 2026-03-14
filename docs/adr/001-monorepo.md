# ADR 001: Monorepo

## Status
**Accepted**

## Context
**What is the problem we are solving?**
Yop will consist of multiple interconnected components:
- **Backend**: A REST API handling high-concurrency booking logic
- **Frontend**: A dashboard for hotel staff
- **Contracts**: OpenAPI specifications to handle the "handshake" between the two
- **Infrastructure/DevOps**: Docker Compose, database migrations, setup scripts, CI/CD pipelines

A `polyrepo` approach, where each is separated into different repos often lead to **"Contract Drift"** - where the backend and frontend desynchronize. It also hurts the developing experience as it would be constant `git clones` etc

## Decision
**What are we doing about it?**
We will use a **Monorepo** architecture to house all project code, documentation and infrastructure.

**Proposed Directory Structure:**
```
.
├── api/                # OpenAPI/Swagger specs (The Source of Truth)
├── cmd/                # Entry points for the application
│   └── server/         # main.go lives here
├── docs/               # ADRs, ideas, business logic diagrams, onboarding etc
├── internal/           # Private code (cannot be imported by other projects)
│   ├── booking/        # Domain: Logic, Interfaces, Service Layer
│   ├── auth/           # Domain: Authentication/Authorization
│   ├── platform/       # Cross-cutting concerns (DB, Logger, Middleware)
│   └── store/          # SQLC generated code & repository implementations
├── web/                # SvelteKit application (replaces 'js/')
├── migrations/         # SQL migration files (raw SQL)
├── scripts/            # Build/Dev scripts (seeders, contract builders, audits)
├── Makefile            # The project's remote control
└── docker-compose.yml
```

Please note that this will continuously be updated, however all updates will be defined in the README.md.


## Consequences
**What is the aftermath of this decision?**

### ✅ Positive (The "Wins")
* **Atomic Commits:** Changes to the backend will automatically sync to the frontend. This ensures there is no discrepancies in production
* **Shared Tooling:** All code is in the same repository, making it easier to maintain and update. A single `docker-compose.yaml` and a single `Makefile` orchestrates the whole system
* **Simplified Onboarding:** A new developer only needs to clone the repo and then run `make setup` to have the entire environment ready
* **Transparency:** All ADRs and other documentation is readily available for both front and backend developers. 

### ⚠️ Negative/Neutral (The "Costs")
* **CI/CD Complexity:** As the project grows, we will need to implement "path-based" triggers in our CI/CD pipeline to avoid rebuilding redundant sides of the project (only frontend changes, and we rebuild both)
* **Tooling Conflicts:** We must be careful to keep `node_modules` and Go binaries separated. However, this is easily handled by `.gitignore` and targeted `Makefile` commands.

## Alternatives Considered
* **Polyrepo:** Rejected due to the high overhead of managing separate versioning for a small, high-velocity team.
* **Git Submodules:** Rejected because submodules are essentially "the worst of both worlds"—manual synchronization and complex Git state.