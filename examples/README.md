<div align="center">

# DataCat Examples

```
██████╗  █████╗ ████████╗ █████╗  ██████╗ █████╗ ████████╗
██╔══██╗██╔══██╗╚══██╔══╝██╔══██╗██╔════╝██╔══██╗╚══██╔══╝
██║  ██║███████║   ██║   ███████║██║     ███████║   ██║
██║  ██║██╔══██║   ██║   ██╔══██║██║     ██╔══██║   ██║
██████╔╝██║  ██║   ██║   ██║  ██║╚██████╗██║  ██║   ██║
╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝   ╚═╝
```

</div>

This directory contains example applications demonstrating various features of DataCat.

## 🎨 Interactive Demo GUI

### [demo_gui/](demo_gui/) - Modern Web-Based Demo

**A comprehensive interactive demo with a modern web UI!**

Features a beautiful web interface (powered by Gradio) that demonstrates all DataCat features:
- 🌙 Dark mode by default using Gradio's native theme
- 📝 State management with JSON editor
- 📢 Event logging
- 📈 Metrics tracking
- ⚠️ Custom logging handler for error messages
- 💥 Exception generation and handling with full stack traces

**Installation:**
```bash
cd examples/demo_gui
pip install -r requirements.txt
pip install -e ../../python
```

**Run:**
```bash
# Option 1: Direct
cd examples/demo_gui
python demo_gui.py

# Option 2: Using PowerShell script (Windows)
.\scripts\run-demo-gui.ps1
```

Opens automatically in your browser at http://127.0.0.1:7860

📖 **[Full Documentation](demo_gui/README.md)** | 🚀 **[Quick Start](demo_gui/QUICKSTART.md)**

---

## Python Examples

All Python examples assume the datacat-server is running on `http://localhost:9090`.

### [basic_example.py](basic_example.py)

Basic usage of the Python client - session creation, state updates, events, and metrics.

```bash
python examples/basic_example.py
```

### [window_tracking_example.py](window_tracking_example.py)

Demonstrates nested state tracking for window management with deep merge updates.

```bash
python examples/window_tracking_example.py
```

### [heartbeat_example.py](heartbeat_example.py)

Shows heartbeat monitoring to detect hung applications using a background thread.

```bash
python examples/heartbeat_example.py
```

### [exception_logging_example.py](exception_logging_example.py)

Demonstrates exception logging with full traceback capture and custom context.

```bash
python examples/exception_logging_example.py
```

### [logging_handler_example.py](logging_handler_example.py)

Shows how to integrate DataCat with Python's standard `logging` module using a custom handler. Demonstrates:
- Custom `DatacatLoggingHandler` implementation
- Exception logging with stack traces
- Log levels (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- Custom log fields
- Integration with existing logging infrastructure

```bash
python examples/logging_handler_example.py
```

### [testing_example.py](testing_example.py)

Example of using DataCat in CI/CD for tracking test runs and build metrics.

```bash
python examples/testing_example.py
```

### [complete_example.py](complete_example.py)

Comprehensive demo showing all features together in a realistic scenario.

```bash
python examples/complete_example.py
```

## Go Examples

### [go-client-example](go-client-example/)

Demonstrates the Go client library usage.

```bash
cd examples/go-client-example
go run main.go
```

## Running the Examples

### 1. Start datacat-server

```bash
cd cmd/datacat-server
go run main.go
```

### 2. (Optional) Start datacat-web

```bash
cd cmd/datacat-web
go run main.go
```

Open `http://localhost:8080` to see sessions in the web UI.

### 3. Run Examples

```bash
# Python examples
python examples/basic_example.py
python examples/heartbeat_example.py

# Go example
cd examples/go-client-example && go run main.go
```

## Example Output

All examples will:
1. Create a session
2. Log data (state, events, metrics)
3. Display session ID
4. Show confirmation when complete

You can view the logged data:
- Via the web UI at `http://localhost:8080`
- By querying the API at `http://localhost:9090/api/sessions/{session_id}`
- Through external tools using the `/api/data/sessions` endpoint
