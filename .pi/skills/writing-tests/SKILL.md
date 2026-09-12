---
name: writing-tests
description: Use when writing or reviewing tests in Go, TypeScript, or Svelte; test names must state behavior and bodies must show given/when/then clearly.
scope: project
---

# Writing Tests

A test reads like a spec. The name states the behavior under test; the body groups setup, action, and assertion so the shape is visible at a glance.

## Procedure

### 1. Name the behavior

Write the test name as a plain statement of behavior, not implementation: `returns only the open orders`, not `test getOrders filter`. Name what the system does, from the caller's perspective.

**Done when:** a reader who has never seen the code understands the expected behavior from the name alone.

### 2. Structure given/when/then

Group the body into three visible blocks, separated by a blank line:

- **Given** — fixtures, seed data, preconditions.
- **When** — the single action under test.
- **Then** — the assertion(s).

```
test('returns only the open orders', async () => {
  givenCustomer('c1');
  givenOrder('o1', 'c1', 'open');
  givenOrder('o2', 'c1', 'shipped');
  givenOrder('o3', 'c2', 'open');

  const orders = await getOrders('c1', { state: 'open' });

  const ids = orders.map((o) => o.id);
  expect(ids).toEqual(['o1']);
});
```

**Done when:** each block is visually separated, contains one concern, and a reader can tell given/when/then apart without reading closely.

### 3. Keep one action per test

If a test exercises more than one behavior, split it. Each test asserts one outcome; add cases for edge conditions as separate tests with their own behavior-stating names, not extra assertions bolted onto one test.

**Done when:** every test has exactly one `when` action and its assertions all check the same outcome.

### 4. Prefer domain-named helpers over inline setup

Extract repeated fixture setup into verb-named helpers (`givenCustomer`, `givenOrder`) instead of repeating raw construction in every test. Keep helpers close to the tests that use them.

Draw the line by repetition, not by anticipation. Don't extract a helper for a single test's one-off setup — that's cognitive overhead for a reader who now has to jump to a definition to see three lines of construction. Do extract once the same shape is written a second or third time in the same file: at that point the helper is paying for itself, and a reader benefits from the domain name more than from seeing the raw fields. Never pre-build a general fixture factory, options bag, or builder for scale the suite doesn't have yet — add parameters only when a real second caller needs them to differ.

**Done when:** the `given` block reads as domain statements, not object literals, and no helper exists to serve only its own single call site.

### 5. Make the test deterministic

A test must produce the same result on every run, in any order, on any machine. Eliminate sources of nondeterminism:

- **Time** — never call the real clock (`time.Now()`, `Date.now()`, `new Date()`) inside code under test without injecting it. Pass a fixed or fake clock; assert against that fixed value.
- **Randomness** — inject a seeded RNG or fake ID generator rather than relying on real UUIDs, `Math.random()`, or `crypto.rand`. Assert on shape/type when a truly random value is unavoidable, never on its exact output.
- **Ordering** — don't assert on map iteration order, unordered query results, or concurrent goroutine/promise completion order unless the system contractually guarantees that order. Sort before comparing, or assert on set membership.
- **External state** — don't depend on real network calls, real databases outside a controlled test instance, the filesystem outside a temp dir, or environment variables that vary by machine. Use fixtures, fakes, or an isolated test database/transaction per test.
- **Shared mutable state** — don't let one test's side effects (global variables, package-level caches, shared fixtures) leak into another. Each test sets up and tears down its own state.

**Done when:** running the test 100 times, in isolation and in parallel with the full suite, produces the same pass/fail result every time.

## Guardrails

- No vague test names (`works`, `test 1`, `handles edge case`).
- No blank-line-free walls of setup, action, and assertion mixed together.
- No multiple unrelated behaviors asserted in one test.
- Do not assert on incidental detail (ordering, internal fields) unless that is the behavior under test.
- No unseeded randomness, uninjected real clocks, or real network/filesystem/env dependencies in a unit test.
- No `sleep`/fixed-delay waits to paper over async timing — wait on the actual condition or fake the clock.

## Verification

Read each changed test name aloud as a sentence describing behavior. Confirm given/when/then blocks are blank-line separated and each test has exactly one action and one behavior under assertion. Search the diff for `time.Now()`, `Date.now()`, `new Date()`, `Math.random()`, unseeded UUID/RNG calls, and `sleep`/`setTimeout` waits — each hit must be injected, faked, or justified.
