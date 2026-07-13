# Agentic Engineering Workflow

Yop PMS uses a 6-stage agentic workflow from idea to shipped feature. Each stage is a skill invocation — the agent drives, human gates at key points.

Use `/ask-yop` at any point for guidance on which skill or stage to invoke next.

## Pipeline Overview

Prerequisite: a draft Linear project with the roles this feature affects. Roles determine which Role Job Specs to create.

```mermaid
flowchart LR
    P[Linear project draft + roles] -.->|prerequisite| A[Role Job Spec]
    A --> B[Research]
    B --> C[Domain Modeling]
    C --> D[to-spec]
    D --> E[to-tickets]
    E --> F[implement]
    F --> G[Shipped]

    A -.->|gate| A1[run-audit]
    D -.->|gate| D1[run-audit]
```

## Stage 1: Role Job Spec

Start from the Linear project draft — it defines which roles this feature affects. Those roles determine what to research and write specs for. Capture domain knowledge per role before feature planning. If no knowledge exists for a role, research first — then interview against the domain model. Advisory review catches blind spots after the draft.

```mermaid
flowchart TD
    A["/grill-with-docs per role"] --> B{role knowledge exists?}
    B -->|no| C["/research on role domain"]
    C --> D[draft Role Job Spec]
    B -->|yes| D
    D --> E{"run-audit full"}
    E --> F[CTO: scale, multi-tenancy]
    E --> G[Boutique Director: ops, staff flows]
    E --> H[Compliancy: GDPR, PII]
    E --> I[UX Expert: power-user speed]
    F --> J[revise spec]
    G --> J
    H --> J
    I --> J
    J --> K[final Role Job Spec]
```

## Stage 2 and 3: Research and Domain Modeling

```mermaid
flowchart LR
    A["/research"] --> B[cited findings]
    B --> C["/domain-modeling"]
    C --> D[update CONTEXT.md]
    C --> E[create ADRs if needed]
```

## Stage 4: to-spec — Feature Job Spec

Synthesize role specs, research, and domain model into a Feature Job Spec. No user stories — role-grouped requirements format. Advisory review gates before ticket creation.

```mermaid
flowchart TD
    A[Role Job Specs] --> D
    B[Research findings] --> D
    C[Domain model / ADRs] --> D
    D["/to-spec"] --> E[draft Feature Job Spec]
    E --> F{"run-audit full"}
    F --> G[CTO review]
    F --> H[Boutique Director review]
    F --> I[Compliancy review]
    F --> J[UX Expert review]
    G --> K[revise spec]
    H --> K
    I --> K
    J --> K
    K --> L[final Feature Job Spec]
```

## Stage 5: to-tickets

Break Feature Job Spec into vertical tracer-bullet tickets with blocking edges. Each ticket is a narrow but complete path through every layer.

```mermaid
flowchart LR
    A[Feature Job Spec] --> B["/to-tickets"]
    B --> C[ticket 1]
    B --> D[ticket 2]
    B --> E[ticket 3]
    E --> F[ticket 4]
    C -.->|blocks| F
    D -.->|blocks| F

    style C fill:#90EE90
    style D fill:#90EE90
    style E fill:#FFD700
    style F fill:#FFB6C1
```

Green = frontier (takeable now). Yellow = in progress. Red = blocked.

## Stage 6: implement

Build one ticket at a time. Agent drives 6 steps autonomously. Single human gate after commit.

```mermaid
flowchart TD
    A[1. Read ticket + spec] --> B[2. Draft E2E happy-path test]
    B --> C[3. TDD build]
    C --> D{UI changed?}
    D -->|yes| E[4. playwright-capture]
    D -->|no| F[5. code-review]
    E --> F
    F --> G[5. code-review]

    G --> H[/"Human: hunk-grill"/]
    H -->|changes requested| I[agent fix loop]
    I --> H
    H -->|approved| J[6. Commit]
    J --> K[push to PR]

    style A fill:#E8E8E8
    style B fill:#E8E8E8
    style C fill:#E8E8E8
    style D fill:#E8E8E8
    style E fill:#E8E8E8
    style F fill:#E8E8E8
    style G fill:#E8E8E8
    style H fill:#FFD700
    style I fill:#FFD700
    style J fill:#E8E8E8
    style K fill:#90EE90
```

Grey = agent-driven. Yellow = human-in-the-loop. Green = done.

Parallel tickets run in separate Git worktrees. Clear context between tickets.

## On-ramps

Two paths merge onto the main flow:

```mermaid
flowchart TD
    subgraph onramps ["On-ramps"]
        A[Bugs / requests piling up] --> B["/triage"]
        B --> C[agent-ready issues]
        D[Something broken] --> E["/diagnosing-bugs"]
        E --> F[post-mortem]
    end

    subgraph main ["Main flow"]
        G["/implement"]
    end

    C --> G
    F --> H["/improve-codebase-architecture"]
```

## Codebase Health

```mermaid
flowchart LR
    A["/improve-codebase-architecture"] --> B[deepening opportunities]
    B --> C["/grill-with-docs"]
    C --> D[ADR or refactor plan]
    D --> E["/implement"]
```

## Skill Map

| Category | Skills | Location |
|----------|--------|----------|
| Workflow | grilling, grill-with-docs, grill-me, research, domain-modeling, to-spec, to-tickets, implement, tdd, code-review, prototype, handoff | `.agents/skills/engineering/` |
| Health | codebase-design, diagnosing-bugs, improve-codebase-architecture | `.agents/skills/health/` |
| Process | wayfinder, triage | `.agents/skills/process/` |
| Learn | teach, writing-great-skills | `.agents/skills/learn/` |
| Yop overrides | to-spec, implement, playwright-capture | `.agents/skills/yop/` |
| Yop custom | ask-yop (router), run-audit (advisory review) | `.pi/skills/` |

## Context Hygiene

- Stages 1 through 4 in one unbroken context window — do not compact until after to-tickets
- Each implement starts fresh from the ticket
- Limit: smart zone (~120k tokens)
