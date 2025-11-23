# Metric Types

Datacat supports four different types of metrics, each designed for specific use cases.

## Quick Reference

| Type | Use Case | Example | Characteristics |
|------|----------|---------|-----------------|
| **Gauge** | Current values | CPU%, memory, temperature | Can go up or down |
| **Counter** | Cumulative totals | Total requests, bytes sent | Only increases |
| **Histogram** | Value distributions | Request latencies, file sizes | Many samples for percentiles |
| **Timer** | Duration measurement | Function execution time | Auto-measured, optional count |

---

## Gauge

**Current point-in-time values that can increase or decrease.**

### When to Use
- System metrics (CPU%, memory usage, disk space)
- Application state (active connections, queue depth)
- Environmental data (temperature, humidity)
- Anything that represents "current value"

### Python API

```python
session.log_gauge("cpu_percent", 45.2, tags=["system"], unit="percent")
session.log_gauge("memory_mb", 1024.5, tags=["system"], unit="megabytes")
session.log_gauge("active_connections", 42, tags=["network"])
```

### Characteristics
- Value can go up or down
- Latest value is what matters
- Each log represents current state at that moment

---

## Counter

**Cumulative values that only increase. The daemon automatically tracks totals for you!**

### How It Works
- Call `log_counter()` with a delta (default: 1)
- **Daemon accumulates the total** automatically
- Server receives cumulative totals
- You don't need to track totals in your code!

### When to Use
- Total requests served
- Total errors encountered
- Bytes transmitted
- Operations completed
- Any cumulative count

### Python API

```python
# Simple increment (by 1)
session.log_counter("http_requests", tags=["method:GET"])

# Increment by specific amount
session.log_counter("bytes_sent", delta=1048576, tags=["network"])  # 1 MB

# Multiple calls accumulate automatically
for request in requests:
    session.log_counter("requests_processed")  # Daemon tracks total!

# Error counting
if error:
    session.log_counter("errors", tags=["type:validation"])
```

### Why This Is Better

```python
# OLD WAY - Manual tracking (don't do this!)
total_requests = 0
def handle_request():
    global total_requests
    total_requests += 1
    session.log_counter("requests", value=total_requests)

# NEW WAY - Let daemon track it!
def handle_request():
    session.log_counter("requests")  # Just increment!
```

### Characteristics
- Value only increases (monotonic)
- Daemon-side aggregation (thread-safe automatically)
- Useful for calculating rates (requests/second)
- Separate totals per unique name + tags combination
- No need to maintain counter variables

### Analysis
- Server can calculate rate of change (derivative)
- Example: 100 requests at t=0s, 150 requests at t=5s → 10 requests/second

---

## Histogram

**Distribution of values across many samples.**

### When to Use
- Request/response latencies
- File sizes
- Query execution times
- Anything where you want percentiles (p50, p95, p99)

### Python API

```python
# Log many samples
for request in requests:
    session.log_histogram(
        "request_latency",
        request.duration,
        tags=["endpoint:/api/users"],
        metadata={"request_id": request.id}
    )
```

### Characteristics
- Log many samples (hundreds or thousands)
- Analyze distribution later
- Calculate percentiles: p50 (median), p95, p99
- Identify outliers and patterns

### Analysis Example
```
100 latency samples:
- p50 (median): 45ms  (half of requests faster than this)
- p95: 250ms          (95% of requests faster than this)
- p99: 1200ms         (99% of requests faster than this)
- Max: 2500ms         (slowest request)
```

**Current Implementation Note:** Datacat currently stores all individual histogram values (raw samples). In the future, we may add client-side bucketing for more efficient storage at high volume.

---

## Timer

**Measure duration of operations with automatic timing.**

### When to Use
- Function/method execution time
- Operation duration
- Performance profiling
- Loop processing time with iteration counts

### Python API

#### Basic Timer
```python
with session.timer("load_config"):
    config = load_config_file()
# Automatically logs duration
```

#### Timer with Count (Known Iterations)
```python
items = get_items()
with session.timer("process_items", count=len(items)):
    for item in items:
        process(item)
# Logs duration AND count (for avg time per item)
```

#### Timer with Incremental Count
```python
with session.timer("process_queue") as timer:
    while queue.has_items():
        timer.count += 1
        process(queue.pop())
# Logs duration and final count
```

#### Specify Time Unit
```python
# Log in milliseconds instead of seconds
with session.timer("render_frame", unit="milliseconds"):
    render_scene()
```

### Characteristics
- Automatically measures duration (start to end of context)
- Optional count field for iterations
- Can calculate average time per iteration
- Default unit: seconds (can use "milliseconds")

### Analysis
```
Timer: "process_items", duration: 5.2s, count: 100
→ Average time per item: 52ms
```

---

## API Reference

### Session Methods

```python
# Gauge
session.log_gauge(name, value, tags=None, unit=None)

# Counter
session.log_counter(name, value, tags=None)

# Histogram
session.log_histogram(name, value, tags=None, metadata=None)

# Timer
session.log_timer(name, duration, count=None, tags=None, unit="seconds")
session.timer(name, count=None, tags=None, unit="seconds")  # Context manager
```

### Low-Level API

```python
# Generic metric logging (all types)
client.log_metric(
    session_id,
    name,
    value,
    tags=None,
    metric_type="gauge",  # "gauge", "counter", "histogram", "timer"
    count=None,
    unit=None,
    metadata=None
)
```

---

## Best Practices

### Naming Conventions

**Be descriptive and consistent:**
```python
# Good
session.log_gauge("memory_used_mb", 1024.5)
session.log_counter("http_requests_total", 1523)
session.log_histogram("api_response_time_ms", 45.2)

# Avoid
session.log_gauge("mem", 1024.5)  # Too vague
session.log_counter("requests", 1523)  # Missing context
```

### Include Units in Name or Parameter

```python
# Option 1: In name
session.log_gauge("cpu_percent", 45.2)
session.log_gauge("memory_mb", 1024.5)

# Option 2: In unit parameter
session.log_gauge("cpu", 45.2, unit="percent")
session.log_gauge("memory", 1024.5, unit="megabytes")
```

### Use Tags for Dimensions

```python
# Group related metrics with tags
session.log_gauge("memory_mb", 1024.5, tags=["type:heap", "process:worker-1"])
session.log_counter("requests_total", 1523, tags=["endpoint:/api/users", "method:GET"])
```

### Choose the Right Type

```python
# WRONG: Using gauge for cumulative count
session.log_gauge("total_requests", 1523)  # Should be counter

# WRONG: Using counter for current value
session.log_counter("active_connections", 42)  # Should be gauge

# RIGHT:
session.log_counter("total_requests", 1523)  # Cumulative
session.log_gauge("active_connections", 42)  # Current
```

### Histogram Sampling

For very high-frequency events, consider sampling:

```python
# Sample 10% of requests for histogram
if random.random() < 0.1:
    session.log_histogram("request_latency", duration)
```

---

## Migration from Old API

If you're upgrading from the old API without metric types:

### Old Code (Implicit Gauge)
```python
session.log_metric("cpu_percent", 45.2)
```

### New Code (Explicit Type)
```python
# Backward compatible (defaults to gauge)
session.log_metric("cpu_percent", 45.2)

# OR be explicit
session.log_gauge("cpu_percent", 45.2)
```

**Note:** Old code continues to work. Metrics without an explicit type default to "gauge" for backward compatibility.

---

## Examples

See `examples/metric_types_example.py` for a complete demonstration of all metric types.

```bash
python examples/metric_types_example.py
```

---

## Future Enhancements

Potential future additions:

1. **Client-side histogram bucketing** - Aggregate histogram data client-side for high-volume scenarios
2. **Summary metrics** - Pre-calculated quantiles (p50, p90, p99) sent with each batch
3. **Rate calculations** - Automatic rate calculation for counters on server side
4. **Metric aggregation** - Time-window aggregations (per minute, per hour)

---

## FAQ

**Q: What's the difference between a timer and a histogram of durations?**
- **Timer**: Convenience feature that automatically measures duration. Still stores as individual samples.
- **Histogram**: Manual logging of many values. Useful for any distribution, not just durations.
- In practice: Timers are histograms with automatic time measurement.

**Q: Should I use a counter or a gauge for "number of items processed"?**
- **Counter**: If it's a cumulative total (keeps increasing)
- **Gauge**: If it's a current count that can decrease (like queue size)

**Q: How many histogram samples should I log?**
- Generally: 100-10,000 samples per time window
- For percentiles: More samples = more accuracy
- For high volume: Consider sampling (log 1-10% of events)

**Q: Can I use timers in high-frequency loops?**
- Yes, but consider the overhead
- For 60 FPS game loops: Timer per frame might be okay
- For 1000s of operations/sec: Consider sampling or batch timing

**Q: What happens if I use the wrong metric type?**
- Server stores it as specified type
- You can still query the data
- But analysis/visualization will be suboptimal
- Example: Using gauge for cumulative total means you can't calculate rates easily

