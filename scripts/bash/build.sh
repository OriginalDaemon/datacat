#!/bin/bash
# Build all datacat binaries

echo "Building datacat binaries..."
cd "$(dirname "$0")/../.." || exit 1

# Create bin directory
mkdir -p bin

# Build server
echo "Building server..."
cd cmd/datacat-server || exit 1
go build -o ../../bin/datacat-server
cd ../..

# Build web UI
echo "Building web UI..."
cd cmd/datacat-web || exit 1
go build -o ../../bin/datacat-web
cd ../..

# Build daemon
echo "Building daemon..."
cd cmd/datacat-daemon || exit 1
go build -o ../../bin/datacat-daemon
cd ../..

# Build Go client example
echo "Building Go client example..."
cd examples/go-client-example || exit 1
go build -o ../../bin/go-client-example
cd ../..

echo "✅ All binaries built successfully in bin/ directory"
