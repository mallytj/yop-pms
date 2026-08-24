---
name: writing-ts
description:
  Use when writing or reviewing TypeScript, Svelte 5, frontend tests, API
  adapters, or type errors; keep boundaries typed and names domain-specific.
scope: project
---

# Writing TypeScript

Use explicit domain types at boundaries. Let the compiler expose contract
problems; do not hide them.

## Procedure

### 1. Read the contract

Check generated OpenAPI types, nearby code, and the Svelte 5 runes conventions
before editing. Treat generated API types as authoritative; adapt external data
in a named boundary function.

**Done when:** inputs, outputs, failure states, and generated types are
identified before implementation.

### 2. Model valid states

Use discriminated unions when states exclude one another. Use interfaces for
component props. Name booleans with `is`, `has`, `can`, or `should`; name
functions with verbs and values with domain meaning.

Prefer `const`, early returns, explicit transformations, and immutable inputs.
Use `satisfies` when validating object shape while preserving literals.

**Done when:** types prevent invalid states and names explain intent without
comments.

### 3. Narrow at boundaries

Treat API responses, URL parameters, form data, local storage, and
external-library values as untrusted. Start with `unknown`, validate with a type
guard or schema parser, then pass the narrowed value inward.

Use `import type` for type-only imports. Fix the source contract or create a
named adapter when data disagrees with generated types.

**Done when:** every untrusted value has one visible validation path before
domain logic uses it.

### 4. Compose Svelte components

Use `$state`, `$derived`, `$effect`, and `$props` consistently. Extract visual
or interactive regions into named components. Keep side effects in `$effect` or
named actions, and keep callback contracts typed.

**Done when:** markup has clear boundaries and no section comment is standing in
for a component.

### 5. Handle failures deliberately

Handle errors where recovery, feedback, or logging is possible. Preserve context
when wrapping errors. Represent expected failure states in types when callers
branch on them.

**Done when:** each failure path either recovers, reports useful context, or
propagates intentionally.

### 6. Verify

Run `npm run check` and relevant tests. Search touched files and the diff for
`as unknown as`, `any`, `var`, vague names (`data`, `value`, `item`, `thing`,
`helper`), and section banners.

**Done when:** checks pass and every search hit is removed, narrowed, renamed,
or explicitly justified at a boundary.

## Guardrails

- Never use `as unknown as X`, chained casts, or casts to silence an error.
- Use `any` only where an external declaration forces it; isolate and document
  that adapter. Prefer `unknown` plus runtime narrowing everywhere else.
- Do not mutate arguments or shared state outside an explicit store/API
  boundary.
- Do not duplicate generated API types.
- Avoid abbreviations, unexplained single-letter names, clever one-liners, and
  deeply nested conditionals.
