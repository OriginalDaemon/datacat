---
layout: default
title: Python Examples
parent: Examples
nav_order: 1
---

# Python Examples
{: .no_toc }

Comprehensive guide to all Python examples demonstrating DataCat features.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Getting Started

All Python examples are in the `examples/` directory. Make sure the DataCat server is running before running examples:

```bash
.\scripts\run-server.ps1
```

---

## Core Examples

### Basic Example

**File**: `examples/basic_example.py`

The simplest possible DataCat example - create a session, log some data, and end the session.

```python
from datacat import create_session

# Create session
session = create_session()

# Log state
session.update_state({"status": "running"})

# Log event
session.log_event("app_started")

# Log metric
session.log_metric("cpu_percent", 45.2)

# End session
session.end()
```

**Run**:
```bash
python examples/basic_example.py
```

---

### Complete Example

**File**: `examples/complete_example.py`

Demonstrates all major DataCat features in one comprehensive example.

**Features**:
- Session lifecycle management
- Nested state tracking
- Event logging with levels
- Metric logging with tags
- Heartbeat monitoring
- Exception logging

**Run**:
```bash
python examples/complete_example.py
```

---

## Game & Real-Time Application Examples

### Interactive Game Demo

**File**: `examples/example_game.py`

A complete simulated game that demonstrates DataCat in a realistic scenario.

**Features**:
- Main update/render loop running at 60 FPS
- Real-time metrics (FPS, memory, player stats)
- Random gameplay events (enemies, powerups, achievements)
- Random errors and exceptions
- Different modes: normal, hang, crash

**Usage**:
```bash
# Run normally for 60 seconds
python examples/example_game.py --duration 60

# Run with hang behavior
python examples/example_game.py --mode hang --duration 30

# Run with crash behavior
python examples/example_game.py --mode crash --duration 20

# Or use PowerShell script
.\scripts\run-example-game.ps1 -Mode normal -Duration 60
```

**What You'll See**:
- Real-time game session in the web UI
- Live metrics updating every second
- Crash detection for crashed games
- Hang detection for frozen games
- Complete event timeline

---

### Game Swarm (Multi-Instance)

**File**: `examples/run_game_swarm.py`

Launch multiple concurrent game sessions to demonstrate DataCat handling many simultaneous sessions.

**Usage**:
```bash
# Launch 10 concurrent games
python examples/run_game_swarm.py --count 10 --duration 60

# Launch 20 games with custom hang/crash rates
python examples/run_game_swarm.py --count 20 --hang-rate 0.2 --crash-rate 0.1

# Or use PowerShell script
.\scripts\run-game-swarm.ps1 -Count 10 -Duration 60
```

**What You'll See**:
- Multiple game sessions running simultaneously
- Real-time session list in web UI
- Different session statuses (active, hung, crashed)
- Server handling high throughput

---

### Game Logging (Async Performance Test)

**File**: `examples/game_logging_example.py`

Demonstrates ultra-fast async logging suitable for game engines and real-time applications.

**Features**:
- Non-blocking logging (< 0.01ms per call)
- 300 frames at 60 FPS simulation
- Performance statistics
- Async vs blocking comparison

**Usage**:
```bash
python examples/game_logging_example.py
```

**Performance Results**:
```
Total frames: 300
Average FPS: 58.6
Logging overhead per frame: 0.0440 ms (0.26% of frame budget)
Events sent: 1585
Events dropped: 0
```

**See Also**: [Game Logging Guide](../GAME_LOGGING.md)

---

## Metric Examples

### Metric Types Demo

**File**: `examples/metric_types_example.py`

Demonstrates all four metric types: Gauges, Counters, Histograms, and Timers.

**Metric Types**:
- **Gauges**: Point-in-time values (CPU%, memory, temperature)
- **Counters**: Cumulative counts (requests, errors, bytes sent)
- **Histograms**: Value distributions (latencies, durations)
- **Timers**: Duration measurements with context manager

**Usage**:
```bash
python examples/metric_types_example.py
```

**Example Code**:
```python
# Gauge - point-in-time value
session.log_gauge("cpu_percent", 45.2, unit="percent")

# Counter - cumulative count (daemon aggregates)
session.log_counter("http_requests")  # Increment by 1
session.log_counter("bytes_sent", delta=1024)  # Increment by amount

# Histogram - value distribution
session.log_histogram("request_latency", 0.045, unit="seconds")

# Timer - context manager
with session.timer("database_query", unit="seconds"):
    result = db.execute(query)
```

**See Also**: [Metric Types Guide](../METRIC_TYPES.md)

---

### Incremental Counters

**File**: `examples/incremental_counters_example.py`

Deep-dive into counter usage with daemon-side aggregation.

**Features**:
- Simple event counting
- Byte counting with deltas
- Concurrent operations (multi-threaded)
- Cache statistics
- Counter vs Gauge comparison

**Usage**:
```bash
python examples/incremental_counters_example.py
```

**See Also**: [Incremental Counters Guide](../INCREMENTAL_COUNTERS.md)

---

### FPS Histogram

**File**: `examples/fps_histogram_example.py`

Demonstrates histograms with custom FPS-aligned buckets.

**Features**:
- Custom FPS buckets (60fps, 30fps, 20fps, 10fps)
- Multiple performance scenarios
- Render phase breakdown
- Default vs custom buckets comparison

**Usage**:
```bash
python examples/fps_histogram_example.py
```

**Example Buckets**:
```python
# Custom FPS buckets (frame time in seconds)
fps_buckets = [
    1.0 / 60.0,  # ~16.67ms (+60 FPS)
    1.0 / 30.0,  # ~33.33ms (60 FPS)
    1.0 / 20.0,  # ~50.00ms (30 FPS)
    1.0 / 10.0,  # ~100.0ms (20 FPS)
    1000.0       # Effectively 'infinity'
]

session.log_histogram("frame_time", frame_time,
                     unit="seconds",
                     buckets=fps_buckets)
```

**See Also**: [Histogram Buckets Guide](../HISTOGRAM_BUCKETS.md)

---

## Feature-Specific Examples

### Exception Logging

**File**: `examples/exception_logging_example.py`

Demonstrates automatic exception capture with stack traces.

**Features**:
- Automatic exception capture
- Stack trace extraction
- Source file/line information
- Custom exception data

**Usage**:
```bash
python examples/exception_logging_example.py
```

**Example Code**:
```python
try:
    risky_operation()
except Exception as e:
    session.log_exception(extra_data={"context": "startup"})
```

---

### Heartbeat Monitoring

**File**: `examples/heartbeat_example.py`

Demonstrates heartbeat monitoring for hang detection.

**Features**:
- Manual heartbeat sending
- Automatic heartbeat monitoring
- Hang detection (60 second timeout)
- Recovery detection

**Usage**:
```bash
python examples/heartbeat_example.py
```

**Example Code**:
```python
# Start automatic heartbeat monitoring
session.start_heartbeat_monitor(timeout=60, check_interval=5)

# Or send manual heartbeats
session.heartbeat()

# Stop monitoring
session.stop_heartbeat_monitor()
```

---

### Window Tracking

**File**: `examples/window_tracking_example.py`

Demonstrates tracking window lifecycle and state.

**Features**:
- Window open/close tracking
- Active window tracking
- Window state changes
- Nested state management

**Usage**:
```bash
python examples/window_tracking_example.py
```

---

### Logging Handler Integration

**File**: `examples/logging_handler_example.py`

Demonstrates integration with Python's standard logging module.

**Features**:
- Custom logging handler for DataCat
- Automatic log level mapping
- Integration with existing logging code
- No code changes needed in logged code

**Usage**:
```bash
python examples/logging_handler_example.py
```

**Example Code**:
```python
import logging
from datacat import create_session, DatacatLogHandler

session = create_session()
handler = DatacatLogHandler(session)
logger = logging.getLogger()
logger.addHandler(handler)

# Now all logging.info/warning/error calls go to DataCat
logger.info("Application started")
logger.warning("Low memory")
logger.error("Failed to connect")
```

---

### Testing Integration

**File**: `examples/testing_example.py`

Demonstrates using DataCat for test tracking and reporting.

**Features**:
- Test case tracking
- Assertion logging
- Test result reporting
- Integration with test frameworks

**Usage**:
```bash
python examples/testing_example.py
```

---

## Testing & Validation Examples

### Offline Demo

**File**: `examples/offline_demo.py`

Demonstrates offline mode - daemon continues working when server is unavailable.

**Features**:
- Daemon-side session creation
- Operation queueing
- Automatic server sync when available
- No data loss

**Usage**:
```bash
# Stop the server first
python examples/offline_demo.py
# Start the server during execution to see sync
```

---

### Crash Detection Test

**File**: `examples/test_crash_detection.py`

Tests crash detection by simulating application crashes.

**Features**:
- Parent process monitoring
- Crash event logging
- Session status updates

**Usage**:
```bash
python examples/test_crash_detection.py
```

---

### Python 2.7.4 Compatibility Test

**File**: `examples/test_async_py27.py`

Comprehensive test suite verifying AsyncSession works in Python 2.7.4.

**Tests**:
- Python 2 vs 3 import compatibility
- Queue.Queue functionality
- Threading with daemon threads
- Non-blocking queue operations
- Queue overflow handling
- Background thread processing

**Usage**:
```bash
python examples/test_async_py27.py
# Or with Python 2.7.4
python2.7 examples/test_async_py27.py
```

---

## Performance Benchmarks

### Async Logging

From `game_logging_example.py`:

| Operation          | Average Time | Frame Budget @ 60 FPS |
| ------------------ | ------------ | --------------------- |
| `log_event()`      | 0.008ms      | 0.05%                 |
| `log_metric()`     | 0.008ms      | 0.05%                 |
| `update_state()`   | 0.008ms      | 0.05%                 |
| **100 logs/frame** | 0.8ms        | **5%** ✅             |

**Conclusion**: Async logging is suitable for 60 FPS, 120 FPS, or even higher frame rates.

---

## Next Steps

- **[Demo GUI](demo-gui.md)** - Interactive web UI demonstration
- **[Go Examples](go-examples.md)** - Go client library examples
- **[Game Logging Guide](../GAME_LOGGING.md)** - Complete async logging guide
- **[Metric Types Guide](../METRIC_TYPES.md)** - Understanding all metric types

