---
layout: default
title: Error Handling
parent: API Reference
nav_order: 5
---

# Error Handling
{: .no_toc }

Error responses and status codes used by the DataCat API.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## HTTP Status Codes

All endpoints return appropriate HTTP status codes:

- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid input
- `404 Not Found` - Resource not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

---

## Error Response Format

All errors return a JSON response with error details:

```json
{
  "error": "Error type",
  "details": "Detailed error message"
}
```

---

## Common Errors

### Missing Required Field

**Status:** `400 Bad Request`

```json
{
  "error": "Missing required field",
  "details": "product is required"
}
```

**Cause:** Required field missing from request body

**Solution:** Include all required fields in your request

---

### Session Not Found

**Status:** `404 Not Found`

```json
{
  "error": "Session not found",
  "details": "No session with ID abc-123"
}
```

**Cause:** Session ID doesn't exist or has been deleted

**Solution:** Verify the session ID is correct and the session exists

---

### Invalid JSON

**Status:** `400 Bad Request`

```json
{
  "error": "Invalid request",
  "details": "Failed to parse JSON"
}
```

**Cause:** Malformed JSON in request body

**Solution:** Validate JSON syntax before sending

---

### Method Not Allowed

**Status:** `405 Method Not Allowed`

```json
{
  "error": "Method not allowed",
  "details": "POST required"
}
```

**Cause:** Using wrong HTTP method (e.g., GET instead of POST)

**Solution:** Use the correct HTTP method for the endpoint

---

## Error Handling Best Practices

### Check Status Codes

Always check the HTTP status code:

```python
try:
    response = client.create_session("MyApp", "1.0.0")
except Exception as e:
    if "404" in str(e):
        print("Session not found")
    elif "400" in str(e):
        print("Invalid request")
    else:
        print(f"Unexpected error: {e}")
```

### Handle Network Errors

```python
import time

def create_session_with_retry(client, product, version, max_retries=3):
    for attempt in range(max_retries):
        try:
            return client.create_session(product, version)
        except Exception as e:
            if "500" in str(e) and attempt < max_retries - 1:
                # Server error - retry with backoff
                time.sleep(2 ** attempt)
                continue
            raise
```

### Validate Input

```python
def log_metric_safe(session, name, value):
    # Validate before sending
    if not isinstance(name, str) or not name:
        raise ValueError("Metric name must be a non-empty string")
    
    if not isinstance(value, (int, float)):
        raise ValueError("Metric value must be a number")
    
    session.log_metric(name, value)
```

---

## Debugging Failed Requests

### Enable Verbose Logging

```python
import logging

logging.basicConfig(level=logging.DEBUG)

# Client requests will now show details
client = DatacatClient("http://localhost:9090")
```

### Inspect Error Details

```python
try:
    client.update_state(session_id, {"status": "running"})
except Exception as e:
    print(f"Error: {e}")
    # Log full error details for debugging
    import traceback
    traceback.print_exc()
```

### Use curl for Testing

Test endpoints directly with curl:

```bash
# Test with verbose output
curl -v -X POST http://localhost:9090/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"product":"TestApp","version":"1.0.0"}'

# Check response headers
curl -I http://localhost:9090/api/sessions/invalid-id
```

---

## Server Errors

### 500 Internal Server Error

**Cause:** Unexpected server-side error

**What to check:**
1. Server logs for error details
2. Database connectivity
3. Disk space availability
4. Memory usage

**Temporary Solutions:**
- Retry the request after a delay
- Check server health endpoint
- Contact administrator if persistent

---

## Client Library Errors

### Connection Refused

```
Exception: Failed to connect to http://localhost:9090
```

**Cause:** Server not running or wrong URL

**Solution:**
1. Verify server is running
2. Check the server URL and port
3. Verify no firewall blocking connection

---

### Timeout

```
Exception: Request timed out after 30 seconds
```

**Cause:** Server taking too long to respond

**Solutions:**
1. Increase timeout if operation is legitimately slow
2. Check server performance
3. Reduce batch size if batching
4. Check network connectivity

---

## Rate Limiting

Currently no rate limiting is implemented. Future versions may add rate limiting for production deployments.

**Planned Response:**
```json
{
  "error": "Rate limit exceeded",
  "details": "Maximum 100 requests per minute",
  "retry_after": 45
}
```

---

## See Also

- [Sessions](sessions.html) - Session endpoints
- [Events](events.html) - Event logging
- [Metrics](metrics.html) - Metric logging
- [Troubleshooting Guide](../guides/troubleshooting.html) - Common issues and solutions
