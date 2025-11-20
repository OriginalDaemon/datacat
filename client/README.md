# DataCat Go Client

Go client library for interacting with the DataCat REST API.

## Installation

```bash
go get github.com/OriginalDaemon/datacat/client
```

## Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/OriginalDaemon/datacat/client"
)

func main() {
    // Create client
    c := client.NewClient("http://localhost:9090")

    // Create session
    sessionID, err := c.CreateSession()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created session: %s\n", sessionID)

    // Update state
    err = c.UpdateState(sessionID, map[string]interface{}{
        "status": "running",
        "version": "1.0.0",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Log event
    err = c.LogEvent(sessionID, "user_login", map[string]interface{}{
        "username": "alice",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Log metric
    err = c.LogMetric(sessionID, "cpu_usage", 45.2, []string{"app:myapp"})
    if err != nil {
        log.Fatal(err)
    }

    // Get session
    session, err := c.GetSession(sessionID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Session: %+v\n", session)

    // End session
    err = c.EndSession(sessionID)
    if err != nil {
        log.Fatal(err)
    }
}
```

## API

### `NewClient(baseURL string) *Client`

Creates a new client instance.

### `CreateSession() (string, error)`

Creates a new session and returns its ID.

### `GetSession(sessionID string) (*Session, error)`

Retrieves session details.

### `UpdateState(sessionID string, state map[string]interface{}) error`

Updates session state with deep merge.

### `LogEvent(sessionID, name string, data map[string]interface{}) error`

Logs an event for the session.

### `LogMetric(sessionID, name string, value float64, tags []string) error`

Logs a metric for the session.

### `EndSession(sessionID string) error`

Marks the session as ended.

## Testing

```bash
go test -v -cover
```

Coverage is >85%.
