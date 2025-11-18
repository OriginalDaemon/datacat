# datacat-web

Interactive web dashboard for browsing datacat sessions and visualizing metrics.

## Features

- Browse all sessions with real-time updates
- View session details (state, events, metrics)
- Advanced metrics visualization with Chart.js
- Filter sessions by current state or state history
- Query sessions by array contains (e.g., find sessions with specific windows open)
- Multiple aggregation modes (all values, peak, average, min per session)
- Built with htmx for reactive UI

## Running

```bash
cd cmd/datacat-web
go run main.go
```

The web UI will be available at `http://localhost:8081` by default.

## Building

```bash
cd cmd/datacat-web
go build -o datacat-web
./datacat-web
```

## Configuration

- `PORT` - Web server port (default: 8081)
- `API_URL` - datacat-server API URL (default: http://localhost:8080)

## Example Queries

**Peak memory for sessions with "space probe" window:**
- Metric: `memory_usage`
- Aggregation: `peak per session`
- Filter Mode: `State Array Contains`
- Filter Path: `window_state.open`
- Filter Value: `space probe`

**CPU usage for currently running applications:**
- Metric: `cpu_usage`
- Aggregation: `all values`
- Filter Mode: `Current State Equals`
- Filter Path: `status`
- Filter Value: `running`
