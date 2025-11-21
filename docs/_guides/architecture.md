---
layout: default
title: Architecture
parent: Guides
nav_order: 2
---

# DataCat Architecture
{: .no_toc }

Understanding the DataCat system architecture and design principles.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## System Overview

DataCat uses a multi-tier architecture optimized for efficiency and reliability:

```
Application → Local Daemon → DataCat Server → BadgerDB
                ↓                    ↓
         Crash Detection    State Management
         Hang Detection     Data Persistence
         Batching
```

### Core Components

1. **Client Libraries** (Python/Go) - Application integration
2. **Local Daemon** - Batching and monitoring subprocess
3. **REST API Server** - Central service and persistence
4. **Web Dashboard** - Visualization and exploration
5. **BadgerDB** - Embedded key-value database

---

## Architecture Patterns

### Pattern 1: Daemon Mode (Recommended)

The recommended architecture uses a local daemon subprocess for intelligent batching:

```
┌─────────────────────────────────────┐
│      Application Process             │
│                                      │
│  ┌──────────────────────────────┐  │
│  │   Python/Go Client Library    │  │
│  │   session.update_state()      │  │
│  │   session.log_event()         │  │
│  │   session.heartbeat()         │  │
│  └──────────┬───────────────────┘  │
│             │ HTTP :8079            │
└─────────────┼───────────────────────┘
              │ subprocess
┌─────────────▼───────────────────────┐
│      Local Daemon Process           │
│                                      │
│  • Batches requests (5s intervals)  │
│  • Filters unchanged state          │
│  • Monitors parent process          │
│  • Detects hangs/crashes            │
│  • 10-100x network reduction        │
└──────────┬──────────────────────────┘
           │ HTTP to server
┌──────────▼──────────────────────────┐
│     DataCat Server (Remote)         │
│                                      │
│  • REST API endpoints               │
│  • BadgerDB persistence             │
│  • State management                 │
│  • Deep merge logic                 │
└─────────────────────────────────────┘
```

**Benefits:**
- **10-100x network traffic reduction** through batching
- **Smart filtering** - only sends changed state
- **Automatic crash detection** - monitors parent process
- **Hang detection** - tracks heartbeat timeouts
- **Offline resilience** - queues requests when server unavailable

### Pattern 2: Direct Mode

For simpler use cases, clients can connect directly to the server:

```
┌─────────────────────────────────────┐
│      Application Process             │
│                                      │
│  ┌──────────────────────────────┐  │
│  │   Python/Go Client Library    │  │
│  │   (Direct connection)         │  │
│  └──────────┬───────────────────┘  │
└─────────────┼───────────────────────┘
              │ HTTP :9090
┌─────────────▼───────────────────────┐
│     DataCat Server                   │
│  • No batching                       │
│  • No crash detection                │
│  • Simple request/response           │
└─────────────────────────────────────┘
```

**Use Cases:**
- Quick prototyping
- Debugging
- Server-side applications
- When overhead of daemon not needed

---

## Component Deep Dive

### Client Libraries

**Python Client** (`python/datacat.py`)
- Python 2.7+ and 3.x compatible
- Automatic daemon management
- Background heartbeat monitoring
- Exception tracking with stack traces
- Type hints for better IDE support

**Go Client** (`client/client.go`)
- Type-safe interface
- Support for both daemon and direct modes
- Timeout handling
- Error propagation

**Key Features:**
- Session lifecycle management
- Deep merge state updates
- Event and metric logging
- Heartbeat monitoring
- Exception capture

### Local Daemon

**Purpose:** Optimize network usage and add intelligence to client-side operations.

**Responsibilities:**

1. **Batching** - Collects API calls for 5 seconds before sending
2. **Smart Filtering** - Only sends state that has changed
3. **Parent Monitoring** - Detects when parent process crashes
4. **Heartbeat Tracking** - Identifies hung applications
5. **Retry Logic** - Queues failed requests for retry

**Endpoints:**
- `POST /register` - Create session with parent PID tracking
- `POST /state` - Buffer state update
- `POST /event` - Buffer event
- `POST /metric` - Buffer metric
- `POST /heartbeat` - Record heartbeat timestamp
- `POST /end` - Flush buffers and end session
- `GET /session` - Get session details (from buffer or server)
- `GET /sessions` - Get all sessions

**Background Workers:**
- **Batch Sender** (5s interval) - Sends buffered data
- **Heartbeat Monitor** (5s interval) - Checks for hung apps
- **Parent Monitor** (5s interval) - Checks parent process status

### REST API Server

**Technology Stack:**
- **Language:** Go
- **Database:** BadgerDB (embedded)
- **Router:** Native Go http.ServeMux

**Key Endpoints:**
```
POST   /api/sessions              Create session
GET    /api/sessions/{id}         Get session details
POST   /api/sessions/{id}/state   Update state (deep merge)
POST   /api/sessions/{id}/events  Log event
POST   /api/sessions/{id}/metrics Log metric
POST   /api/sessions/{id}/end     End session
POST   /api/sessions/{id}/heartbeat Update heartbeat
GET    /api/data/sessions         Export all sessions
```

**Features:**
- **Deep Merge State Updates** - Preserves nested data
- **Session Lifecycle** - Active, ended, crashed, hung states
- **Machine Tracking** - MAC address-based machine IDs
- **Crash Detection** - Identifies crashed sessions
- **Data Retention** - Configurable cleanup policies

### BadgerDB Storage

**Why BadgerDB?**
- **Embedded** - No separate database process
- **Fast** - LSM tree design for writes
- **Persistent** - Data survives restarts
- **Simple** - Key-value interface
- **Go-native** - Excellent Go integration

**Data Model:**
```
Key: "session:<session-id>"
Value: {
  "id": "uuid",
  "created_at": "timestamp",
  "active": boolean,
  "crashed": boolean,
  "hung": boolean,
  "suspended": boolean,
  "state": {...},
  "events": [...],
  "metrics": [...],
  "machine_id": "mac-address",
  "last_heartbeat": "timestamp"
}
```

### Web Dashboard

**Technology:**
- **Backend:** Go http server
- **Frontend:** htmx for dynamic updates
- **Charts:** Chart.js for visualization
- **Styling:** Custom CSS with dark mode

**Features:**
- Session browser with filtering
- Real-time metrics visualization
- Event timeline
- State history viewer
- Crash/hang detection display

---

## Data Flow

### Session Creation Flow

1. **Client** calls `create_session(product, version)`
2. **Daemon** receives request at `/register`
3. **Daemon** tracks parent process PID
4. **Daemon** forwards to **Server** at `/api/sessions`
5. **Server** creates session in BadgerDB
6. **Server** returns session ID
7. **Daemon** caches session ID
8. **Client** receives session ID

### State Update Flow

1. **Client** calls `session.update_state({...})`
2. **Daemon** buffers update in memory
3. **Daemon** waits for batch interval (5s)
4. **Daemon** compares with last sent state
5. **Daemon** sends only changed fields
6. **Server** deep merges with existing state
7. **Server** persists to BadgerDB

### Crash Detection Flow

1. **Daemon** monitors parent process every 5s
2. **Daemon** detects parent process terminated
3. **Daemon** immediately sends crash event
4. **Server** marks session as crashed
5. **Server** logs `session_crashed_detected` event
6. **Dashboard** displays crashed status

### Hang Detection Flow

1. **Client** sends heartbeats via `session.heartbeat()`
2. **Daemon** records heartbeat timestamp
3. **Daemon** checks heartbeat age every 5s
4. **Daemon** detects heartbeat timeout (60s default)
5. **Daemon** logs `application_appears_hung` event
6. **Server** marks session as hung
7. **Client** resumes heartbeats
8. **Daemon** logs `application_recovered` event

---

## Performance Characteristics

### Network Optimization

**Without Daemon (Direct Mode):**
- 1 request per API call
- No batching
- Full state sent each time

**With Daemon (Recommended):**
- Batched every 5 seconds
- Only changed state sent
- **10-100x reduction** in network traffic

**Example:**
```
Application makes 100 state updates in 10 seconds

Direct Mode:    100 HTTP requests
Daemon Mode:    2 HTTP requests (2 batches)

Network Reduction: 50x
```

### Database Performance

**BadgerDB Characteristics:**
- **Writes:** O(log n) with LSM tree
- **Reads:** O(log n) for single key
- **Scans:** Sequential iteration
- **Memory:** Configurable cache size

**Typical Performance:**
- Session creation: <1ms
- State update: <2ms
- Query single session: <1ms
- Query all sessions: <10ms for 1000 sessions

---

## Scalability Considerations

### Vertical Scaling

The server is single-threaded with mutex protection but highly efficient:
- **100+ concurrent sessions** on modest hardware
- **1000+ sessions** with proper tuning
- **10,000+ sessions** possible with optimization

### Horizontal Scaling

Current architecture is single-server. For horizontal scaling:
- Load balancer in front of multiple servers
- Shared database (replace BadgerDB with distributed store)
- Session affinity or distributed state management

### Data Retention

Configure retention to manage database size:
```json
{
  "retention_days": 365,
  "cleanup_interval_hours": 24
}
```

---

## Security Considerations

### Network Security

- **Local Daemon:** Binds to localhost only
- **Server:** Can bind to specific interface
- **No Authentication:** Currently assumes trusted network
- **Future:** Add API keys or OAuth

### Data Security

- **No Encryption:** Data stored plaintext in BadgerDB
- **File Permissions:** Database files inherit system permissions
- **Future:** Add encryption at rest

### Process Security

- **Daemon Isolation:** Runs as subprocess
- **No Privilege Escalation:** Inherits parent permissions
- **Clean Shutdown:** Handles SIGTERM/SIGINT

---

## Reliability Features

### Crash Detection

**Mechanism:**
1. Daemon tracks parent process PID
2. Monitors process existence every 5s
3. Detects abnormal termination
4. Logs crash event immediately

**States:**
- **Active** - Normal operation
- **Suspended** - No heartbeats (sleep/pause)
- **Crashed** - Parent process terminated
- **Ended** - Normal shutdown

### Data Persistence

**BadgerDB Guarantees:**
- Write-ahead logging
- Atomic operations
- Crash recovery
- ACID properties

**Backup Strategy:**
1. Stop server
2. Copy `datacat_data/` directory
3. Restart server

### Error Handling

**Client Errors:**
- Timeout after 30s
- Retry logic in daemon
- Graceful degradation

**Server Errors:**
- Structured error messages
- HTTP status codes
- Detailed logging

---

## Monitoring DataCat

### Health Checks

```bash
# Check server health
curl http://localhost:9090/api/data/sessions

# Check daemon health
curl http://localhost:8079/sessions
```

### Logs

**Server Logs:**
```
2024-01-01 12:00:00 Loaded 0 sessions from database
2024-01-01 12:00:00 Server listening on :9090
2024-01-01 12:00:05 Session created: abc-123
2024-01-01 12:00:10 Session ended: abc-123
```

**Daemon Logs:**
```
2024-01-01 12:00:00 Daemon listening on :8079
2024-01-01 12:00:05 Registered session: local-session-1
2024-01-01 12:00:10 Batch sent: 5 updates
2024-01-01 12:00:15 Parent process alive: PID 1234
```

### Metrics

Monitor these metrics:
- Active sessions count
- Crashed sessions count
- Hung sessions count
- Average session duration
- Events per session
- Metrics per session
- Database size

---

## Next Steps

- [Daemon Batching](daemon-batching.html) - Deep dive into batching logic
- [Crash Detection](crash-detection.html) - Understanding crash detection
- [State Management](state-management.html) - Deep merge internals
- [Deployment Guide](deployment.html) - Production deployment
