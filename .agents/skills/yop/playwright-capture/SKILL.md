---
name: playwright-capture
description: Capture screenshots, full-page, and video of frontend changes. Camera, not a judge — outputs to .qa-review/ for human review.
disable-model-invocation: true
---

# Playwright Capture

**Camera, not a judge.** Captures raw visual evidence of frontend changes. No AI self-grading — just the artifacts for human review before hunk-grill.

## Prerequisites

Playwright MCP server must be installed and configured in your Pi setup. Without it, capture commands will fail silently — the agent will skip and note the gap.

## Trigger

Invoke when `git diff` shows changes to `.svelte` or `.css` files. Skip when diff has zero UI changes.

Three capture modes, each used only when the change warrants it:

- **Screenshot** — page at a specific state. Use when a single state changed (new button, modified form field).
- **Full-page screenshot** — entire scrollable page. Use when layout or content length changed.
- **Video** — interaction recording. Use when a multi-step flow changed (form submit, drag-and-drop, page navigation).

## Output

```
.qa-review/<ticket-slug>/
├── check-in.png            # page-based naming
├── booking-form.png
├── check-in-full.png       # full-page variant
└── booking-flow.webm       # video of interaction
```

`.qa-review/` is gitignored — artifacts are local only.

## Procedure

1. Identify affected pages from the diff and ticket scope.
2. Navigate to each affected page at the relevant state.
3. Capture screenshots. Add full-page variants where layout changed.
4. Record video of interaction flows where multi-step behaviour changed.
5. Save all artifacts to `.qa-review/<ticket-slug>/`.

**Done when** captures exist for every page with UI changes, or diff has zero Svelte/CSS changes (skip).
