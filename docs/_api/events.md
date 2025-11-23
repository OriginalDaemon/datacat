---
layout: default
title: Events
parent: API Reference
nav_order: 3
---

# Event Logging Endpoint
{: .no_toc }

Log events with optional metadata and exception information.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Log Event

Log an event with optional metadata.

**Endpoint:** `POST /api/sessions/{id}/events`

**Path Parameters:**
- `id` (string, required) - Session UUID

**Request Body:**
```json
{
  "name": "user_action",
  "level": "info",
  "category": "ui",
  "labels": ["button", "click"],
  "message": "User clicked submit button",
  "data": {
    "button_id": "submit",
    "form": "login"
  }
}
```

**Parameters:**
- `name` (string, required) - Event name
- `level` (string, optional) - Log level: debug, info, warning, error, critical
- `category` (string, optional) - Event category
- `labels` (array, optional) - List of labels/tags
- `message` (string, optional) - Human-readable message
- `data` (object, optional) - Event data

**Response:** `200 OK`
```json
{
  "success": true
}
```

**Example:**
```bash
curl -X POST http://localhost:9090/api/sessions/550e8400/events \
  -H "Content-Type: application/json" \
  -d '{
    "name": "error_occurred",
    "level": "error",
    "data": {
      "error_type": "ValidationError",
      "message": "Invalid email"
    }
  }'
```

---

## Exception Events

For exception/error events, additional fields are available:

**Request Body:**
```json
{
  "name": "exception",
  "level": "error",
  "category": "exception",
  "labels": ["exception", "ValueError"],
  "message": "Invalid input provided",
  "data": {
    "context": "user_input"
  },
  "exception_type": "ValueError",
  "exception_msg": "Invalid input provided",
  "stacktrace": [
    "Traceback (most recent call last):",
    "  File \"app.py\", line 42, in process",
    "    validate(data)",
    "ValueError: Invalid input provided"
  ],
  "source_file": "/app/validator.py",
  "source_line": 15,
  "source_function": "validate"
}
```

**Exception Parameters:**
- `exception_type` (string, optional) - Exception type (e.g., "ValueError", "NullPointerException")
- `exception_msg` (string, optional) - Exception message
- `stacktrace` (array, optional) - Array of stack trace lines
- `source_file` (string, optional) - File where exception occurred
- `source_line` (int, optional) - Line number where exception occurred
- `source_function` (string, optional) - Function where exception occurred

---

## Event Levels

Standard log levels (in order of severity):

1. **debug** - Detailed debugging information
2. **info** - Informational messages (default)
3. **warning** - Warning messages
4. **error** - Error messages
5. **critical** - Critical failures

---

## Common Event Patterns

### User Actions
```json
{
  "name": "button_click",
  "level": "info",
  "category": "ui",
  "labels": ["user_action"],
  "data": {
    "button_id": "submit",
    "screen": "login"
  }
}
```

### System Events
```json
{
  "name": "background_task_started",
  "level": "info",
  "category": "system",
  "labels": ["background", "task"],
  "data": {
    "task_type": "sync",
    "estimated_duration": 30
  }
}
```

### Errors
```json
{
  "name": "api_error",
  "level": "error",
  "category": "network",
  "message": "Failed to connect to API",
  "data": {
    "endpoint": "/api/users",
    "status_code": 500,
    "retry_count": 3
  }
}
```

---

## Best Practices

### Use Descriptive Names
```python
# Good
log_event("user_logged_in", data={"username": "alice"})
log_event("payment_failed", data={"amount": 50, "reason": "insufficient_funds"})

# Avoid
log_event("event1", data={"type": "login"})
log_event("error", data={"msg": "failed"})
```

### Add Context with Data
Include relevant context that helps debugging:
```python
log_event("database_query_slow",
    level="warning",
    data={
        "query": "SELECT * FROM users WHERE...",
        "duration_ms": 5000,
        "table": "users",
        "row_count": 10000
    }
)
```

### Use Appropriate Levels
- `debug`: Verbose details for development
- `info`: Normal operations and significant events
- `warning`: Issues that don't prevent operation
- `error`: Failures that need attention
- `critical`: Severe failures requiring immediate action

### Categorize Events
Use categories to group related events:
```python
# UI events
log_event("button_click", category="ui", ...)

# Network events
log_event("api_call", category="network", ...)

# Business logic
log_event("order_placed", category="business", ...)
```

---

## See Also

- [Sessions](sessions.html) - Session management
- [Metrics](metrics.html) - Recording numeric measurements
- [Hung Tracking](../_guides/hung-tracking.html) - Automatic hang detection
- [REST API Reference](rest-api.html) - Complete API documentation
