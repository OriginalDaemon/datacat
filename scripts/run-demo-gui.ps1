#!/usr/bin/env pwsh
# Run the datacat Demo GUI
# This script launches the modern web-based demo GUI for the datacat Python client

# Import common functions
. "$PSScriptRoot\common.ps1"

$python = Get-PythonExe

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "datacat Demo GUI Launcher" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Check prerequisites
Write-Host "Checking prerequisites..." -ForegroundColor Yellow
Write-Host ""

# Show which Python is being used
if ($python -like "*\.venv\*") {
    Write-Host "Using Python from virtual environment (.venv)" -ForegroundColor Green
} else {
    Write-Host "Using system Python (venv not found)" -ForegroundColor Yellow
}

# Check Python version
$pythonVersion = & $python --version 2>&1
Write-Host "Python: $pythonVersion (at: $python)" -ForegroundColor Green

# Check if Gradio is installed
Write-Host "Checking for Gradio..." -ForegroundColor Yellow
$gradioCheck = & $python -c "import gradio; print(gradio.__version__)" 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "Gradio: $gradioCheck" -ForegroundColor Green
} else {
    Write-Host "Gradio: NOT INSTALLED" -ForegroundColor Red
    Write-Host ""
    Write-Host "Gradio is required to run the demo GUI." -ForegroundColor Yellow
    if ($python -like "*\.venv\*") {
        Write-Host "It will be installed in the virtual environment (.venv)" -ForegroundColor Cyan
    }
    Write-Host ""
    $install = Read-Host "Would you like to install it now? (y/n)"
    if ($install -eq "y") {
        Write-Host ""
        if ($python -like "*\.venv\*") {
            Write-Host "Installing Gradio into virtual environment..." -ForegroundColor Yellow
        } else {
            Write-Host "Installing Gradio..." -ForegroundColor Yellow
        }
        & $python -m pip install gradio
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Failed to install Gradio!" -ForegroundColor Red
            if ($python -like "*\.venv\*") {
                Write-Host "Please install manually:" -ForegroundColor Yellow
                Write-Host "  .\.venv\Scripts\pip.exe install gradio" -ForegroundColor White
            } else {
                Write-Host "Please install manually: pip install gradio" -ForegroundColor Yellow
            }
            exit 1
        }
        Write-Host "Gradio installed successfully!" -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "Please install Gradio to continue:" -ForegroundColor Yellow
        if ($python -like "*\.venv\*") {
            Write-Host "  .\.venv\Scripts\pip.exe install gradio" -ForegroundColor White
            Write-Host ""
            Write-Host "Or install from requirements:" -ForegroundColor Yellow
            Write-Host "  .\.venv\Scripts\pip.exe install -r examples/demo_gui/requirements.txt" -ForegroundColor White
        } else {
            Write-Host "  pip install gradio" -ForegroundColor White
            Write-Host ""
            Write-Host "Or install from requirements:" -ForegroundColor Yellow
            Write-Host "  cd examples/demo_gui" -ForegroundColor White
            Write-Host "  pip install -r requirements.txt" -ForegroundColor White
        }
        exit 1
    }
}

# Check if datacat client is available
Write-Host "Checking for datacat Python client..." -ForegroundColor Yellow
& $python -c "import sys; sys.path.insert(0, 'python'); import datacat; print('OK')" 2>&1 | Out-Null
if ($LASTEXITCODE -eq 0) {
    Write-Host "datacat client: Available" -ForegroundColor Green
} else {
    Write-Host "datacat client: NOT AVAILABLE" -ForegroundColor Red
    Write-Host ""
    Write-Host "The datacat Python client must be installed." -ForegroundColor Yellow
    Write-Host "Please run:" -ForegroundColor Yellow
    Write-Host "  cd python" -ForegroundColor White
    Write-Host "  pip install -e ." -ForegroundColor White
    exit 1
}

Write-Host ""

# Check if server is running
Write-Host "Checking if datacat server is running..." -ForegroundColor Yellow
try {
    Invoke-WebRequest -Uri "http://localhost:9090/health" -Method GET -UseBasicParsing -ErrorAction Stop -TimeoutSec 2 | Out-Null
    Write-Host "Server: Running and healthy on http://localhost:9090" -ForegroundColor Green
} catch {
    Write-Host "Server: NOT RUNNING" -ForegroundColor Red
    Write-Host ""
    Write-Host "The datacat server doesn't appear to be running." -ForegroundColor Yellow
    Write-Host "Please start it before using the demo:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Option 1: Using PowerShell script" -ForegroundColor Cyan
    Write-Host "    .\scripts\run-server.ps1" -ForegroundColor White
    Write-Host ""
    Write-Host "  Option 2: Run from source" -ForegroundColor Cyan
    Write-Host "    cd cmd/datacat-server" -ForegroundColor White
    Write-Host "    go run main.go config.go" -ForegroundColor White
    Write-Host ""
    Write-Host "  Option 3: Run built binary" -ForegroundColor Cyan
    Write-Host "    .\bin\datacat-server.exe" -ForegroundColor White
    Write-Host ""

    $continue = Read-Host "Continue anyway? (y/n)"
    if ($continue -ne "y") {
        exit 0
    }
}

Write-Host ""
Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Launching Demo GUI..." -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "The demo will open in your browser at:" -ForegroundColor Yellow
Write-Host "  http://127.0.0.1:7860" -ForegroundColor White
Write-Host ""
Write-Host "View sessions in the web UI at:" -ForegroundColor Yellow
Write-Host "  http://localhost:8080" -ForegroundColor White
Write-Host ""
Write-Host "Press Ctrl+C to stop the demo" -ForegroundColor Gray
Write-Host ""

# Change to the demo_gui directory and run
Push-Location "$PSScriptRoot\..\examples\demo_gui"
try {
    & $python demo_gui.py
} catch {
    Write-Host ""
    Write-Host "Error launching demo: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

