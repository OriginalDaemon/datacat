# datacat Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application                              │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                 Python Client                              │  │
│  │                                                             │  │
│  │  ┌───────────────┐  ┌─────────────────────────────────┐  │  │
│  │  │   Main App    │  │   Heartbeat Monitor (Thread)    │  │  │
│  │  │               │  │                                   │  │  │
│  │  │ session.      │  │  Runs independently in           │  │  │
│  │  │  heartbeat()  │──┼─>background                      │  │  │
│  │  │               │  │                                   │  │  │
│  │  │ session.      │  │  If no heartbeat for 60s:        │  │  │
│  │  │  update_state │  │   -> logs "app_appears_hung"     │  │  │
│  │  │  log_event    │  │                                   │  │  │
│  │  │  log_metric   │  │                                   │  │  │
│  │  └───────┬───────┘  └──────────────┬──────────────────┘  │  │
│  │          │                         │                      │  │
│  └──────────┼─────────────────────────┼──────────────────────┘  │
│             │                         │                          │
└─────────────┼─────────────────────────┼──────────────────────────┘
              │                         │
              │ HTTP/JSON              │ HTTP/JSON
              │                         │
┌─────────────▼─────────────────────────▼──────────────────────────┐
│                     Go REST API Service                           │
│                     (http://localhost:8080)                       │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                      Endpoints                            │   │
│  │                                                            │   │
│  │  POST   /api/sessions              - Create session       │   │
│  │  GET    /api/sessions/{id}         - Get session          │   │
│  │  POST   /api/sessions/{id}/state   - Update state         │   │
│  │  POST   /api/sessions/{id}/events  - Log event            │   │
│  │  POST   /api/sessions/{id}/metrics - Log metric           │   │
│  │  POST   /api/sessions/{id}/end     - End session          │   │
│  │  GET    /api/grafana/sessions      - Export all sessions  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              In-Memory Session Store                      │   │
│  │                                                            │   │
│  │  Sessions Map:                                             │   │
│  │  {                                                         │   │
│  │    "session-id-1": {                                       │   │
│  │      id: "...",                                            │   │
│  │      created_at: "...",                                    │   │
│  │      active: true,                                         │   │
│  │      state: {                                              │   │
│  │        window_state: {                                     │   │
│  │          open: ["w1", "w2"],                               │   │
│  │          active: "w1"                                      │   │
│  │        },                                                  │   │
│  │        memory: { footprint_mb: 75 }                        │   │
│  │      },                                                    │   │
│  │      events: [...],                                        │   │
│  │      metrics: [...]                                        │   │
│  │    }                                                       │   │
│  │  }                                                         │   │
│  │                                                            │   │
│  │  Features:                                                 │   │
│  │  • Thread-safe with mutex                                  │   │
│  │  • Deep merge for nested state updates                     │   │
│  │  • Preserves hierarchy on partial updates                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                    │
└──────────────────────────────┼────────────────────────────────────┘
                               │
                               │ HTTP/JSON
                               │
┌──────────────────────────────▼────────────────────────────────────┐
│                          Grafana                                  │
│                                                                   │
│  Queries via JSON API data source:                               │
│  • Active sessions (never ended)                                 │
│  • Hung sessions (last event = "application_appears_hung")       │
│  • Reliability metrics                                           │
│  • Time-series metrics                                           │
│  • Event timelines                                               │
└───────────────────────────────────────────────────────────────────┘

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

## Heartbeat Flow

1. App starts: session.start_heartbeat_monitor(timeout=60)
2. Monitor thread starts, records last_heartbeat = now
3. App loop: session.heartbeat() every few seconds
4. Monitor checks every 5 seconds:
   - If (now - last_heartbeat) > 60s:
     - Log "application_appears_hung" event
5. If app resumes heartbeats after hanging:
   - Log "application_recovered" event
6. App ends: session.end()
   - Stops monitor thread
   - Marks session as inactive
```
