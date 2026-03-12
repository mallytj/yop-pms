#!/bin/bash

# 1. Generate Go Code (using sqlc)
echo "Running sqlc generate..."
sqlc generate

# 2. Generate TypeScript Interfaces from OpenAPI spec
# We use npx to avoid global dependencies
echo "Generating TypeScript types from OpenAPI spec..."
npx openapi-typescript ./api/openapi.yaml -o ./web/src/lib/types/api.d.ts

echo "Done! Contracts are synced."