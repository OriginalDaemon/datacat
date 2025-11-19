# Quick Start Guide

## Start the Service

```bash
# Build
go build -o datacat

# Run
./datacat
```

The service will start on `http://localhost:9090`

## Quick Python Example

```python
from datacat import create_session

# Create session
session = create_session()

# Track nested state
session.update_state({
    "window_state": {
        "open": ["window 1", "window 2"]
    }
})

# Log events
session.log_event("user_action", {"action": "click"})

# Log metrics
session.log_metric("response_time", 0.5)

# Enable hang detection
session.start_heartbeat_monitor()

# Send heartbeats regularly
while running:
    session.heartbeat()
    do_work()

# End session
session.end()
```

## Test the API

```bash
# Create session
curl -X POST http://localhost:9090/api/sessions

# Update state
curl -X POST http://localhost:9090/api/sessions/{id}/state \
  -H "Content-Type: application/json" \
  -d '{"status":"running"}'

# Get session
curl http://localhost:9090/api/sessions/{id}

# Get all sessions (Grafana)
curl http://localhost:9090/api/data/sessions
```

## Run Examples

```bash
# Basic usage
python3 examples/basic_example.py

# Window tracking
python3 examples/window_tracking_example.py

# Heartbeat monitoring
python3 examples/heartbeat_example.py

# Complete demo
python3 examples/complete_example.py
```
