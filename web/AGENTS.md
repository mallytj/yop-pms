# Web AGENTS.md

SvelteKit 5 (runes). Vitest + jsdom + testing-library. Pure CSS + Tokens.

---

## 1. Principles

- **Self-documenting code**: Expressive names > comments, add comments
  explaining nonsensical code
- **No emojis**: Lucide icons (`@lucide/svelte`) only.
- **Notion/Linear UX**: High contrast, subtle borders, focus states, clear hierarchy.
- **Functional colors**: No raw hex/decorative colors. Use semantic tokens (`var(--color-text-muted)`).

---

## 2. API & Data (OpenAPI First)

1. Read **OpenAPI spec** first before writing/changing API calls.
2. Only check backend source if OpenAPI spec has bugs/inconsistencies.
3. Infer client types from generated OpenAPI types.

---

## 3. Comments Policy

- **Banned**: What code does, state updates, section dividers, dead code.
- **Allowed**: Why non-obvious business/domain logic exists, browser/library workarounds.
- If a comment names a visual or logical section, extract a component or function instead.
- Never use `as unknown as X`, `any`, or a cast to silence a type error. Validate and narrow untrusted values at boundaries.
- Use domain-specific names. Avoid vague names such as `data`, `value`, `item`, and `helper` when a precise name exists.

---

## 4. Components & Bits UI

- Structure: `$lib/components/` (shared) | `src/routes/<name>/_components/` (route-local).
- **Bits UI**: Use as unstyled backbone for all interactive primitives (Dialog, Dropdown, Popover, Select, Tooltip).
- Style Bits UI primitives with scoped Pure CSS + CSS variables.

```svelte
<script lang="ts">
	import { Dialog } from 'bits-ui';
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		open?: boolean;
		title: string;
		children: Snippet;
	}
	let { open = $bindable(false), title, children }: Props = $props();
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="overlay" />
		<Dialog.Content class="content">
			<header>
				<Dialog.Title>{title}</Dialog.Title>
				<Dialog.Close><X class="icon-sm" /></Dialog.Close>
			</header>
			{@render children()}
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>

<style>
	.overlay {
		position: fixed;
		inset: 0;
		background: var(--color-bg-overlay);
	}
	.content {
		position: fixed;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		background: var(--color-bg-surface);
		border: 1px solid var(--color-border-subtle);
		border-radius: var(--radius-md);
	}
</style>
```

---

## 5. Design Tokens (`src/app.css`)

View `src/app.css` for our design tokens

## 6. State Management

- **Module `$state**`: Global app/shell state.
- **Class + Context**: Scoped component/route state.

```ts
// Global
const currentTheme = $state<'light' | 'dark'>('light');
export const userPrefs = {
	get theme() {
		return currentTheme;
	},
	setTheme(t: 'light' | 'dark') {
		currentTheme = t;
	}
};

// Scoped
class SessionStore {
	activeId = $state<string | null>(null);
}
const KEY = Symbol('session');
export const setSession = () => setContext(KEY, new SessionStore());
export const getSession = () => getContext<SessionStore>(KEY);
```

---

## 7. Testing & Quality

- **Type Check**: Always run `npm run check` post-changes. Never skip.
- **Framework**: Vitest colocated (`foo.ts` -> `foo.test.ts`). No `__tests__/` dirs.
- **Rules**: Test public interfaces/user flows only. 1 test file per source file. Set input state explicitly (no default reliance).

### Mutation testing (`npm run test:mutation`)

Stryker mutates `src/routes/tape-chart/**/_utils/**/*.ts`, `src/lib/helpers/dates.ts`,
`src/lib/api/tape-chart.ts` (see `stryker.config.js`). A surviving mutant is a real test
gap until proven otherwise — kill it with a real assertion first.

- Reports print truncated (`and N more tests!`). For the full survived/no-coverage list,
  rerun with `--reporters json,clear-text` and read `reports/mutation/mutation.json`.
- Boundary/guard survivors (`<` vs `<=`, `>` vs `>=`, short-circuit conditions) usually
  mean the test never exercised the exact boundary, or never asserted a value that would
  differ if the guard were skipped (e.g. flip which side of a comparison "wins").
- `??` vs `&&` survivors on a Map/object key built from a fallback value: write a test
  where two distinct inputs collide onto the same fallback key, not just one input with
  the fallback active.
- Only mark a survivor `// Stryker disable next-line <Mutator>` (with a one-line comment
  explaining why) when you've proven — empirically (e.g. in `node -e`), not just by
  inspection — that no possible input makes the mutant's output observably differ from
  the original, across every code path that reads the value. Never disable a mutator just
  to avoid writing a legitimate test.

---

## 8. Aliases

Use aliases over relative imports (`../../` banned past 1 level):
`$lib`, `$components` (`src/lib/components`), `$helpers`, `$types`, `$actions`
