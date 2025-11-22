# Session Lifecycle Improvements

## Summary

This document describes the improvements made to properly handle daemon lifecycle, crash detection, and session status reporting.

## Issues Fixed

### 1. Daemon Not Shutting Down When Sessions End ✅

**Problem:** When a client application ended a session normally, the daemon would continue running indefinitely.

**Solution:**
- Added automatic daemon shutdown when all sessions are removed
- Daemon now monitors session count and initiates graceful shutdown when count reaches zero
- Added shutdown channel and HTTP server graceful shutdown support
- Applies to both normal session end and crash scenarios

**Implementation:**
- `cmd/datacat-daemon/main.go`: Added `shutdownChan`, `httpServer`, and `shutdownMonitor()` function
- After ending or crashing a session, daemon checks if any sessions remain
- If no sessions remain, triggers shutdown after 2-second delay

### 2. Crash Detection and Reporting ✅

**Problem:** When a client process crashed (terminated without calling `session.end()`), the daemon would detect it but only log an event - it wouldn't properly mark the session as crashed on the server.

**Solution:**
- Added new `/api/sessions/{id}/crash` endpoint to explicitly mark sessions as crashed
- Daemon now calls crash endpoint instead of end endpoint when detecting parent process termination
- Crash endpoint sets `crashed=true` flag and logs a `session_crashed_detected` event
- Distinguishes between normal termination (`session.end()`) and abnormal termination (crash)

**Implementation:**
- `cmd/datacat-server/main.go`:
  - Added `CrashSession()` method to Store
  - Added `/crash` endpoint handler
  - Crash marks session with `crashed=true`, `active=false`, and logs crash event

- `cmd/datacat-daemon/main.go`:
  - Modified `checkParentProcess()` to call `/crash` endpoint instead of `/end`
  - Added `retryCrashSession()` for retry queue handling
  - Added "crash" operation type to retry queue processor

### 3. Improved Parent Process Detection ✅

**Problem:** Parent process crash detection had limitations, especially on Windows.

**Solution:**
- Improved `isProcessRunning()` function for better cross-platform support
- Better Windows process checking (though still has OS limitations)
- Parent process check now properly ends the session when crash detected

**Implementation:**
- `cmd/datacat-daemon/main.go`: Improved `isProcessRunning()` with better Windows handling
- Still some limitations on Windows due to OS API constraints

### 4. UI Status Badge Consistency ✅

**Problem:** Sessions list view only showed "Active" or "Ended" badges, while detail views showed full status including Crashed, Suspended, and Hung.

**Solution:**
- Made all session list views consistent with complete status badge display
- Now shows: Active (green), Crashed (red), Suspended (orange), Ended (gray), Hung (yellow)

**Implementation:**
- `cmd/datacat-web/main.go`: Updated `handleSessionsTable()` to show all status badges
- `cmd/datacat-web/templates/sessions.html`: Added full status badge logic
- `cmd/datacat-web/templates/sessions_enhanced.html`: Added full status badge logic

## API Changes

### New Endpoint: Mark Session as Crashed

**Endpoint:** `POST /api/sessions/{id}/crash`

**Purpose:** Mark a session as crashed due to abnormal termination

**Request Body:**
```json
{
  "reason": "parent_process_terminated"
}
```

**Behavior:**
- Sets `crashed` flag to `true`
- Sets `active` to `false`
- Sets `ended_at` timestamp
- Logs `session_crashed_detected` event
- Persists to database

**Used by:** Daemon when detecting parent process termination without proper `session.end()` call

## Session Lifecycle States

### Normal Flow
```
CREATE → ACTIVE → END → (daemon shutdown)
                  ↑
                  └── Client calls session.end()
```

### Crash Flow
```
CREATE → ACTIVE → CRASH → (daemon shutdown)
                   ↑
                   └── Parent process dies without calling session.end()
```

### Hang Flow (Active but Unresponsive)
```
CREATE → ACTIVE → ACTIVE + HUNG → (may recover or crash)
                   ↑
                   └── No heartbeats for 60+ seconds
```

### Suspend Flow (Machine Sleep)
```
CREATE → ACTIVE → SUSPENDED → ACTIVE (on wake)
                   ↑            or
                   ↓          CRASHED (if new session starts)
             No daemon heartbeats for 60+ seconds
```

## Status Flags

- **Active:** Session is running and sending heartbeats (daemon → server)
- **Ended:** Session terminated normally via `session.end()`
- **Crashed:** Session terminated abnormally (parent process died)
- **Suspended:** Daemon stopped sending heartbeats (likely machine sleep)
- **Hung:** Application stopped sending heartbeats to daemon (unresponsive app)

## Testing

### Manual Test for Crash Detection

Run the provided test script:

```bash
python examples/test_crash_detection.py
```

This script:
1. Creates a session
2. Logs some activity
3. Exits abruptly without calling `session.end()` (simulates crash)
4. Daemon detects crash within 5 seconds
5. Session marked as "Crashed" in web UI

### Expected Behavior

After running the test:
1. Check web UI at `http://localhost:8080`
2. Find the session (Product: "CrashTest")
3. Status should show **Crashed** badge (red)
4. Events should include `parent_process_crashed` event
5. Daemon should have shut down automatically

## Files Modified

### Server
- `cmd/datacat-server/main.go` - Added crash endpoint and CrashSession method

### Daemon
- `cmd/datacat-daemon/main.go`:
  - Added shutdown monitoring and graceful shutdown
  - Modified parent process checking to call crash endpoint
  - Added crash retry handling
  - Improved Windows process detection

### Web UI
- `cmd/datacat-web/main.go` - Updated status badge rendering
- `cmd/datacat-web/templates/sessions.html` - Added full status badges
- `cmd/datacat-web/templates/sessions_enhanced.html` - Added full status badges

### Documentation
- `docs/_api/sessions.md` - Added crash endpoint documentation
- `examples/test_crash_detection.py` - Added crash detection test script

## Breaking Changes

**None.** All changes are backwards compatible. The new `/crash` endpoint is additive.

## Migration Notes

Existing sessions will continue to work as before. The improvements only affect:
- New session terminations (normal vs crashed)
- Daemon lifecycle management
- UI status display consistency

No database migrations required - the `crashed` flag already exists in the session model.

