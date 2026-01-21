#!/bin/bash

set -e

echo "======================================"
echo "Ollerod PMS Setup Script"
echo "======================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Setup Backend
echo -e "${BLUE}Setting up Backend (Go)...${NC}"
cd backend

if [ ! -f "go.mod" ]; then
    echo "Initializing Go module..."
    go mod init github.com/lexxcode1/ollerod-pms/backend
fi

echo "Downloading Go dependencies..."
go mod download
go mod tidy

echo -e "${GREEN}✓ Backend setup complete${NC}"
echo ""

cd ..

# Setup Frontend
echo -e "${BLUE}Setting up Frontend (Astro + React + TypeScript)...${NC}"
cd frontend

if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install
else
    echo "Dependencies already installed"
fi

echo -e "${GREEN}✓ Frontend setup complete${NC}"
echo ""

cd ..

# Setup Database
echo -e "${BLUE}Setting up Database...${NC}"
if command -v docker &> /dev/null; then
    echo "Starting PostgreSQL container..."
    docker compose up -d postgres
    
    echo "Waiting for PostgreSQL to be ready..."
    sleep 5
    
    # Run migrations if psql is available
    if command -v psql &> /dev/null; then
        echo "Running database migrations..."
        PGPASSWORD=pms_password psql -h localhost -U pms_user -d pms_db -f backend/migrations/001_initial_schema.sql 2>/dev/null || echo "Migration may have already been applied"
        echo -e "${GREEN}✓ Database setup complete${NC}"
    else
        echo "psql not found. Please run migrations manually:"
        echo "  psql \"postgres://pms_user:pms_password@localhost:5432/pms_db\" -f backend/migrations/001_initial_schema.sql"
    fi
else
    echo "Docker not found. Please install Docker to use the database."
    echo "You can start the database later with: make db-up"
fi

echo ""
echo "======================================"
echo -e "${GREEN}Setup Complete!${NC}"
echo "======================================"
echo ""
echo "Next steps:"
echo "  1. Start the database:    make db-up"
echo "  2. Start the backend:     make run-backend"
echo "  3. Start the frontend:    make run-frontend"
echo ""
echo "Or use individual commands:"
echo "  Backend:  cd backend && go run ./cmd/api"
echo "  Frontend: cd frontend && npm run dev"
echo ""
echo "Access the application:"
echo "  Frontend: http://localhost:4321"
echo "  Backend:  http://localhost:8080"
echo "======================================"