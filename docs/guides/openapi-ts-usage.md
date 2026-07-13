# Using OpenAPI in Typescript

In a production-grade SaaS, manually managing interfaces that mirror the backend
wiill become heavily timeconsuming.

## The Strategy

Use `openapi` combined with `swagger` for contract generation Use
`openapi-fetch` for the requests

See [ADR-001](../adr/001-schema-first-api.md)

## Setup

Create one instance of the openapi fetch client

```ts
// src/lib/api/client.ts
import createClient from "openapi-fetch";
import type { paths } from "./schema";

const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export const api = createClient<paths>({
  baseUrl: API_BASE_URL,
});
```

## Type-safe Data Fetching

Use `openapi-fetch` alongside the `client.ts` to make requests to defined routes

```ts
// src/routes/health/+page.server.ts
import { api } from "$lib/api/client";
import { error } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ fetch }) => {
  // Pass SvelteKit's fetch to the client to handle SSR correctly
  const {
    data,
    error: apiError,
    response,
  } = await api.GET("/healthz", {
    fetch: fetch,
  });

  if (apiError || !data) {
    throw error(response.status || 500, {
      message: data?.message || "Service Unavailable",
    });
  }

  return { health: data };
};
```

## Response Types

To keep components consistent - create specific types from generated
`components` interface

```ts
// src/lib/types/api.ts
import type { components } from "$lib/api/schema";

export type HealthResponse = components["schemas"]["cmd_server.HealthResponse"];
export type ServiceHealth = components["schemas"]["cmd_server.ServiceHealth"];
```
