---
name: writing-comments
description: Use when writing or reviewing comments, names, component boundaries, or exported API documentation in Go, TypeScript, Svelte, SQL, or tests.
scope: project
---

# Writing Comments

Comments are a last-mile explanation. Code carries intent through names, types, boundaries, and decomposition.

## Procedure

### 1. Find the missing explanation

Read the changed code and ask what a new maintainer cannot infer from its names and types. Identify business rules, external contracts, safety constraints, and surprising consequences.

**Done when:** every proposed comment names a specific non-obvious fact.

### 2. Improve the code first

If a comment labels a visual or logical section, extract a named Svelte component, function, module, or helper. If a comment repeats an operation, improve names or types instead.

**Done when:** no comment is compensating for a missing boundary or vague name.

### 3. Write the smallest useful comment

Explain why, not what. Include the source of a constraint when useful: requirement, ADR, protocol, browser behavior, or library limitation. Keep the comment beside the code it governs.

Use Go doc comments for exported identifiers. Use JSDoc only when an exported TypeScript contract remains unclear after typing.

When JSDoc is warranted, keep it a one-line summary by default: `/** Rejects a booking that overlaps an existing hold. */`. Reach for a tag only when it carries information types can't:

- `@remarks` — a short paragraph for context the summary line can't hold: why this exists, an invariant, a non-obvious side effect. Not a place to restate the signature.
- `@throws {ErrorType}` — when a function throws instead of returning a typed failure, name the error and the condition.
- `@deprecated` — with the replacement.
- `@see` — link to an ADR, spec, or the identifier to use instead.

Don't add `@param`/`@returns` when the TypeScript signature already names and types them clearly — that's the "restated code" the audit step removes. Use `@param`/`@returns` only to explain a specific non-obvious value (a magic sentinel, a unit, a constraint the type can't express).

**Done when:** each remaining comment explains a reason, constraint, contract, or consequence in precise domain language, and every JSDoc tag present adds information the signature doesn't already carry.

### 4. Audit the diff

Remove section banners, restated code, stale comments, dead-code comments, vague TODOs, and comments that merely say a value is important or magic. Link TODOs to an issue and owner, or finish the work.

**Done when:** changed comments are true, local, actionable, and no banned pattern remains in the diff.

## Guardrails

- `// -- HERO SECTION --` means a component boundary is missing.
- Comments must not narrate control flow, state updates, CSS declarations, or names.
- Comments must not justify poor structure.
- Keep comments short and complete. Prefer one precise sentence over a paragraph.
- Don't scatter JSDoc tags by habit. A full `@param`/`@returns`/`@example` block on every exported function is noise; most exports need only a one-line summary, or nothing at all if the name and type already say it.
- Don't use `@remarks` to pad a summary that already said everything — it's for the second fact, not a restatement of the first.

## Verification

Search the diff for section dividers, `TODO`, `FIXME`, `magic`, `important`, and comments that repeat nearby code. Review every hit; delete, rewrite, or link it. For JSDoc, check each `@param`/`@returns`/`@remarks` tag against the signature it documents; remove any tag that only restates the type.
