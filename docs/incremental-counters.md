# Incremental Counters - Implementation Guide

## Overview

Datacat implements **daemon-side counter aggregation**, which means you don't need to track cumulative totals in your application code. Just call `log_counter()` whenever an event happens, and the daemon handles the rest!

## Architecture

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Client    │         │    Daemon    │         │    Server    │
│             │         │              │         │              │
│  log_counter│────────▶│  Accumulate  │────────▶│  Store total │
│  ("req")    │  delta  │  in memory   │ total   │  metrics     │
│             │         │              │         │              │
└─────────────┘         └──────────────┘         └──────────────┘
                              │
                              │ Every 5 seconds (batch flush)
                              │
                              ▼
                        Send cumulative
                        totals to server
```

## How It Works

### 1. Client Side (Python)

When you call `log_counter()`, the client sends a **delta** (increment) to the daemon:

```python
# Increment by 1 (default)
session.log_counter("http_requests")

# Increment by specific amount
session.log_counter("bytes_sent", delta=1024)
```

**JSON sent to daemon:**
```json
{
  "session_id": "abc-123",
  "name": "http_requests",
  "type": "counter",
  "value": 0,
  "delta": 1,
  "tags": ["method:GET"]
}
```

### 2. Daemon Side (Go)

The daemon:
1. Receives the delta from the client
2. Creates a unique key from `name + tags`
3. Accumulates deltas in memory:

```go
type SessionBuffer struct {
    Counters map[string]float64  // name+tags -> cumulative value
    // ... other fields
}

// On receiving counter delta:
key := makeCounterKey(name, tags)  // "http_requests:["method:GET"]"
buffer.Counters[key] += delta
```

4. Every 5 seconds (batch interval), sends **cumulative totals** to server:

```go
// Flush counters
for key, value := range buffer.Counters {
    name, tags := parseCounterKey(key)
    sendMetric(sessionID, MetricData{
        Name:  name,
        Type:  "counter",
        Value: value,  // Cumulative total!
        Tags:  tags,
    })
}
// Counters are NOT reset - they keep accumulating!
```

### 3. Server Side (Go)

The server receives cumulative totals and stores them:

```json
{
  "timestamp": "2025-11-23T01:32:11Z",
  "name": "http_requests",
  "type": "counter",
  "value": 315,
  "tags": ["method:GET"]
}
```

The server can later calculate **rates** by looking at changes over time:
- t=0s: 100 requests
- t=5s: 150 requests
- **Rate: (150-100)/5 = 10 requests/second**

## Benefits

### ✅ Simpler Code

**Old way (manual tracking):**
```python
class MyApp:
    def __init__(self):
        self.total_requests = 0
        self.lock = threading.Lock()

    def handle_request(self):
        with self.lock:
            self.total_requests += 1
            session.log_counter("requests", value=self.total_requests)
```

**New way (daemon tracking):**
```python
class MyApp:
    def handle_request(self):
        session.log_counter("requests")  # That's it!
```

### ✅ Thread-Safe by Default

The daemon handles concurrency automatically. Multiple threads can log to the same counter without any synchronization in your code:

```python
# Thread 1
session.log_counter("items_processed", tags=["worker:1"])

# Thread 2 (same time!)
session.log_counter("items_processed", tags=["worker:1"])

# Daemon safely accumulates both
```

### ✅ Network Efficient

Only deltas are sent to the daemon (small payloads). The daemon batches and sends totals to the server periodically.

```
Client → Daemon:  {"delta": 1}          # 20 bytes
Client → Daemon:  {"delta": 1}          # 20 bytes
Client → Daemon:  {"delta": 1}          # 20 bytes
...

Daemon → Server:  {"value": 100}        # 25 bytes (once per batch)
```

### ✅ Separate Counters per Tag Combination

Each unique `name + tags` combination gets its own counter:

```python
session.log_counter("requests", tags=["method:GET"])   # Counter A
session.log_counter("requests", tags=["method:POST"])  # Counter B
session.log_counter("requests")                        # Counter C (no tags)
```

Result:
- `requests:["method:GET"]` → 150
- `requests:["method:POST"]` → 75
- `requests:[]` → 25

## Implementation Details

### Counter Key Generation

The daemon creates a unique key for each counter:

```go
func makeCounterKey(name string, tags []string) string {
    if len(tags) == 0 {
        return name
    }
    // Sort tags for consistent key
    sortedTags := make([]string, len(tags))
    copy(sortedTags, tags)
    sort.Strings(sortedTags)
    tagsJSON, _ := json.Marshal(sortedTags)
    return name + ":" + string(tagsJSON)
}
```

Examples:
- `makeCounterKey("requests", [])` → `"requests"`
- `makeCounterKey("requests", ["http", "GET"])` → `"requests:["GET","http"]"`
- `makeCounterKey("requests", ["GET", "http"])` → `"requests:["GET","http"]"` (same!)

### Counter Persistence

Counters live in the daemon's memory **for the duration of the session**:
- Created when session starts
- Accumulate throughout session lifetime
- Sent to server periodically (every 5 seconds)
- **Not reset** between flushes
- Cleared when session ends

### Backward Compatibility

The old API still works (sending absolute values):

```python
# Old way (still supported)
session.log_metric("requests", value=100, metric_type="counter")

# Sends: {"type": "counter", "value": 100} (no delta field)
# Daemon treats as a regular metric (not accumulated)
```

## Example Use Cases

### HTTP Server

```python
@app.route('/api/users')
def get_users():
    session.log_counter("http_requests", tags=["endpoint:/api/users", "method:GET"])

    try:
        users = db.query_users()
        return jsonify(users)
    except Exception as e:
        session.log_counter("http_errors", tags=["endpoint:/api/users", "type:500"])
        raise
```

### File Transfer

```python
def transfer_file(filename):
    size = os.path.getsize(filename)
    with open(filename, 'rb') as f:
        data = f.read()
        send_over_network(data)

    # Increment bytes counter by file size
    session.log_counter("bytes_transferred", delta=size, tags=["protocol:https"])
```

### Cache Monitoring

```python
def get_from_cache(key):
    value = redis.get(key)
    if value:
        session.log_counter("cache_hits", tags=["cache:redis"])
        return value
    else:
        session.log_counter("cache_misses", tags=["cache:redis"])
        return None
```

### Multi-threaded Processing

```python
def worker(thread_id):
    while True:
        item = queue.get()
        if item is None:
            break

        process(item)

        # Thread-safe! No need for locks
        session.log_counter("items_processed", tags=[f"worker:{thread_id}"])
```

## Testing

See `examples/incremental_counters_example.py` for a comprehensive demonstration:

```bash
python examples/incremental_counters_example.py
```

Then check the results:

```bash
# Get session data
curl http://localhost:9090/api/sessions/{session_id}
```

Look for metrics with `"type": "counter"` to see the accumulated totals.

## Server-Side Analysis

Since the server receives cumulative totals at regular intervals, you can:

1. **Calculate rates:**
   - Requests per second: `(total_t2 - total_t1) / (t2 - t1)`

2. **Compare periods:**
   - Morning vs Evening traffic patterns
   - Weekday vs Weekend differences

3. **Detect anomalies:**
   - Sudden spikes in error counters
   - Unusual traffic patterns

4. **Aggregate across sessions:**
   - Total requests across all users
   - System-wide error rates

## Design Decisions

### Why Daemon-Side Aggregation?

**Alternative 1: Client-side tracking**
- ❌ Requires app to maintain state
- ❌ Thread synchronization needed
- ❌ More complex application code

**Alternative 2: Server-side aggregation**
- ❌ More network traffic
- ❌ Server must handle high-frequency updates
- ❌ Complicates server logic

**✅ Daemon-side aggregation (our choice):**
- ✅ Client just sends deltas (simple)
- ✅ Daemon handles thread-safety
- ✅ Network efficient (batching)
- ✅ Server gets clean cumulative values

### Why Not Reset Counters After Flush?

The counters represent **cumulative totals for the session**. Resetting would lose information:

```
BAD (reset after flush):
t=0s:  value=50   → Server receives: 50
t=5s:  value=30   → Server receives: 30  (only +30 from last flush)
t=10s: value=75   → Server receives: 75

Server sees: [50, 30, 75] - Can't calculate true total!

GOOD (cumulative):
t=0s:  value=50   → Server receives: 50
t=5s:  value=80   → Server receives: 80  (50+30)
t=10s: value=155  → Server receives: 155 (80+75)

Server sees: [50, 80, 155] - Can calculate rates and totals!
```

## See Also

- [Metric Types Overview](metric-types.md)
- [Python Client API](../python/README.md)
- [Daemon Configuration](../cmd/datacat-daemon/README.md)

