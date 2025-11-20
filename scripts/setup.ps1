#!/usr/bin/env pwsh
# Setup development environment

$ErrorActionPreference = "Stop"

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Setting up datacat development environment" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Define venv path relative to repository root
$repoRoot = Split-Path -Parent $PSScriptRoot
$venvPath = Join-Path $repoRoot ".venv"

# Check Go installation
Write-Host "Checking Go installation..." -ForegroundColor Green
try {
    $goVersion = go version
    Write-Host "  $goVersion" -ForegroundColor Gray
} catch {
    Write-Host "  Go is not installed! Please install Go from https://go.dev/dl/" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Check Python installation
Write-Host "Checking Python installation..." -ForegroundColor Green
try {
    $pythonVersion = python --version
    Write-Host "  $pythonVersion" -ForegroundColor Gray
} catch {
    Write-Host "  Python is not installed! Please install Python from https://python.org/" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Create Python virtual environment if it doesn't exist
if (-not (Test-Path $venvPath)) {
    Write-Host "Creating Python virtual environment at .venv..." -ForegroundColor Green
    python -m venv $venvPath
    Write-Host "  Virtual environment created successfully!" -ForegroundColor Gray
} else {
    Write-Host "Python virtual environment already exists at .venv" -ForegroundColor Green
}
Write-Host ""

# Activate virtual environment and install dependencies
Write-Host "Installing Python dependencies in virtual environment..." -ForegroundColor Green
$pipPath = Join-Path $venvPath "Scripts\pip.exe"
& $pipPath install --upgrade pip
& $pipPath install -r (Join-Path $repoRoot "requirements-dev.txt")
Write-Host ""

# Install Python client in development mode
Write-Host "Installing Python client in development mode..." -ForegroundColor Green
Push-Location (Join-Path $repoRoot "python")
try {
    & $pipPath install -e .
} finally {
    Pop-Location
}
Write-Host ""

# Download Go dependencies
Write-Host "Downloading Go dependencies..." -ForegroundColor Green
go mod download
Write-Host ""

Write-Host "======================================" -ForegroundColor Green
Write-Host "Setup complete!" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Run the server: .\scripts\run-server.ps1" -ForegroundColor Gray
Write-Host "  2. Run the web UI: .\scripts\run-web.ps1" -ForegroundColor Gray
Write-Host "  3. Run tests: .\scripts\test-all.ps1" -ForegroundColor Gray
Write-Host "  4. Build binaries: .\scripts\build.ps1" -ForegroundColor Gray
Write-Host ""
Write-Host "Note: All Python commands will automatically use the virtual environment at .venv" -ForegroundColor Yellow
