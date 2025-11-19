#!/bin/bash
# Run the datacat web UI

echo "Starting datacat web UI..."
echo "Web UI will be available at http://localhost:8081"
cd "$(dirname "$0")/../.." || exit 1
cd cmd/datacat-web || exit 1
go run main.go
