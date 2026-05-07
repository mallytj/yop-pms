#!/bin/bash
set -e

echo "🧹 Running pre-commit formatting..."

# Stash unstaged changes (if any) so only staged files get reformatted
# Use git diff to check since git stash push does nothing if clean, and
# git stash pop would then fail on an empty stash stack.
if ! git diff --quiet; then
  git stash push --keep-index -m "pre-commit-format"
  stashed=true
else
  stashed=false
fi

make format

# Re-stage only formatting changes to already-tracked files
git add -u

# Restore unstaged changes
if [ "$stashed" = true ]; then
  git stash pop --index
fi

echo "✅ Code formatted. Committing..."
exit 0