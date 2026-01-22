# Ollerod PMS (Property Management System)

A comprehensive property management system built with Go and PostgreSQL, designed for managing hotels, properties, licences, users, and related operations.

## 🏗️ Project Overview

Ollerod PMS is a backend API system for property management with a focus on:
- Multi-tenant architecture through licences
- Property and user management
- Amenity and room type tracking
- Reservation and guest management
- Audit logging for all operations

## 🛠️ Technology Stack

### Backend
- **Language**: Go 1.25.6
- **Web Framework**: Chi Router v5
- **Database**: PostgreSQL 16
- **Database Toolkit**: pgx/v5 with connection pooling (pgxpool)
- **SQL Generation**: SQLC for type-safe queries
- **Migrations**: Goose
- **Testing**: testify, faker
- **Configuration**: godotenv for environment management

### Infrastructure
- **Containerization**: Docker & Docker Compose
- **Database Admin**: pgAdmin 4
- **Development Tools**: Make for task automation

## 📁 Project Structure

```
ollerod-pms/
├── backend/
│   ├── cmd/                          # Application entrypoints
│   │   ├── main.go                   # Main application setup
│   │   └── api.go                    # API routes and server configuration
│   ├── internal/
│   │   ├── adapters/
│   │   │   └── postgresql/
│   │   │       ├── migrations/       # Database migrations
│   │   │       └── sqlc/             # Generated SQLC code
│   │   ├── config/                   # Configuration management
│   │   ├── middleware/               # HTTP middleware (context extraction)
│   │   ├── helpers/                  # Utility functions
│   │   ├── users/                    # User domain logic
│   │   ├── licences/                 # Licence domain logic
│   │   ├── properties/               # Property domain logic
│   │   ├── property_amenities/       # Amenities domain logic
│   │   └── types/                    # Shared types
│   ├── docker-compose.yaml           # Database services
│   ├── sqlc.yaml                     # SQLC configuration
│   ├── go.mod                        # Go dependencies
│   └── go.sum
├── Makefile                          # Build and development commands
├── setup.sh                          # Automated setup script
└── README.md
```

## 🚀 Getting Started

### Prerequisites
- Go 1.25.6 or higher
- Docker and Docker Compose
- Make (optional, for using Makefile commands)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/lexxcode1/ollerod-pms.git
   cd ollerod-pms
   ```

2. **Run the setup script**
   ```bash
   chmod +x setup.sh
   ./setup.sh
   ```
   
   Or manually:
   ```bash
   # Start PostgreSQL
   make db-up
   
   # Install backend dependencies
   cd backend && go mod download && go mod tidy
   ```

3. **Configure environment variables**
   
   Create a `.env` file in the `backend/` directory:
   ```env
   DB_HOST=localhost
   DB_PORT=5433
   DB_USER=admin
   DB_PASSWORD=password123
   DB_NAME=hotel_pms
   ```

4. **Start the backend server**
   ```bash
   make run-backend
   # or
   cd backend && go run ./cmd/api
   ```

The API will be available at `http://localhost:8080`

## 🗄️ Database Schema

The system uses PostgreSQL with the following main entities:

### Core Tables
- **licences** - Multi-tenant licence management
- **properties** - Hotel/property information
- **users** - System users with role-based access
- **property_amenities** - Amenities available at properties
- **room_types** - Room categories and configurations
- **rooms** - Individual room inventory
- **guests** - Guest information and preferences
- **reservations** - Booking management
- **rate_plans** - Pricing strategies
- **availability** - Daily room availability tracking
- **audit_logs** - System activity logging

## 📡 API Endpoints

### Health Check
```
GET /health - Server health status
```

### Users
```
POST   /users                    - Create a new user
GET    /users                    - List all users
GET    /users/{userID}           - Get user by ID
PUT    /users/{userID}           - Update user
DELETE /users/{userID}           - Delete user
GET    /users/{userID}/licence   - Get user's licence
```

### Licences
```
POST   /licences                        - Create a new licence
GET    /licences                        - List all licences
GET    /licences/{licenceID}            - Get licence by ID
PUT    /licences/{licenceID}            - Update licence
DELETE /licences/{licenceID}            - Delete licence
GET    /licences/{licenceID}/users      - Get users by licence
```

### Properties
```
POST   /properties                      - Create a new property
GET    /properties                      - List all properties
GET    /properties/{propertyID}         - Get property by ID
PUT    /properties/{propertyID}         - Update property
DELETE /properties/{propertyID}         - Delete property
GET    /properties/{propertyID}/licence - Get property's licence
GET    /properties/{propertyID}/users   - Get property's users
```

### Property Amenities
```
POST   /property-amenities              - Create a property amenity
```

## 🏗️ Architecture

### Service Layer Pattern
The application follows a clean architecture with:
- **Handlers**: HTTP request/response handling
- **Services**: Business logic and orchestration
- **Repository**: Database access via SQLC
- **Middleware**: Cross-cutting concerns (context, logging, recovery)

### Key Features
1. **Connection Pooling**: Uses pgxpool for efficient database connections
2. **Context Middleware**: Extracts IDs from URL parameters and injects into request context
3. **Type-Safe Queries**: SQLC generates type-safe Go code from SQL
4. **Error Handling**: Centralized error helpers with PostgreSQL error code constants
5. **Audit Logging**: Tracks all entity changes for compliance

## 🧪 Testing

Run tests with:
```bash
make test-backend
# or
cd backend && go test -v ./...
```

Test coverage includes:
- Integration tests for API endpoints
- Service layer unit tests
- Helper function tests

## 📋 Available Make Commands

```bash
make help           # Show all available commands
make setup          # Run setup script
make db-up          # Start PostgreSQL database
make db-down        # Stop database
make db-reset       # Reset database (removes data)
make build-backend  # Build backend binary
make test-backend   # Run backend tests
make run-backend    # Run backend server
make clean          # Clean build artifacts
```

## 🔧 Development Workflow

1. **Database Changes**
   - Add migration in `backend/internal/adapters/postgresql/migrations/`
   - Run migration: `make db-reset`
   - Update SQLC queries in `backend/internal/adapters/postgresql/sqlc/queries.sql`
   - Generate code: `cd backend && sqlc generate`

2. **Adding New Endpoints**
   - Create domain package in `backend/internal/{domain}/`
   - Define types, service, and handlers
   - Register routes in `backend/cmd/api.go`
   - Add middleware if needed

3. **Testing Changes**
   - Write tests in `{domain}/{domain}_test.go`
   - Run tests: `make test-backend`
   - Check coverage: `go test -cover ./...`

## 🔐 Security Features

- Password hashing with bcrypt
- UUID-based identifiers
- Audit logging for all mutations
- SQL injection protection via parameterized queries
- Connection pooling with timeout controls

## 🌐 Database Access

### Via Docker Container
```bash
docker exec -it hotel_pms_db psql -U admin -d hotel_pms
```

### Via pgAdmin
- URL: `http://localhost:5050`
- Email: `admin@admin.com`
- Password: `admin`

## 📝 Configuration

The application uses environment variables for configuration:
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5433)
- `DB_USER` - Database user (default: admin)
- `DB_PASSWORD` - Database password (default: password123)
- `DB_NAME` - Database name (default: hotel_pms)

## 🚧 Roadmap (Commented Features)

The following endpoints are planned but not yet implemented:
- Room management endpoints
- Reservation system
- Rate plan management
- Guest management
- Daily availability tracking
- Room type management

## 📄 License

This project is private and proprietary.

## 👥 Contributing

This is a private project. Contact the repository owner for contribution guidelines.

---

**Server Start Banner**
```
 ▗▄▄▖▗▄▄▄▖▗▄▄▖ ▗▖  ▗▖▗▄▄▄▖▗▄▄▖      ▗▄▄▖▗▄▄▄▖▗▄▖ ▗▄▄▖▗▄▄▄▖▗▄▄▄▖▗▄▄▄ 
▐▌   ▐▌   ▐▌ ▐▌▐▌  ▐▌▐▌   ▐▌ ▐▌    ▐▌     █ ▐▌ ▐▌▐▌ ▐▌ █  ▐▌   ▐▌  █
 ▝▀▚▖▐▛▀▀▘▐▛▀▚▖▐▌  ▐▌▐▛▀▀▘▐▛▀▚▖     ▝▀▚▖  █ ▐▛▀▜▌▐▛▀▚▖ █  ▐▛▀▀▘▐▌  █
▗▄▄▞▘▐▙▄▄▖▐▌ ▐▌ ▝▚▞▘ ▐▙▄▄▖▐▌ ▐▌    ▗▄▄▞▘  █ ▐▌ ▐▌▐▌ ▐▌ █  ▐▙▄▄▖▐▙▄▄▀
```