# Histogram Buckets - Daemon-Side Aggregation

## Overview

Datacat implements **daemon-side histogram aggregation** with support for custom bucket boundaries. This allows you to efficiently track distributions of values without sending every individual sample to the server.

## How It Works

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Client    │         │    Daemon    │         │    Server    │
│             │         │              │         │              │
│ log_histogram│───────▶│  Accumulate  │────────▶│Store buckets,│
│  (value)    │  sample │ into buckets │ counts  │ sum, count   │
│             │         │              │         │              │
└─────────────┘         └──────────────┘         └──────────────┘
```

### 1. Client Side (Python)

```python
# Default buckets (covers wide range)
session.log_histogram("request_latency", 0.045)

# Custom buckets for specific use case (FPS tracking)
fps_buckets = [1.0/60, 1.0/30, 1.0/20, 1.0/10, 10.0]
session.log_histogram("frame_time", 0.016, buckets=fps_buckets)
```

### 2. Daemon Side (Go)

The daemon:
1. Receives the value and optional bucket configuration
2. Creates/finds the histogram with matching name + tags + buckets
3. Increments the appropriate bucket counter
4. Tracks sum and total count
5. Every 5 seconds, sends bucket data to server

**Default Buckets** (if not specified):
```go
[0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0]
```
These cover microseconds to minutes, suitable for most latency tracking.

### 3. Server Side (Go)

The server stores histogram metrics with metadata:

```json
{
  "timestamp": "2025-11-23T01:55:16Z",
  "name": "fps",
  "type": "histogram",
  "value": 0.02,
  "metadata": {
    "buckets": [
      {"le": 0.0167, "count": 1},
      {"le": 0.0333, "count": 1}
    ],
    "count": 2,
    "sum": 0.04
  }
}
```

Where:
- `le`: "Less than or equal to" (bucket upper bound)
- `count`: Number of samples in this bucket (cumulative)
- `sum`: Sum of all observed values
- `count` (top-level): Total number of observations
- `value`: Average (sum / count)

## Custom Buckets

### Why Custom Buckets?

Different metrics have different meaningful thresholds:

**FPS/Frame Time:**
- +60 FPS: < 16.67ms
- 30-60 FPS: 16.67-33.33ms
- <30 FPS: > 33.33ms

**HTTP Latency:**
- Fast: < 100ms
- OK: 100-500ms
- Slow: > 500ms

**File Sizes:**
- Tiny: < 1KB
- Small: 1KB-100KB
- Medium: 100KB-10MB
- Large: > 10MB

### Defining Custom Buckets

Buckets are defined as **upper bounds** (inclusive). The daemon automatically sorts them and places values in the first bucket where `value <= upper_bound`.

#### Example 1: FPS Tracking

```python
# Frame time buckets aligned with FPS thresholds
fps_buckets = [
    1.0/60.0,   # 16.67ms - anything less is 60+ FPS
    1.0/30.0,   # 33.33ms - 30-60 FPS range
    1.0/20.0,   # 50.00ms - 20-30 FPS range
    1.0/10.0,   # 100.0ms - 10-20 FPS range
    10.0        # 10s - catch-all for extremely slow frames
]

# Log frame times
session.log_histogram("frame_time", frame_duration,
                     unit="seconds",
                     tags=["graphics:high"],
                     buckets=fps_buckets)
```

**Result:** Easy to see how many frames fell into each performance tier.

#### Example 2: API Latency with SLA Buckets

```python
# Buckets based on SLA thresholds
sla_buckets = [
    0.1,    # 100ms - target latency
    0.25,   # 250ms - acceptable
    0.5,    # 500ms - warning threshold
    1.0,    # 1s - SLA violation
    5.0     # 5s - severe issue
]

session.log_histogram("api_latency", response_time,
                     tags=["endpoint:/api/users"],
                     buckets=sla_buckets)
```

#### Example 3: File Size Distribution

```python
# Buckets for file sizes (in bytes)
size_buckets = [
    1024,           # 1 KB
    1024 * 100,     # 100 KB
    1024 * 1024,    # 1 MB
    1024 * 1024 * 10,   # 10 MB
    1024 * 1024 * 100,  # 100 MB
]

session.log_histogram("file_size", size_in_bytes,
                     unit="bytes",
                     tags=["type:image"],
                     buckets=size_buckets)
```

## Separate Histograms

Each unique combination of **name + tags + buckets** gets its own histogram:

```python
# Three separate histograms
session.log_histogram("latency", 0.1, tags=["region:us"])
session.log_histogram("latency", 0.2, tags=["region:eu"])
session.log_histogram("latency", 0.3, buckets=[0.1, 0.5, 1.0])  # Different buckets
```

## Advantages Over Raw Samples

### Storage Efficiency

**Without Aggregation** (storing every sample):
```
1000 samples × 50 bytes = 50 KB per batch
```

**With Histogram Aggregation:**
```
15 buckets × 20 bytes = 300 bytes per batch
```

**Savings: ~165x reduction!**

### Network Efficiency

Only bucket counts are sent, not individual values. For high-frequency metrics, this dramatically reduces bandwidth.

### Analysis Performance

Server-side queries are faster when working with pre-aggregated buckets vs. processing thousands of individual samples.

### Approximate Percentiles

While you lose exact percentiles, you can calculate approximate percentiles from buckets, which is usually sufficient:

```
Buckets: [0.01: 100, 0.05: 850, 0.1: 950, 0.5: 995, 1.0: 1000]

p50 (median): ~0.05s (850th sample)
p95: ~0.5s (950th sample)
p99: ~0.5s (990th sample)
```

## Default Buckets vs Custom Buckets

### Use Default Buckets When:
- General latency tracking
- You want standard percentiles (p50, p95, p99)
- Exploratory analysis
- You don't have specific thresholds in mind

### Use Custom Buckets When:
- You have specific performance thresholds (SLA, FPS targets)
- Domain-specific buckets make more sense (file sizes, age groups, etc.)
- You want to track "good/ok/bad" categories
- Business logic depends on specific boundaries

## Histogram State

**Important:** Histograms accumulate samples for the **entire session** (like counters):

```python
# Time 0-5s
session.log_histogram("latency", 0.01, buckets=[0.1, 1.0])
session.log_histogram("latency", 0.05, buckets=[0.1, 1.0])
# Daemon sends: {buckets: [{le:0.1, count:2}], sum:0.06, count:2}

# Time 5-10s
session.log_histogram("latency", 0.15, buckets=[0.1, 1.0])
# Daemon sends: {buckets: [{le:0.1, count:2}, {le:1.0, count:3}], sum:0.21, count:3}
```

The histogram **does not reset** between flushes - it keeps accumulating!

## Complete Example

```python
from datacat import create_session
import time
import random

session = create_session("http://localhost:9090",
                        product="GameEngine",
                        version="1.0.0")

# Define FPS buckets
fps_buckets = [1.0/60, 1.0/30, 1.0/20, 1.0/10, 10.0]

# Simulate 1000 frames
for frame in range(1000):
    frame_start = time.time()

    # ... render frame ...

    frame_time = time.time() - frame_start

    # Log to histogram (daemon aggregates)
    session.log_histogram("frame_time",
                         frame_time,
                         unit="seconds",
                         tags=["graphics:high", "scene:complex"],
                         buckets=fps_buckets)

session.end()
```

**Network traffic:**
- Without aggregation: ~1000 metrics × 50 bytes = 50 KB
- With aggregation: ~5 histogram updates × 300 bytes = 1.5 KB
- **Savings: 97%!**

## API Reference

### Python Client

```python
session.log_histogram(name, value, unit=None, tags=None, buckets=None, metadata=None)
```

**Parameters:**
- `name` (str): Histogram name
- `value` (float): Sample value to record
- `unit` (str, optional): Unit of measurement ("seconds", "bytes", etc.)
- `tags` (list, optional): Tags for grouping
- `buckets` (list, optional): Custom bucket boundaries. If omitted, uses defaults.
- `metadata` (dict, optional): Additional metadata

**Examples:**

```python
# Default buckets
session.log_histogram("request_latency", 0.045)

# Custom buckets
session.log_histogram("frame_time", 0.016,
                     unit="seconds",
                     buckets=[0.0167, 0.0333, 0.05])

# With tags
session.log_histogram("file_size", 1024000,
                     unit="bytes",
                     tags=["type:image", "format:png"])
```

## See Also

- [FPS Histogram Example](../examples/fps_histogram_example.py) - Complete working example
- [Metric Types Overview](metric-types.md) - All metric types explained
- [Incremental Counters](incremental-counters.md) - Counter aggregation
- [Python Client API](../python/README.md) - Full API documentation

