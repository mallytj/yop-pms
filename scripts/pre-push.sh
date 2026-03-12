echo "🚀 Running pre-push audit..."

# Run the audit
make audit

# Run all testsH
make test

# Capture the exit code
RESULT=$?

if [ $RESULT -ne 0 ]; then
    echo "❌ Audit failed! Push aborted. Please fix the issues and try again."
    exit 1
fi

echo "✅ Audit passed. Pushing code..."
exit 0