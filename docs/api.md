---
layout: default
title: API Reference
nav_order: 3
has_children: true
permalink: /api
---

# API Reference

Complete API documentation for DataCat REST API and client libraries.
{: .fs-6 .fw-300 }

---

## REST API Endpoints

The DataCat REST API provides HTTP endpoints for all operations.

### Core Endpoints

- **[Sessions](api/sessions.html)** - Create, retrieve, and manage sessions
  - POST /api/sessions - Create session
  - GET /api/sessions/{id} - Get session details
  - POST /api/sessions/{id}/end - End session
  - POST /api/sessions/{id}/heartbeat - Send heartbeat
  - GET /api/data/sessions - Get all sessions

- **[State Updates](api/state.html)** - Deep merge state management
  - POST /api/sessions/{id}/state - Update session state

- **[Events](api/events.html)** - Log events and exceptions
  - POST /api/sessions/{id}/events - Log event

- **[Metrics](api/metrics.html)** - Record numeric measurements
  - POST /api/sessions/{id}/metrics - Log metric

- **[Error Handling](api/errors.html)** - Error codes and troubleshooting
  - Common errors and solutions
  - HTTP status codes
  - Debugging tips

---

## Client Libraries

Language-specific API references.

- [Python API](python-api.html) - Python client library reference
- [Go API](go-api.html) - Go client library reference

---

## Quick Examples

### Create Session and Log Data

**Python:**
```python
from datacat import create_session

session = create_session(
    "http://localhost:9090",
    product="MyApp",
    version="1.0.0"
)

session.update_state({"status": "running"})
session.log_event("startup", data={"config": "prod"})
session.log_metric("memory_mb", 512.5)
session.end()
```

**curl:**
```bash
# Create session
SESSION_ID=$(curl -s -X POST http://localhost:9090/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"product":"MyApp","version":"1.0.0"}' | jq -r '.id')

# Update state
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/state \
  -H "Content-Type: application/json" \
  -d '{"status":"running"}'

# Log event
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/events \
  -H "Content-Type: application/json" \
  -d '{"name":"startup","data":{"config":"prod"}}'

# Log metric
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/metrics \
  -H "Content-Type: application/json" \
  -d '{"name":"memory_mb","value":512.5}'

# End session
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/end
```

---

## API Versioning

The current API is version 1. The base path is `/api/`.

Future versions will use `/api/v2/` etc. to maintain backwards compatibility.

---

## Base URL

```
http://localhost:9090/api
```

All endpoints are prefixed with `/api`. The server runs on port 9090 by default.

---

## Authentication

Currently, DataCat does not require authentication. Ensure the server is deployed in a trusted network environment.

---

## Best Practices

### Use the Daemon

For optimal performance, use the local daemon instead of calling the API directly:
- 10-100x reduction in network traffic
- Automatic batching and filtering
- Built-in crash and hang detection

See the [Architecture Guide](guides/architecture.html) for more details.

### Heartbeat Frequency

Send heartbeats based on your application's characteristics:
- **Interactive apps:** Every 5-10 seconds
- **Batch processing:** Every 30-60 seconds
- **Long-running jobs:** Every 1-2 minutes

Configure timeout to 3-5x the heartbeat interval.

### State Updates

- Use deep merge to update nested state
- Avoid sending entire state on every update
- Use null to delete fields you no longer need

### Events vs Metrics

- **Events:** Discrete occurrences (user actions, errors, state changes)
- **Metrics:** Numeric measurements (performance, resource usage)

---

## See Also

- [Quick Start Guide](guides/quickstart.html) - Get started quickly
- [Architecture Guide](guides/architecture.html) - System architecture
- [Python Client Guide](guides/python-client.html) - Using the Python client
- [Go Client Guide](guides/go-client.html) - Using the Go client
