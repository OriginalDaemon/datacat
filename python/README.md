# datacat Python Client

Python client library for interacting with the datacat REST API. Compatible with Python 2.7+ and Python 3.x.

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

### Basic Usage

```python
from datacat import create_session

# Create session
session = create_session("http://localhost:8080")
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
session.log_event("user_action", {"action": "click", "button": "submit"})

# Log metric
session.log_metric("memory_usage", 1024.5, tags=["app:myapp"])

# End session
session.end()
```

### Exception Logging

```python
from datacat import create_session

session = create_session("http://localhost:8080")

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

session = create_session("http://localhost:8080")

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

### `create_session(base_url: str) -> Session`

Creates a new session and returns a Session object.

### `Session` Class

#### Methods

- `update_state(state: dict) -> None` - Update session state (deep merge)
- `log_event(name: str, data: dict = None) -> None` - Log an event
- `log_metric(name: str, value: float, tags: list = None) -> None` - Log a metric
- `log_exception(extra_data: dict = None) -> None` - Log current exception with traceback
- `start_heartbeat_monitor(timeout: int = 60, check_interval: int = 5) -> None` - Start heartbeat monitoring thread
- `heartbeat() -> None` - Send heartbeat signal
- `stop_heartbeat_monitor() -> None` - Stop heartbeat monitoring
- `end() -> None` - End the session

#### Properties

- `session_id` - The session ID (string)

## Client API

For lower-level access, use the `DatacatClient` class:

```python
from datacat import DatacatClient

client = DatacatClient("http://localhost:8080")

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
