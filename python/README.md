# DataCat Python Client

Python client library for interacting with the DataCat REST API.

**Features:**

- ⚡ **Async logging** for games and real-time apps (< 0.01ms per call)
- 🐍 **Python 2.7.4+ and 3.x** compatible
- 🚀 Zero external dependencies (uses standard library only)
- 📦 Local daemon for batching and crash detection

## Installation

```bash
cd python
pip install -e .
```

Or directly:

```bash
pip install -e git+https://github.com/OriginalDaemon/datacat.git#egg=datacat&subdirectory=python
```

## Usage

### Basic Usage (Blocking Mode)

```python
from datacat import create_session

# Create session
session = create_session(
    "http://localhost:9090",
    product="MyApp",
    version="1.0.0"
)
print("Session ID:", session.session_id)

# Update state
session.update_state({
    "status": "running",
    "version": "1.0.0"
})

# Nested state updates merge intelligently
session.update_state({
    "window_state": {"open": ["w1", "w2"], "active": "w1"}
})
session.update_state({
    "window_state": {"open": ["w1", "w2", "w3"]}  # preserves "active": "w1"
})

# Log event
session.log_event("user_action", data={"action": "click", "button": "submit"})

# Log metric
session.log_metric("memory_usage", 1024.5, tags=["app:myapp"])

# End session
session.end()
```

### ⚡ Async Mode (Non-Blocking - for Games & Real-Time Apps)

```python
from datacat import create_session

# Create async session - all calls return in < 0.01ms!
session = create_session(
    "http://localhost:9090",
    product="MyGame",
    version="1.0.0",
    async_mode=True,      # Enable non-blocking logging
    queue_size=10000      # Buffer up to 10K events
)

# In your game loop (60 FPS) - returns immediately!
for frame in range(1000):
    session.log_event("frame_update", data={"frame": frame})
    session.log_metric("fps", 60.0)
    session.update_state({"frame": frame, "health": 100})

    # Your game logic here...
    render_frame()

# Graceful shutdown - flushes remaining logs
session.shutdown()
```

**🎮 Game Developers**: See [docs/GAME_LOGGING.md](../docs/GAME_LOGGING.md) for complete guide.

**📊 Performance**: Async logging adds < 0.1ms overhead per frame at 60 FPS.

### Exception Logging

```python
from datacat import create_session

session = create_session("http://localhost:9090")

try:
    risky_operation()
except Exception:
    # Logs exception with full traceback
    session.log_exception(extra_data={"context": "user_action"})
```

### Heartbeat Monitoring

```python
from datacat import create_session
import time

session = create_session("http://localhost:9090")

# Start heartbeat monitor with 60s timeout
session.start_heartbeat_monitor(timeout=60)

# Main application loop
while running:
    session.heartbeat()  # Call this regularly
    do_work()
    time.sleep(5)

# If heartbeat not called for 60s, auto-logs "application_appears_hung" event
# When heartbeats resume, logs "application_recovered" event

session.end()
```

## API

### `create_session(base_url, daemon_port="auto", product=None, version=None, async_mode=False, queue_size=10000)`

Creates a new session and returns a Session or AsyncSession object.

**Arguments:**

- `base_url` (str): DataCat server URL
- `daemon_port` (str): Daemon port ("auto" finds available port)
- `product` (str): Product name (required)
- `version` (str): Product version (required)
- `async_mode` (bool): Enable non-blocking async logging (default: False)
- `queue_size` (int): Queue size for async mode (default: 10000)

**Returns:** `Session` or `AsyncSession` (if async_mode=True)

### `Session` Class

#### Methods

- `update_state(state: dict) -> None` - Update session state (deep merge)
- `log_event(name: str, level=None, category=None, labels=None, message=None, data=None) -> None` - Log an event
- `log_metric(name: str, value: float, tags: list = None) -> None` - Log a metric
- `log_exception(exc_info=None, extra_data: dict = None) -> None` - Log exception with traceback
- `start_heartbeat_monitor(timeout: int = 60, check_interval: int = 5) -> HeartbeatMonitor` - Start heartbeat monitoring thread
- `heartbeat() -> None` - Send heartbeat signal
- `pause_heartbeat_monitoring() -> None` - Pause heartbeat monitoring
- `resume_heartbeat_monitoring() -> None` - Resume heartbeat monitoring
- `stop_heartbeat_monitor() -> None` - Stop heartbeat monitoring
- `end() -> dict` - End the session

#### Properties

- `session_id` (str) - The session ID
- `client` (DatacatClient) - The underlying client

### `AsyncSession` Class

Non-blocking wrapper around `Session` for real-time applications.

#### Methods

All methods are **non-blocking** and return immediately (< 0.01ms):

- `log_event(name, level=None, category=None, labels=None, message=None, data=None)` - Log event (non-blocking)
- `log_metric(name, value, tags=None)` - Log metric (non-blocking)
- `update_state(state)` - Update state (non-blocking)
- `log_exception(exc_info=None, extra_data=None)` - Log exception (non-blocking)
- `heartbeat()` - Send heartbeat (non-blocking)
- `get_stats() -> dict` - Get logging statistics (sent, dropped, queued)
- `flush(timeout=2.0)` - Wait for queue to drain (blocking)
- `shutdown(timeout=2.0)` - Graceful shutdown (flushes + ends session)
- `end()` - Alias for shutdown()

#### Properties

- `session_id` (str) - The session ID
- `client` (DatacatClient) - The underlying client

## Client API

For lower-level access, use the `DatacatClient` class:

```python
from datacat import DatacatClient

client = DatacatClient("http://localhost:9090")

# Create session
session_id = client.create_session()

# Update state
client.update_state(session_id, {"status": "running"})

# Log event
client.log_event(session_id, "user_login", {"user": "alice"})

# Log metric
client.log_metric(session_id, "cpu_usage", 45.2, tags=["app:myapp"])

# Get session
session_data = client.get_session(session_id)

# End session
client.end_session(session_id)
```

## Testing

```bash
pytest tests/ -v --cov=python --cov-report=term
```
