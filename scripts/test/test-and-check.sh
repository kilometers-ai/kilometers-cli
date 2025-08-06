#!/bin/bash

# Wrapper script demonstrating different ways to run tests and check exit codes

echo "=== Method 1: Using && operator (only run second command if first succeeds) ==="
./scripts/run-tests.sh --coverage && echo "✅ Tests passed! Exit code: $?"

echo ""
echo "=== Method 2: Using || operator (only run second command if first fails) ==="
./scripts/run-tests.sh --coverage || echo "❌ Tests failed! Exit code: $?"

echo ""
echo "=== Method 3: Capture exit code explicitly ==="
./scripts/run-tests.sh --coverage
EXIT_CODE=$?
echo "Tests completed with exit code: $EXIT_CODE"
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Tests passed!"
else
    echo "❌ Tests failed!"
fi

echo ""
echo "=== Method 4: One-liner with exit code check ==="
./scripts/run-tests.sh --coverage; [ $? -eq 0 ] && echo "✅ Success" || echo "❌ Failed"

echo ""
echo "=== Method 5: For CI/CD - fail fast if tests fail ==="
./scripts/run-tests.sh --coverage && echo "✅ Ready for deployment" || (echo "❌ Deployment blocked" && exit 1) 