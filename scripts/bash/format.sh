#!/bin/bash
# Auto-format Python code with Black

echo "Formatting Python code with Black..."
cd "$(dirname "$0")/../.." || exit 1
black python/ examples/ tests/
echo "✅ Formatting complete"
