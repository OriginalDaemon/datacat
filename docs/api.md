# DataCat API Reference

## Health Endpoints

Both the server and web UI provide health check endpoints for monitoring and service discovery.

### Server Health Check

**Endpoint:** `GET /health`

**URL:** `http://localhost:9090/health`

**Response:**

```json
{
  "status": "healthy",
  "service": "datacat-server",
  "version": "1.0.0"
}
```

**Status Code:** `200 OK`

**Usage:**

```bash
# Check if server is healthy
curl http://localhost:9090/health

# PowerShell
Invoke-WebRequest -Uri "http://localhost:9090/health"
```

### Web UI Health Check

**Endpoint:** `GET /health`

**URL:** `http://localhost:8080/health`

**Response:**

```json
{
  "status": "healthy",
  "service": "datacat-web",
  "version": "1.0.0"
}
```

**Status Code:** `200 OK`

**Usage:**

```bash
# Check if web UI is healthy
curl http://localhost:8080/health

# PowerShell
Invoke-WebRequest -Uri "http://localhost:8080/health"
```

---

## Session Management

### Create Session

**Endpoint:** `POST /api/sessions`

**Request Body:**

```json
{
  "product": "MyApp",
  "version": "1.0.0",
  "initial_state": {
    "status": "starting"
  }
}
```

**Response:**

```json
{
  "session_id": "abc123...",
  "product": "MyApp",
  "version": "1.0.0",
  "state": {
    "status": "starting"
  },
  "created_at": "2025-01-01T12:00:00Z"
}
```

### Get Session

**Endpoint:** `GET /api/sessions/{session_id}`

**Response:**

```json
{
  "session_id": "abc123...",
  "product": "MyApp",
  "version": "1.0.0",
  "state": {...},
  "events": [...],
  "metrics": [...],
  "created_at": "2025-01-01T12:00:00Z",
  "last_updated": "2025-01-01T12:05:00Z"
}
```

### Update State

**Endpoint:** `POST /api/sessions/{session_id}/state`

**Request Body:**

```json
{
  "status": "running",
  "window_state": {
    "open": ["w1", "w2"],
    "active": "w1"
  }
}
```

**Response:** `200 OK`

**Note:** State updates are deep-merged with existing state.

### Log Event

**Endpoint:** `POST /api/sessions/{session_id}/events`

**Request Body:**

```json
{
  "name": "user_action",
  "level": "info",
  "category": "interaction",
  "labels": ["button", "click"],
  "message": "User clicked submit button",
  "data": {
    "button_id": "submit",
    "page": "checkout"
  }
}
```

**Response:** `200 OK`

### Log Metric

**Endpoint:** `POST /api/sessions/{session_id}/metrics`

**Request Body:**

```json
{
  "name": "response_time",
  "value": 123.45,
  "tags": ["http", "api", "checkout"]
}
```

**Response:** `200 OK`

### End Session

**Endpoint:** `POST /api/sessions/{session_id}/end`

**Response:** `200 OK`

---

## Data Retrieval

### Get All Sessions

**Endpoint:** `GET /api/data/sessions`

**Response:**

```json
{
  "sessions": [
    {
      "session_id": "abc123...",
      "product": "MyApp",
      "version": "1.0.0",
      "created_at": "2025-01-01T12:00:00Z",
      "last_updated": "2025-01-01T12:05:00Z",
      "status": "active"
    },
    ...
  ]
}
```

---

## Error Responses

All API endpoints may return error responses:

### 400 Bad Request

```json
{
  "error": "Invalid request body"
}
```

### 404 Not Found

```json
{
  "error": "Session not found"
}
```

### 500 Internal Server Error

```json
{
  "error": "Internal server error"
}
```

---

## Rate Limiting

Currently, there are no rate limits on the API. For production use, consider implementing rate limiting at the proxy/gateway level.

---

## Authentication

Currently, the API does not require authentication. For production use, consider implementing authentication at the proxy/gateway level or by extending the server code.

---

## CORS

The server accepts requests from all origins by default. For production use, configure CORS headers appropriately.
