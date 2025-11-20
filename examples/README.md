# Examples

This directory contains example applications demonstrating various features of datacat.

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

### [testing_example.py](testing_example.py)

Example of using datacat in CI/CD for tracking test runs and build metrics.

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
