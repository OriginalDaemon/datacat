---
layout: default
title: State Updates
parent: API Reference
nav_order: 2
---

# State Update Endpoint
{: .no_toc }

Update session state using deep merge.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Update State

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

---

## Deep Merge Behavior

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

**Key Points:**
- New top-level fields are added
- Existing top-level fields are updated
- Nested objects are merged recursively
- Fields not mentioned in the update are preserved

---

## Delete Fields

Use `null` to delete:
```json
{
  "status": null
}
```

This removes the `status` field from the state.

---

## Response

**Status:** `200 OK`
```json
{
  "success": true
}
```

---

## Example

```bash
curl -X POST http://localhost:9090/api/sessions/550e8400/state \
  -H "Content-Type: application/json" \
  -d '{
    "status": "running",
    "progress": 50
  }'
```

---

## Best Practices

### Use Deep Merge Effectively

Only send state changes:
```python
# Good - only send what changed
session.update_state({"progress": 75})

# Avoid - sending entire state unnecessarily
session.update_state({
    "product": "MyApp",
    "version": "1.0.0",
    "progress": 75,
    ...
})
```

### Organize State Hierarchically

Use nested objects for related fields:
```json
{
  "window_state": {
    "open": ["window1", "window2"],
    "active": "window1",
    "minimized": false
  },
  "memory": {
    "footprint_mb": 512,
    "peak_mb": 768
  }
}
```

This allows updating window state without affecting memory tracking.

### Clean Up Temporary Fields

Delete temporary state when no longer needed:
```json
{
  "download_progress": null,
  "download_url": null
}
```

---

## See Also

- [Sessions](sessions.html) - Session management
- [State Management Guide](../guides/state-management.html) - Advanced state patterns
