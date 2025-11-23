# Process Isolation - One Daemon Per Application

## Overview

DataCat ensures that each application instance runs its own dedicated daemon process. Daemons are never shared across different applications, ensuring **process isolation** and preventing port conflicts.

## Architecture Principle

```
Application A                    Application B
     |                                |
     ├─> Daemon A (port: 50180)      ├─> Daemon B (port: 50183)
     |   └─> Server                  |   └─> Server
     |                                |
     ✓ Isolated                      ✓ Isolated
```

**NOT:**

```
Application A ──┐
                ├─> Shared Daemon (port: 8079) ✗ WRONG
Application B ──┘
```

## How It Works

### Dynamic Port Allocation

When you create a `DatacatClient` (Python) or `Client` with daemon (Go), the daemon manager automatically:

1. **Finds an available port** using OS-level port allocation
2. **Creates a unique config file** for that daemon instance
3. **Starts the daemon** on the allocated port
4. **Cleans up** the config file when the daemon stops

### Python Client

```python
from datacat import DatacatClient

# Each client gets its own daemon on a unique port
client = DatacatClient("http://localhost:9090")  # daemon_port="auto" by default

# The daemon will automatically start on an available port
session_id = client.register_session("MyApp", "1.0")
```

### Go Client

```go
import "github.com/OriginalDaemon/datacat/client"

// Create client with automatic daemon management
c, err := client.NewClientWithDaemon(
    "http://localhost:9090",  // server URL
    "auto",                   // daemon port (auto-allocates)
    "",                       // daemon binary (auto-detected)
)
```

## Port Selection

### Default Behavior ("auto")

```python
# Uses dynamic port allocation
client = DatacatClient("http://localhost:9090")
# or explicitly:
client = DatacatClient("http://localhost:9090", daemon_port="auto")
```

The daemon will:

- Find an available port using the OS
- Typical ports: 50000-65535 range
- No conflicts with other applications

### Fixed Port (Not Recommended)

```python
# Only use if you need a specific port
client = DatacatClient("http://localhost:9090", daemon_port="8080")
```

⚠️ **Warning**: Fixed ports can cause conflicts if:

- Multiple instances of your app run simultaneously
- The port is already in use
- Multiple apps use the same fixed port

## Benefits

### 1. **No Port Conflicts**

Each application gets its own daemon on a unique port, preventing "address already in use" errors.

### 2. **Process Isolation**

- Daemon lifecycle tied to parent application
- Daemon terminates when application exits
- No shared state between applications

### 3. **Clean Separation**

- Each app's data is batched independently
- Crash of one app doesn't affect others
- Easy to debug and monitor per-app

### 4. **Multi-Instance Support**

Run multiple instances of the same application simultaneously without conflicts:

```bash
# Terminal 1
python my_app.py  # Daemon on port 50180

# Terminal 2
python my_app.py  # Daemon on port 50183

# No conflicts!
```

## Implementation Details

### Config File Naming

Each daemon instance creates a uniquely named config file:

```
daemon_config_50180.json
daemon_config_50183.json
daemon_config_50186.json
```

These files are automatically cleaned up when the daemon stops.

### Environment Variable

The daemon binary reads its config file path from the `DATACAT_CONFIG` environment variable:

```bash
export DATACAT_CONFIG=daemon_config_50180.json
./datacat-daemon
```

### Daemon Discovery

The client automatically discovers its daemon's port after starting:

```python
client = DatacatClient("http://localhost:9090")
# client.base_url is now "http://localhost:50180" (or whatever port was allocated)
```

## Testing

Verify process isolation works correctly:

```python
# test_isolation.py
from datacat import DatacatClient

# Start 3 separate clients
clients = []
for i in range(3):
    client = DatacatClient("http://localhost:9090")
    print(f"Client {i+1} daemon: {client.base_url}")
    clients.append(client)

# Each should have a different daemon port
ports = [c.base_url.split(":")[-1] for c in clients]
assert len(set(ports)) == 3, "Each client should have unique daemon!"
print(f"✓ All clients isolated on ports: {ports}")
```

Expected output:

```
Client 1 daemon: http://localhost:50180
Client 2 daemon: http://localhost:50183
Client 3 daemon: http://localhost:50186
✓ All clients isolated on ports: ['50180', '50183', '50186']
```

## Troubleshooting

### "Port already in use" Error

If you still see port conflicts:

1. **Don't use fixed ports** - Use `daemon_port="auto"` (default)
2. **Check for zombie daemons** - Kill any leftover daemon processes
3. **Clean up config files** - Remove `daemon_config_*.json` files

### Daemon Not Starting

Check that:

- The daemon binary exists and is executable
- No firewall blocking local ports
- Sufficient system resources (ports available)

### Finding Active Daemons

```bash
# Windows
netstat -ano | findstr :50

# Linux/Mac
lsof -i :50000-60000
```

## Migration from Shared Daemon

If you were using a fixed port before:

### Before (Problematic)

```python
# All apps used the same port
client = DatacatClient("http://localhost:9090", daemon_port="8079")
```

### After (Correct)

```python
# Each app gets its own daemon
client = DatacatClient("http://localhost:9090")  # daemon_port="auto" is default
```

No code changes needed - just remove the `daemon_port` argument or set it to `"auto"`.

## Related Documentation

- [Architecture Overview](../ARCHITECTURE.md)
- [Daemon Configuration](daemon-config.md)
- [Troubleshooting Guide](troubleshooting.md)
