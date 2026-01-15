# Ollerod PMS - Hotel Property Management System

A modern Hotel Property Management System built with Clean Architecture principles.

## Architecture

### Backend (Go)
- **Clean Architecture** with clear separation of concerns
- **Repository Pattern** for data access
- **pgx/v5** for PostgreSQL connectivity
- **testify** for comprehensive unit testing
- **UUIDs** for all entity identifiers

### Frontend (Astro + React + TypeScript)
- **Astro** for optimized static site generation
- **React** components with TypeScript
- **Partial Hydration** for better performance

### Infrastructure
- **PostgreSQL 16** database via Docker Compose
- **Makefile** for common development tasks
- **setup.sh** for automated project initialization

## Project Structure

```
ollerod-pms/
├── backend/
│   ├── cmd/
│   │   └── api/              # API entry point
│   ├── internal/
│   │   ├── models/           # Domain models (UUIDs)
│   │   ├── service/          # Business logic (Booking service)
│   │   └── store/
│   │       └── postgres/     # Repository implementations (pgx/v5)
│   ├── migrations/           # Database migrations
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/       # React components
│   │   ├── layouts/          # Astro layouts
│   │   └── pages/            # Astro pages
│   ├── public/               # Static assets
│   ├── astro.config.mjs
│   ├── tsconfig.json
│   └── package.json
├── docker-compose.yml        # PostgreSQL 16 container
├── Makefile                  # Development tasks
└── setup.sh                  # Setup automation script
```

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Docker and Docker Compose
- PostgreSQL client (optional, for migrations)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/lexxcode1/ollerod-pms.git
cd ollerod-pms
```

2. Run the setup script:
```bash
chmod +x setup.sh
./setup.sh
```

Or use Make:
```bash
make setup
```

### Running the Application

Start all services:
```bash
# Start database
make db-up

# In one terminal - run backend
make run-backend

# In another terminal - run frontend
make run-frontend
```

Access the application:
- Frontend: http://localhost:4321
- Backend API: http://localhost:8080
- Health check: http://localhost:8080/health

## Development

### Available Make Commands

```bash
make help          # Show all available commands
make setup         # Initialize the project
make db-up         # Start PostgreSQL database
make db-down       # Stop database
make db-reset      # Reset database (fresh start)
make build         # Build both backend and frontend
make test          # Run all tests
make run-backend   # Run backend server
make run-frontend  # Run frontend dev server
make clean         # Clean build artifacts
```

### Backend Development

```bash
cd backend

# Run tests
go test -v ./...

# Run the API server
go run ./cmd/api

# Build binary
go build -o ../bin/api ./cmd/api
```

### Frontend Development

```bash
cd frontend

# Install dependencies
npm install

# Start dev server
npm run dev

# Build for production
npm run build
```

## Database

The application uses PostgreSQL 16 with the following schema:
- **guests**: Guest information
- **rooms**: Room inventory with pricing
- **bookings**: Booking records with relationships

Migrations are located in `backend/migrations/`.

## API Endpoints

### Health
- `GET /health`: Health check

### Bookings
- `POST /api/bookings`: Create a new booking
- `GET /api/bookings/{id}`: Get booking details
- `PATCH /api/bookings/{id}`: Update booking status (confirm/cancel)

## Testing

Run backend tests:
```bash
make test-backend
# or
cd backend && go test -v ./...
```

The test suite includes unit tests for:
- Booking service business logic
- Date validation
- Room availability checking
- Overlapping booking detection

## Features

### Implemented
- ✅ Clean Architecture with Go backend
- ✅ Repository Pattern with pgx/v5
- ✅ Booking service with validation logic
- ✅ Unit tests with testify
- ✅ Astro + React + TypeScript frontend
- ✅ Docker Compose with PostgreSQL 16
- ✅ Automated setup script
- ✅ Makefile for development tasks

### Domain Models
- **Guest**: Hotel guest information
- **Room**: Room details with pricing and status
- **Booking**: Reservation with check-in/out dates

## License

MIT