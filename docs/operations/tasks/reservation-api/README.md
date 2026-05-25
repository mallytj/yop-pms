# Reservation API Retrospective

This directory records how `feat/reservation-api` **should** have been split
into execution tasks. Each file maps to one branch that would have merged
independently.

## What went wrong

The work was done in a single monolithic branch instead of 9 serial tasks.

## Task status

| # | Task | Status | Branch |
| :--- | :--- | :--- | :--- |
| 01 | Schema | done | `feat/reservation/schema` |
| 02 | Platform additions | done | `feat/reservation/platform-additions` |
| 03 | Types + state machine | done | `feat/reservation/types-sm` |
| 04 | Service core | done | `feat/reservation/service-core` |
| 05 | Business logic | in-progress | `feat/reservation/business-logic` |
| 06 | Middleware | done | `feat/reservation/middleware` |
| 07 | Handlers + router | in-progress | `feat/reservation/handlers-router` |
| 08 | Workers | not-started | `feat/reservation/workers` |
| 09 | SSE | not-started | `feat/reservation/sse` |
