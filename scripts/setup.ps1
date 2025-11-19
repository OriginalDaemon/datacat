#!/usr/bin/env pwsh
# Setup development environment

$ErrorActionPreference = "Stop"

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Setting up datacat development environment" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

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

# Install Python dependencies
Write-Host "Installing Python dependencies..." -ForegroundColor Green
pip install -r requirements-dev.txt
Write-Host ""

# Install Python client in development mode
Write-Host "Installing Python client in development mode..." -ForegroundColor Green
Push-Location $PSScriptRoot/../python
try {
    pip install -e .
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
