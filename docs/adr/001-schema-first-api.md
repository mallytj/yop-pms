# ADR 001: Schema-First API

## Status

**Accepted**

API is defined via Swagger comments in Go handlers; `make swag` produces `swagger.json`; `openapi-typescript` generates `web/src/lib/types/api.d.ts`. SvelteKit compile fails if backend field renames go unmatched in frontend. Live docs at `/swagger/index.html`.

Alternatives: manual TS type sync (error-prone), tRPC (frontend coupling to RPC shape), GraphQL (overshoot for 10-domain PMS).

---

See: `cmd/server/api.go`, `Makefile` (`make gen`)
