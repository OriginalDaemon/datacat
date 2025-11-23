---
layout: default
title: Demo GUI
parent: Examples
nav_order: 2
---

# DataCat Demo GUI
{: .no_toc }

Interactive web-based demonstration GUI built with Gradio.
{: .fs-6 .fw-300 }

## Table of Contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

The Demo GUI is a modern, web-based demonstration interface for the DataCat Python client, built with Gradio. It provides an interactive way to explore all DataCat features through a clean, dark-mode UI.

**Location**: `examples/demo_gui/`

---

## Features

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
- Real-time metric updates

### 💥 Exception Logging
- Log exceptions with stack traces
- Automatic source information capture
- Custom exception data

### 💓 Heartbeat Monitoring
- Manual heartbeat sending
- View last heartbeat timestamp
- Hang detection testing

### 🔍 Session Information
- View current session ID
- Check session details
- Monitor session status

---

## Installation

### Prerequisites

1. **DataCat server running** on http://localhost:9090
2. **Python 3.6+** (or Python 2.7+ with limited features)

### Step 1: Install DataCat Python Client

```bash
cd python
pip install -e .
```

### Step 2: Install Gradio

```bash
pip install gradio
```

Or use the requirements file:
```bash
cd examples/demo_gui
pip install -r requirements.txt
```

---

## Usage

### Method 1: PowerShell Script (Recommended for Windows)

```powershell
.\scripts\run-demo-gui.ps1
```

**Features**:
- Automatic prerequisite checking
- Offers to install Gradio if missing
- Verifies DataCat server is running
- Clean error messages

### Method 2: Direct Python

```bash
cd examples/demo_gui
python demo_gui.py
```

### Method 3: From Repository Root

```bash
python examples/demo_gui/demo_gui.py
```

---

## Getting Started

### 1. Launch the Demo

```bash
.\scripts\run-demo-gui.ps1
```

The demo will open in your default browser at `http://localhost:7860`

### 2. Create a Session

When the demo launches, a session is automatically created. You'll see:
- Session ID displayed at the top
- Current state in the "Current State" box

### 3. Try the Features

#### Update State

1. Go to the "State Management" tab
2. Edit the JSON in the text area:
   ```json
   {
       "level": 2,
       "player": {
           "health": 80,
           "position": {"x": 100, "y": 200}
       }
   }
   ```
3. Click "Send State Update"
4. View the updated state in "Current State"

#### Log an Event

1. Go to the "Event Logging" tab
2. Enter event name: `player_action`
3. Select level: `info`
4. Enter event data:
   ```json
   {
       "action": "jump",
       "height": 10
   }
   ```
5. Click "Log Event"

#### Log a Metric

1. Go to the "Metrics" tab
2. Enter metric name: `fps`
3. Enter value: `60.0`
4. Enter tags: `performance,realtime`
5. Click "Log Metric"

#### Send Heartbeat

1. Go to the "Heartbeat" tab
2. Click "Send Heartbeat"
3. View last heartbeat timestamp

---

## UI Layout

### Top Section: Session Info
- **Session ID**: Current session identifier
- **Session Details**: View complete session data
- **Current State**: Real-time state display

### Tabs

1. **State Management**
   - JSON editor for state updates
   - Send button
   - Success/error feedback

2. **Event Logging**
   - Event name input
   - Level selector (debug, info, warning, error, critical)
   - JSON editor for event data
   - Log button

3. **Metrics**
   - Metric name input
   - Value input (numeric)
   - Tags input (comma-separated)
   - Log button

4. **Exception Logging**
   - Exception type input
   - Exception message input
   - JSON editor for additional data
   - Log button

5. **Heartbeat**
   - Manual heartbeat button
   - Last heartbeat timestamp display

---

## Configuration

### Custom Server URL

Edit the `demo_gui.py` file to change the server URL:

```python
client = datacat.DatacatClient(base_url="http://your-server:9090")
```

### Custom Port

Change the Gradio launch port:

```python
demo.launch(server_port=8080)  # Default is 7860
```

### Dark Mode Toggle

Dark mode is enabled by default. Users can toggle it using the Gradio settings menu (⚙️ icon).

---

## JSON Format Examples

### State Update
```json
{
    "level": 1,
    "player": {
        "health": 100,
        "mana": 50,
        "position": {"x": 0, "y": 0}
    },
    "inventory": ["sword", "potion"]
}
```

### Event Data
```json
{
    "action": "attack",
    "target": "enemy_01",
    "damage": 25,
    "critical": true
}
```

### Exception Data
```json
{
    "context": "startup",
    "user_id": "12345",
    "operation": "load_config"
}
```

---

## Troubleshooting

### "DataCat server is not running"

**Problem**: The demo cannot connect to the DataCat server.

**Solution**:
1. Start the server:
   ```bash
   .\scripts\run-server.ps1
   ```
2. Verify it's running at http://localhost:9090/health

### "Gradio is not installed"

**Problem**: Gradio module not found.

**Solution**:
```bash
pip install gradio
```

### "Invalid JSON"

**Problem**: JSON syntax error in state/event data.

**Solution**:
- Check for missing commas, quotes, or brackets
- Use a JSON validator
- Example valid JSON:
  ```json
  {"key": "value", "number": 123}
  ```

### Browser doesn't open automatically

**Problem**: Gradio launches but browser doesn't open.

**Solution**:
- Manually navigate to http://localhost:7860
- Check the terminal output for the correct URL

---

## Features in Detail

### State Management

The state management tab allows you to update the session's state using JSON. DataCat performs a **deep merge** on state updates:

```python
# Initial state
{"player": {"health": 100, "mana": 50}}

# Update with
{"player": {"health": 80}}

# Result
{"player": {"health": 80, "mana": 50}}  # mana preserved
```

### Event Logging

Events are logged with:
- **Name**: Identifier for the event type
- **Level**: Severity (debug, info, warning, error, critical)
- **Data**: Structured data about the event (JSON)

Events appear in the web UI timeline with timestamps and color-coding by level.

### Metrics

Metrics can be logged with:
- **Name**: Metric identifier (e.g., "fps", "memory_mb")
- **Value**: Numeric value
- **Tags**: Comma-separated labels for filtering

Tags can be:
- Simple: `performance,realtime`
- Key-value: `env:prod,region:us-west`

### Heartbeat

The heartbeat feature prevents hang detection. If heartbeats stop for more than 60 seconds, the daemon logs a hang event.

Use this to:
- Test hang detection
- Keep session alive during long operations
- Monitor application responsiveness

---

## Advanced Usage

### Custom Session Configuration

Create a custom session with specific parameters:

```python
session = client.create_session(
    product="MyApp",
    version="1.0.0",
    metadata={
        "environment": "production",
        "region": "us-west"
    }
)
```

### Programmatic Control

The demo is a standard Python script. You can modify it to:
- Add custom tabs
- Add pre-defined actions
- Integrate with your own systems
- Add validation logic

---

## File Structure

```
examples/demo_gui/
├── demo_gui.py           # Main demo application
├── requirements.txt      # Python dependencies (Gradio)
├── README.md            # Basic documentation
├── QUICKSTART.md        # Quick start guide
├── ICONS.md             # Icon setup instructions
└── daemon_config.json   # Daemon configuration
```

---

## Next Steps

- **[Python Examples](python-examples.md)** - Explore other Python examples
- **[Go Examples](go-examples.md)** - Go client library examples
- **[Quick Start Guide](../_guides/quickstart.md)** - Get started with DataCat
- **[API Reference](../_api/rest-api.md)** - Complete API documentation

