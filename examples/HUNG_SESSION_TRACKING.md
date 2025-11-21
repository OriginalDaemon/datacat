# Hung Session Tracking & Filtering

DataCat automatically detects when applications stop responding (hang) and provides powerful filtering to analyze different hang scenarios.

## How It Works

### 1. Hang Detection

The daemon monitors heartbeats from the application:
- If no heartbeat received for 60 seconds → Logs `application_appears_hung` event
- Session marked as `Hung=true` on the server

### 2. Recovery Detection

When the application resumes sending heartbeats:
- Daemon logs `application_recovered` event
- Session marked as `Hung=false` (no longer hung)

### 3. Crash While Hung

If the application crashes while hung:
- Session stays `Hung=true`
- Session also marked as `Crashed=true`
- Distinguishes "hung then crashed" from clean errors

## Session States

### Hung Status (Flag)
- **True**: Application is currently not responding
- **False**: Application is responding normally
- **Note**: This flag changes dynamically as the app hangs/recovers

### Event History
The server tracks all hang/recovery events, allowing you to search for:
- Sessions that were hung at some point (even if recovered)
- Patterns of repeated hangs
- Duration of hangs

## Filtering Options

The web UI provides specialized filters for analyzing hung sessions:

### 1. **Currently Hung**
```
Filter: Status → Currently Hung
```
Shows sessions that are **right now** not responding.

**Use case**: Find applications that are stuck right now and may need intervention.

### 2. **Ever Hung (Any)**
```
Filter: Status → Ever Hung (Any)
```
Shows **all sessions** that experienced a hang at any point, regardless of current status.

**Use case**: Find which applications have responsiveness issues, even if they recovered.

### 3. **Hung & Recovered**
```
Filter: Status → Hung & Recovered
```
Shows sessions that:
- Had an `application_appears_hung` event
- Had an `application_recovered` event
- Are no longer hung

**Use case**: Analyze **transient hangs** - find what causes temporary freezes that resolve themselves.

**Example scenarios**:
- Long GC pauses
- Blocking I/O operations
- Temporary resource contention
- Deadlocks that eventually resolve

### 4. **Hung When Ended**
```
Filter: Status → Hung When Ended
```
Shows sessions that:
- Had a hang event
- Ended normally (graceful shutdown)
- Were not crashed

**Use case**: Applications that **froze, then user closed normally**. The hang wasn't fatal.

**Example scenarios**:
- Application hung during long operation
- User waited for it to finish (or manually recovered it)
- Application eventually responded and shut down cleanly

### 5. **Hung When Crashed** ⚠️
```
Filter: Status → Hung When Crashed
```
Shows sessions that:
- Had a hang event
- Crashed (machine came back but session didn't resume)

**Use case**: **Critical failure mode** - applications that froze permanently and had to be force-killed.

**Example scenarios**:
- Deadlocks requiring process termination
- Infinite loops causing CPU spike
- Resource exhaustion leading to kill
- User forced termination (Task Manager, kill -9)

## Analyzing Hung Sessions

### Scenario Analysis

#### Transient Performance Issue
```
Filter: Hung & Recovered
Look for: Multiple sessions from same product/version
Action: Investigate what operation was running during hang
```
**Diagnosis**: Likely a performance bottleneck that eventually completes.

#### Fatal Hang (Permanent Freeze)
```
Filter: Hung When Crashed
Look for: Common state/operations before hang
Action: Review state history and events leading to hang
```
**Diagnosis**: Likely a deadlock or infinite loop requiring code fix.

#### User Patience Issues
```
Filter: Hung When Ended
Look for: Short-lived sessions with quick hang→end
Action: Check if operation is just slow or actually hung
```
**Diagnosis**: May be user impatience rather than actual bug.

### Timeline Analysis

For each hung session, the timeline shows:
1. **Before**: Normal heartbeats and activity
2. **Hang Event**: `application_appears_hung` marker
3. **During**: Gap in heartbeats (red zone)
4. **After**: Either:
   - `application_recovered` (green) - Resumed
   - Session end (gray) - Closed normally
   - Machine return without session (red) - Crashed

## Status Badge Display

Sessions show status with multiple badges:

```
[Active] [Hung]           - Currently running but not responding
[Crashed] [Hung]          - Crashed while hung (force-killed)
[Suspended]               - Sleeping, not hung
[Ended]                   - Ended normally (not hung)
```

The **Hung** badge only appears when `Hung=true` (currently hung or was hung when crashed).

## Event Integration

### Hang Event
```json
{
  "name": "application_appears_hung",
  "data": {
    "last_heartbeat": "2025-11-21T14:30:00Z",
    "seconds_since_heartbeat": 65
  }
}
```

### Recovery Event
```json
{
  "name": "application_recovered",
  "data": {}
}
```

## Best Practices

### 1. Regular Monitoring
Set up alerts for:
- `Hung When Crashed` count > threshold
- `Currently Hung` sessions lasting > X minutes
- Repeated `Hung & Recovered` from same machine

### 2. Pattern Analysis
Use filters to identify:
- Products/versions with frequent hangs
- Machines with repeated hang issues
- Time patterns (hangs during backups, etc.)

### 3. State Correlation
When investigating hangs, check:
- **State before hang**: What operation was running?
- **Metrics before hang**: Memory/CPU patterns?
- **Events before hang**: Any errors logged?

### 4. Reproduction
For `Hung & Recovered` sessions:
- Note the recovery time (hang duration)
- Check state changes during hang
- Look for patterns in successful recoveries

## Integration with Heartbeat Monitoring

Your application should:

```python
# Start heartbeat monitoring
session.start_heartbeat_monitor(timeout=60)

# Send heartbeats regularly
while running:
    session.heartbeat()  # Call more frequently than 60s
    do_work()
    time.sleep(5)
```

The daemon will automatically:
1. Detect when heartbeats stop
2. Log the hang event after 60s
3. Detect when heartbeats resume
4. Log the recovery event

## Limitations

### False Positives
- **Intentional pauses**: Use `pause_heartbeat_monitoring()` for planned delays
- **System sleep**: Will show as "Suspended", not "Hung"
- **Debugging**: Breakpoints will trigger hang detection

### False Negatives
- **Busy loops**: CPU-intensive operations that call `heartbeat()` won't be detected
- **Partial hangs**: Only main thread frozen, heartbeat thread still running

### Timeout Configuration
- Default: 60 seconds
- Too short: More false positives during slow operations
- Too long: Delayed detection of real hangs

## Example Queries

### Find products with most hangs
```
1. Filter: Ever Hung (Any)
2. Group by: Product/Version
3. Sort by: Count descending
```

### Find critical failures
```
1. Filter: Hung When Crashed
2. Time range: Last week
3. Look for: Patterns in state/events
```

### Analyze recovery times
```
1. Filter: Hung & Recovered
2. Timeline: Measure time between hang and recovery events
3. Identify: Sessions with quick vs. slow recovery
```

### Debug specific product
```
1. Product filter: MyApp
2. Status: Ever Hung (Any)
3. State filter: Look for common operations
```

## Future Enhancements

Potential improvements:
- **Hang duration tracking**: Measure exact hang time
- **Hang frequency**: Track sessions with multiple hang/recovery cycles
- **Auto-recovery rate**: Percentage of hangs that recovered vs. crashed
- **Hang severity**: Distinguish minor delays from critical freezes
- **Stack traces**: Capture thread states during hang (requires OS integration)

