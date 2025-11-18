# datacat-server

The main REST API server for datacat. This service provides endpoints for session management, state tracking, event logging, and metrics collection.

## Features

- Session lifecycle management (create, update, end)
- Deep merge for nested state updates
- Event and metric logging
- BadgerDB persistence (data survives restarts)
- Grafana JSON export endpoint

## Running

```bash
cd cmd/datacat-server
go run main.go
```

The server will start on `http://localhost:8080` by default.

## Building

```bash
cd cmd/datacat-server
go build -o datacat-server
./datacat-server
```

## API Endpoints

- `POST /api/sessions` - Create new session
- `GET /api/sessions/{id}` - Get session details
- `POST /api/sessions/{id}/state` - Update session state (deep merge)
- `POST /api/sessions/{id}/events` - Log event
- `POST /api/sessions/{id}/metrics` - Log metric
- `POST /api/sessions/{id}/end` - End session
- `GET /api/grafana/sessions` - Export all sessions for Grafana

## Environment

- `PORT` - Server port (default: 8080)
- `BADGER_DIR` - Database directory (default: ./badger_data)
