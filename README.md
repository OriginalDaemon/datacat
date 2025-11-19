# datacat

[![Tests](https://github.com/OriginalDaemon/datacat/workflows/Tests/badge.svg)](https://github.com/OriginalDaemon/datacat/actions)
[![codecov](https://codecov.io/gh/OriginalDaemon/datacat/branch/main/graph/badge.svg)](https://codecov.io/gh/OriginalDaemon/datacat)
[![Go Report Card](https://goreportcard.com/badge/github.com/OriginalDaemon/datacat?v=2)](https://goreportcard.com/report/github.com/OriginalDaemon/datacat)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A complete data logging system with REST API service, client libraries, and web UI for tracking application sessions, state, events, and metrics.

## 📦 Repository Structure

This repository contains multiple independent programs and libraries:

```
datacat/
├── cmd/
│   ├── datacat-server/    # Main REST API service (Go)
│   ├── datacat-daemon/    # Local batching daemon (Go) ⭐ NEW
│   └── datacat-web/       # Web UI dashboard (Go + htmx)
├── client/                # Go client library
├── python/                # Python client library (2.7+ and 3.x)
├── examples/              # Example applications
│   ├── *.py              # Python examples
│   └── go-client-example/ # Go example
├── scripts/               # PowerShell scripts for Windows
├── tests/                 # Integration tests
└── docs/                  # Documentation site (just-the-docs)
```

## 🚀 Quick Start

### Option A: Using Scripts (Windows PowerShell)

```powershell
# Setup environment
.\scripts\setup.ps1

# Start both server and web UI
.\scripts\run-both.ps1

# In another terminal, run an example
.\scripts\run-example.ps1 -Example complete
```

See [scripts/README.md](scripts/README.md) for all available scripts.

### Option B: Manual Setup

#### 1. Start the API Server

```bash
cd cmd/datacat-server
go run main.go
```

Server runs on `http://localhost:8080` with BadgerDB persistence.

#### 2. Start the Web UI (Optional)

```bash
cd cmd/datacat-web
go run main.go
```

Dashboard available at `http://localhost:8081`

#### 3. Use a Client Library

**Python:**
```bash
cd python && pip install -e .
python ../examples/basic_example.py
```

**Go:**
```bash
cd examples/go-client-example
go run main.go
```

## 📚 Components

### [datacat-server](cmd/datacat-server/) - REST API Service

The core service providing session management, state tracking, and data persistence.

- **Technology:** Go with BadgerDB embedded database
- **Features:** Session lifecycle, deep merge state updates, event/metric logging
- **Port:** 8080 (default)
- **[Full Documentation](cmd/datacat-server/README.md)**

### [datacat-daemon](cmd/datacat-daemon/) - Local Batching Daemon ⭐ RECOMMENDED

Intelligent local subprocess that reduces network traffic through batching and smart filtering.

- **Technology:** Go HTTP server subprocess
- **Features:** 
  - **10-100x network reduction** through intelligent batching
  - **Smart state filtering** (only sends changed state)
  - **Parent process monitoring** (detects crashes/abnormal exits)
  - **Hang detection** (monitors heartbeats)
  - **Auto-retry with queueing**
- **Port:** 8079 (default)
- **Architecture:** Application → Daemon (subprocess) → Server (remote)
- **Usage:** Automatically started by client libraries when `use_daemon=True`

### [datacat-web](cmd/datacat-web/) - Web Dashboard

Interactive web UI for browsing sessions and visualizing metrics.

- **Technology:** Go + htmx + Chart.js
- **Features:** Session browser, advanced metrics visualization, filtering
- **Port:** 8081 (default)
- **[Full Documentation](cmd/datacat-web/README.md)**

### [Go Client Library](client/)

Type-safe Go client for the datacat API.

- **Coverage:** >85%
- **Features:** Full API support, timeout handling
- **[Full Documentation](client/README.md)**

### [Python Client Library](python/)

Python 2.7+ and 3.x compatible client with advanced features.

- **Features:** Session management, exception logging, heartbeat monitoring
- **Special:** Automatic hang detection with background thread
- **[Full Documentation](python/README.md)**

## 💡 Usage Examples

### Python Client

**With Local Daemon (Recommended):**

```python
from datacat import create_session

# Create session with local daemon (automatic batching and crash detection)
session = create_session("http://localhost:8080", use_daemon=True)

# Nested state updates with deep merge
session.update_state({
    "window_state": {"open": ["w1", "w2"], "active": "w1"}
})

# Log events and metrics (batched by daemon)
session.log_event("user_action", {"action": "click"})
session.log_metric("memory_usage", 1024.5)

# Exception logging with traceback
try:
    risky_operation()
except Exception:
    session.log_exception(extra_data={"context": "user_action"})

# Heartbeat monitoring - daemon detects hangs and crashes
session.start_heartbeat_monitor(timeout=60)
while running:
    session.heartbeat()
    do_work()

session.end()
```

**Direct to Server (No Daemon):**

```python
# For simple use cases without batching/monitoring
session = create_session("http://localhost:8080", use_daemon=False)
# ... same API as above
```

### Go Client

**With Local Daemon (Recommended):**

```go
import "github.com/OriginalDaemon/datacat/client"

// Create client with local daemon
c, err := client.NewClientWithDaemon("http://localhost:8080", "8079")
if err != nil {
    log.Fatal(err)
}
defer c.Close()

sessionID, err := c.CreateSession()
if err != nil {
    log.Fatal(err)
}
c.UpdateState(sessionID, map[string]interface{}{"status": "running"})

// Send heartbeats - daemon detects hangs and crashes
c.Heartbeat(sessionID)

c.EndSession(sessionID)
```

**Direct to Server (No Daemon):**

```go
import "github.com/OriginalDaemon/datacat/client"

c := client.NewClient("http://localhost:8080")
sessionID, err := c.CreateSession()
if err != nil {
    log.Fatal(err)
}

c.UpdateState(sessionID, map[string]interface{}{"status": "running"})
c.LogEvent(sessionID, "user_login", map[string]interface{}{"user": "alice"})
c.LogMetric(sessionID, "cpu_usage", 45.2, []string{"app:myapp"})

c.EndSession(sessionID)
```

## 🔌 API Endpoints

The REST API provides the following endpoints:

- `POST /api/sessions` - Create new session
- `GET /api/sessions/{id}` - Get session details  
- `POST /api/sessions/{id}/state` - Update state (deep merge)
- `POST /api/sessions/{id}/events` - Log event
- `POST /api/sessions/{id}/metrics` - Log metric
- `POST /api/sessions/{id}/end` - End session
- `GET /api/grafana/sessions` - Export all sessions (Grafana)

## ✨ Key Features

- **🔄 Session Lifecycle Management** - Track application sessions from start to end
- **📊 Deep Merge State Updates** - Update nested state without losing other data
- **⚡ BadgerDB Persistence** - Fast embedded database, data survives restarts
- **🐍 Python 2/3 Compatible** - Works with Python 2.7+ and 3.x
- **💓 Heartbeat Monitoring** - Auto-detect hung applications with background thread
- **📈 Advanced Metrics Visualization** - Filter, aggregate, and chart metrics
- **🔍 State History Queries** - Find sessions that ever had specific state values
- **🚨 Exception Logging** - Capture full tracebacks with context
- **✅ Production Ready** - 85%+ test coverage, security scanned

## 🎯 Use Cases

**Application Monitoring**
```python
session.update_state({"app": "myapp", "status": "starting"})
session.log_metric("requests_per_second", 1000)
```

**Testing & CI/CD**
```python
session.update_state({"test_suite": "integration"})
for test in tests:
    session.log_event("test_completed", {"name": test.name, "result": result})
```

**Window/UI Tracking**
```python
session.update_state({
    "window_state": {
        "open": ["window 1", "window 2", "space probe"],
        "active": "space probe"
    }
})
```

**Crash Detection**
```python
# Heartbeat monitor auto-logs "application_appears_hung" if no heartbeat for 60s
session.start_heartbeat_monitor(timeout=60)
```

## 📖 Documentation

- **[Full Documentation Site](https://OriginalDaemon.github.io/datacat/)** - Complete guides and API reference
- **[Quick Start Guide](QUICKSTART.md)** - Get up and running quickly
- **[Architecture](ARCHITECTURE.md)** - System design and components
- **[Branch Protection Rules](.github/BRANCH_PROTECTION.md)** - PR requirements

## Contributing

We welcome contributions! Please ensure your PR meets the following requirements:

### Code Quality Requirements
- ✅ **Python code** must pass Black formatting (`black --check`)
- ✅ **Python code** must pass mypy type checking
- ✅ **Go code** must build successfully
- ✅ **All tests** must pass
- ✅ **Code coverage** must be at least 85%

### Before Submitting a PR

Run the following commands locally:

```bash
# Format Python code
black python/ examples/ tests/

# Type check Python code
mypy python/ --ignore-missing-imports

# Test Go code with coverage
go test -v -coverprofile=coverage.out ./...

# Test Python code with coverage
pytest tests/ -v --cov=python --cov-report=term
```

### Branch Protection

The `main` branch is protected with the following requirements:
- All status checks must pass (linting, formatting, tests)
- Code coverage must be at least 85%
- At least one approval required

See [Branch Protection Rules](.github/BRANCH_PROTECTION.md) for detailed information.

## License

MIT License - see LICENSE file for details.
