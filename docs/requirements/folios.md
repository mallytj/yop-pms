# Folios RTM

> **⚠️ PLACEHOLDER — Folio API + finance details defer to a future PR**
>
> This file captures the intended folio model so other docs (reservations,
> reservation groups) can reference it consistently. The **Folio API endpoints**,
> the **finance / payments / refunds detail**, the **optimistic-lock contract**, and
> the **cross-reservation transfer flow** all land in a later finance PR. Section
> headings below mark what is committed vs what is deferred.

## 1. Folio Parts

| Part | Purpose                              | Created When                                     |
| ---- | ------------------------------------ | ------------------------------------------------ |
| A    | Stay charges (room, tax)             | Auto on reservation creation (R-RES-CRUD-010)    |
| B    | Incidentals (F&B, mini-bar, extras)  | Lazy: first incidental charge OR staff opens it  |
| C    | Company / travel-agent billed        | Lazy: when travel_agent_id or company set        |

> No automatic routing rules yet. Staff posts cross-folio transfers manually.
>
> **Group master folio:** a group master is one of A / B / C on the master
> reservation that has been *designated* as the group master — there is no fourth
> folio kind. Designation rules deferred to the reservation-groups PR.

## 2. Folio Lifecycle (committed)

| ID          | Requirement                                                                                |
| ----------- | ------------------------------------------------------------------------------------------ |
| R-FOLIO-001 | Folio belongs to a reservation. One reservation may have 1..3 folios (A, B, C)             |
| R-FOLIO-002 | Folio has running balance derived from `folio_transactions` (cached on `folio.balance_pence`). Cache invalidation rule deferred to finance PR |
| R-FOLIO-003 | Folio cannot be deleted if **any** transaction has ever been posted against it (audit). Independent of balance |
| R-FOLIO-004 | Folio cannot be deleted while balance ≠ 0 OR while linked to non-terminal reservation       |
| R-FOLIO-005 | Folio close = invoice generation (see `finance.invoices`); closed folios are read-only      |
| R-FOLIO-006 | Concurrency control will be optimistic (`version` column per ADR-007 conventions). Exact contract deferred to finance PR |

## 3. Currency

| ID            | Requirement                                                                          |
| ------------- | ------------------------------------------------------------------------------------ |
| R-FOLIO-CCY-001 | Currency is set **per transaction** (`folio_transactions.currency`)                |
| R-FOLIO-CCY-002 | Default currency on a new transaction is the property's configured currency        |
| R-FOLIO-CCY-003 | Folio balance cache is denominated in the property currency. Mixed-currency aggregation rules (FX snapshot vs live) deferred to finance PR |

## 4. Transactions (committed)

| ID             | Requirement                                                                                          |
| -------------- | ---------------------------------------------------------------------------------------------------- |
| R-FOLIO-TX-001 | Transaction status: `pending`, `posted`, `voided`, `reversed`                                        |
| R-FOLIO-TX-002 | Posted transactions are immutable; corrections via `reversed` + new transaction                      |
| R-FOLIO-TX-003 | Tax rate is snapshotted at post time (`tax_rate_snapshot`); subsequent tax rule changes do not apply |
| R-FOLIO-TX-004 | Transactions tagged with `ledger_code_id` for reporting                                              |
| R-FOLIO-TX-005 | `pending` = "ghost" charge known but not yet due. Example: a 4-night stay creates 4 room-night charges on day 0; nights 2–4 sit `pending` and flip to `posted` by the night-audit on each respective day |
| R-FOLIO-TX-006 | `pending` transactions are excluded from `posted` balance but are visible to staff (forecast view); inclusion in cached `balance_pence` deferred to finance PR |
| R-FOLIO-TX-007 | Payments are `folio_transactions` rows with a payments `ledger_code_id`, included in balance. Allocation strategy across charges (even split vs FIFO vs explicit) deferred to finance PR |

## 5. Deposits & Pre-payments

| ID             | Requirement                                                                                       |
| -------------- | ------------------------------------------------------------------------------------------------- |
| R-FOLIO-DEP-001 | Deposits / pre-payments post as payment transactions before stay charges, producing a **negative folio balance** (folio owes guest) |
| R-FOLIO-DEP-002 | Negative balance is balanced down by subsequent night postings as the stay progresses              |
| R-FOLIO-DEP-003 | Deposit refund handling (when balance still negative at checkout) routed through the refund flow (R-FOLIO-REF, deferred) |

## 6. Cross-Folio Transfer (within same reservation)

| ID                | Requirement                                                                          |
| ----------------- | ------------------------------------------------------------------------------------ |
| R-FOLIO-XFER-001  | Transfer creates linked credit + debit transactions on source + destination folios   |
| R-FOLIO-XFER-002  | Transfer requires `folios:transfer` permission (defined when authz lands)            |
| R-FOLIO-XFER-003  | Cross-property transfer prohibited                                                   |
| R-FOLIO-XFER-004  | VAT / tax components move with the transfer (tax follows the charge to the destination folio; tax is **not** retained on source) |

### Cross-reservation transfer (deferred)

> Transferring a charge from a folio on reservation X to a folio on reservation Y
> (same property) goes via a dedicated **"transfer to foreign folio"** route.
> Specification deferred to the finance PR.

## 7. Refunds (deferred)

> Refund issuance — cash, card, or credit note — is **not** specified yet.
> Will be designed in the finance PR. Negative balances at checkout, deposit returns,
> chargebacks, and refund-against-closed-folio all live there.

## 8. Endpoints (deferred)

> The Folio HTTP API ships in a later PR. Endpoint shapes will follow the OpenAPI
> contract pattern (see `docs/adr/001-schema-first-api.md`). The reservation-flow PR
> only depends on internal folio creation (auto on reservation create) and the
> folio model described in §1–§6.

## 9. Authorization

> Per `docs/requirements/authorization.md`, authz is itself a placeholder. Folio
> permissions (`folios:open`, `folios:post`, `folios:void`, `folios:reverse`,
> `folios:transfer`, `folios:close`, `folios:invoice`) will be defined when the
> auth PR lands. Until then, handlers stub the perm check via the standard
> `requirePermission(...)` middleware hook.

## 10. Edge Cases

_Captured for the finance PR; not solved here._

- Reversing a transaction whose tax rule no longer exists (R-FOLIO-TX-003 protects: snapshot wins)
- Folio close with zero balance (still generate invoice? — TBD)
- Reopen of closed folio (operational reality vs immutability — TBD)
- Voiding a payment that was an authorisation (release auth? — TBD)
- Posting to closed folio (rejected)
- Invoice number collision (DB UNIQUE)
- Negative balance at checkout (refund flow — deferred)
- Mixed-currency folio aggregation
