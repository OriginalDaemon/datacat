---
layout: default
title: Examples
nav_order: 4
has_children: true
---

# DataCat Examples

{: .no_toc }

Example code demonstrating how to use DataCat in various scenarios.
{: .fs-6 .fw-300 }

## Quick Start

```bash
# Start the server
.\scripts\run-server.ps1

# Run an example
python examples/game_logging_example.py
```

---

## Example Categories

### Python Examples

Comprehensive Python examples demonstrating all features:

**[Python Examples Guide →](python-examples.md)**

- Game logging with async mode (< 0.01ms overhead)
- All metric types (Gauges, Counters, Histograms, Timers)
- Exception tracking with stack traces
- Heartbeat monitoring and hang detection
- State management with nested objects

### Demo GUI

Interactive web-based demonstration interface:

**[Demo GUI Guide →](demo-gui.md)**

- Built with Gradio
- Dark mode enabled by default
- All DataCat features in one interactive UI
- Perfect for exploring and testing

### Go Examples

Go client library examples:

**[Go Examples Guide →](go-examples.md)**

- Session management
- Event and metric logging
- State updates
- Daemon integration

---

## Featured Examples

### Interactive Game Demo

Realistic game simulation with 60 FPS loop:

```bash
# Single game
python examples/example_game.py --duration 60

# Multiple concurrent games
python examples/run_game_swarm.py --count 10
```

### Async Logging Performance Test

Test ultra-fast async logging:

```bash
python examples/game_logging_example.py
```

**Result**: < 0.01ms per log call, < 1% overhead at 60 FPS

### Metric Types Demo

See all four metric types in action:

```bash
python examples/metric_types_example.py
```

**Metric Types**:

- Gauges - Point-in-time values
- Counters - Cumulative counts
- Histograms - Value distributions
- Timers - Duration measurements

---

## Next Steps

- **[Python Examples](python-examples.md)** - Complete Python examples guide
- **[Demo GUI](demo-gui.md)** - Interactive web UI demo
- **[Go Examples](go-examples.md)** - Go client library examples
- **[Game Logging](../game-logging.md)** - Async logging for real-time apps
- **[Metric Types](../metric-types.md)** - Understanding all metric types
