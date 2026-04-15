# Frontend Constraints

Single source of truth for validation rules synced from PostgreSQL. **Never edit `web/src/lib/types/constraints.g.ts` manually.**

## 1. Import

```typescript
import { CONSTRAINTS } from '$types/constraints.g';
```

The `$types` alias resolves to `web/src/lib/types/`.

## 2. Production Usage

### Native HTML Binding (Simplest)

Bind constraints directly to attributes. The browser enforces these for free.

```svelte
<script lang="ts">
  import { CONSTRAINTS } from '$types/constraints.g';
  const u = CONSTRAINTS['auth.users'].fields;
</script>

<input
  name="username"
  required={u.username.required}
  maxlength={u.username.maxLength}
  pattern={u.username.pattern?.source}
/>
```

### Svelte 5 Reactive Validation

Use `$derived` for custom error messages and UI feedback.

```svelte
<script lang="ts">
  import { CONSTRAINTS } from '$types/constraints.g';
  const f = CONSTRAINTS['auth.users'].fields.username;

  let username = $state('');

  const error = $derived(
    f.required && !username ? 'Required' :
    f.pattern && !f.pattern.test(username) ? 'Invalid format' :
    null
  );
</script>

<input bind:value={username} />
{#if error}<span class="error">{error}</span>{/if}
```

## 3. Supported Rules

| Rule          | HTML Attribute | Description                             |
| :------------ | :------------- | :-------------------------------------- |
| `required`    | `required`     | Field cannot be empty                   |
| `maxLength`   | `maxlength`    | Maximum string length                   |
| `min` / `max` | `min` / `max`  | Numeric range (for `type="number"`)     |
| `pattern`     | `pattern`      | Native `RegExp` object                  |
| `exactLength` | -              | Fixed width (e.g. currency codes)       |
| `validRange`  | -              | For date ranges (enforce `end > start`) |

## 4. Regeneration

Run after any database schema change:

```bash
make gen-constraints
git add config/constraints.g.yml web/src/lib/types/constraints.g.ts
```
