---
name: implement
description: Implement a ticket from to-tickets. Yop override: 6-step flow with E2E, TDD, playwright-capture, code-review, and hunk-grill human gate.
disable-model-invocation: true
---

# Implement — Yop Override

Implement a **tracer bullet** ticket from to-tickets. Work from the ticket body and parent Feature Job Spec. Each step gates the next — no step completes until its criterion is met.

## Flow

### 1. Read ticket + spec

Load the ticket body and parent Feature Job Spec from Linear. Read every section. **Done when** you can state the exact end-to-end behaviour this ticket delivers, in one sentence.

### 2. Draft E2E happy-path test

Write a Playwright test exercising the full vertical slice: page → API → DB → response → page. **Done when** the test file exists, compiles, and describes the happy path without gaps.

Self-validate — no human approval gate. The test proves the ticket's behaviour is reachable.

### 3. TDD build

Red-green-refactor at pre-agreed seams. Drive implementation from the E2E test down through integration and unit tests.

- Run typechecking after every file change
- Run affected test files after every behaviour change
- Run full test suite at the end

**Done when** full test suite is green and typecheck passes clean.

### 4. playwright-capture

Inspect `git diff` for `.svelte` or `.css` changes. If UI changed, invoke `/playwright-capture`. **Done when** `.qa-review/<ticket-slug>/` contains captures for affected pages, or no UI changes existed.

Agent decides to invoke — not automatic. Skip if diff has zero Svelte/CSS changes.

### 5. code-review

Self-review the diff via `/code-review`. Two axes:
- **Standards**: follows repo conventions?
- **Spec**: matches what the ticket asked for?

**Done when** review completes and any self-found issues are fixed.

### Human gate: hunk-grill

Stop here. Human reviews unstaged changes via `/hunk-grill`. Agent addresses hunk comments in a fix loop until human approves.

### 6. Commit

After human approval, organize changes into clean, logical commits. Message references the ticket identifier. **Done when** commits are recorded and `git status` is clean.

Human then pushes to PR.

## Context hygiene

Each `/implement` runs fresh. Use `/using-git-worktrees` for parallel tickets — one worktree per ticket, clear context between them.
