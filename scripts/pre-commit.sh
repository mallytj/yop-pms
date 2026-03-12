#!/bin/bash

echo "🧹 Running pre-commit formatting..."

make format

git add .

echo "✅ Code formatted. Committing..."
exit 0