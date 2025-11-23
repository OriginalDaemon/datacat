#!/bin/bash
# Test script for security features

set -e

echo "========================================="
echo "Testing Datacat Security Features"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Generate test API key
API_KEY=$(openssl rand -hex 16)
echo "Generated test API key: $API_KEY"
echo ""

# Create test configs
echo "Creating test configurations..."

# Server config with API key
cat > test_server_config.json <<EOF
{
  "data_path": "./test_data",
  "retention_days": 30,
  "cleanup_interval_hours": 24,
  "server_port": "19090",
  "heartbeat_timeout_seconds": 60,
  "api_key": "$API_KEY",
  "require_api_key": true
}
EOF

# Daemon config with API key and compression
cat > test_daemon_config.json <<EOF
{
  "daemon_port": "18079",
  "server_url": "http://localhost:19090",
  "batch_interval_seconds": 2,
  "max_batch_size": 100,
  "heartbeat_timeout_seconds": 60,
  "api_key": "$API_KEY",
  "enable_compression": true,
  "tls_verify": true,
  "tls_insecure_skip_verify": false
}
EOF

echo "✓ Configs created"
echo ""

# Build binaries
echo "Building binaries..."
cd cmd/datacat-server && go build -o ../../datacat-server && cd ../..
cd cmd/datacat-daemon && go build -o ../../datacat-daemon && cd ../..
echo "✓ Binaries built"
echo ""

# Start server
echo "Starting server..."
./datacat-server -config test_server_config.json > test_server.log 2>&1 &
SERVER_PID=$!
sleep 2

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}✗ Server failed to start${NC}"
    cat test_server.log
    exit 1
fi
echo "✓ Server started (PID: $SERVER_PID)"
echo ""

# Test 1: Health check (should work without API key)
echo "Test 1: Health check (no auth required)"
if curl -s -f http://localhost:19090/health > /dev/null; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed${NC}"
    kill $SERVER_PID
    exit 1
fi
echo ""

# Test 2: Create session without API key (should fail)
echo "Test 2: Create session without API key (should fail)"
RESPONSE=$(curl -s -w "%{http_code}" -X POST http://localhost:19090/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"product":"TestApp","version":"1.0"}' \
  -o /dev/null)

if [ "$RESPONSE" = "401" ]; then
    echo -e "${GREEN}✓ Correctly rejected unauthorized request${NC}"
else
    echo -e "${RED}✗ Expected 401, got $RESPONSE${NC}"
    kill $SERVER_PID
    exit 1
fi
echo ""

# Test 3: Create session with API key (should succeed)
echo "Test 3: Create session with valid API key (should succeed)"
SESSION_RESPONSE=$(curl -s -X POST http://localhost:19090/api/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -d '{"product":"TestApp","version":"1.0"}')

SESSION_ID=$(echo $SESSION_RESPONSE | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)

if [ -n "$SESSION_ID" ]; then
    echo -e "${GREEN}✓ Session created: $SESSION_ID${NC}"
else
    echo -e "${RED}✗ Failed to create session${NC}"
    echo "Response: $SESSION_RESPONSE"
    kill $SERVER_PID
    exit 1
fi
echo ""

# Test 4: Send compressed data
echo "Test 4: Send compressed event data"
EVENT_DATA='{"name":"test_event","level":"info","message":"Test message","data":{"key":"value"}}'

# Create gzip compressed data
echo -n "$EVENT_DATA" | gzip | curl -s -X POST \
  "http://localhost:19090/api/sessions/$SESSION_ID/events" \
  -H "Content-Type: application/json" \
  -H "Content-Encoding: gzip" \
  -H "Authorization: Bearer $API_KEY" \
  --data-binary @- > /dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Compressed event sent successfully${NC}"
else
    echo -e "${RED}✗ Failed to send compressed event${NC}"
    kill $SERVER_PID
    exit 1
fi
echo ""

# Test 5: Verify data was received
echo "Test 5: Verify data was stored"
sleep 1
SESSION_DATA=$(curl -s -H "Authorization: Bearer $API_KEY" \
  "http://localhost:19090/api/sessions/$SESSION_ID")

if echo "$SESSION_DATA" | grep -q "test_event"; then
    echo -e "${GREEN}✓ Event data verified in session${NC}"
else
    echo -e "${RED}✗ Event data not found${NC}"
    echo "Session data: $SESSION_DATA"
    kill $SERVER_PID
    exit 1
fi
echo ""

# Cleanup
echo "Cleaning up..."
kill $SERVER_PID
sleep 1
rm -rf test_data test_server_config.json test_daemon_config.json
rm -f datacat-server datacat-daemon test_server.log
echo "✓ Cleanup complete"
echo ""

echo "========================================="
echo -e "${GREEN}All tests passed!${NC}"
echo "========================================="
echo ""
echo "Summary:"
echo "  ✓ Health check works without auth"
echo "  ✓ API key authentication working"
echo "  ✓ Compressed data transfer working"
echo "  ✓ Data integrity maintained"

