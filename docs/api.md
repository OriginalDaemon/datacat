---
layout: default
title: API Reference
nav_order: 3
has_children: true
permalink: /api
---

# API Reference

Complete API documentation for DataCat REST API and client libraries.
{: .fs-6 .fw-300 }

---

## REST API

The DataCat REST API provides HTTP endpoints for all operations.

- [REST API Reference](rest-api.html) - Complete HTTP API documentation

---

## Client Libraries

Language-specific API references.

- [Python API](python-api.html) - Python client library reference
- [Go API](go-api.html) - Go client library reference

---

## Quick Links

### Common Operations

**Create Session:**
```bash
POST /api/sessions
{
  "product": "MyApp",
  "version": "1.0.0"
}
```

**Update State:**
```bash
POST /api/sessions/{id}/state
{
  "status": "running",
  "user": "alice"
}
```

**Log Event:**
```bash
POST /api/sessions/{id}/events
{
  "name": "user_action",
  "data": {"action": "click"}
}
```

**Log Metric:**
```bash
POST /api/sessions/{id}/metrics
{
  "name": "response_time",
  "value": 123.45,
  "tags": ["api", "user"]
}
```

### Authentication

Currently, DataCat does not require authentication. Ensure the server is deployed in a trusted network environment.

---

## API Versioning

The current API is version 1. The base path is `/api/`.

Future versions will use `/api/v2/` etc. to maintain backwards compatibility.

---

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid input
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

Error responses include a message:

```json
{
  "error": "Session not found",
  "details": "No session with ID abc-123"
}
```
