# Ollerod PMS — Frontend

SvelteKit + TypeScript frontend for the Ollerod property management system. Talks to the Go backend at `localhost:8080`.

## Tech Stack

- **SvelteKit** (Svelte 5) with TypeScript
- **Tailwind CSS v4**
- **Vite 7**

## Getting Started

```sh
npm install
npm run dev
```

Opens at [http://localhost:5173](http://localhost:5173).

## Project Structure

```
src/
├── routes/           # SvelteKit pages
│   └── planner/      # /planner — interactive reservation grid
├── lib/
│   ├── actions/      # Svelte use: actions (draggable, etc.)
│   ├── components/   # UI components (planner grid, modals, toolbar)
│   ├── helpers/      # Business logic (drag handling, date math, overlap detection)
│   ├── stores/       # Svelte stores (planner state, reservation modal, toasts)
│   ├── styles/       # Global CSS
│   └── types/        # TypeScript type definitions
```

### Path Aliases

Configured in `svelte.config.js`:

| Alias          | Path              |
| -------------- | ----------------- |
| `$lib`         | `src/lib`         |
| `$components`  | `src/lib/components` |
| `$helpers`     | `src/lib/helpers` |
| `$stores`      | `src/lib/stores`  |
| `$types`       | `src/lib/types`   |
| `$actions`     | `src/lib/actions` |

### Naming Conventions

-   **Components**: `PascalCase.svelte` (e.g., `PlannerGrid.svelte`, `RoomCard.svelte`).
-   **TypeScript Files** (`.ts`): `snake_case.ts` (e.g., `planner_store.ts`, `fetchers.ts`).
-   **TypeScript Types**: `PascalCase` for interfaces and type aliases (e.g., `interface PlannerData`, `type Booking`).
-   **CSS Variables**: `--kebab-case` (e.g., `--primary-brand-color`).

### Styling

The project uses **Tailwind CSS v4** with its Vite plugin.

Global styles and CSS custom properties (variables) are defined in `src/lib/styles/app.css`. This file defines the base theme colors, fonts, and layout variables used throughout the application.

To help with development and visualization of the color palette, a utility route is available at `/dev/colorhelper`. This page displays all the defined CSS color variables, making it easy to see the available theme colors at a glance.

```css
/* src/lib/styles/app.css */
:root {
  --background: 0 0% 100%;
  --foreground: 222.2 84% 4.9%;
  /* ... other color variables */
}
```

## Core Architecture & Logic

This section outlines the core architectural patterns for data flow and state management.

### 1. Data Fetching & API Interaction

To maintain clean components and a clear separation of concerns, all backend API interactions are handled by dedicated functions located in `src/lib/helpers`.

**Example: Fetching Planner Data**

-   **Location**: `src/lib/helpers/planner/fetchers.ts`
-   **Function**: `fetchPlannerData(startDate: Date, endDate: Date): Promise<PlannerData>`
-   **Description**: This function is responsible for calling the `GET /v1/planner` backend endpoint and returning the data required to render the planner grid. Components do not call `fetch` directly; they use this helper.

```typescript
// src/lib/helpers/planner/fetchers.ts
import type { PlannerData } from '$types/planner_data';

export async function fetchPlannerData(startDate: Date, endDate: Date): Promise<PlannerData> {
  const start = startDate.toISOString().split('T');
  const end = endDate.toISOString().split('T');
  
  // The base URL is proxied by Vite during development.
  const response = await fetch(`/api/v1/planner?startDate=${start}&endDate=${end}`);

  if (!response.ok) {
    throw new Error('Failed to fetch planner data');
  }

  return await response.json();
}
```

### 2. State Management with Svelte Stores

We use class-based Svelte stores to manage application state, especially for complex, interactive components.

#### Planner Store (`$stores/planner_store`)

This store holds the state for the interactive planner grid, managing UI concerns like drag-and-drop operations, cell selections, and modal visibility.

#### Booking Engine Store (`$stores/booking_store.svelte.ts`)

The booking engine is the most complex piece of state in the application. Its architecture is detailed in the `src/lib/components/booking-engine/plan.md` document and follows a critical pattern:

1.  **Separation of Data**: The store clearly separates immutable **Reference Data** (e.g., `RatePlans`, `DailyRates` fetched from the server) from the mutable **Booking State** that the user modifies (e.g., selected dates, occupancy, price adjustments).

2.  **Context-Based Store**: The store is not a global singleton. It's created and provided via Svelte's context API within the main `<BookingEngine />` component. This encapsulates the booking logic and allows for multiple instances if ever needed.

3.  **Data Transformation on Save**: The backend expects data in a specific format (one row per booked day). The frontend store's structure is optimized for UI reactivity and calculations. Therefore, a transformation occurs before submitting the data. A `getFinalSubmission()` method in the store is responsible for converting the rich, nested frontend state into the flat structure the backend API requires.

    -   **Frontend State (Simplified)**:
        ```json
        { "room_1": { "rate_id": "rp1", "adjustment": -10 } }
        ```
    -   **Transformed Payload for Backend**:
        ```json
        [{ "date": "2026-10-12", "room_id": "...", "final_amount": 290.00 }]
        ```

## Scripts

| Command              | Description                  |
| -------------------- | ---------------------------- |
| `npm run dev`        | Start dev server             |
| `npm run build`      | Production build             |
| `npm run preview`    | Preview production build     |
| `npm run check`      | Run svelte-check type checks |
| `npm run check:watch`| Type checks in watch mode    |