# DataCat Examples

This directory contains example code demonstrating how to use DataCat in various scenarios.

## Examples

### Game Logging (Async, Non-Blocking)

**[game_logging_example.py](game_logging_example.py)**

Demonstrates ultra-fast async logging suitable for game engines and real-time applications with strict frame timing requirements (e.g., 60 FPS = 16.7ms per frame).

**Features:**

- Non-blocking logging (< 0.01ms per call)
- Python 2.7.4+ compatible
- Zero external dependencies
- Queue-based batching
- Performance statistics

**Usage:**

```bash
# Make sure server is running first
python examples/game_logging_example.py
```

**Key Takeaway:** Async logging adds < 0.1ms overhead per frame (< 1% of frame budget at 60 FPS).

---

### Python 2.7.4 Compatibility Test

**[test_async_py27.py](test_async_py27.py)**

Comprehensive test suite verifying AsyncSession works correctly in Python 2.7.4 using only standard library features.

**Tests:**

- Python 2 vs 3 import compatibility
- Queue.Queue functionality
- Threading with daemon threads
- Non-blocking queue operations
- Queue overflow handling
- Background thread processing

**Usage:**

```bash
python examples/test_async_py27.py
```

---

### Python GUI Demo

**[demo_gui.py](demo_gui.py)**

Simple tkinter-based GUI demonstrating DataCat session management, event logging, and metrics.

**Features:**

- Session creation and management
- Event logging with different levels
- Metric logging
- State updates
- Exception logging
- Heartbeat monitoring

**Usage:**

```bash
python examples/demo_gui.py
```

---

### Go Client Example

**[client-example.go](client-example.go)**

Demonstrates using the Go client library with daemon-based logging.

**Features:**

- Session creation
- Event logging
- Metric logging
- State updates
- Daemon management

**Usage:**

```bash
cd examples
go run client-example.go
```

---

## Quick Comparison

| Example                   | Language | Use Case          | Blocking         | Python 2.7.4        |
| ------------------------- | -------- | ----------------- | ---------------- | ------------------- |
| `game_logging_example.py` | Python   | Real-time / Games | ❌ No (< 0.01ms) | ✅ Yes              |
| `test_async_py27.py`      | Python   | Testing           | ❌ No            | ✅ Yes              |
| `demo_gui.py`             | Python   | General Demo      | ✅ Yes (~2ms)    | ⚠️ Tkinter required |
| `client-example.go`       | Go       | Go Applications   | ✅ Yes           | N/A                 |

---

## Getting Started

### 1. Start the DataCat Server

```bash
# Windows
.\scripts\run-server.ps1

# Or manually
cd cmd/datacat-server
go run main.go config.go
```

### 2. Start the Web UI (optional)

```bash
# Windows
.\scripts\run-web.ps1

# Or manually
cd cmd/datacat-web
go run main.go
```

### 3. Run an Example

```bash
# Game logging example
python examples/game_logging_example.py

# Test Python 2.7.4 compatibility
python examples/test_async_py27.py

# GUI demo
python examples/demo_gui.py
```

---

## For Game Developers

If you're building a game or real-time application:

1. **Read the guide**: [docs/GAME_LOGGING.md](../docs/GAME_LOGGING.md)
2. **Run the example**: `python examples/game_logging_example.py`
3. **Test compatibility**: `python examples/test_async_py27.py` (if using Python 2.7.4)
4. **Use async mode**:

```python
from datacat import create_session

session = create_session(
    "http://localhost:9090",
    product="YourGame",
    version="1.0.0",
    async_mode=True  # Non-blocking!
)

# In game loop - returns in < 0.01ms
session.log_event("player_action", data={...})
```

---

## Performance Benchmarks

### Game Logging (Async Mode)

Tested on: Windows 11, Python 3.12, 60 FPS simulation

| Operation          | Average Time | Frame Budget @ 60 FPS |
| ------------------ | ------------ | --------------------- |
| `log_event()`      | 0.008ms      | 0.05%                 |
| `log_metric()`     | 0.008ms      | 0.05%                 |
| `update_state()`   | 0.008ms      | 0.05%                 |
| **100 logs/frame** | 0.8ms        | **5%** ✅             |

**Conclusion:** Async logging is suitable for 60 FPS, 120 FPS, or even higher frame rates.

---

## Need Help?

- **Game logging**: See [docs/GAME_LOGGING.md](../docs/GAME_LOGGING.md)
- **Process isolation**: See [docs/PROCESS_ISOLATION.md](../docs/PROCESS_ISOLATION.md)
- **Architecture**: See [ARCHITECTURE.md](../ARCHITECTURE.md)
- **Main README**: See [README.md](../README.md)
