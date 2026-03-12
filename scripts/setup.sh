#!/bin/bash
# scripts/setup.sh

echo "🚀 Starting Yop PMS Setup..."

# 1. Check for Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install it first."
    exit 1
fi

# 2. Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed."
    exit 1
fi

echo "📦 Installing Go tools (sqlc, goose, air, swag, golangci)..."
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/air-verse/air@latest
go install github.com/swaggo/swag/cmd/swag@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5

# 4. Handle .env
if [ ! -f .env ]; then
    echo "📄 Creating .env from .env.example..."
    cp .env.example .env
    echo "⚠️  Please check .env and update secrets if necessary."
else
    echo "✅ .env already exists."
fi

# 5. Frontend setup
echo "🏗️  Setting up frontend dependencies..."
cd web && npm install

echo "✨ Setup complete! Run 'make dev' to start the engine."