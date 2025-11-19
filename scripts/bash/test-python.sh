#!/bin/bash
# Run Python integration tests

echo "Running Python integration tests..."
cd "$(dirname "$0")/../.." || exit 1

# Start the server in background
echo "Starting test server..."
cd cmd/datacat-server
go run main.go config.go &
SERVER_PID=$!
cd ../..

# Wait for server to start
sleep 3

# Run Python tests
echo "Running tests..."
pytest tests/ -v

TEST_EXIT=$?

# Kill the server
echo "Stopping test server..."
kill $SERVER_PID 2>/dev/null

exit $TEST_EXIT
