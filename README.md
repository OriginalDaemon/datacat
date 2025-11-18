# datacat

[![Tests](https://github.com/OriginalDaemon/datacat/workflows/Tests/badge.svg)](https://github.com/OriginalDaemon/datacat/actions)
[![codecov](https://codecov.io/gh/OriginalDaemon/datacat/branch/main/graph/badge.svg)](https://codecov.io/gh/OriginalDaemon/datacat)
[![Go Report Card](https://goreportcard.com/badge/github.com/OriginalDaemon/datacat)](https://goreportcard.com/report/github.com/OriginalDaemon/datacat)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A REST API service for logging arbitrary application data, events, and metrics with session tracking.

## Overview

datacat provides a Go-based REST API service that allows applications to:
- Register sessions and receive unique session IDs
- Log and update state, events, and metrics associated with each session
- Export data in JSON format for visualization tools like Grafana

## Components

### Go Service (Backend)

A lightweight REST API server written in Go with BadgerDB for fast data persistence.

### Go Client Library

A Go client library (`client/`) for interacting with the datacat API from Go applications.

### Python Client (Frontend)

A Python module compatible with both Python 2.7+ and Python 3.x for easy interaction with the datacat service.

### Web UI (htmx)

An interactive web dashboard built with htmx for browsing sessions and visualizing metrics with advanced filtering capabilities.

## Installation

### Go Service

1. Build the service:
```bash
go build -o datacat
```

2. Run the service:
```bash
./datacat
```

The service will start on `http://localhost:8080` by default.

### Python Client

Install the Python client:
```bash
cd python
pip install .
```

Or for development:
```bash
cd python
pip install -e .
```

## API Endpoints

### Session Management

**Create a new session**
```
POST /api/sessions
Response: {"session_id": "uuid-string"}
```

**Get session details**
```
GET /api/sessions/{session_id}
Response: Session object with all data
```

**End a session**
```
POST /api/sessions/{session_id}/end
Response: {"status": "ok"}
```

### Data Logging

**Update session state (supports nested objects)**
```
POST /api/sessions/{session_id}/state
Body: {"key": "value", "nested": {"key": "value"}}
Response: {"status": "ok"}

Note: State updates are deep-merged, so you can update nested keys without 
losing other data in the same parent object.
```

**Log an event**
```
POST /api/sessions/{session_id}/events
Body: {"name": "event_name", "data": {"key": "value"}}
Response: {"status": "ok"}
```

**Log a metric**
```
POST /api/sessions/{session_id}/metrics
Body: {"name": "metric_name", "value": 123.45, "tags": ["tag1", "tag2"]}
Response: {"status": "ok"}
```

### Grafana Integration

**Get all sessions (for Grafana)**
```
GET /api/grafana/sessions
Response: Array of all session objects
```

## Web UI

The datacat web UI provides an interactive dashboard for browsing sessions and visualizing metrics.

### Starting the Web UI

```bash
cd web
go run main.go
```

The web UI will be available at `http://localhost:8081`

### Features

- **Dashboard**: Overview of all sessions with statistics
- **Session Browser**: View detailed session information including state, events, and metrics
- **Advanced Metrics Visualization**:
  - Interactive timeseries charts using Chart.js
  - Multiple filtering modes:
    - Current state filtering
    - State history filtering (sessions that ever had a value)
    - Array contains filtering (e.g., find sessions with "space probe" in open_windows)
  - Aggregation modes:
    - All values: Show every metric point
    - Peak per session: Show highest value from each session
    - Average per session: Show average value from each session
    - Min per session: Show lowest value from each session
  - Real-time statistics (peak, average, min values)

### Example Queries

**Peak memory for sessions with "space probe" window open:**
- Metric: `memory_usage`
- Aggregation: `peak`
- Filter Mode: `State Array Contains`
- Filter Path: `window_state.open`
- Filter Value: `space probe`

**CPU usage for currently running applications:**
- Metric: `cpu_usage`
- Aggregation: `all`
- Filter Mode: `Current State`
- Filter Path: `application.status`
- Filter Value: `running`

## Python Client Usage

### Basic Usage

```python
from datacat import create_session

# Create a new session
session = create_session("http://localhost:8080")

# Update state
session.update_state({"status": "running", "progress": 0})

# Log events
session.log_event("started", {"user": "alice"})

# Log metrics
session.log_metric("cpu_usage", 45.2, tags=["host:server1"])

# Get session details
details = session.get_details()
print(details)
```

### Advanced Usage

```python
from datacat import DatacatClient, Session

# Create client
client = DatacatClient("http://localhost:8080")

# Register a session manually
session_id = client.register_session()

# Create session object
session = Session(client, session_id)

# Use session methods
session.update_state({"phase": "initialization"})
session.log_event("initialization_complete")
session.log_metric("init_time", 1.234)

# Get all sessions (for monitoring/debugging)
all_sessions = client.get_all_sessions()
```

### Python 2 Compatibility

The client is designed to work with both Python 2.7+ and Python 3.x:

```python
# Python 2.7
from __future__ import print_function
from datacat import create_session

session = create_session()
session.update_state({"version": "2.7"})
print("Session ID:", session.session_id)
```

### Nested State Tracking

Track complex application state with hierarchical data structures:

```python
from datacat import create_session

session = create_session()

# Set initial nested state
session.update_state({
    "window_state": {
        "open": ["window 1", "window 2"],
        "active": "window 1"
    },
    "memory": {
        "footprint_mb": 50.2
    },
    "settings": {
        "theme": "dark"
    }
})

# Partial update - only changes window_state.open
# Preserves window_state.active and all other state
session.update_state({
    "window_state": {
        "open": ["window 1", "window 2", "window 3"]
    }
})

# Update different part of state
session.update_state({
    "memory": {
        "footprint_mb": 75.5
    }
})
```

### Heartbeat Monitoring

Automatically detect when your application appears to be hung:

```python
from datacat import create_session
import time

session = create_session()

# Start heartbeat monitor (default: 60 second timeout)
session.start_heartbeat_monitor(timeout=60, check_interval=5)

# Your application loop
while running:
    # Send heartbeat to indicate app is alive
    session.heartbeat()
    
    # Do work...
    do_work()
    
    time.sleep(1)

# End session normally
session.end()
```

If your application stops sending heartbeats for 60 seconds (or the configured timeout), 
the monitor will automatically log an `application_appears_hung` event. This allows you to:

- Identify sessions that crashed or hung without proper shutdown
- Track reliability metrics in Grafana
- Find sessions where `active=true` and last event is `application_appears_hung`

### Session Lifecycle Management

Properly track session start and end:

```python
session = create_session()

# Log session start
session.log_event("session_started", {"user": "alice"})

# Do work...
session.update_state({"status": "running"})

# End session properly
session.log_event("session_ending")
session.end()
```

Sessions that are ended will have:
- `active: false`
- `ended_at` timestamp set

This allows Grafana queries to distinguish between:
- Active sessions (`active=true`)
- Properly ended sessions (`active=false`, has `ended_at`)
- Crashed/hung sessions (`active=true`, last event is `application_appears_hung`)

## Use Cases

### Application Monitoring

Track application lifecycle and performance:

```python
session = create_session()
session.update_state({"app": "myapp", "status": "starting"})
session.log_event("startup", {"version": "1.0.0"})
session.log_metric("startup_time", 2.5)

# During execution
session.update_state({"status": "running"})
session.log_metric("requests_per_second", 1000)

# On shutdown
session.log_event("shutdown")
session.update_state({"status": "stopped"})
```

### Testing and CI/CD

Track test runs and build metrics:

```python
session = create_session()
session.update_state({"test_suite": "integration", "status": "running"})

for test in tests:
    session.log_event("test_started", {"name": test.name})
    result = run_test(test)
    session.log_event("test_completed", {
        "name": test.name,
        "result": result.status,
        "duration": result.duration
    })
    session.log_metric("test_duration", result.duration, 
                      tags=[f"test:{test.name}", f"status:{result.status}"])

session.update_state({"status": "completed"})
```

### Grafana Integration

The `/api/grafana/sessions` endpoint returns all session data in JSON format that can be consumed by Grafana using the JSON API data source plugin. This allows you to:

- Visualize metrics over time
- Track events across sessions
- Monitor application state changes
- Create dashboards for session analytics

#### Querying for Hung/Crashed Sessions

To identify problematic sessions in Grafana:

**Active sessions that may have crashed:**
```javascript
// Filter sessions where active=true (no end event)
sessions.filter(s => s.active === true)
```

**Sessions that appeared to hang:**
```javascript
// Find sessions with "application_appears_hung" as last event
sessions.filter(s => {
  if (s.events.length === 0) return false;
  const lastEvent = s.events[s.events.length - 1];
  return lastEvent.name === "application_appears_hung";
})
```

**Sessions that recovered after hanging:**
```javascript
// Find sessions with both hung and recovered events
sessions.filter(s => {
  const eventNames = s.events.map(e => e.name);
  return eventNames.includes("application_appears_hung") && 
         eventNames.includes("application_recovered");
})
```

**Calculate reliability metrics:**
```javascript
const total = sessions.length;
const completed = sessions.filter(s => !s.active).length;
const hung = sessions.filter(s => {
  const lastEvent = s.events[s.events.length - 1];
  return lastEvent && lastEvent.name === "application_appears_hung";
}).length;

const reliability = ((total - hung) / total) * 100;
```

## Data Model

### Session Object

```json
{
  "id": "uuid-string",
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:05:00Z",
  "ended_at": "2023-01-01T00:10:00Z",
  "active": false,
  "state": {
    "key": "value",
    "nested": {
      "key": "value"
    }
  },
  "events": [
    {
      "timestamp": "2023-01-01T00:01:00Z",
      "name": "event_name",
      "data": {"key": "value"}
    }
  ],
  "metrics": [
    {
      "timestamp": "2023-01-01T00:02:00Z",
      "name": "metric_name",
      "value": 123.45,
      "tags": ["tag1", "tag2"]
    }
  ]
}
```

## License

MIT License - see LICENSE file for details.
