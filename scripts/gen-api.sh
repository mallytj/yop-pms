#!/bin/bash
# scripts/gen-api.sh

# 1. Generate Swagger 2.0 from Go comments
echo "🛠️  Generating Swagger 2.0 from Go comments..."
make swag

# 2. Convert Swagger 2.0 to OpenAPI 3.0
# We use npx to run it on the fly without a global install
echo "🌉 Converting Swagger 2.0 to OpenAPI 3.0..."
npx swagger2openapi ./api/yop_swagger.json -o ./api/openapi.yaml --patch --yaml

# 3. Generate TypeScript Types from the new v3.0 spec
echo "🟦 Generating SvelteKit TypeScript types..."
npx openapi-typescript ./api/openapi.yaml -o ./web/src/lib/types/api.d.ts

# 4. Run sqlc
echo "🗄️  Running sqlc generate..."
sqlc generate

echo "✅ Generation complete!"