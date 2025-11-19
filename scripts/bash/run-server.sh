#!/bin/bash
# Run the datacat server

echo "Starting datacat server..."
cd "$(dirname "$0")/../.." || exit 1
cd cmd/datacat-server || exit 1
go run main.go config.go
