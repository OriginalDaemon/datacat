#!/usr/bin/env pwsh
# Run the datacat web UI dashboard

$ErrorActionPreference = "Stop"

# Build web binary only
Write-Host "Building datacat web UI..." -ForegroundColor Yellow
& "$PSScriptRoot\build.ps1" -Components web | Out-Null
Write-Host ""

Write-Host "Starting datacat web UI..." -ForegroundColor Green
Write-Host "Web UI will be available at http://localhost:8080" -ForegroundColor Cyan
Write-Host "Make sure the datacat server is running at http://localhost:9090" -ForegroundColor Yellow
Write-Host ""
Write-Host "Opening web UI in your browser..." -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop the web UI" -ForegroundColor Yellow
Write-Host ""

# Run the built binary
$repoRoot = Join-Path $PSScriptRoot ".."
$webBin = Join-Path $repoRoot "bin/datacat-web.exe"

if (-not (Test-Path $webBin)) {
    Write-Host "Error: Web UI binary not found at $webBin" -ForegroundColor Red
    exit 1
}

# Start the web UI in a background job
$webJob = Start-Job -ScriptBlock {
    param($binPath, $workDir)
    Set-Location $workDir
    & $binPath
} -ArgumentList $webBin, $repoRoot

# Wait a moment for the web UI to start
Start-Sleep -Seconds 2

# Open the browser
$webUrl = "http://localhost:8080"
try {
    Start-Process $webUrl
    Write-Host "Browser opened to $webUrl" -ForegroundColor Green
} catch {
    Write-Host "Could not automatically open browser. Please navigate to $webUrl" -ForegroundColor Yellow
}

Write-Host ""

try {
    # Wait for the job to finish (or Ctrl+C)
    Wait-Job -Job $webJob | Out-Null
    Receive-Job -Job $webJob
} finally {
    Write-Host "Stopping web UI..." -ForegroundColor Yellow
    Stop-Job -Job $webJob -ErrorAction SilentlyContinue
    Remove-Job -Job $webJob -Force -ErrorAction SilentlyContinue
}
