---
layout: home
title: Home
nav_order: 1
description: "DataCat is a complete data logging system with REST API service, client libraries, and web UI for tracking application sessions, state, events, and metrics."
permalink: /
---

# DataCat Documentation

{: .fs-9 }

A complete data logging system with REST API service, client libraries, and web UI for tracking application sessions, state, events, and metrics.
{: .fs-6 .fw-300 }

[Get Started](#quick-start){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/OriginalDaemon/datacat){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## Features

{: .text-delta }

### 🔄 Session Lifecycle Management
Track application sessions from start to end with automatic crash and hang detection.

### 📊 Deep Merge State Updates
Update nested state without losing other data - perfect for complex application state.

### ⚡ Intelligent Batching Daemon
Reduce network traffic by 10-100x through smart batching and filtering.

### 🐍 Python 2/3 Compatible
Works with Python 2.7+ and 3.x for maximum compatibility.

### 💓 Heartbeat Monitoring
Auto-detect hung applications with background thread and configurable timeouts.

### 📈 Advanced Metrics Visualization
Filter, aggregate, and chart metrics with the built-in web dashboard.

### 🔍 State History Queries
Find sessions that ever had specific state values for debugging and analytics.

### 🚨 Exception Logging
Capture full tracebacks with context for comprehensive error tracking.

---

## Quick Start

### Prerequisites

- **Go 1.21+** for server and daemon
- **Python 2.7+ or 3.x** for Python client
- **Modern web browser** for dashboard

### Installation

#### 1. Start the API Server

```bash
cd cmd/datacat-server
go run main.go
```

Server runs on `http://localhost:9090` with BadgerDB persistence.

#### 2. Start the Web UI (Optional)

```bash
cd cmd/datacat-web
go run main.go
```

Dashboard available at `http://localhost:8080`

#### 3. Install Python Client

```bash
cd python
pip install -e .
```

### First Steps

**Python Client:**

```python
from datacat import create_session

# Create session (automatically uses local daemon)
session = create_session(
    "http://localhost:9090",
    product="MyApp",
    version="1.0.0"
)

# Update state with deep merge
session.update_state({
    "status": "running",
    "user": "alice"
})

# Log events and metrics
session.log_event("user_action", data={"action": "click"})
session.log_metric("memory_usage", 1024.5)

# Exception logging
try:
    risky_operation()
except Exception:
    session.log_exception()

# End session
session.end()
```

**Go Client:**

```go
import "github.com/OriginalDaemon/datacat/client"

// Create client with daemon
c, err := client.NewClientWithDaemon("http://localhost:9090", "8079")
if err != nil {
    log.Fatal(err)
}
defer c.Close()

sessionID, _ := c.CreateSession()
c.UpdateState(sessionID, map[string]interface{}{"status": "running"})
c.LogEvent(sessionID, "user_action", map[string]interface{}{"action": "click"})
c.EndSession(sessionID)
```

---

## Architecture

DataCat uses a multi-tier architecture for optimal performance:

```
Application → Local Daemon → DataCat Server → BadgerDB
                ↓
           Web Dashboard
```

### Components

- **datacat-server**: REST API service with embedded BadgerDB
- **datacat-daemon**: Local batching daemon (10-100x network reduction)
- **datacat-web**: Interactive web dashboard
- **Client Libraries**: Python and Go implementations

See the [Architecture Guide](guides/architecture.html) for detailed information.

---

## Documentation

### Guides
- [Quick Start Guide](guides/quickstart.html)
- [Architecture Overview](guides/architecture.html)
- [Python Client Guide](guides/python-client.html)
- [Go Client Guide](guides/go-client.html)
- [Web Dashboard Guide](guides/web-dashboard.html)
- [Deployment Guide](guides/deployment.html)

### API Reference
- [REST API Reference](api/rest-api.html)
- [Python Client API](api/python-api.html)
- [Go Client API](api/go-api.html)

### Advanced Topics
- [Daemon Batching](guides/daemon-batching.html)
- [Crash Detection](guides/crash-detection.html)
- [State Management](guides/state-management.html)
- [Troubleshooting](guides/troubleshooting.html)

---

## Contributing

We welcome contributions! Please see our [Contributing Guide](guides/contributing.html) for details.

### Requirements

- Python code must pass Black formatting and mypy type checking
- Go code must build successfully
- All tests must pass with 85%+ coverage
- Branch protection enforced on `main`

---

## License

DataCat is distributed under the [MIT License](https://opensource.org/licenses/MIT).
