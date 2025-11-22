# DataCat Examples

This directory contains example code demonstrating how to use DataCat in various scenarios.

## Examples

### Example Game (Interactive Demo)

**[example_game.py](example_game.py)** + **[run_game_swarm.py](run_game_swarm.py)**

A complete simulated game that demonstrates DataCat in a realistic scenario. Features:
- Main update/render loop running at 60 FPS
- Real-time metrics (FPS, memory, player stats)
- Random gameplay events (enemies, powerups, achievements)
- Random errors and exceptions
- Different modes: normal, hang, crash
- Multi-instance swarm launcher

**Single Game Usage:**
```bash
# Run normally for 60 seconds
python examples/example_game.py --duration 60

# Run with hang behavior
python examples/example_game.py --mode hang --duration 30

# Run with crash behavior
python examples/example_game.py --mode crash --duration 20

# Or use PowerShell script
.\scripts\run-example-game.ps1 -Mode normal -Duration 60
```

**Swarm Mode (Multiple Players):**
```bash
# Launch 10 concurrent games
python examples/run_game_swarm.py --count 10 --duration 60

# Launch 20 games with custom hang/crash rates
python examples/run_game_swarm.py --count 20 --hang-rate 0.2 --crash-rate 0.1

# Or use PowerShell script
.\scripts\run-game-swarm.ps1 -Count 10 -Duration 60
```

**What You'll See:**
- Real-time game sessions in the web UI
- Live metrics updating every second
- Crash detection for crashed games
- Hang detection for frozen games
- Complete event timeline for each player

---

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

| Example                   | Language | Use Case                  | Blocking         | Python 2.7.4        |
| ------------------------- | -------- | ------------------------- | ---------------- | ------------------- |
| `example_game.py`         | Python   | **Interactive Game Demo** | ❌ No (< 0.01ms) | ✅ Yes              |
| `run_game_swarm.py`       | Python   | **Multi-Player Demo**     | ❌ No (< 0.01ms) | ✅ Yes              |
| `game_logging_example.py` | Python   | Performance Testing       | ❌ No (< 0.01ms) | ✅ Yes              |
| `test_async_py27.py`      | Python   | Compatibility Testing     | ❌ No            | ✅ Yes              |
| `demo_gui.py`             | Python   | General Demo              | ✅ Yes (~2ms)    | ⚠️ Tkinter required |
| `client-example.go`       | Go       | Go Applications           | ✅ Yes           | N/A                 |

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

1. **Try the interactive demo**: `python examples/example_game.py` or `.\scripts\run-example-game.ps1`
2. **Launch the swarm**: `python examples/run_game_swarm.py --count 10` to see multiple concurrent sessions
3. **Read the guide**: [docs/GAME_LOGGING.md](../docs/GAME_LOGGING.md)
4. **Run performance tests**: `python examples/game_logging_example.py`
5. **Test compatibility**: `python examples/test_async_py27.py` (if using Python 2.7.4)
6. **Use async mode in your game**:

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
