# 🐱 datacat Demo GUI

A modern, web-based demonstration GUI for the datacat Python client.

![Python](https://img.shields.io/badge/python-3.6%2B-blue)
![Gradio](https://img.shields.io/badge/UI-Gradio-orange)

## Features

This demo showcases all major features of the datacat Python client:

### 🌙 Dark Mode (Built-in)
- Launches in dark mode by default
- Use Gradio's settings menu (⚙️ icon in bottom left) to toggle light/dark mode
- Preference is saved automatically
- Uses Gradio's native dark theme for optimal compatibility

### 📝 State Management
- Edit and send JSON state updates
- View current state in real-time
- Deep merge support for nested state objects

### 📢 Event Logging
- Log custom events with structured data
- JSON editor for event payloads
- Automatic event tracking

### 📈 Metrics Logging
- Send numeric metrics with tags
- Support for multiple tag formats
- Real-time metric tracking

### ⚠️ Error Logging with Custom Handler
- Custom `logging.Handler` implementation
- Automatically formats log messages and stack traces
- Supports all Python logging levels (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- Integrates seamlessly with Python's logging module

### 💥 Exception Generation & Handling
- Generate various Python exception types:
  - `ValueError`
  - `TypeError`
  - `KeyError`
  - `IndexError`
  - `ZeroDivisionError`
- Support for nested exceptions (exception chaining)
- Full stack trace capture and formatting
- Dual logging: both via handler and direct API

### 📊 Session Statistics
- Real-time statistics display
- Track events, metrics, state updates, and errors
- Session ID and web UI links

## Installation

### 1. Install the datacat Python client

```bash
cd python
pip install -e .
```

### 2. Install the demo requirements

```bash
cd examples/demo_gui
pip install -r requirements.txt
```

Or install Gradio directly:
```bash
pip install gradio
```

### 3. Make sure datacat server is running

```bash
# In one terminal, start the server
cd cmd/datacat-server
go run main.go config.go

# Or if you have built binaries
./bin/datacat-server

# Or use the PowerShell script (Windows)
.\scripts\run-server.ps1
```

### 4. Start the web UI (optional, for viewing sessions)

```bash
# In another terminal
cd cmd/datacat-web
go run main.go

# Or if you have built binaries
./bin/datacat-web

# Or use the PowerShell script (Windows)
.\scripts\run-web.ps1
```

## Usage

### Starting the Demo

**Option 1: Direct**
```bash
cd examples/demo_gui
python demo_gui.py
```

**Option 2: Using PowerShell script (Windows - Recommended)**
```powershell
.\scripts\run-demo-gui.ps1
```

**Benefits:**
- Automatically uses the Python virtual environment (`.venv`) if available
- Checks prerequisites and offers to install Gradio if missing
- Verifies server connectivity
- Provides helpful error messages

The demo will automatically:
- Start a local web server on http://127.0.0.1:7860
- Open your default browser to the demo interface

### Using the Demo

**Theme:** The demo starts in dark mode by default. Use Gradio's settings menu (⚙️) in the bottom left corner to switch to light mode if preferred.

1. **Start a Session**
   - **Enter your product name** (required) - e.g., "my-app", "web-service"
   - **Enter your product version** (required) - e.g., "1.0.0", "2.1.3"
   - Set the server URL (default: `http://localhost:9090`)
   - Choose whether to use the daemon (optional)
   - Click **"Start New Session"**
   - Your session ID will be displayed
   - All feature controls become active

2. **Update State** (requires active session)
   - Edit the JSON in the State Management section
   - Click "Update State" to send it to datacat
   - View the current state in the result

3. **Log Events** (requires active session)
   - Enter an event name (e.g., `user_action`, `button_click`)
   - Provide JSON data for the event (optional)
   - Click "Log Event"

4. **Log Metrics** (requires active session)
   - Enter a metric name (e.g., `response_time`, `cpu_usage`)
   - Provide a numeric value
   - Add tags in comma-separated format (e.g., `env:prod, service:api`)
   - Click "Log Metric"

5. **Log Errors via Handler** (requires active session)
   - Select a log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
   - Enter an error message
   - Click "Log Error"
   - The custom logging handler formats and sends the error

6. **Generate Exceptions** (requires active session)
   - Select an exception type from the dropdown
   - Optionally enable "Include Nested Exception" for exception chaining
   - Click "Generate & Log Exception"
   - The exception is logged via both the custom handler AND direct API
   - Full stack traces are captured and formatted

7. **View Statistics** (requires active session)
   - Click "Refresh Stats" to see current session statistics
   - View total events, metrics, state updates, and errors

8. **End Session**
   - Click **"End Session"** (the button changes after starting)
   - The session is properly closed and logged
   - All feature controls become disabled
   - You can start a new session by clicking "Start New Session" again

### Custom Logging Handler

The demo includes a custom `DatacatLoggingHandler` that integrates with Python's standard logging module:

```python
class DatacatLoggingHandler(logging.Handler):
    """
    Custom logging handler that sends log messages to datacat.

    Formats exceptions with full stack traces and sends them as events.
    """
```

**Features:**
- Automatically captures exception info with `exc_info=True`
- Formats stack traces using `traceback.format_exception()`
- Sends logs as structured events to datacat
- Includes metadata: level, logger name, timestamp
- Gracefully handles logging errors

**Example Usage:**
```python
logger = logging.getLogger("my_app")
handler = DatacatLoggingHandler(session)
logger.addHandler(handler)

# This will be logged to datacat with full traceback
try:
    risky_operation()
except Exception:
    logger.error("Operation failed", exc_info=True)
```

### Exception Demonstration

The demo can generate and log various exception types with full stack traces:

**Single Exception:**
```python
try:
    int("not_a_number")  # ValueError
except Exception as e:
    logger.error("Error occurred", exc_info=True)
    session.log_exception()
```

**Nested Exception (Exception Chaining):**
```python
try:
    try:
        raise ValueError("Inner error")
    except ValueError as inner:
        raise RuntimeError("Outer error") from inner
except RuntimeError as e:
    logger.error("Nested error", exc_info=True)
    session.log_exception()
```

Both the logging handler and direct API capture:
- Exception type
- Exception message
- Full stack trace with file names and line numbers
- Exception chaining information

## Viewing Sessions

Once you've created a session and logged data, view it in the web UI:

```
http://localhost:8080/session/{session_id}
```

The session ID is displayed in the demo GUI after creation.

## Architecture

```
┌─────────────────┐
│   Demo GUI      │
│   (Gradio)      │
│  Port: 7860     │
└────────┬────────┘
         │
         ├── Creates sessions
         ├── Updates state
         ├── Logs events
         ├── Logs metrics
         └── Handles exceptions
         │
         v
┌─────────────────┐
│ datacat Client  │
│   (Python)      │
└────────┬────────┘
         │
         v
┌─────────────────┐
│ datacat Server  │
│  Port: 9090     │
└────────┬────────┘
         │
         v
┌─────────────────┐
│  BadgerDB       │
│  (./data_path)  │
└─────────────────┘
         │
         v
┌─────────────────┐
│  Web UI         │
│  Port: 8080     │
└─────────────────┘
```

## Tips

### For Development
- Use "Use Daemon" mode for automatic batching and crash detection
- Watch the terminal for error messages
- Check the web UI to see how data is structured

### For Testing
- Try different exception types to see how they're formatted
- Enable nested exceptions to test exception chaining
- Use various log levels to see how they're categorized
- Send metrics with different tag combinations

### For Demonstrations
- Create a session before starting your demo
- Show the web UI side-by-side with the demo GUI
- Generate exceptions to showcase error tracking
- Use the statistics panel to show activity counts

## Troubleshooting

**"Error creating session: Connection refused"**
- Make sure the datacat server is running on port 9090
- Check the server URL is correct

**"Module 'gradio' not found"**
- Install requirements: `pip install -r requirements.txt`

**"Error: Daemon binary not found"**
- Don't worry! Just uncheck "Use Daemon" to connect directly to the server
- Or build the daemon: `cd cmd/datacat-daemon && go build`

**Browser doesn't open automatically**
- Manually navigate to http://127.0.0.1:7860
- Check the terminal output for the actual URL

## Why Gradio?

Gradio was chosen for this demo because it:
- ✅ Creates modern, professional-looking web UIs
- ✅ Requires minimal dependencies (just `gradio`)
- ✅ Provides excellent form controls and layouts
- ✅ Supports custom themes and styling
- ✅ Automatically handles the web server
- ✅ Opens the browser automatically
- ✅ Is actively maintained and well-documented
- ✅ Works with Python 2.7+ and Python 3.x
- ✅ Looks much better than tkinter!

## Dark Mode

The demo launches in dark mode by default using Gradio's built-in dark theme:
- **Default dark theme**: Starts in dark mode automatically
- **Easy switching**: Use Gradio's settings menu (⚙️ icon in bottom left) to toggle
- **Native integration**: Uses Gradio's official dark theme for best compatibility
- **Automatic saving**: Your theme preference is saved automatically

Perfect for late-night coding sessions or reducing eye strain!

## License

Same as the datacat project.

