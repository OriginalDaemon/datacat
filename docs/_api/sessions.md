---
layout: default
title: Sessions
parent: API Reference
nav_order: 1
---

# Session Endpoints
{: .no_toc }

Endpoints for creating, retrieving, and managing sessions.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Create Session

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

## Get Session

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
  "events": [...],
  "metrics": [...],
  "state_history": [...],
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

## End Session

Mark a session as ended (normal termination).

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
- Use this for normal session termination (when application calls `session.end()`)

**Example:**
```bash
curl -X POST http://localhost:9090/api/sessions/550e8400/end
```

---

## Mark Session as Crashed

Mark a session as crashed (abnormal termination). This is typically called by the daemon when it detects the parent process has terminated without calling `end()`.

**Endpoint:** `POST /api/sessions/{id}/crash`

**Path Parameters:**
- `id` (string, required) - Session UUID

**Request Body:**
```json
{
  "reason": "parent_process_terminated"
}
```

**Response:** `200 OK`
```json
{
  "status": "ok"
}
```

**Behavior:**
- Sets `ended_at` timestamp
- Sets `crashed` flag to `true`
- Sets `active` to `false`
- Logs a `session_crashed_detected` event with the provided reason
- Used to distinguish abnormal termination from normal session end

**Common Reasons:**
- `parent_process_terminated` - Daemon detected parent process ended without calling `end()`
- `abnormal_termination` - Generic crash reason

**Example:**
```bash
curl -X POST http://localhost:9090/api/sessions/550e8400/crash \
  -H "Content-Type: application/json" \
  -d '{"reason":"parent_process_terminated"}'
```

---

## Update Heartbeat

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

## Get All Sessions

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

## See Also

- [State Updates](state.html) - Managing session state
- [Events](events.html) - Logging events
- [Metrics](metrics.html) - Recording metrics
