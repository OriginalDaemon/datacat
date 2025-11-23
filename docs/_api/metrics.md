---
layout: default
title: Metrics
parent: API Reference
nav_order: 4
---

# Metric Logging Endpoint
{: .no_toc }

Record numeric metrics with optional tags.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Log Metric

Record a numeric metric with optional tags.

**Endpoint:** `POST /api/sessions/{id}/metrics`

**Path Parameters:**
- `id` (string, required) - Session UUID

**Request Body:**
```json
{
  "name": "response_time_ms",
  "value": 123.45,
  "tags": ["api", "v2", "user"]
}
```

**Parameters:**
- `name` (string, required) - Metric name
- `value` (number, required) - Metric value
- `tags` (array, optional) - List of tags

**Response:** `200 OK`
```json
{
  "success": true
}
```

**Example:**
```bash
curl -X POST http://localhost:9090/api/sessions/550e8400/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "name": "memory_mb",
    "value": 512.5,
    "tags": ["backend"]
  }'
```

---

## Metric Types

### Performance Metrics
```json
{
  "name": "response_time_ms",
  "value": 45.2,
  "tags": ["api", "endpoint:users"]
}
```

### Resource Metrics
```json
{
  "name": "memory_usage_mb",
  "value": 1024.5,
  "tags": ["heap", "java"]
}
```

### Business Metrics
```json
{
  "name": "transactions_processed",
  "value": 1523,
  "tags": ["payment", "success"]
}
```

### Counter Metrics
```json
{
  "name": "error_count",
  "value": 3,
  "tags": ["validation", "user_input"]
}
```

---

## Best Practices

### Use Consistent Naming

Follow a naming convention:
```python
# Good - descriptive and consistent
log_metric("api_response_time_ms", 123.4)
log_metric("database_query_time_ms", 45.6)
log_metric("memory_usage_mb", 512.0)

# Avoid - inconsistent or vague
log_metric("time", 123.4)
log_metric("db_ms", 45.6)
log_metric("mem", 512.0)
```

### Include Units in Name

Make units explicit:
```python
# Good
log_metric("duration_seconds", 5.2)
log_metric("file_size_bytes", 1048576)
log_metric("temperature_celsius", 22.5)

# Avoid
log_metric("duration", 5.2)  # Seconds? Milliseconds?
log_metric("size", 1048576)   # Bytes? KB? MB?
```

### Use Tags for Dimensions

Use tags to add context without creating separate metrics:
```python
# Good - single metric with tags
log_metric("http_requests", 100, tags=["endpoint:users", "method:GET"])
log_metric("http_requests", 50, tags=["endpoint:posts", "method:POST"])

# Avoid - separate metrics for each combination
log_metric("http_requests_users_get", 100)
log_metric("http_requests_posts_post", 50)
```

### Record at Appropriate Frequency

Balance detail with overhead:
```python
# High frequency for critical metrics
while processing:
    log_metric("queue_size", len(queue), tags=["priority:high"])
    time.sleep(1)  # Every second

# Lower frequency for less critical metrics
every_minute:
    log_metric("disk_usage_gb", get_disk_usage())
```

---

## Common Metric Patterns

### Response Times
```python
import time

start = time.time()
result = api_call()
duration_ms = (time.time() - start) * 1000

session.log_metric("api_response_time_ms", duration_ms,
                   tags=["endpoint:users", "status:200"])
```

### Resource Usage
```python
import psutil

process = psutil.Process()
memory_mb = process.memory_info().rss / 1024 / 1024
cpu_percent = process.cpu_percent(interval=1)

session.log_metric("memory_usage_mb", memory_mb)
session.log_metric("cpu_usage_percent", cpu_percent)
```

### Throughput
```python
items_processed = process_batch(items)
session.log_metric("items_processed", items_processed,
                   tags=["batch_id:123", "type:user_data"])
```

### Error Rates
```python
total = 100
errors = 5
error_rate = (errors / total) * 100

session.log_metric("error_rate_percent", error_rate,
                   tags=["operation:import"])
```

---

## Metric vs Event

**Use Metrics for:**
- Numeric measurements
- Performance data
- Resource usage
- Counters and rates
- Time series data

**Use Events for:**
- Discrete occurrences
- User actions
- State changes
- Errors and exceptions
- Audit logs

**Example:**
```python
# Metric - numeric measurement
session.log_metric("api_latency_ms", 123.4)

# Event - discrete occurrence
session.log_event("api_call_completed",
                  data={"endpoint": "/users", "status": 200})
```

---

## See Also

- [Events](events.html) - Logging discrete events
- [Sessions](sessions.html) - Session management
- [Metric Types](../metric-types.html) - Understanding all metric types
- [REST API Reference](rest-api.html) - Complete API documentation
