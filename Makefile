.PHONY: help clean swag dev docker-up gen audit setup reset-db test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf backend/vendor/
	rm -rf frontend/dist/
	rm -rf frontend/node_modules/

swag: ## Makes the swagger files
	swag init -g cmd/server/main.go -o ./api --parseInternal --parseDependency --instanceName yop

dev: ## Runs the backend & frontend
	(trap 'kill 0' SIGINT; air & cd web && npm run dev)
docker-up: ## Runs docker
	docker-compose up -d

gen: ## Sync backend & frontend contracts
	chmod +x scripts/gen-api.sh
	./scripts/gen-api.sh

audit: ## Run quality checks
	@echo "🔍 Auditing Backend..."
	go mod verify
	go mod tidy
	go vet ./...
	# Run golangci-lint (checks for dead code, shadowing, etc.)
	golangci-lint run ./...
	# Run Go tests with the race detector (CRITICAL for a booking engine!)
	go test -v -race -buildvcs ./...
	# Vulnerability check
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

	@echo "🔍 Auditing Frontend..."
	cd web && npm run lint
	# svelte-check does the heavy lifting of type-checking your .svelte files
	cd web && npm run check
	@echo "✅ All checks passed!"

setup: ## Run to init the project
	chmod +x scripts/setup.sh
	./scripts/setup.sh

reset-db: ## Run to reset the docker
	docker-compose down -v
	docker-compose up -d
	@echo "Waiting for database to be ready..."
	@sleep 3

test-backend: ## Run all tests in the backend
	go test ./...

test-frontend: ## Run all tests in the frontend
	npm run test

test: ## Run all tests
	make test-backend && make test-frontend

format: 
	go fmt ./...
	cd web && npm run format 