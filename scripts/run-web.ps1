#!/usr/bin/env pwsh
# Run the datacat web UI dashboard

$ErrorActionPreference = "Stop"

# Build binary first
Write-Host "Building datacat web UI..." -ForegroundColor Yellow
& "$PSScriptRoot\build.ps1" | Out-Null
Write-Host ""

Write-Host "Starting datacat web UI..." -ForegroundColor Green
Write-Host "Web UI will be available at http://localhost:8080" -ForegroundColor Cyan
Write-Host "Make sure the datacat server is running at http://localhost:9090" -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop the web UI" -ForegroundColor Yellow
Write-Host ""

# Run the built binary
$repoRoot = Join-Path $PSScriptRoot ".."
$webBin = Join-Path $repoRoot "bin/datacat-web.exe"

if (-not (Test-Path $webBin)) {
    Write-Host "Error: Web UI binary not found at $webBin" -ForegroundColor Red
    exit 1
}

# Change to repository root before running for consistency
Push-Location $repoRoot
try {
    & $webBin
} finally {
    Pop-Location
}
