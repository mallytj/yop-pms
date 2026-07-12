---
name: run-audit
description: |
  Run role-specific audit agents (CTO, Boutique Director, Compliancy, UX Expert)
  against a feature, design, or codebase. No extensions required.
  Usage: /run-audit full "context" or /run-audit cto,compliancy "context"
scope: project
---

# Run Audit — Role-Specific Advisory Review

Lightweight multi-perspective audit. Reads agent personas from `.pi/agents/`
and applies each as an analysis lens. Zero extension dependencies.

## When to Use

- Before implementing a feature — catch issues early
- After designing an API or schema — validate against role concerns
- During code review — get domain-specific feedback
- When unsure about tradeoffs — get conflicting perspectives

## Agent Map

| Trigger word | Agent file | Severity scale |
|-------------|-----------|----------------|
| `cto` | `.pi/agents/cto.md` | blocker / warning / observation |
| `boutique` | `.pi/agents/boutique-director.md` | blocker / friction / nice-to-have |
| `compliancy` | `.pi/agents/compliancy.md` | violation / risk / improvement |
| `ux` | `.pi/agents/ux-expert.md` | blocker / friction / polish |
| `full` or `all` | All four | Per-agent scale |

## Procedure

### Step 1: Parse the request

From user input, extract:
- **Scope:** `full`, `all`, or comma-separated agent names (`cto,boutique,ux`)
- **Context:** The feature, design, file, or question to audit. Everything after the scope word and before end of message.

If context is a file path or diff, read it first. If context is a feature name, read relevant source files before auditing.

### Step 2: Load agent personas

For each agent in scope, read `.pi/agents/<file>.md`. Extract the system prompt:
everything after the second `---` (frontmatter closer). This is the agent's
persona, expertise lens, severity scale, and output format.

Do NOT copy the frontmatter — only the system prompt body.

### Step 3: Apply each persona

For each agent, apply its system prompt as an analysis lens against the context.
Think as that persona. Use their exact severity scale. Follow their output
conventions. Reference specific files and line numbers when auditing code.

If an agent has tools listed in its frontmatter (e.g., `tools: read, grep, find`),
use those tools to inspect the codebase as part of the analysis. If `tools: ""`
or no tools, work from the provided context only.

### Step 4: Report findings

For each agent, output findings structured as:

```
## {Agent Name} — {N} findings

**Rating scale:** {their scale}

| Severity | Finding | Evidence |
|----------|---------|----------|
| ... | ... | ... |

**Verdict:** 1-2 sentence summary from this role's perspective.
```

### Step 5: Cross-reference

After all agents report, identify:
- Issues flagged by multiple agents (higher priority)
- Conflicts between agent perspectives (flag for human decision)
- Overlapping recommendations (single fix solves multiple concerns)

Output a consolidated priority table sorted by:
1. Issues flagged by multiple agents
2. Highest severity per issue
3. Estimated effort to fix

## Pitfalls

- Do NOT skip reading the agent files. Personas must be fresh from source.
- Do NOT merge agent perspectives into one blob. Keep them separate — the user needs to see conflict.
- Do NOT invent severity ratings. Use exactly the scale from each agent's system prompt.
- If an agent file is missing, skip it and warn. Do not improvise the persona.
- The AI/ML engineer agent (`engineering.ai-engineer`) is NOT included in audit runs — it's an implementation agent, not advisory.

## Verification

- Each requested agent produced findings in its own section
- Every finding has a severity rating from that agent's scale
- Cross-reference table identifies overlapping concerns
- Agent personas were read from actual `.pi/agents/` files, not from memory
