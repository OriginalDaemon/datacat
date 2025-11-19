#!/bin/bash
# Clean build artifacts and temporary files

echo "Cleaning build artifacts..."
cd "$(dirname "$0")/../.." || exit 1

# Remove binaries
rm -rf bin/
echo "Removed bin/"

# Remove Python cache
find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null
find . -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null
find . -type f -name "*.pyc" -delete
echo "Removed Python cache files"

# Remove test coverage
rm -f .coverage coverage.out
echo "Removed coverage files"

# Remove BadgerDB data (be careful with this!)
rm -rf datacat_data/
echo "Removed datacat_data/"

echo "✅ Cleanup complete"
