# Go Client Example

Demonstrates using the datacat Go client library to track application sessions.

## Prerequisites

Make sure the datacat-server is running:

```bash
cd ../../cmd/datacat-server
go run main.go
```

## Running

```bash
go run main.go
```

## What It Does

This example:
1. Creates a new session
2. Sets initial application state
3. Logs startup event
4. Simulates work by:
   - Generating random CPU and memory metrics
   - Logging metrics every 2 seconds
   - Updating state with current values
5. Logs completion event
6. Ends the session
7. Retrieves session to verify data was saved

## Output

```
Created session: <uuid>
Iteration 1: CPU=45.23%, Memory=234.56MB
Iteration 2: CPU=62.45%, Memory=189.34MB
...
Session ended successfully
Final session state:
  Active: false
  Events: 2
  Metrics: 20
```

## View Results

- **Web UI**: http://localhost:8080 (if datacat-web is running)
- **API**: http://localhost:9090/api/sessions/<session_id>
- **External Tools**: Use the `/api/data/sessions` endpoint
