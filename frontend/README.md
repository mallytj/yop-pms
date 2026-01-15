# Frontend - Hotel PMS

Modern frontend for the Hotel Property Management System built with Astro, React, and TypeScript.

## Technologies

- **Astro**: Static site generator with partial hydration
- **React**: UI component library
- **TypeScript**: Type-safe JavaScript

## Getting Started

### Prerequisites

- Node.js 18+ and npm

### Setup

1. Install dependencies:
```bash
npm install
```

2. Start the development server:
```bash
npm run dev
```

The frontend will be available at `http://localhost:4321`

## Available Scripts

- `npm run dev`: Start development server
- `npm run build`: Build for production
- `npm run preview`: Preview production build

## Features

- **Booking Form**: Create hotel bookings with React component
- **Responsive Design**: Works on all device sizes
- **Type Safety**: Full TypeScript support
- **Partial Hydration**: Only interactive components load JavaScript

## Project Structure

```
frontend/
├── src/
│   ├── components/     # React components
│   ├── layouts/        # Astro layouts
│   └── pages/          # Astro pages (routes)
├── public/             # Static assets
├── astro.config.mjs    # Astro configuration
├── tsconfig.json       # TypeScript configuration
└── package.json        # Dependencies
```
