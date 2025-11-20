# 🚀 datacat Demo GUI - Quick Start Guide

Get up and running with the datacat Demo GUI in under 5 minutes!

## Prerequisites

1. **datacat server running** on http://localhost:9090
2. **Python 3.6+** (or Python 2.7+ with limited features)

## Installation (3 steps)

### Step 1: Install the datacat Python client

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

### Step 3: Launch the demo

**Option A: Using PowerShell script (Windows - Recommended)**
```powershell
.\scripts\run-demo-gui.ps1
```

This automatically:
- Uses the virtual environment (`.venv`) if available
- Checks and installs Gradio if needed
- Verifies server is running
- Provides helpful diagnostics

**Option B: Direct**
```bash
cd examples/demo_gui
python demo_gui.py
```

Note: Make sure to activate your virtual environment first if using this method.

**That's it!** The demo will open in your browser at http://127.0.0.1:7860

## Quick Test

Once the demo opens (in dark mode by default):

**Tip:** Use Gradio's settings (⚙️ icon, bottom left) to switch to light mode if preferred

1. **Enter Product Info** - Fill in product name (e.g., "my-app") and version (e.g., "1.0.0")
2. **Start Session** - Click the "Start New Session" button (all controls become active)
3. **Update State** - Click "Update State" with the default JSON
4. **Log Event** - Click "Log Event" with the default values
5. **Generate Exception** - Select "ValueError" and click "Generate & Log Exception"
6. **Refresh Stats** - Click "Refresh Stats" to see your activity
7. **End Session** - Click "End Session" when done (button text changes dynamically)

**View your session**: The demo will show you a link to view the session in the web UI at http://localhost:8080

## Troubleshooting

### "Error creating session: Connection refused"

**Solution**: Start the datacat server first:
```bash
# Option 1: Using PowerShell script (Windows)
.\scripts\run-server.ps1

# Option 2: Manual
cd cmd/datacat-server
go run main.go config.go
```

### "Module 'gradio' not found"

**Solution**: Install Gradio:
```bash
pip install gradio
```

### "Module 'datacat' not found"

**Solution**: Install the datacat Python client:
```bash
cd python
pip install -e .
```

## What Can You Do?

### 📝 State Management
Send structured state updates with JSON editing. Try updating nested objects to see deep merge in action.

### 📢 Event Logging
Log events with custom names and data payloads. Great for tracking user actions or system events.

### 📈 Metrics Tracking
Send numeric metrics with tags. Useful for performance monitoring and analytics.

### ⚠️ Error Logging via Handler
Use the custom `DatacatLoggingHandler` to send log messages at different levels (DEBUG, INFO, WARNING, ERROR, CRITICAL).

### 💥 Exception Generation
Generate Python exceptions and see how they're captured with full stack traces. Try nested exceptions to see exception chaining.

### 📊 Session Statistics
View real-time statistics about your session including event counts, metric counts, and more.

## Next Steps

- **Explore the code**: Check out `examples/demo_gui/demo_gui.py` to see how it's built
- **Try the logging handler example**: Run `python examples/logging_handler_example.py`
- **Build your own integration**: Use the demo as a reference for your own applications
- **View the full documentation**: See [README.md](README.md) for complete details

## Features Demonstrated

✅ Session creation and management
✅ State updates with JSON editing
✅ Event logging with structured data
✅ Metrics with tags
✅ Custom logging handler integration
✅ Exception handling with stack traces
✅ Nested exception support
✅ Real-time statistics
✅ Direct and daemon modes
✅ Modern, responsive web UI

## Questions?

- Check the [full documentation](README.md)
- Look at other [examples](../README.md)
- Read the [Python client docs](../../python/README.md)
- View the [main README](../../README.md)

Enjoy exploring datacat! 🐱

