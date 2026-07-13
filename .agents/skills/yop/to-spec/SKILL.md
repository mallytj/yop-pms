---
name: to-spec
description: Turn the current conversation into a Feature Job Spec and publish it to Linear — no interview, just synthesis of what you've already discussed. Yop override: Problem/Solution format with role-grouped requirements.
disable-model-invocation: true
---

# To Spec — Yop Override

Turn the current conversation into a **Feature Job Spec** and publish it to Linear. No interview — synthesize what's already in context.

The issue tracker and triage labels are configured. Templates live in Linear.

## Process

1. Explore the repo to understand current codebase state. Use the project's domain glossary vocabulary and respect ADRs in the area.

2. Identify the Role Job Specs that feed into this feature. A Feature Job Spec draws from one or more Role Job Specs. Reference them in the Inputs section.

3. Sketch testing seams — highest seam possible, prefer existing seams. Check with the user that seams match expectations.

4. Write the Feature Job Spec using the template below. Publish to Linear with `ready-for-agent` label.

<spec-template>

## Problem Statement

The problem from the user's perspective.

## Solution

High-level approach.

## Inputs

Role Job Specs consulted:
- [Role Job Spec: <Role>](link)

Research: <link or summary>
Relevant ADRs: [ADR-XXX](link)

## Requirements

<!-- Grouped by role, matching Role Job Spec sources -->

### <Role> Requirements

- **<What>:** <Why>
- ...

## State Machine

<!-- Only when feature has meaningful state transitions -->

States: <list>
Transitions: <list>

## Edge Cases

- What happens when <condition>?
- ...

## Routes

### API Endpoints

| Method | Path | Purpose | Auth |
|--------|------|---------|------|
| GET | /api/... | ... | ... |

### Page Routes

| Path | Component | Purpose |
|------|-----------|---------|
| /... | ... | ... |

## Workers

<!-- Only when feature needs background jobs -->

| Worker | Trigger | What it does |
|--------|---------|--------------|
| ... | ... | ... |

## Migrations

<!-- Only when feature needs DB schema changes -->

| Migration | Table | Change |
|-----------|-------|--------|
| ... | ... | ... |

## Implementation Decisions

- Modules built/modified
- Interface changes
- Architectural decisions
- API contracts

No file paths or code snippets. Exception: prototype output that encodes a decision — inline trimmed to decision-rich parts.

## Testing Decisions

- What makes a good test (external behaviour, not implementation)
- Which modules tested
- Seams (highest possible, prefer existing)
- Prior art in codebase

## Out of Scope

What's explicitly excluded.

## Further Notes

Additional context, caveats, follow-ups.

</spec-template>
