---
layout: default
title: Go Examples
parent: Examples
nav_order: 3
---

# Go Client Examples
{: .no_toc }

Examples demonstrating the DataCat Go client library.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

The Go client library provides a native Go interface to DataCat with support for daemon-based logging for process isolation and crash detection.

**Location**: `examples/go-client-example/`

---

## Prerequisites

- Go 1.21+
- DataCat server running on http://localhost:9090

```bash
# Start the server
.\scripts\run-server.ps1
```

---

## Basic Example

**File**: `examples/go-client-example/main.go`

Demonstrates basic usage of the Go client library.

### Features

- Session creation with daemon
- Event logging
- Metric logging
- State updates
- Automatic daemon lifecycle management

### Run

```bash
cd examples/go-client-example
go run main.go
```

---

## Code Walkthrough

### Import the Client

```go
import (
    "github.com/OriginalDaemon/datacat/client"
)
```

### Create a Client with Daemon

```go
// Create client with automatic daemon startup
c, err := client.NewClient(
    "http://localhost:9090",
    client.WithDaemon(),  // Enable daemon mode
)
if err != nil {
    log.Fatal(err)
}
defer c.Close()  // Cleanup daemon on exit
```

### Create a Session

```go
session, err := c.CreateSession(client.SessionOptions{
    Product: "MyGoApp",
    Version: "1.0.0",
})
if err != nil {
    log.Fatal(err)
}
defer session.End()
```

### Log Events

```go
// Simple event
err = session.LogEvent("app_started", nil)

// Event with data
err = session.LogEvent("user_action", map[string]interface{}{
    "action": "click",
    "button": "submit",
})

// Event with level
err = session.LogEvent("error_occurred", map[string]interface{}{
    "error": "connection timeout",
}, client.WithLevel("error"))
```

### Log Metrics

```go
// Simple metric (gauge)
err = session.LogMetric("cpu_percent", 45.2)

// Metric with tags
err = session.LogMetric("memory_mb", 128.5,
    client.WithTags([]string{"system", "monitoring"}))

// Metric with unit
err = session.LogMetric("temperature", 72.5,
    client.WithUnit("celsius"))
```

### Update State

```go
// Update state with nested data
err = session.UpdateState(map[string]interface{}{
    "status": "running",
    "config": map[string]interface{}{
        "debug": true,
        "port": 8080,
    },
})
```

### Send Heartbeat

```go
// Manual heartbeat
err = session.Heartbeat()
```

---

## Complete Example

Here's a complete working example:

```go
package main

import (
    "log"
    "time"

    "github.com/OriginalDaemon/datacat/client"
)

func main() {
    // Create client with daemon
    c, err := client.NewClient(
        "http://localhost:9090",
        client.WithDaemon(),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // Create session
    session, err := c.CreateSession(client.SessionOptions{
        Product: "MyGoApp",
        Version: "1.0.0",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer session.End()

    // Log startup event
    session.LogEvent("app_started", map[string]interface{}{
        "environment": "production",
    })

    // Set initial state
    session.UpdateState(map[string]interface{}{
        "status": "initializing",
        "workers": 0,
    })

    // Simulate application work
    for i := 0; i < 10; i++ {
        // Update state
        session.UpdateState(map[string]interface{}{
            "status": "running",
            "workers": i + 1,
        })

        // Log metrics
        session.LogMetric("active_workers", float64(i+1))
        session.LogMetric("cpu_percent", 45.0 + float64(i))

        // Log event
        session.LogEvent("worker_started", map[string]interface{}{
            "worker_id": i + 1,
        })

        // Send heartbeat
        session.Heartbeat()

        time.Sleep(time.Second)
    }

    // Log shutdown event
    session.LogEvent("app_stopping", nil)
}
```

---

## Client Options

### WithDaemon

Enable daemon mode for process isolation and crash detection:

```go
c, err := client.NewClient(
    "http://localhost:9090",
    client.WithDaemon(),
)
```

**Benefits**:
- Automatic crash detection
- Process isolation
- Offline operation support
- No blocking on network errors

### WithoutDaemon

Direct connection to server (no daemon):

```go
c, err := client.NewClient(
    "http://localhost:9090",
)
```

**Use When**:
- Daemon not needed
- Server-side application
- Simplified deployment

---

## Session Options

### Product and Version

```go
session, err := c.CreateSession(client.SessionOptions{
    Product: "MyApp",
    Version: "1.0.0",
})
```

### Custom Metadata

```go
session, err := c.CreateSession(client.SessionOptions{
    Product: "MyApp",
    Version: "1.0.0",
    Metadata: map[string]interface{}{
        "environment": "production",
        "region": "us-west",
    },
})
```

---

## Metric Options

### WithTags

Add tags to metrics for filtering:

```go
session.LogMetric("request_count", 1234,
    client.WithTags([]string{"api", "v1", "production"}))
```

### WithUnit

Specify measurement unit:

```go
session.LogMetric("response_time", 0.045,
    client.WithUnit("seconds"))

session.LogMetric("memory_usage", 128.5,
    client.WithUnit("megabytes"))
```

### WithType

Specify metric type:

```go
// Gauge (default)
session.LogMetric("cpu_percent", 45.2,
    client.WithType("gauge"))

// Counter
session.LogMetric("requests_total", 1000,
    client.WithType("counter"))

// Histogram
session.LogMetric("request_duration", 0.045,
    client.WithType("histogram"))

// Timer
session.LogMetric("operation_duration", 0.123,
    client.WithType("timer"))
```

---

## Event Options

### WithLevel

Set event severity level:

```go
session.LogEvent("debug_info", data,
    client.WithLevel("debug"))

session.LogEvent("info_message", data,
    client.WithLevel("info"))

session.LogEvent("warning_alert", data,
    client.WithLevel("warning"))

session.LogEvent("error_occurred", data,
    client.WithLevel("error"))

session.LogEvent("critical_failure", data,
    client.WithLevel("critical"))
```

### WithCategory

Categorize events:

```go
session.LogEvent("database_query", data,
    client.WithCategory("database"))

session.LogEvent("api_request", data,
    client.WithCategory("api"))
```

### WithLabels

Add labels for filtering:

```go
session.LogEvent("user_action", data,
    client.WithLabels([]string{"ui", "interaction", "button"}))
```

---

## Error Handling

Always check errors:

```go
// Check session creation
session, err := c.CreateSession(options)
if err != nil {
    log.Printf("Failed to create session: %v", err)
    return
}

// Check log operations
if err := session.LogEvent("event", data); err != nil {
    log.Printf("Failed to log event: %v", err)
}

if err := session.LogMetric("metric", value); err != nil {
    log.Printf("Failed to log metric: %v", err)
}

if err := session.UpdateState(state); err != nil {
    log.Printf("Failed to update state: %v", err)
}
```

---

## Best Practices

### 1. Always Close Resources

```go
c, err := client.NewClient(url, client.WithDaemon())
if err != nil {
    log.Fatal(err)
}
defer c.Close()  // Important!

session, err := c.CreateSession(options)
if err != nil {
    log.Fatal(err)
}
defer session.End()  // Important!
```

### 2. Use Daemon Mode

```go
// Recommended: With daemon
c, err := client.NewClient(url, client.WithDaemon())

// For server-side: Without daemon
c, err := client.NewClient(url)
```

### 3. Send Heartbeats

```go
// In long-running operations
ticker := time.NewTicker(5 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        session.Heartbeat()
    case <-done:
        return
    }
}
```

### 4. Handle Errors Gracefully

```go
if err := session.LogEvent("event", data); err != nil {
    // Log but don't crash
    log.Printf("Warning: Failed to log event: %v", err)
    // Continue application logic
}
```

---

## Advanced Usage

### Concurrent Sessions

```go
// Create multiple sessions
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()

        session, err := c.CreateSession(client.SessionOptions{
            Product: "MyApp",
            Version: "1.0.0",
            Metadata: map[string]interface{}{
                "worker_id": id,
            },
        })
        if err != nil {
            log.Printf("Worker %d failed: %v", id, err)
            return
        }
        defer session.End()

        // Worker logic...
    }(i)
}

wg.Wait()
```

### Structured Logging Integration

```go
import (
    "log/slog"
    "github.com/OriginalDaemon/datacat/client"
)

// Custom slog handler
type DatacatHandler struct {
    session *client.Session
}

func (h *DatacatHandler) Handle(r slog.Record) error {
    data := make(map[string]interface{})
    r.Attrs(func(a slog.Attr) bool {
        data[a.Key] = a.Value.Any()
        return true
    })

    return h.session.LogEvent(r.Message, data,
        client.WithLevel(r.Level.String()))
}
```

---

## Troubleshooting

### "Failed to start daemon"

**Problem**: Daemon fails to start.

**Solutions**:
- Check if Go is installed: `go version`
- Verify daemon binary exists in project
- Check file permissions
- Try running without daemon mode first

### "Connection refused"

**Problem**: Cannot connect to server.

**Solutions**:
- Start the server: `.\scripts\run-server.ps1`
- Verify server URL: `http://localhost:9090`
- Check server health: `curl http://localhost:9090/health`

### "Session not found"

**Problem**: Session operations fail.

**Solutions**:
- Ensure session was created successfully
- Check if session was already ended
- Verify server is running and accessible

---

## Next Steps

- **[Go Client Library](../../client/)** - Complete client documentation
- **[API Reference](../_api/rest-api.md)** - REST API documentation
- **[Python Examples](python-examples.md)** - Python examples
- **[Architecture](../_guides/architecture.md)** - System architecture

