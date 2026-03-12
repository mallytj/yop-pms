.PHONY: help setup db-up db-down db-reset build-backend build-frontend build test-backend test-frontend test run-backend run-frontend clean build-model

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Run setup script to initialize project
	chmod +x setup.sh
	./setup.sh

db-up: ## Start database with Docker Compose
	docker compose up -d postgres

db-down: ## Stop database
	docker compose down

db-reset: ## Reset database (stop, remove volumes, and start fresh)
	docker compose down -v
	docker compose up -d postgres

build-backend: ## Build backend binary
	cd backend && go build -o ../bin/api ./cmd/api

build-frontend: ## Build frontend for production
	cd frontend && npm run build

build: build-backend build-frontend ## Build both backend and frontend

test-backend: ## Run backend tests
	cd backend && go test -v ./...

test-frontend: ## Run frontend tests
	cd frontend && npm test

test: test-backend test-frontend ## Run all tests

run-backend: ## Run backend server
	cd backend && go run ./cmd/...

run-frontend: ## Run frontend dev server
	cd frontend/client && npm run dev

seed-backend: 
	cd backend && go run ./cmd/... -seed

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf backend/vendor/
	rm -rf frontend/dist/
	rm -rf frontend/.astro/
	rm -rf frontend/node_modules/
