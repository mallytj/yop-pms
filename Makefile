.PHONY: help clean swag dev docker-up gen audit setup reset-db test sqlc _guard-local-db goose-circle

COVERAGE_DIR = /tmp/yop-pms-coverage
COVERAGE_FILE = $(COVERAGE_DIR)/cover.out
COVERAGE_HTML = $(COVERAGE_DIR)/cover.html

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

sqlc: ## Generate SQLC code
	sqlc generate

swag: ## Makes the swagger files
	swag init -g cmd/server/main.go -o ./api --parseInternal --parseDependency --instanceName yop

dev: ## Runs the backend & frontend
	(trap 'kill 0' SIGINT; air & cd web && npm run dev)
docker-up: ## Runs docker
	docker-compose up -d

db-up: ## Runs just the databases
	docker-compose up -d postgres
	docker-compose up -d redis

gen: ## Sync backend & frontend contracts
	chmod +x scripts/gen-api.sh
	./scripts/gen-api.sh

GOBIN := $(shell go env GOPATH)/bin

audit: ## Run quality checks
	@echo "🔍 Auditing Backend..."
	go mod verify
	go mod tidy
	go vet ./...

	gotestsum -- -v -race -buildvcs ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

	@echo "🔍 Auditing Frontend..."
	cd web && npm run check
	@echo "✅ All checks passed!"

setup: ## Run to init the project
	chmod +x scripts/setup.sh
	./scripts/setup.sh

lint: ## Lints both front and backend
	@echo "Linting Backend..."
	golangci-lint run ./...

	@echo "Linting Frontend..."
	cd web && npm run lint


_guard-local-db:
	@if [ ! -f .env ]; then echo "refusing: .env file is missing. Run 'make setup'."; exit 1; fi
	@APP_ENV_VAL=$$(grep -E '^APP_ENV=' .env | cut -d= -f2- | xargs); \
	[ "$$APP_ENV_VAL" = "dev" ] || { echo "refusing: APP_ENV in .env must be 'dev' (got '$$APP_ENV_VAL')"; exit 1; }
	@[ "$$CONFIRM" = "YES" ] || { echo "refusing: set CONFIRM=YES to proceed with destructive DB action"; exit 1; }

reset-db: _guard-local-db ## Run to reset the docker (requires CONFIRM=YES and local GOOSE_DBSTRING)
	docker-compose down -v
	docker-compose up -d
	@echo "Waiting for database to be ready..."
	@sleep 3

test-backend: ## Run all tests in the backend
	gotestsum -- -buildvcs -race ./...

test-frontend: ## Run all tests in the frontend
	cd web && npm run test

test: ## Run all tests
	make test-backend && make test-frontend

format: ## Formats all code
	go fmt ./...
	cd web && npm run format 

goose-circle: _guard-local-db ## Completely reset goose (requires CONFIRM=YES and local GOOSE_DBSTRING)
	goose reset
	goose up

gen-constraints: ## Sync constraints.yml and constraints.ts from live DB check constraints
	go run ./cmd/tools/sync-constraints/...

## test-cover: Run tests and open coverage report in browser
test-cover:
	@mkdir -p $(COVERAGE_DIR)
	@echo "Running tests and generating coverage..."
	gotestsum -- -v -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"
	@if [ "$$(uname)" = "Darwin" ]; then \
		open $(COVERAGE_HTML); \
	elif [ "$$(uname)" = "Linux" ]; then \
		xdg-open $(COVERAGE_HTML); \
	else \
		echo "Please open $(COVERAGE_HTML) manually."; \
	fi

## clean-cover: Remove coverage files
clean-cover:
	rm -rf $(COVERAGE_DIR)
