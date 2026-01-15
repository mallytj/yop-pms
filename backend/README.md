# Backend - Hotel PMS

Clean Architecture Go backend for the Hotel Property Management System.

## Architecture

This backend follows Clean Architecture principles with the following structure:

- **cmd/api**: Application entry point and HTTP handlers
- **internal/models**: Domain models (entities)
- **internal/service**: Business logic layer
- **internal/store/postgres**: Data access layer using Repository Pattern

## Technologies

- **Go**: 1.21+
- **Database**: PostgreSQL 16 with pgx/v5 driver
- **Testing**: testify for unit tests
- **UUID**: google/uuid for unique identifiers

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 16 (via Docker Compose)

### Setup

1. Initialize dependencies:
```bash
go mod download
```

2. Start the database:
```bash
cd .. && make db-up
```

3. Run migrations:
```bash
psql "postgres://pms_user:pms_password@localhost:5432/pms_db?sslmode=disable" -f migrations/001_initial_schema.sql
```

4. Run the application:
```bash
go run ./cmd/api
```

The server will start on `http://localhost:8080`

## Testing

Run unit tests:
```bash
go test -v ./...
```

## API Endpoints

### Health Check
- `GET /health`: Check server health

### Bookings
- `POST /api/bookings`: Create a new booking
- `GET /api/bookings/{id}`: Get booking by ID
- `PATCH /api/bookings/{id}`: Update booking status (confirm/cancel)

## Environment Variables

- `DATABASE_URL`: PostgreSQL connection string (default: `postgres://pms_user:pms_password@localhost:5432/pms_db?sslmode=disable`)
- `PORT`: Server port (default: `8080`)
