---
layout: default
title: Quick Start
parent: Guides
nav_order: 1
---

# Quick Start Guide
{: .no_toc }

Get up and running with DataCat in minutes.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Installation

### Prerequisites

- **Go 1.21+** - For server and daemon components
- **Python 2.7+ or 3.x** - For Python client library
- **Git** - To clone the repository

### Clone the Repository

```bash
git clone https://github.com/OriginalDaemon/datacat.git
cd datacat
```

---

## Starting the Server

The DataCat server provides the REST API and data persistence layer.

### Build and Run

```bash
cd cmd/datacat-server
go build -o datacat-server
./datacat-server
```

The server will start on `http://localhost:9090` by default.

### Verify Server is Running

```bash
curl http://localhost:9090/api/data/sessions
```

You should see an empty array `[]` indicating the server is ready.

### Configuration

The server can be configured via `config.json`:

```json
{
  "port": 9090,
  "data_path": "./datacat_data",
  "retention_days": 365,
  "cleanup_interval_hours": 24,
  "heartbeat_timeout_seconds": 60
}
```

---

## Starting the Web Dashboard (Optional)

The web dashboard provides a visual interface for exploring sessions and metrics.

### Build and Run

```bash
cd cmd/datacat-web
go build -o datacat-web
./datacat-web
```

The dashboard will be available at `http://localhost:8080`

### Features

- **Session Browser** - View all sessions with filtering and search
- **Metrics Visualization** - Interactive charts with Chart.js
- **State History** - Track state changes over time
- **Event Timeline** - View events chronologically
- **Crash Detection** - Identify crashed and hung sessions

---

## Using the Python Client

### Installation

```bash
cd python
pip install -e .
```

This installs the `datacat` package in development mode.

### Basic Usage

```python
from datacat import create_session

# Create a session (daemon starts automatically)
session = create_session(
    "http://localhost:9090",
    product="MyApp",
    version="1.0.0"
)

# Update state - supports deep merging
session.update_state({
    "status": "running",
    "user": "alice",
    "window_state": {
        "open": ["main", "settings"]
    }
})

# Log an event
session.log_event("user_login", data={
    "username": "alice",
    "login_method": "oauth"
})

# Log a metric
session.log_metric("response_time_ms", 125.5, tags=["api", "user"])

# End the session
session.end()
```

### Heartbeat Monitoring

Enable automatic hang detection:

```python
# Start heartbeat monitor (60 second timeout)
session.start_heartbeat_monitor(timeout=60)

# In your main loop
while application_running:
    # Send heartbeat to indicate app is alive
    session.heartbeat()
    do_work()

# Stop monitoring when done
session.stop_heartbeat_monitor()
```

If the application stops sending heartbeats, the system will log an `application_appears_hung` event.

### Exception Logging

Automatically capture exceptions with full stack traces:

```python
try:
    risky_operation()
except Exception:
    session.log_exception(extra_data={
        "context": "processing user request",
        "user_id": 123
    })
```

---

## Using the Go Client

### Installation

```bash
go get github.com/OriginalDaemon/datacat/client
```

### Basic Usage with Daemon

```go
package main

import (
    "log"
    "github.com/OriginalDaemon/datacat/client"
)

func main() {
    // Create client with local daemon (recommended)
    c, err := client.NewClientWithDaemon("http://localhost:9090", "8079")
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // Create session
    sessionID, err := c.CreateSession()
    if err != nil {
        log.Fatal(err)
    }

    // Update state
    err = c.UpdateState(sessionID, map[string]interface{}{
        "status": "running",
        "version": "1.0.0",
    })

    // Log event
    err = c.LogEvent(sessionID, "startup", map[string]interface{}{
        "config": "production",
    })

    // Log metric
    err = c.LogMetric(sessionID, "startup_time_ms", 250.0, []string{"prod"})

    // Send heartbeat
    err = c.Heartbeat(sessionID)

    // End session
    err = c.EndSession(sessionID)
}
```

### Direct Server Mode

For simpler use cases without the daemon:

```go
c := client.NewClient("http://localhost:9090")
// Use the same methods as above
```

---

## Testing the REST API

You can interact with the API directly using curl or any HTTP client.

### Create a Session

```bash
curl -X POST http://localhost:9090/api/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "product": "TestApp",
    "version": "1.0.0",
    "hostname": "localhost"
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-01-01T12:00:00Z",
  "active": true,
  "state": {
    "product": "TestApp",
    "version": "1.0.0"
  }
}
```

### Update Session State

```bash
curl -X POST http://localhost:9090/api/sessions/550e8400-e29b-41d4-a716-446655440000/state \
  -H "Content-Type: application/json" \
  -d '{
    "status": "running",
    "progress": 50
  }'
```

### Log an Event

```bash
curl -X POST http://localhost:9090/api/sessions/550e8400-e29b-41d4-a716-446655440000/events \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user_action",
    "level": "info",
    "data": {"action": "click", "button": "submit"}
  }'
```

### Get Session Details

```bash
curl http://localhost:9090/api/sessions/550e8400-e29b-41d4-a716-446655440000
```

### Get All Sessions

```bash
curl http://localhost:9090/api/data/sessions
```

---

## Running Examples

The repository includes several example applications.

### Python Examples

```bash
# Install Python client first
cd python && pip install -e . && cd ..

# Basic usage example
python examples/basic_example.py

# Window state tracking
python examples/window_tracking_example.py

# Heartbeat monitoring
python examples/heartbeat_example.py

# Complete feature demo
python examples/complete_example.py

# Exception logging
python examples/exception_logging_example.py

# Custom logging handler
python examples/logging_handler_example.py
```

### Go Example

```bash
cd examples/go-client-example
go run main.go
```

### Interactive Demo GUI

Try the interactive web demo with Gradio:

```bash
cd examples/demo_gui
pip install -r requirements.txt
python demo_gui.py
```

Open http://127.0.0.1:7860 in your browser for an interactive interface.

---

## Next Steps

Now that you have DataCat running, explore these topics:

- [Architecture Overview](architecture.html) - Understand how DataCat works
- [REST API Reference](../_api/rest-api.html) - Complete API documentation
- [Python Examples](../_examples/python-examples.html) - More Python usage examples
- [Game Logging](../game-logging.html) - Ultra-fast async logging
- [Metric Types](../metric-types.html) - Understanding all metric types

---

## Troubleshooting

### Server Won't Start

**Error: Port already in use**

```bash
# Find process using port 9090 (Linux/Mac)
lsof -i :9090

# Windows PowerShell
Get-NetTCPConnection -LocalPort 9090

# Kill the process (replace <PID> with actual process ID)
kill -9 <PID>  # Linux/Mac
Stop-Process -Id <PID>  # Windows
```

**Error: Database corruption**

```bash
# Remove database and restart (Linux/Mac)
rm -rf ./datacat_data
./datacat-server

# Windows PowerShell
Remove-Item -Recurse -Force .\datacat_data
.\datacat-server.exe
```

### Client Connection Issues

**Cannot connect to server**

1. Verify server is running: `curl http://localhost:9090/api/data/sessions`
2. Check firewall settings
3. Verify correct URL and port

**Daemon won't start**

The daemon binary must be accessible. Check these locations:
- Current directory: `./datacat-daemon`
- Repository root: `../datacat-daemon`
- Built binaries: `./bin/datacat-daemon`

Build the daemon manually if needed:
```bash
cd cmd/datacat-daemon
go build -o ../../datacat-daemon
```

### Getting Help

- Review [GitHub Issues](https://github.com/OriginalDaemon/datacat/issues)
- Check the [Architecture Guide](architecture.html) for system design
- Read the [API documentation](../_api/rest-api.html)
