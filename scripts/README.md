# PowerShell Scripts for Windows

This directory contains convenient PowerShell scripts for running datacat services, tests, and examples on Windows.

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
- Install Python dependencies
- Install the Python client in development mode
- Download Go dependencies

## Running Services

### Start the REST API Server

```powershell
.\scripts\run-server.ps1
```

The server will be available at `http://localhost:9090`

### Start the Web UI Dashboard

```powershell
.\scripts\run-web.ps1
```

The web UI will be available at `http://localhost:8080`

**Note:** The web UI requires the server to be running at `http://localhost:9090`

### Start Both Services

```powershell
.\scripts\run-both.ps1
```

This starts both the server and web UI in parallel. Press Ctrl+C to stop both services.

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
- `bin/go-client-example.exe` - Go client example

Custom output directory:

```powershell
.\scripts\build.ps1 -Output "custom/path"
```

## Running Examples

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

**Note:** The server must be running before executing examples.

## Cleanup

### Remove Build Artifacts

```powershell
.\scripts\clean.ps1
```

This removes:
- `bin/` - Compiled binaries
- `badger_data/` - Database files
- Coverage reports
- Python cache files (`__pycache__`, `*.egg-info`)

## Example Workflow

1. **Initial setup:**
   ```powershell
   .\scripts\setup.ps1
   ```

2. **Start services:**
   ```powershell
   .\scripts\run-both.ps1
   ```

3. **In another terminal, run an example:**
   ```powershell
   .\scripts\run-example.ps1 -Example complete
   ```

4. **Run tests:**
   ```powershell
   .\scripts\test-all.ps1 -Coverage
   ```

5. **Build binaries for distribution:**
   ```powershell
   .\scripts\build.ps1
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
   - Server: Edit `cmd/datacat-server/config.json` and change `server_port`
   - Web UI: Edit `cmd/datacat-web/main.go` and change the `port` variable

### Python Dependencies Issues

If you encounter Python dependency issues, try:

```powershell
pip install --upgrade pip
pip install -r requirements-dev.txt --force-reinstall
```

## Script Reference

| Script | Description |
|--------|-------------|
| `setup.ps1` | Setup development environment |
| `run-server.ps1` | Start REST API server |
| `run-web.ps1` | Start web UI dashboard |
| `run-both.ps1` | Start both services in parallel |
| `run-example.ps1` | Run example applications |
| `test-all.ps1` | Run all tests (Go + Python) |
| `test-python.ps1` | Run Python integration tests |
| `lint.ps1` | Check code quality (Black + mypy) |
| `format.ps1` | Format Python code with Black |
| `build.ps1` | Build all Go binaries |
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
