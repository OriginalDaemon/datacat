# datacat Architecture

## Architecture with Daemon (Recommended)

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application Process                      │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Python/Go Client Library                      │  │
│  │                                                             │  │
│  │  ┌───────────────┐           ┌──────────────────────────┐ │  │
│  │  │   Main App    │           │ Heartbeat Sending        │ │  │
│  │  │               │           │                          │ │  │
│  │  │ session.      │           │ Periodically sends       │ │  │
│  │  │  heartbeat()  │──────────>│ heartbeat to daemon      │ │  │
│  │  │               │           │                          │ │  │
│  │  │ session.      │           │                          │ │  │
│  │  │  update_state │           │                          │ │  │
│  │  │  log_event    │           │                          │ │  │
│  │  │  log_metric   │           │                          │ │  │
│  │  └───────┬───────┘           └──────────────────────────┘ │  │
│  │          │                                                  │  │
│  │          │ Starts subprocess on init                       │  │
│  │          │ All API calls go to daemon                      │  │
│  └──────────┼──────────────────────────────────────────────────┘  │
│             │                                                      │
│             │ HTTP to localhost:8079                               │
└─────────────┼──────────────────────────────────────────────────────┘
              │
              │ Daemon is subprocess of application
              │
┌─────────────▼──────────────────────────────────────────────────────┐
│             Local Daemon Subprocess (http://localhost:8079)        │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                   Daemon Responsibilities                   │  │
│  │                                                              │  │
│  │  1. Batching: Collects state/events/metrics for 5s          │  │
│  │  2. Smart Filtering: Only sends changed state               │  │
│  │  3. Heartbeat Monitoring: Detects hung applications         │  │
│  │  4. Parent Process Monitoring: Detects crashes/exits        │  │
│  │  5. Network Optimization: 10-100x reduction                 │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  Endpoints:                                                        │
│  • POST /register          - Create session (+ track parent PID)  │
│  • POST /state             - Buffer state update                  │
│  • POST /event             - Buffer event                         │
│  • POST /metric            - Buffer metric                        │
│  • POST /heartbeat         - Record heartbeat                     │
│  • POST /end               - Flush & end session                  │
│                                                                     │
│  Background Workers:                                               │
│  • Batch sender (every 5s): Flushes buffered data to server       │
│  • Heartbeat monitor (every 5s): Checks for hung apps             │
│  • Parent monitor (every 5s): Checks if parent crashed            │
└─────────────┬──────────────────────────────────────────────────────┘
              │
              │ HTTP/JSON to remote server
              │ Only sends when data changes or batch interval
              │
┌─────────────▼──────────────────────────────────────────────────────┐
│                  Go REST API Service (Server)                      │
│                  (http://localhost:8080 or remote)                 │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                      Endpoints                              │  │
│  │                                                              │  │
│  │  POST   /api/sessions              - Create session         │  │
│  │  GET    /api/sessions/{id}         - Get session            │  │
│  │  POST   /api/sessions/{id}/state   - Update state           │  │
│  │  POST   /api/sessions/{id}/events  - Log event              │  │
│  │  POST   /api/sessions/{id}/metrics - Log metric             │  │
│  │  POST   /api/sessions/{id}/end     - End session            │  │
│  │  GET    /api/grafana/sessions      - Export all sessions    │  │
│  └────────────────────────────────────────────────────────────┘  │
│                              │                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │           BadgerDB Persistent Storage                       │  │
│  │                                                              │  │
│  │  Sessions persisted to disk:                                │  │
│  │  {                                                           │  │
│  │    "session-id-1": {                                         │  │
│  │      id: "...",                                              │  │
│  │      created_at: "...",                                      │  │
│  │      active: true,                                           │  │
│  │      state: {                                                │  │
│  │        window_state: {                                       │  │
│  │          open: ["w1", "w2"],                                 │  │
│  │          active: "w1"                                        │  │
│  │        },                                                    │  │
│  │        memory: { footprint_mb: 75 }                          │  │
│  │      },                                                      │  │
│  │      events: [...],                                          │  │
│  │      metrics: [...]                                          │  │
│  │    }                                                         │  │
│  │  }                                                           │  │
│  │                                                              │  │
│  │  Features:                                                   │  │
│  │  • Thread-safe with mutex                                    │  │
│  │  • Deep merge for nested state updates                       │  │
│  │  • Preserves hierarchy on partial updates                    │  │
│  │  • Survives server restarts                                  │  │
│  └────────────────────────────────────────────────────────────┘  │
│                              │                                    │
└──────────────────────────────┼────────────────────────────────────┘
                               │
                               │ HTTP/JSON
                               │
┌──────────────────────────────▼────────────────────────────────────┐
│                      Web UI / Grafana                             │
│                                                                   │
│  Queries via JSON API data source:                               │
│  • Active sessions (never ended)                                 │
│  • Hung sessions (event = "application_appears_hung")            │
│  • Crashed sessions (event = "parent_process_crashed")           │
│  • Reliability metrics                                           │
│  • Time-series metrics                                           │
│  • Event timelines                                               │
└───────────────────────────────────────────────────────────────────┘
```

## Daemon Features

### 1. Batching & Network Optimization
- Collects data for 5 seconds before sending
- Combines multiple state updates into one
- **10-100x reduction** in network requests

### 2. Smart State Filtering
- Tracks last known state
- Only sends state updates that actually changed
- Deep merge comparison

### 3. Heartbeat Monitoring
- Daemon monitors heartbeats from application
- If no heartbeat for 60s, logs "application_appears_hung"
- If heartbeat resumes, logs "application_recovered"

### 4. Parent Process Monitoring  
- Daemon tracks parent process PID
- Checks every 5s if parent is still alive
- If parent crashes/exits abnormally, logs "parent_process_crashed"
- Immediately flushes data to server

## Architecture without Daemon (Simple Mode)

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application                              │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Python/Go Client Library                      │  │
│  │              (use_daemon=False)                            │  │
│  │                                                             │  │
│  │  session.update_state()  ──────────────────────────────>   │  │
│  │  session.log_event()     ──────────────────────────────>   │  │
│  │  session.log_metric()    ──────────────────────────────>   │  │
│  │                                                             │  │
│  │  Direct HTTP calls to server (no batching)                 │  │
│  └─────────────────────────────────────────────────────────────┘  │
└─────────────┬──────────────────────────────────────────────────────┘
              │
              │ HTTP/JSON directly to server
              │
┌─────────────▼──────────────────────────────────────────────────────┐
│                  Go REST API Service (Server)                      │
│                  (http://localhost:8080)                           │
└────────────────────────────────────────────────────────────────────┘
```

## State Update Flow (Deep Merge)

Initial state:
{
  "window_state": { "open": ["w1"], "active": "w1" },
  "memory": { "footprint_mb": 50 }
}

Update 1: { "window_state": { "open": ["w1", "w2"] } }
Result:
{
  "window_state": { "open": ["w1", "w2"], "active": "w1" },  ← "active" preserved
  "memory": { "footprint_mb": 50 }
}

Update 2: { "memory": { "footprint_mb": 75 } }
Result:
{
  "window_state": { "open": ["w1", "w2"], "active": "w1" },  ← preserved
  "memory": { "footprint_mb": 75 }
}

## Heartbeat Flow (With Daemon)

```
1. App starts with daemon:
   - Client library starts daemon subprocess
   - Daemon tracks parent PID
   
2. App creates session:
   - session = create_session(..., use_daemon=True)
   - Client sends parent PID to daemon
   
3. App sends heartbeats:
   - session.heartbeat() sends to daemon
   - Daemon updates LastHeartbeat timestamp
   
4. Daemon monitors (every 5 seconds):
   a) Heartbeat monitoring:
      - If (now - LastHeartbeat) > 60s && !HangLogged:
        -> Logs "application_appears_hung" event
        -> Sets HangLogged = true
      - If heartbeat received after hang:
        -> Logs "application_recovered" event
        
   b) Parent process monitoring:
      - If parent PID not running && !CrashLogged:
        -> Logs "parent_process_crashed" event
        -> Sets CrashLogged = true
        -> Immediately flushes to server
        
5. App ends normally:
   - session.end()
   - Daemon flushes remaining data
   - Daemon stops (child process cleanup)
```

## Benefits of Daemon Architecture

### Network Efficiency
- **Before:** 1000 state updates = 1000 HTTP requests
- **After:** 1000 state updates batched into ~200 HTTP requests (10-100x reduction)

### Crash Detection
- **Before:** If app crashes, no notification sent
- **After:** Daemon detects parent crash and logs event immediately

### Hang Detection
- **Before:** Client-side thread monitors, but requires app to be responsive
- **After:** Daemon monitors independently, works even if app hangs

### Smart Filtering
- **Before:** All state updates sent
- **After:** Only changed state sent (daemon tracks last known state)
```
