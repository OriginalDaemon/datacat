# DataCat

**Session-based application monitoring with automatic crash and hang detection.**

DataCat is a lightweight, session-based monitoring system that tracks application state, events, and metrics. Perfect for development, testing, and debugging - it helps you understand what your application was doing before it crashed or hung.

## Key Features

- 🔍 **Session-Based Tracking** - Every run creates a unique session with complete history
- 💥 **Automatic Crash Detection** - Knows when your app crashed vs clean exit
- ⏸️ **Hang Detection** - Detects when your app stops responding
- 📊 **Real-Time Metrics** - Gauges, Counters, Histograms, and Timers
- 🌐 **Web UI** - Beautiful interface for browsing sessions and viewing timelines
- ⚡ **Ultra-Fast Logging** - <0.01ms overhead with async mode (perfect for games)
- 🔌 **Offline Support** - Works even when server is down
- 🐍 **Python 2.7.4+** - Compatible with Python 2 and 3
- 🔧 **Multiple Languages** - Python and Go clients (more coming)

## Quick Start

### 1. Start the Services

```bash
# Windows
.\scripts\run-server.ps1   # Start server on :9090
.\scripts\run-web.ps1      # Start web UI on :8080

# Linux/Mac
cd cmd/datacat-server && go run main.go config.go
cd cmd/datacat-web && go run main.go
```

### 2. Install Python Client

```bash
cd python
pip install -e .
```

### 3. Use in Your App

```python
from datacat import create_session

# Create session (daemon starts automatically)
session = create_session(
    product="MyApp",
    version="1.0.0",
    async_mode=True  # Non-blocking for games/real-time apps
)

# Track state
session.update_state({
    "level": 1,
    "player": {"health": 100, "position": {"x": 0, "y": 0}}
})

# Log events
session.log_event("player_moved", {"x": 10, "y": 20})

# Log metrics
session.log_gauge("fps", 60.0, unit="fps")
session.log_counter("enemies_defeated")
session.log_histogram("frame_time", 0.016, unit="seconds")

# Automatic crash detection - daemon watches parent process
# If your app crashes, DataCat knows!

# Clean exit
session.end()
```

### 4. View in Web UI

Open http://localhost:8080 to see:

- All sessions with status (active, ended, crashed, hung)
- Complete timeline of events
- Metrics with charts
- State history
- Exception details with stack traces

## Documentation

### Getting Started

- **[Quick Start Guide](docs/_guides/quickstart.md)** - Get up and running in minutes
- **[Architecture](docs/_guides/architecture.md)** - Understand how it works
- **[Examples](docs/_examples/)** - Python, Go, and Demo GUI examples

### Features

- **[Game Logging](docs/game-logging.md)** - Ultra-fast async logging for 60+ FPS
- **[Metric Types](docs/metric-types.md)** - Gauges, Counters, Histograms, Timers
- **[Hung Session Tracking](docs/_guides/hung-tracking.md)** - Detect when apps freeze
- **[Machine Tracking](docs/_guides/machine-tracking.md)** - Track which machine ran what
- **[Process Isolation](docs/process-isolation.md)** - One daemon per app

### API Reference

- **[REST API](docs/_api/rest-api.md)** - Complete API documentation
- **[Sessions API](docs/_api/sessions.md)** - Session management
- **[Events API](docs/_api/events.md)** - Event logging
- **[Metrics API](docs/_api/metrics.md)** - Metrics logging
- **[State API](docs/_api/state.md)** - State management

### Examples

- **[Python Examples](docs/_examples/python-examples.md)** - Complete Python guide
- **[Demo GUI](docs/_examples/demo-gui.md)** - Interactive web demo
- **[Go Examples](docs/_examples/go-examples.md)** - Go client usage

## Use Cases

### Game Development

- Track FPS, frame times, and player actions
- Async logging with <0.01ms overhead
- Perfect for 60+ FPS games
- See: [Game Logging Guide](docs/game-logging.md)

### Testing & QA

- Track test execution
- Identify flaky tests
- Compare test runs
- Automatic crash detection

### Development

- Debug hard-to-reproduce issues
- Track application state before crashes
- Monitor long-running operations
- Detect hangs automatically

### Production Monitoring

- Track user sessions
- Monitor application health
- Collect metrics and events
- Offline operation support

## Architecture

```
Application → Local Daemon → DataCat Server → BadgerDB
                ↓                    ↓
         Crash Detection    State Management
         Hang Detection     Data Persistence
         Offline Queue      Web UI (Port 8080)
```

**Process Isolation**: Each application gets its own daemon - no shared state, no port conflicts.

**Crash Detection**: Daemon monitors parent process - knows the difference between crashes and clean exits.

**Hang Detection**: Automatic heartbeat monitoring - detects when applications stop responding.

**Offline Support**: Daemon queues data when server is unavailable - no data loss.

## Performance

### Async Logging (Python)

| Operation          | Time    | Frame Budget @ 60 FPS |
| ------------------ | ------- | --------------------- |
| `log_event()`      | 0.008ms | 0.05%                 |
| `log_metric()`     | 0.008ms | 0.05%                 |
| `update_state()`   | 0.008ms | 0.05%                 |
| **100 logs/frame** | 0.8ms   | **5%** ✅             |

**Perfect for real-time applications!** See [Game Logging Guide](docs/game-logging.md)

## Requirements

### Server & Daemon

- Go 1.21+
- BadgerDB (included)
- ~50MB RAM per session

### Python Client

- Python 2.7.4+ or Python 3.x
- Zero external dependencies for core features
- Gradio for demo GUI (optional)

### Go Client

- Go 1.21+

## Components

- **`cmd/datacat-server`** - Main server with REST API
- **`cmd/datacat-daemon`** - Per-application monitoring daemon
- **`cmd/datacat-web`** - Web UI for browsing sessions
- **`python/`** - Python client library
- **`client/`** - Go client library
- **`examples/`** - Example applications
- **`scripts/`** - Utility scripts

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

See [SECURITY.md](SECURITY.md) for security policy and vulnerability reporting.

## License

[Add your license here]

## Support

- 📖 [Documentation](docs/)
- 💬 [Issues](https://github.com/OriginalDaemon/datacat/issues)
- 📧 [Contact](mailto:your-email@example.com)

---

**Made with ❤️ for developers who want to understand what their apps are actually doing**
