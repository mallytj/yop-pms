# Booking Engine Implementation Plan

## 1. Data Flow Architecture

We separate **Reference Data** (immutable from server) from **Booking State** (mutable by user).

1. **Server Load (`+page.server.ts`):** Fetches `RatePlans` and `DailyRates`.
2. **Store Initialization:** Hydrates the `bookingStore` with default values (0 adults, no rate selected).
3. **User Interaction:** Updates `BookingState` (adjustments, selection).
4. **Derived Calculation:** Combines `Reference Data` + `Booking State` to show final prices in real-time.
5. **Save Action:** Submits the fully calculated `ReservationDraft` to the backend.


### 1.1 Load Sequence Diagram
```mermaid
sequenceDiagram
    actor User
    participant App as SvelteKit (Client/Server)
    participant API as Backend

    User->>App: Selects dates & rooms
    Note over App, API: Reservation flow starts
    App->>API: POST /v1/inventory/lock
    API-->>App: 200 OK (Locked room)

    rect rgba(3, 58, 73, 1)
    Note right of App: Parallel Data Fetching
        App->>API: GET /v1/rate-plans
        API-->>App: 200 OK (Rate Plans)
        App->>API: GET /v1/rate-map?startDate=X&endDate=Y
        API-->>App: 200 OK (Daily Rates)
    end
    App-->>App: Render BookingEngine.svelte
    App->>App: Initialize BookingStore()
    App->>App: Set default state (2 adults, no rates)
    App-->>User: Show booking summary
```

### 1.2 Booking Engine Process
```mermaid
stateDiagram-v2
    [*] --> INIT
    INIT --> SELECT_DATES: User selects dates
    SELECT_DATES --> SELECT_ROOMS: User selects rooms
    SELECT_ROOMS --> SELECT_RATES: User selects rates
    SELECT_RATES --> ADD_GUEST: User adds or creates guests
    ADD_GUEST --> REVIEW: User reviews
    REVIEW --> CONFIRM: User confirms
    CONFIRM --> [*]: Reservation created
```

---

## 2. File Structure (Flattened & Domain-Driven)

We avoid the "Russian Doll" directory structure. Components are grouped by **Domain** (Booking) or **Type** (UI Primitives).

```mermaid
graph LR
    subgraph Root [src/lib]
        direction TB
        
        subgraph Components [Components Layer]
            direction LR
            subgraph Domain [booking/]
                BE[BookingEngine.svelte]
                RC[RoomCard.svelte]
                RR[RateRow.svelte]
                DB[DailyBreakdown.svelte]
                Head[Header.svelte]
                Foot[Footer.svelte]
            end
            
            subgraph UI [ui/]
                CI[CurrencyInput.svelte]
                IS[IconSwitch.svelte]
                ST[Stepper.svelte]
                MOD[Modal.svelte]
            end
        end

        subgraph Logic [Logic Layer]
            Store[booking.svelte.ts]
        end

        subgraph Definitions [Type Layer]
            Types[booking.d.ts]
        end
    end

    %% Dependencies
    BE -.-> Store
    RC -.-> Store
    RR -.-> Store
    Types -.-> Store
    Types -.-> BE
    
    %% UI Usage
    RR --> CI
    RR --> IS
    RC --> ST
    BE --> MOD

    %% Styling
    style BE fill:#033a49,color:#fff,stroke:#fff
    style Store fill:#ff3e00,color:#fff,stroke:#fff
    style Types fill:#3178c6,color:#fff,stroke:#fff
    style Domain fill:#f4f4f4,stroke:#333,stroke-dasharray: 5 5
    style UI fill:#fff,stroke:#f9f,stroke-width:2px
```

---

## 3. Store Architecture (The "CPU")
We use a **Context-based Store**. This allows you to have multiple booking engines on one page if needed, but mostly it prevents global state pollution.

**`src/lib/stores/booking.svelte.ts`** (Using Svelte 5 Runes)

### 3.1 Planned Store Structure
```mermaid
classDiagram
    namespace State_Manager {
        class BookingStore {
            +draft: ReservationDraft
            +plans: RatePlan[]
            -rateMap: RateMap
            
            <<get>> +items: ReservationItemDraft[]
            <<get>> +grandTotal: number
            
            +selectRatePlan(itemId, ratePlanId)
            +updateOccupancy(itemId, adults, children)
            +setDailyAdjustment(itemId, date, adj)
            +setGlobalAdjustment(itemId, adj)
            +applyFirstToAll()
            
            -distributeFixedAmount(item, adj)
            -getItem(itemId)
        }
    }

    namespace Domain_Data_Interfaces {
        class ReservationDraft {
            <<interface>>
            +globalCheckInDate: ISO8601Date
            +globalCheckOutDate: ISO8601Date
            +items: ReservationItemDraft[]
        }

        class ReservationItemDraft {
            <<interface>>
            +tempId: UUID
            +bookedRoomTypeId: UUID
            +selectedRatePlanId: UUID
            +adults: number
            +children: number
            +dailyRates: DailyRate[]
            +computed: ItemTotals
        }

        class DailyRate {
            <<interface>>
            +date: ISO8601Date
            +basePricePence: number
            +adjustment: Adjustment
            +adjustmentApproved: boolean
            +computedFinalPricePence: number
        }

        class Adjustment {
            <<interface>>
            +type: "fixed_amount" | "percentage"
            +value: number
            +reason: string
        }

        class ItemTotals {
            <<interface>>
            +baseTotal: Money
            +adjustmentTotal: Money
            +finalTotal: Money
        }
    }

    BookingStore ..> ReservationDraft : manages
    ReservationDraft "1" *-- "many" ReservationItemDraft
    ReservationItemDraft "1" *-- "many" DailyRate
    DailyRate o-- Adjustment
    ReservationItemDraft "1" *-- "1" ItemTotals
```

---

## 4. Component Hierarchy
```mermaid
graph TD
    %% Main Orchestrator
    BE[BookingEngine.svelte]
    
    %% Top Level Layout
    BE --> Head[Header.svelte]
    BE --> Rates[Rates.svelte]
    Rates --> RC[RoomCard.svelte]
    BE --> Foot[Footer.svelte]

    %% Header Branch
    Rates --> SD[StayDates.svelte]
    
    %% RoomCard Branch
    RC --> OS[OccupancySetter.svelte]
    OS --> STEP[Stepper.svelte]
    RC --> RR[RateRow.svelte]

    
    %% RateRow Branch
    RR --> RAD[RadioBtn.svelte]
    RR --> CA[CurrencyAdjuster.svelte]
    RR --> DB[DailyBreakdown.svelte]
    
    %% DailyBreakdown Branch
    DB --> DBR[ResetBtn.svelte]
    DB --> DR[DayRow.svelte]
    DR --> SEL[Select.svelte]
    DR --> CA2[CurrencyAdjuster.svelte]
    DR --> REAS2[Reason.svelte]

    %% UI Primitives Styling
    classDef ui fill:#f9f,stroke:#333,stroke-width:2px;
    class STEP,RAD,CA,CA2,SEL,DBR ui;
    
    %% Domain Containers Styling
    classDef domain fill:#bbf,stroke:#333,stroke-width:2px;
    class BE,RC,RR,DB,DR domain;
```

## 5. Future Improvements

### 5.1 Adding Authorisation

We should add authorisation to the booking engine to prevent unauthorized access to the booking engine.
It should also have a cap for the adjustments per user/role to prevent abuse.
Adjustments must also be approved by a set user
Adding a quick approval workflow such as a manager pin will allow this without ruining the UX

### 5.2 Adding Keyboard Macros

We should add keyboard macros to the booking engine to allow for quick and easy adjustments to the booking engine.
This will need to be planned and talked through with reception staff to provide actually useful shortcuts

### 5.3 Adding Maintenace Blocks

Add an option at rate selection to decommission a room
Also enforce authorisation settings for this

### 5.4 Adding reservation groups

In the rates page, allow the creation of groups of rooms

