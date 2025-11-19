#!/bin/bash
# Run linters and code quality checks

echo "Running code quality checks..."
cd "$(dirname "$0")/../.." || exit 1

# Run Black
echo "Checking Python formatting with Black..."
black --check python/ examples/ tests/
BLACK_EXIT=$?

# Run mypy
echo "Running mypy type checking..."
mypy python/ --ignore-missing-imports
MYPY_EXIT=$?

if [ $BLACK_EXIT -eq 0 ] && [ $MYPY_EXIT -eq 0 ]; then
    echo "✅ All checks passed"
    exit 0
else
    echo "❌ Some checks failed"
    exit 1
fi
