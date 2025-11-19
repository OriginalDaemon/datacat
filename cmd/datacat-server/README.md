# datacat-server

The main REST API server for datacat. This service provides endpoints for session management, state tracking, event logging, and metrics collection.

## Features

- Session lifecycle management (create, update, end)
- Deep merge for nested state updates
- Event and metric logging
- **BadgerDB persistence** (data survives restarts)
- **Configurable data retention** (default: 1 year)
- **Automatic cleanup** of old sessions
- Grafana JSON export endpoint

## Configuration

The server uses a `config.json` file for configuration. If not found, it creates a default configuration.

### Configuration Options

Create a `config.json` file in the same directory as the server:

```json
{
  "data_path": "./datacat_data",
  "retention_days": 365,
  "cleanup_interval_hours": 24,
  "server_port": "9090"
}
```

**Options:**
- `data_path` - Directory for BadgerDB data storage (default: `./datacat_data`)
- `retention_days` - Number of days to keep session data (default: `365` = 1 year)
- `cleanup_interval_hours` - Hours between automatic cleanup runs (default: `24`)
- `server_port` - Port to run the server on (default: `9090`)

**Note:** The cleanup routine runs automatically in the background and removes sessions older than `retention_days`.

## Running

```bash
cd cmd/datacat-server
go run main.go config.go
```

The server will:
1. Load or create `config.json`
2. Initialize BadgerDB at the configured `data_path`
3. Start the cleanup routine
4. Start listening on the configured port

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
- `GET /api/data/sessions` - Export all sessions for Grafana

## Data Management

### Manual Cleanup

While automatic cleanup runs based on `cleanup_interval_hours`, you can also trigger cleanup by restarting the server or implementing a manual cleanup endpoint if needed.

### Changing Retention Period

1. Edit `config.json` and update `retention_days`
2. Restart the server
3. Old sessions will be removed on the next cleanup cycle

### Backup and Migration

The data is stored in the directory specified by `data_path`. To backup:

```bash
# Stop the server first
cp -r ./datacat_data ./datacat_data_backup
```

To migrate to a new location, update `data_path` in `config.json`.

