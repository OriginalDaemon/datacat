---
layout: default
title: REST API
parent: API Reference
nav_order: 1
---

# REST API Reference
{: .no_toc }

Complete reference for the DataCat REST API.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Base URL

```
http://localhost:9090/api
```

All endpoints are prefixed with `/api`. The server runs on port 9090 by default.

---

## Sessions

### Create Session

Create a new session for tracking.

**Endpoint:** `POST /api/sessions`

**Request Body:**
```json
{
  "product": "MyApplication",
  "version": "1.0.0",
  "hostname": "localhost",
  "machine_id": "00:11:22:33:44:55"
}
```

**Parameters:**
- `product` (string, required) - Product name
- `version` (string, required) - Product version
- `hostname` (string, optional) - Hostname
- `machine_id` (string, optional) - MAC address for crash detection

**Response:** `201 Created`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z",
  "active": true,
  "crashed": false,
  "hung": false,
  "suspended": false,
  "state": {
    "product": "MyApplication",
    "version": "1.0.0"
  },
  "events": [],
  "metrics": [],
  "machine_id": "00:11:22:33:44:55",
  "hostname": "localhost"
}
```

**Example:**
```bash
curl -X POST http://localhost:9090/api/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "product": "TestApp",
    "version": "1.0.0"
  }'
```

---

### Get Session

Retrieve details for a specific session.

**Endpoint:** `GET /api/sessions/{id}`

**Path Parameters:**
- `id` (string, required) - Session UUID

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:05Z",
  "ended_at": null,
  "active": true,
  "crashed": false,
  "hung": false,
  "suspended": false,
  "state": {
    "product": "TestApp",
    "version": "1.0.0",
    "status": "running",
    "user": "alice"
  },
  "events": [
    {
      "timestamp": "2024-01-01T12:00:01Z",
      "name": "user_login",
      "level": "info",
      "data": {
        "username": "alice"
      }
    }
  ],
  "metrics": [
    {
      "timestamp": "2024-01-01T12:00:02Z",
      "name": "response_time",
      "value": 123.45,
      "tags": ["api"]
    }
  ],
  "state_history": [
    {
      "timestamp": "2024-01-01T12:00:00Z",
      "state": {
        "product": "TestApp",
        "version": "1.0.0"
      }
    }
  ],
  "last_heartbeat": "2024-01-01T12:00:05Z",
  "machine_id": "00:11:22:33:44:55",
  "hostname": "localhost"
}
```

**Errors:**
- `404 Not Found` - Session does not exist

**Example:**
```bash
curl http://localhost:9090/api/sessions/550e8400-e29b-41d4-a716-446655440000
```

---

### Update State

Update session state using deep merge. New fields are added, existing fields are updated, null values delete fields.

**Endpoint:** `POST /api/sessions/{id}/state`

**Path Parameters:**
- `id` (string, required) - Session UUID

**Request Body:**
```json
{
  "status": "running",
  "user": "alice",
  "nested": {
    "field1": "value1"
  }
}
```

**Deep Merge Behavior:**

Given existing state:
```json
{
  "product": "TestApp",
  "nested": {
    "field1": "old",
    "field2": "preserved"
  }
}
```

Update with:
```json
{
  "status": "running",
  "nested": {
    "field1": "new"
  }
}
```

Result:
```json
{
  "product": "TestApp",
  "status": "running",
  "nested": {
    "field1": "new",
    "field2": "preserved"
  }
}
```

**Delete Fields:**

Use `null` to delete:
```json
{
  "status": null
}
```

**Response:** `200 OK`
```json
{
  "success": true
}
```

**Example:**
```bash
curl -X POST http://localhost:9090/api/sessions/550e8400/state \
  -H "Content-Type: application/json" \
  -d '{
    "status": "running",
    "progress": 50
  }'
```

---

### Log Event

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

### Log Metric

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

### Update Heartbeat

Update the last heartbeat timestamp to indicate the application is alive.

**Endpoint:** `POST /api/sessions/{id}/heartbeat`

**Path Parameters:**
- `id` (string, required) - Session UUID

**Request Body:** Empty or `{}`

**Response:** `200 OK`
```json
{
  "success": true
}
```

**Behavior:**
- Updates `last_heartbeat` to current time
- Resets `hung` status if previously hung
- Logs `application_recovered` event if recovering from hang

**Example:**
```bash
curl -X POST http://localhost:9090/api/sessions/550e8400/heartbeat
```

---

### End Session

Mark a session as ended.

**Endpoint:** `POST /api/sessions/{id}/end`

**Path Parameters:**
- `id` (string, required) - Session UUID

**Request Body:** Empty or `{}`

**Response:** `200 OK`
```json
{
  "success": true
}
```

**Behavior:**
- Sets `active` to `false`
- Sets `ended_at` timestamp
- Session remains in database until retention cleanup

**Example:**
```bash
curl -X POST http://localhost:9090/api/sessions/550e8400/end
```

---

## Data Export

### Get All Sessions

Retrieve all sessions (with pagination planned for future).

**Endpoint:** `GET /api/data/sessions`

**Query Parameters:** None (pagination planned)

**Response:** `200 OK`
```json
[
  {
    "id": "session-1",
    "created_at": "2024-01-01T12:00:00Z",
    "active": true,
    "state": {...},
    "events": [...],
    "metrics": [...]
  },
  {
    "id": "session-2",
    "created_at": "2024-01-01T13:00:00Z",
    "active": false,
    "ended_at": "2024-01-01T14:00:00Z",
    ...
  }
]
```

**Example:**
```bash
curl http://localhost:9090/api/data/sessions
```

---

## Error Responses

All errors return appropriate HTTP status codes with error details:

```json
{
  "error": "Error type",
  "details": "Detailed error message"
}
```

### Common Error Codes

- `400 Bad Request` - Invalid input, missing required fields
- `404 Not Found` - Session not found
- `500 Internal Server Error` - Server-side error

**Examples:**

**Missing Required Field:**
```json
{
  "error": "Missing required field",
  "details": "product is required"
}
```

**Session Not Found:**
```json
{
  "error": "Session not found",
  "details": "No session with ID abc-123"
}
```

---

## Rate Limiting

Currently no rate limiting is implemented. Future versions may add rate limiting for production deployments.

---

## Best Practices

### Use the Daemon

For optimal performance, use the local daemon instead of calling the API directly:
- 10-100x reduction in network traffic
- Automatic batching and filtering
- Built-in crash and hang detection

### Heartbeat Frequency

Send heartbeats based on your application's characteristics:
- **Interactive apps:** Every 5-10 seconds
- **Batch processing:** Every 30-60 seconds
- **Long-running jobs:** Every 1-2 minutes

Configure timeout to 3-5x the heartbeat interval.

### State Updates

- Use deep merge to update nested state
- Avoid sending entire state on every update
- Use null to delete fields you no longer need

### Events vs Metrics

- **Events:** Discrete occurrences (user actions, errors, state changes)
- **Metrics:** Numeric measurements (performance, resource usage)

---

## Examples

### Complete Session Lifecycle

```bash
# 1. Create session
SESSION_ID=$(curl -s -X POST http://localhost:9090/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"product":"MyApp","version":"1.0.0"}' \
  | jq -r '.id')

# 2. Update state
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/state \
  -H "Content-Type: application/json" \
  -d '{"status":"running","user":"alice"}'

# 3. Log event
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/events \
  -H "Content-Type: application/json" \
  -d '{"name":"startup","data":{"config":"prod"}}'

# 4. Log metric
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/metrics \
  -H "Content-Type: application/json" \
  -d '{"name":"memory_mb","value":512.5}'

# 5. Send heartbeat
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/heartbeat

# 6. End session
curl -X POST http://localhost:9090/api/sessions/$SESSION_ID/end

# 7. Retrieve session
curl http://localhost:9090/api/sessions/$SESSION_ID | jq .
```

---

## Next Steps

- [Sessions API](sessions.html) - Session management endpoints
- [Events API](events.html) - Event logging endpoints
- [Metrics API](metrics.html) - Metrics logging endpoints
- [State API](state.html) - State management endpoints
- [Architecture Guide](../_guides/architecture.html) - System architecture
- [Quick Start Guide](../_guides/quickstart.html) - Getting started
