# Web AGENTS.md

SvelteKit 5 (runes). Vitest + jsdom + testing-library. Pure CSS.

## Rules
- **No emojis anywhere** — use Lucide icons (or other icon components) instead.
- **Icons over emojis always.**

## Testing and Type Checking
- **Type check:** Always run `npm run check` after any code changes. Never skip.
- **Framework:** Vitest (`*.test.ts`, colocated besides source file — never a
  shared `__tests__/` directory)
- **Rule:** Cover everything testable. Public interface only, not internals.
- **Svelte components:** Use `@testing-library/svelte` render + queries.
- **Store defaults:** Never rely on default values in tests. Always set input
  explicitly before rendering so a default change won't break the test.
- **One test file per source file.** Split multi-concern test files (e.g.
  `stores.test.ts` testing 2+ stores) into individual colocated files.

## Components

```
src/lib/components/       ← Shared (used by ≥2 routes)
src/routes/<name>/_components/  ← Route-local (single route only)
```

Break components into smallest readable units.

When a route-local component becomes shared, move it to `src/lib/components/`.

## Stores

**Module-level `$state`** for global UI state (shell, topbar).
**Class + context** for scoped state (per-component/per-route).

```ts
// ✅ Module-level for globals
const state = $state(0)
export const myStore = { get value() { return state }, set value(v) { state = v } }

// ✅ Class + context for scoped
class MyStore { value = $state(0) }
const key = Symbol('my-store')
export function getMyStore() { return getContext<MyStore>(key) }
```

## Path Aliases

```
$lib        → src/lib
$components  → src/lib/components
$helpers    → src/helpers
$types      → src/lib/types
$actions    → src/actions
```
