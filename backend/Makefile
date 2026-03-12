.PHONY: help build-entity-dir db-up db-down db-reset sql

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build-entity-dir: ## Usage: make build-entity-dir DIR=internal/room_types
	@mkdir -p $(DIR)
	@name=$(notdir $(DIR)); \
	touch $(DIR)/handlers.go $(DIR)/helpers.go $(DIR)/types.go $(DIR)/service.go $(DIR)/$${name}_test.go; \
	for file in handlers.go helpers.go types.go service.go $${name}_test.go; do \
		echo "package $${name}" > $(DIR)/$$file; \
	done
	@echo "Created handlers.go, helpers.go, types.go, service.go, and $${name}_test.go in $(DIR)"


db-up: ## Start database with Docker Compose
	docker compose up -d postgres

db-down: ## Stop database
	docker compose down

db-reset: ## Reset database (stop, remove volumes, and start fresh)
	docker compose down -v
	docker compose up -d postgres

sql: ## Generate SQLC code
	sqlc generate

swag: ## Generate Swagger documentation
	cd ./cmd && swag init --parseInternal --parseDependency --dir .,../internal/handlers,../internal/service

