# PowerShell Scripts for Windows

This directory contains convenient PowerShell scripts for running DataCat services, tests, and examples on Windows.

## Prerequisites

- **PowerShell 5.1+** or **PowerShell Core 7+** (recommended)
- **Go 1.19+** - Download from [go.dev](https://go.dev/dl/)
- **Python 3.7+** - Download from [python.org](https://python.org/)

## Setup

First, set up your development environment:

```powershell
.\scripts\setup.ps1
```

This will:
- Check Go and Python installations
- Create a Python virtual environment at `.venv/`
- Install Python dependencies in the virtual environment
- Install the Python client in development mode
- Download Go dependencies

**Note:** All Python-related scripts automatically use the virtual environment at `.venv/`. If the virtual environment is not found, scripts will fall back to system Python with a warning.

## Running Services

### Start the REST API Server

```powershell
.\scripts\run-server.ps1
```

This will build the server binary (if not already built) and start it. The server will be available at `http://localhost:9090`

### Start the Web UI Dashboard

```powershell
.\scripts\run-web.ps1
```

This will build the web UI binary (if not already built) and start it. The web UI will be available at `http://localhost:8080`

**Note:** The web UI requires the server to be running at `http://localhost:9090`

### Start Both Services

```powershell
.\scripts\run-both.ps1
```

This will build both binaries (if not already built) and start both the server and web UI in parallel. Press Ctrl+C to stop both services.

## Running Tests

### Run All Tests

```powershell
.\scripts\test-all.ps1
```

Run with coverage:

```powershell
.\scripts\test-all.ps1 -Coverage
```

### Run Python Integration Tests Only

```powershell
.\scripts\test-python.ps1
```

This automatically starts the server, runs the tests, and stops the server.

## Code Quality

### Check Code Formatting and Type Checking

```powershell
.\scripts\lint.ps1
```

This runs:
- Black formatter check (Python)
- mypy type checking (Python)

### Format Python Code

```powershell
.\scripts\format.ps1
```

This automatically formats all Python code with Black.

## Building

### Build All Binaries

```powershell
.\scripts\build.ps1
```

Binaries will be output to the `bin/` directory:
- `bin/datacat-server.exe` - REST API server
- `bin/datacat-web.exe` - Web UI dashboard
- `bin/datacat-daemon.exe` - Local batching daemon
- `bin/go-client-example.exe` - Go client example

Custom output directory:

```powershell
.\scripts\build.ps1 -Output "custom/path"
```

**Note:** The build script is automatically run by `run-server.ps1`, `run-web.ps1`, and `run-both.ps1` to ensure binaries are up to date before running.

## Running Examples

### Run the Interactive Demo GUI

```powershell
.\scripts\run-demo-gui.ps1
```

This launches the modern web-based demo GUI, which provides an interactive interface for exploring all datacat features:
- 📝 State management with JSON editing
- 📢 Event logging
- 📈 Metrics tracking
- ⚠️ Error logging via custom handler
- 💥 Exception generation with full stack traces

The script will:
- **Automatically use the virtual environment** (`.venv`) if available
- Check prerequisites (Python, Gradio, datacat client)
- Offer to install Gradio if missing (into the venv)
- Check if the server is running
- Launch the demo at http://127.0.0.1:7860

**Note:** The demo requires the datacat server to be running. If you haven't set up the virtual environment yet, run `.\scripts\setup.ps1` first. See [examples/demo_gui/](../examples/demo_gui/) for more details.

### Run an Example Application

```powershell
# Basic example
.\scripts\run-example.ps1 -Example basic

# Window tracking example
.\scripts\run-example.ps1 -Example window_tracking

# Heartbeat monitoring example
.\scripts\run-example.ps1 -Example heartbeat

# Exception logging example
.\scripts\run-example.ps1 -Example exception_logging

# Complete feature demo
.\scripts\run-example.ps1 -Example complete

# Testing/CI example
.\scripts\run-example.ps1 -Example testing

# Go client example
.\scripts\run-example.ps1 -Example go-client
```

**Note:** The server must be running before executing examples. Examples use the daemon mode by default, which requires the `datacat-daemon.exe` binary to be built. The build script handles this automatically.

## Cleanup

### Remove Build Artifacts

```powershell
.\scripts\clean.ps1
```

This removes:
- `bin/` - Compiled binaries
- `datacat_data/` - Database files (session data)
- `config.json` - Server configuration file
- `badger_data/` - Legacy database directory (if it exists)
- Coverage reports
- Python cache files (`__pycache__`, `*.egg-info`)

**Note:** This will delete all session data from the server. Make sure to backup any important data before running.

### Remove Everything Including Virtual Environment

```powershell
.\scripts\clean.ps1 -All
```

This removes all of the above plus:
- `.venv/` - Python virtual environment

**Note:** After using `-All`, you'll need to run `.\scripts\setup.ps1` again to recreate the virtual environment.

## Example Workflow

1. **Initial setup:**
   ```powershell
   .\scripts\setup.ps1
   ```

2. **Build binaries (optional - run scripts do this automatically):**
   ```powershell
   .\scripts\build.ps1
   ```

3. **Start services:**
   ```powershell
   .\scripts\run-both.ps1
   ```

4. **In another terminal, run an example:**
   ```powershell
   .\scripts\run-example.ps1 -Example complete
   ```

5. **Run tests:**
   ```powershell
   .\scripts\test-all.ps1 -Coverage
   ```

## Troubleshooting

### Script Execution Policy

If you get an error about script execution being disabled, run PowerShell as Administrator and execute:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Port Already in Use

If port 9090 or 8080 is already in use, you'll need to:
1. Stop the process using the port, or
2. Modify the port in the respective configuration or source files:
   - Server: Edit `config.json` (in repository root when using scripts) and change `server_port`
   - Web UI: Edit `cmd/datacat-web/main.go` and change the `port` variable

### Data Storage Location

When running the server with scripts (`run-server.ps1` or `run-both.ps1`), the database and configuration files are created in the **repository root directory**:

- **Data directory:** `./datacat_data/` (contains BadgerDB files)
- **Config file:** `./config.json`

The scripts explicitly change to the repository root directory before executing the binaries to ensure data is stored in a consistent, predictable location.

**To delete all session data:**

```powershell
# Stop the server first (Ctrl+C), then:
Remove-Item -Recurse -Force ./datacat_data
Remove-Item -Force ./config.json  # Optional - removes custom config
```

Or use the cleanup script:
```powershell
.\scripts\clean.ps1
```

See [FAQ in main README](../README.md#-faq) for more details on data management.

### Python Dependencies Issues

If you encounter Python dependency issues, try:

```powershell
# Delete the virtual environment and recreate it
Remove-Item -Recurse -Force .venv
.\scripts\setup.ps1
```

Alternatively, you can manually reinstall dependencies in the virtual environment:

```powershell
.\.venv\Scripts\pip.exe install --upgrade pip
.\.venv\Scripts\pip.exe install -r requirements-dev.txt --force-reinstall
```

## Script Reference

| Script | Description |
|--------|-------------|
| `setup.ps1` | Setup development environment |
| `build.ps1` | Build all Go binaries |
| `run-server.ps1` | Build and start REST API server |
| `run-web.ps1` | Build and start web UI dashboard |
| `run-both.ps1` | Build and start both services in parallel |
| `run-demo-gui.ps1` | Launch the interactive demo GUI |
| `run-example.ps1` | Run example applications |
| `test-all.ps1` | Run all tests (Go + Python) |
| `test-python.ps1` | Run Python integration tests |
| `lint.ps1` | Check code quality (Black + mypy) |
| `format.ps1` | Format Python code with Black |
| `clean.ps1` | Remove build artifacts |

## Contributing

When developing on Windows, use these scripts to ensure consistency with the CI/CD pipeline:

1. **Before committing:**
   ```powershell
   .\scripts\format.ps1
   .\scripts\lint.ps1
   .\scripts\test-all.ps1
   ```

2. **Before creating a PR:**
   ```powershell
   .\scripts\test-all.ps1 -Coverage
   ```

All PRs must pass Black formatting, mypy type checking, and have 80% minimum code coverage.
