# DataCat Examples

Example code demonstrating how to use DataCat in various scenarios.

## Quick Start

```bash
# Start the server
.\scripts\run-server.ps1

# Run an example
python examples/game_logging_example.py
```

## Documentation

📖 **Complete examples documentation**: [docs/examples/](../docs/_examples/)

- **[Python Examples](../docs/_examples/python-examples.md)** - Complete guide to all Python examples
- **[Demo GUI](../docs/_examples/demo-gui.md)** - Interactive web UI demo
- **[Go Examples](../docs/_examples/go-examples.md)** - Go client library examples

## Featured Examples

### Interactive Game Demo
```bash
python examples/example_game.py --duration 60
python examples/run_game_swarm.py --count 10
```

### Performance Testing
```bash
python examples/game_logging_example.py
```

### Metric Types
```bash
python examples/metric_types_example.py
python examples/incremental_counters_example.py
python examples/fps_histogram_example.py
```

### Demo GUI
```bash
.\scripts\run-demo-gui.ps1
```

## All Examples

| File                            | Description                                    |
| ------------------------------- | ---------------------------------------------- |
| `basic_example.py`              | Simplest possible example                      |
| `complete_example.py`           | All features in one example                    |
| `example_game.py`               | Interactive game simulation (60 FPS)           |
| `run_game_swarm.py`             | Multi-instance game demo                       |
| `game_logging_example.py`       | Async logging performance test                 |
| `metric_types_example.py`       | Gauges, Counters, Histograms, Timers          |
| `incremental_counters_example.py` | Counter patterns and aggregation             |
| `fps_histogram_example.py`      | Custom histogram buckets for FPS tracking      |
| `exception_logging_example.py`  | Exception capture with stack traces            |
| `heartbeat_example.py`          | Heartbeat monitoring and hang detection        |
| `window_tracking_example.py`    | Window lifecycle tracking                      |
| `logging_handler_example.py`    | Python logging module integration              |
| `testing_example.py`            | Test tracking and reporting                    |
| `offline_demo.py`               | Offline mode demonstration                     |
| `test_crash_detection.py`       | Crash detection testing                        |
| `test_async_py27.py`            | Python 2.7.4 compatibility tests               |
| `demo_gui/`                     | Interactive web UI (Gradio)                    |
| `go-client-example/`            | Go client library usage                        |

## Need Help?

- 📖 **[Full Documentation](../docs/_examples/)** - Complete examples guide
- 🎮 **[Game Logging](../docs/GAME_LOGGING.md)** - Async logging for real-time apps
- 📊 **[Metric Types](../docs/METRIC_TYPES.md)** - Understanding metrics
- 🏗️ **[Architecture](../docs/_guides/architecture.md)** - System design
