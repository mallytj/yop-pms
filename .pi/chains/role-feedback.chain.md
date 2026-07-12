---
name: role-feedback
package: advisory
description: Interview all advisory agents (CTO, Boutique Director, Compliancy, UX) in parallel for role-specific feedback
---

# Role Feedback Chain

Runs all four advisory agents in parallel against the same task, context, or design. Each reviews from their role-specific lens. Returns consolidated multi-perspective feedback.

## Usage

```
/run-chain advisory.role-feedback "Review the new booking flow design"
```

Or programmatically:

```typescript
subagent({
  chain: [
    { parallel: [
      { agent: "advisory.cto", as: "cto", task: "Review: {task}. Evaluate scale, multi-tenancy, reliability." },
      { agent: "advisory.boutique-director", as: "boutique", task: "Review: {task}. Evaluate for boutique hotel operations, multi-role staff, personalization." },
      { agent: "advisory.compliancy", as: "compliancy", task: "Review: {task}. Audit for GDPR compliance, PII exposure, data protection risks." },
      { agent: "advisory.ux-expert", as: "ux", task: "Review: {task}. Evaluate for power-user speed, keyboard efficiency, role-specific workflows." }
    ], concurrency: 4 }
  ],
  context: "fresh"
})
```
