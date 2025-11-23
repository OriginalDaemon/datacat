# Example Applications - Metric Types Usage

## Summary

All example applications now demonstrate the use of all four metric types:
- **Gauges**: Point-in-time values (FPS, CPU%, memory, temperature)
- **Counters**: Cumulative totals (requests, errors, items processed)
- **Histograms**: Value distributions (latencies, frame times, durations)
- **Timers**: Duration measurements (operation timing, batch processing)

## Examples by Metric Type Coverage

### ✅ `metric_types_example.py` - ALL 4 TYPES
**Purpose**: Dedicated demonstration of all metric types

**Metrics Used**:
- **Gauges**: `cpu_percent`, `memory_usage`, `fps`, `temperature`
- **Counters**: `http_requests`, `http_errors`, `bytes_sent` (incremental)
- **Histograms**: `request_latency`, `query_time`, `file_size`
- **Timers**: `database_query`, `api_call`, `processing_loop` (with context manager)

**Run**: `python examples/metric_types_example.py`

---

### ✅ `game_logging_example.py` - ALL 4 TYPES (Updated)
**Purpose**: Real-time game loop at 60 FPS with minimal logging overhead

**Metrics Used**:
- **Gauges**: `fps` (current frames per second)
- **Counters**: `frames_rendered`, `enemies_spawned`, `player_moves`
- **Histograms**: `frame_time` (with FPS-specific buckets: 120fps, 60fps, 30fps, 15fps)
- **Timers**: `ai_update`, `physics_update` (context manager for per-frame operations)

**Key Features**:
- Async non-blocking logging
- < 0.1ms overhead per frame
- FPS histogram with custom buckets

**Run**: `python examples/game_logging_example.py`

---

### 🔸 `complete_example.py` - COMPREHENSIVE FEATURE DEMO
**Purpose**: Demonstrates all datacat features (state, events, exceptions, heartbeat)

**Metrics Used**:
- **Gauges**: `memory_mb`, `cpu_percent`, `window_count` (implicitly via `log_metric`)

**Key Features**:
- Nested state tracking
- Heartbeat monitoring
- Exception logging
- Session lifecycle management

**Note**: Focuses on comprehensive feature coverage rather than metric type diversity. See `metric_types_example.py` and `game_logging_example.py` for all 4 metric types in action.

**Run**: `python examples/complete_example.py`

---

### ✅ `fps_histogram_example.py` - FOCUS: Histograms
**Purpose**: Dedicated histogram demonstration with custom FPS buckets

**Metrics Used**:
- **Histograms**: `frame_time` with custom FPS buckets

**Scenarios**:
1. Normal mixed performance (1000 frames)
2. High-performance mode (500 frames)
3. Performance degradation
4. Graphics settings comparison
5. Render phase breakdown
6. Default vs custom buckets

**Run**: `python examples/fps_histogram_example.py`

---

### ✅ `incremental_counters_example.py` - FOCUS: Counters
**Purpose**: Dedicated counter demonstration with daemon-side aggregation

**Metrics Used**:
- **Counters**: `http_requests`, `http_errors`, `bytes_transferred`, `items_processed`, `cache_hits`, `cache_misses`, `cumulative_events`
- **Gauges**: `requests_per_sec`, `events_this_period` (for comparison)

**Scenarios**:
1. Simple event counting (web server)
2. Byte counting with deltas
3. Concurrent operations (multi-threaded)
4. Cache statistics
5. Counter vs Gauge comparison

**Run**: `python examples/incremental_counters_example.py`

---

### 🔸 `basic_example.py` - GAUGES ONLY
**Purpose**: Minimal getting-started example

**Metrics Used**:
- **Gauges**: `cpu_percent`, `memory_usage` (implicitly via `log_metric`)

**Note**: Intentionally simple for beginners. No need to add all types here.

---

### 🔸 Other Examples
These examples focus on specific features (events, exceptions, heartbeat, window tracking):
- `exception_logging_example.py` - Exception handling
- `heartbeat_example.py` - Heartbeat monitoring
- `window_tracking_example.py` - Window lifecycle
- `testing_example.py` - Testing patterns
- `logging_handler_example.py` - Python logging integration

**Note**: These don't need all metric types as they're feature-specific demos.

## Quick Reference: When to Use Each Type

### Gauge
**Use for**: Point-in-time values that can go up or down
```python
session.log_gauge("cpu_percent", 45.2, unit="percent")
session.log_gauge("temperature", 72.5, unit="celsius")
session.log_gauge("fps", 60.0, unit="fps")
```

### Counter
**Use for**: Events you want to count (daemon tracks total automatically)
```python
session.log_counter("requests")  # Increment by 1
session.log_counter("bytes_sent", delta=1024)  # Increment by amount
session.log_counter("errors", tags=["type:validation"])  # With tags
```

### Histogram
**Use for**: Value distributions (latencies, durations, sizes)
```python
# Default buckets (microseconds to minutes)
session.log_histogram("request_latency", 0.045)

# Custom buckets for specific thresholds
session.log_histogram("frame_time", 0.016,
                     buckets=[1/120, 1/60, 1/30, 1/15])
```

### Timer
**Use for**: Measuring operation duration
```python
# Context manager (automatic timing)
with session.timer("database_query", unit="seconds"):
    result = db.execute(query)

# With iteration count (for averaging)
with session.timer("process_items", count=len(items)) as timer:
    for item in items:
        process(item)
```

## Running All Examples

To see all metric types in action:

```bash
# Start services
.\scripts\run-server.ps1
.\scripts\run-daemon.ps1
.\scripts\run-web.ps1

# Run examples
cd D:\github\datacat

# All 4 types - comprehensive demo
python examples/metric_types_example.py

# All 4 types - game scenario (60 FPS)
python examples/game_logging_example.py

# All 4 types - application scenario
python examples/complete_example.py

# Focused demos
python examples/incremental_counters_example.py  # Counter deep-dive
python examples/fps_histogram_example.py         # Histogram deep-dive
```

## Web UI Visualization

After running examples, view the sessions at:
```
http://localhost:8080
```

**What to Look For**:
- **Gauges**: Line charts with statistics (avg, min, max)
- **Counters**: Large cumulative value + trend line
- **Histograms**: Bar charts showing bucket distributions
- **Timers**: Treated as histograms (distribution of durations)

## Summary

✅ **2 examples demonstrate all 4 metric types**:
- `metric_types_example.py` - Dedicated demo showcasing each type
- `game_logging_example.py` - Real-time 60 FPS game scenario

✅ **2 examples deep-dive specific types**:
- `incremental_counters_example.py` - Counter patterns and daemon aggregation
- `fps_histogram_example.py` - Custom histogram buckets for performance tracking

✅ **1 example demonstrates comprehensive feature coverage**:
- `complete_example.py` - All datacat features (state, events, exceptions, heartbeat)

✅ **All examples are tested and working**

Users can now see all metric types in action across different real-world scenarios!

