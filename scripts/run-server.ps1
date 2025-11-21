#!/usr/bin/env pwsh
# Run the datacat REST API server

$ErrorActionPreference = "Stop"

# Build server binary only
Write-Host "Building datacat server..." -ForegroundColor Yellow
& "$PSScriptRoot\build.ps1" -Components server | Out-Null
Write-Host ""

Write-Host "Starting datacat server..." -ForegroundColor Green
Write-Host "API will be available at http://localhost:9090" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
Write-Host ""

# Run the built binary
$repoRoot = Join-Path $PSScriptRoot ".."
$serverBin = Join-Path $repoRoot "bin/datacat-server.exe"

if (-not (Test-Path $serverBin)) {
    Write-Host "Error: Server binary not found at $serverBin" -ForegroundColor Red
    exit 1
}

# Change to repository root before running to ensure data is stored there
Push-Location $repoRoot
try {
    & $serverBin
} finally {
    Pop-Location
}
