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

Server runs on `http://localhost:9090` with BadgerDB persistence.

#### 2. Start the Web UI (Optional)

```bash
cd cmd/datacat-web
go run main.go
```

Dashboard available at `http://localhost:8080`

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
- **Port:** 8080 (default)
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

### 🎨 Try the Interactive Demo

Want to try datacat before diving into code? Check out the **Demo GUI**!

```bash
# Install requirements
cd examples/demo_gui
pip install -r requirements.txt
pip install -e ../../python

# Run the demo
python demo_gui.py
```

Or use the PowerShell script (Windows):
```powershell
.\scripts\run-demo-gui.ps1
```

Opens a modern web interface at http://127.0.0.1:7860 with:
- 🌙 Dark mode by default (Gradio's native theme)
- 📝 State updates with JSON editor
- 📢 Event logging
- 📈 Metrics tracking
- ⚠️ Custom logging handler for errors
- 💥 Exception generation with full stack traces

**[Full Demo Documentation](examples/demo_gui/README.md)** | **[Quick Start](examples/demo_gui/QUICKSTART.md)**

---

### Python Client

**With Local Daemon (Recommended):**

```python
from datacat import create_session

# Create session with local daemon (automatic batching and crash detection)
session = create_session("http://localhost:9090", use_daemon=True)

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
session = create_session("http://localhost:9090", use_daemon=False)
# ... same API as above
```

### Go Client

**With Local Daemon (Recommended):**

```go
import "github.com/OriginalDaemon/datacat/client"

// Create client with local daemon
c, err := client.NewClientWithDaemon("http://localhost:9090", "8079")
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

c := client.NewClient("http://localhost:9090")
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
- `GET /api/data/sessions` - Export all sessions

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

## ❓ FAQ

### Where is my data stored?

All session data is stored in a **BadgerDB database** on the server. The location is configurable in `config.json`:

- **Default location:** `./datacat_data` (relative to where the server binary is executed)
- **Configurable via:** `data_path` setting in `config.json`

**Important - Data Location Depends on How You Run the Server:**

- **When using PowerShell scripts** (`.\scripts\run-server.ps1` or `.\scripts\run-both.ps1`):
  - Data is stored in the **repository root directory**: `./datacat_data`
  - Config file is in the **repository root**: `./config.json`
  - (Scripts explicitly set the working directory to ensure consistent location)

- **When running from cmd/datacat-server** (`cd cmd/datacat-server && go run main.go`):
  - Data is stored in **cmd/datacat-server/datacat_data**
  - Config file is in **cmd/datacat-server/config.json**

- **When running the binary directly** without the scripts:
  - Data is stored **relative to your current working directory** when you run the binary
  - Check the server startup logs to see the exact path being used

**To find your data directory:**
- Check the server startup logs - they show the data path:
  ```
  Configuration loaded: Data path=./datacat_data, Retention=365 days, Port=9090
  ```
- Look in the directory where you ran the server from
- Search your system for `datacat_data` directory

The data directory contains BadgerDB files including MANIFEST, LOCK, .vlog, and .mem files.

### How do I delete all data from the server?

To completely reset the server and delete all session data:

**Step 1: Stop the server**

Stop the running server (Ctrl+C in the terminal/PowerShell window)

**Step 2: Locate and delete the data**

**If you used PowerShell scripts** (`.\scripts\run-server.ps1` or `.\scripts\run-both.ps1`):
```powershell
# Data is in repository root
Remove-Item -Recurse -Force ./datacat_data
Remove-Item -Force ./config.json  # Optional - removes custom config
```

**If you ran manually from cmd/datacat-server**:
```bash
cd cmd/datacat-server
rm -rf ./datacat_data
rm config.json  # Optional
```

**If you're unsure where the data is:**
1. Check the server logs when it started - they show the data path
2. Search for `datacat_data` directory in your repository
3. Use the clean script: `.\scripts\clean.ps1` (cleans repository directory)

**Step 3: Restart the server**

The server will create a fresh database on startup.

**Important:** Always stop the server before deleting the data directory to prevent corruption.

See [cmd/datacat-server/README.md](cmd/datacat-server/README.md#data-management) for more details on data management.

### How long is data retained?

By default, session data is retained for **365 days** (1 year). You can configure this in `config.json`:

```json
{
  "retention_days": 365,
  "cleanup_interval_hours": 24
}
```

The cleanup routine runs automatically every 24 hours by default.

### Can I backup my data?

Yes! To backup your data:

```bash
# Stop the server first
cp -r ./datacat_data ./datacat_data_backup
```

To restore, stop the server and copy the backup back to the original location.

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
