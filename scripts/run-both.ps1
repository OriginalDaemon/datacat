#!/usr/bin/env pwsh
# Run both the datacat server and web UI in parallel

$ErrorActionPreference = "Stop"

# Build server and web binaries only
Write-Host "Building datacat server and web UI..." -ForegroundColor Yellow
& "$PSScriptRoot\build.ps1" -Components server,web | Out-Null
Write-Host ""

Write-Host "Starting datacat server and web UI..." -ForegroundColor Green
Write-Host "Server API: http://localhost:9090" -ForegroundColor Cyan
Write-Host "Web UI: http://localhost:8080" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop both services" -ForegroundColor Yellow
Write-Host ""

# Get binary paths
$repoRoot = Join-Path $PSScriptRoot ".."
$serverBin = Join-Path $repoRoot "bin/datacat-server.exe"
$webBin = Join-Path $repoRoot "bin/datacat-web.exe"

if (-not (Test-Path $serverBin)) {
    Write-Host "Error: Server binary not found at $serverBin" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $webBin)) {
    Write-Host "Error: Web UI binary not found at $webBin" -ForegroundColor Red
    exit 1
}

# Start server in background job
$serverJob = Start-Job -ScriptBlock {
    param($binPath, $workDir)
    Set-Location $workDir
    & $binPath
} -ArgumentList $serverBin, $repoRoot

# Wait a moment for server to start
Start-Sleep -Seconds 2

# Start web UI in background job
$webJob = Start-Job -ScriptBlock {
    param($binPath, $workDir)
    Set-Location $workDir
    & $binPath
} -ArgumentList $webBin, $repoRoot

Write-Host "Both services started!" -ForegroundColor Green
Write-Host "Server job ID: $($serverJob.Id)" -ForegroundColor Gray
Write-Host "Web UI job ID: $($webJob.Id)" -ForegroundColor Gray
Write-Host ""

try {
    # Wait for either job to finish (shouldn't happen unless error)
    Wait-Job -Job $serverJob, $webJob -Any | Out-Null

    # Show output from completed jobs
    Receive-Job -Job $serverJob, $webJob
} finally {
    Write-Host "Stopping services..." -ForegroundColor Yellow
    Stop-Job -Job $serverJob, $webJob -ErrorAction SilentlyContinue
    Remove-Job -Job $serverJob, $webJob -Force -ErrorAction SilentlyContinue
}
