---
layout: default
title: Machine Tracking
parent: Guides
nav_order: 6
---

# Machine Tracking & Crash Detection
{: .no_toc }

DataCat automatically tracks which machine each session runs on and uses this information to distinguish between system sleep/hibernate and actual application crashes.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## How It Works

### 1. Machine Identification

When a session is created, the daemon automatically captures:
- **Machine ID**: MD5 hash of the primary network interface's MAC address
- **Hostname**: The machine's hostname for display purposes

This information is sent to the server and associated with the session.

### 2. Intelligent Status Tracking

The server tracks three distinct states:

#### **Active** (Green badge)
- Heartbeats are being received regularly
- Application is running normally

#### **Suspended** (Orange badge)
- Heartbeats have stopped
- Session is marked as "likely sleeping/hibernating"
- Happens automatically when heartbeat timeout is exceeded

#### **Crashed** (Red badge)
- Machine came back online (new session from same machine)
- But the old "suspended" session didn't resume
- This indicates the application crashed or was terminated

### 3. The Smart Detection Logic

```
1. Application stops sending heartbeats
   └─> Server marks session as "Suspended" (might be sleeping)

2. New session starts from same machine
   └─> Server checks: Are there suspended sessions from this machine?
       ├─> YES: Mark those as "Crashed" (app didn't resume, so it crashed)
       └─> NO: Continue normally

3. Suspended session resumes sending heartbeats
   └─> Server marks session as "Active" again (was just sleeping)
```

## Benefits

### Accurate Crash Detection
- **Before**: All heartbeat timeouts looked the same (could be sleep, crash, or hang)
- **After**: Clear distinction between system sleep and actual crashes

### No False Positives
- Laptop closed for lunch? Marked as "Suspended", not crashed
- Desktop put to sleep overnight? Stays "Suspended"
- Only when the machine comes back WITHOUT the session do we know it crashed

### Cross-Machine Visibility
- See which machine each session ran on
- Track sessions across multiple developers/machines
- Identify machine-specific issues

## Example Scenarios

### Scenario 1: System Sleep (Laptop Closed)
```
1. User closes laptop → Application stops, heartbeats stop
2. Server marks session as "Suspended" (orange badge)
3. User opens laptop → Application resumes, heartbeats resume
4. Server marks session as "Active" again (green badge)
```
**Result**: No crash detected ✓

### Scenario 2: Application Crash
```
1. Application crashes → Heartbeats stop
2. Server marks session as "Suspended" (orange badge)
3. User restarts application → New session created
4. Server sees: "This machine has suspended sessions!"
5. Server marks old session as "Crashed" (red badge)
```
**Result**: Crash correctly detected ✓

### Scenario 3: Machine Reboot
```
1. Machine reboots → All processes terminate
2. Server marks all sessions as "Suspended"
3. Machine comes back, user starts application
4. New session created from same machine
5. Server marks all old sessions as "Crashed"
```
**Result**: Terminated sessions correctly identified ✓

## Implementation Details

### Machine ID Generation

The daemon generates a consistent machine ID using:
```go
// Find first non-loopback network interface
// Hash its MAC address for privacy
machineID := MD5(macAddress)
```

**Why MD5 hash?**
- Privacy: Doesn't expose actual MAC address
- Consistency: Same hash for same machine across sessions
- Uniqueness: Different machines have different IDs

### Server-Side Detection

When a new session is created:
```go
func CreateSession(product, version, machineID, hostname string) *Session {
    // Check for suspended sessions from same machine
    for existing := range suspendedSessions {
        if existing.MachineID == machineID {
            existing.Crashed = true  // Mark as crashed
            existing.Suspended = false
            logEvent("session_crashed_detected")
        }
    }
    // Create new session...
}
```

## Viewing Session Status

In the web UI, each session shows:
- **Status badge**: Active / Suspended / Crashed / Ended
- **Machine**: Hostname of the machine running the session
- **Last Heartbeat**: When the last heartbeat was received

You can filter sessions by status to find:
- All crashed sessions (debugging)
- Currently suspended sessions (might be sleeping)
- Active sessions (currently running)

## Limitations

### False Negatives
- **Multiple network interfaces**: If MAC address changes (e.g., switching from WiFi to Ethernet), the machine ID changes
- **VM cloning**: Cloned VMs may have same MAC until reboot
- **Container environments**: May need additional identification

### False Positives
- **Manual termination**: `kill -9` or task manager termination will be marked as "crashed"
  - This is actually correct behavior - abnormal termination
- **Rapid restart**: If application restarts very quickly (< heartbeat timeout), may not be detected

## Privacy Considerations

- **MAC addresses are hashed**: The actual MAC address is never stored or sent
- **Hostname is optional**: Can be omitted if privacy is a concern
- **No personal data**: Only machine identifier and hostname collected

## Future Enhancements

Potential improvements:
- **Configurable machine ID**: Allow users to provide their own identifier
- **Multi-factor machine ID**: Combine MAC, hostname, and other factors
- **Grace period**: Option to wait N minutes before marking as crashed
- **Container awareness**: Better support for Docker/Kubernetes environments

