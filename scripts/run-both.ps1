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

# Wait for server to be healthy
Write-Host "Waiting for server to be ready..." -ForegroundColor Yellow
$maxAttempts = 30
$attempt = 0
$serverReady = $false

while ($attempt -lt $maxAttempts -and -not $serverReady) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:9090/health" -TimeoutSec 1 -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            $serverReady = $true
            Write-Host "✓ Server is healthy!" -ForegroundColor Green
        }
    } catch {
        # Not ready yet
    }
    if (-not $serverReady) {
        Start-Sleep -Milliseconds 500
        $attempt++
    }
}

if (-not $serverReady) {
    Write-Host "✗ Server failed to start within 15 seconds" -ForegroundColor Red
    Stop-Job -Job $serverJob -ErrorAction SilentlyContinue
    Remove-Job -Job $serverJob -Force -ErrorAction SilentlyContinue
    exit 1
}

# Start web UI in background job
$webJob = Start-Job -ScriptBlock {
    param($binPath, $workDir)
    Set-Location $workDir
    & $binPath
} -ArgumentList $webBin, $repoRoot

# Wait for web UI to be healthy
Write-Host "Waiting for web UI to be ready..." -ForegroundColor Yellow
$attempt = 0
$webReady = $false

while ($attempt -lt $maxAttempts -and -not $webReady) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -TimeoutSec 1 -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            $webReady = $true
            Write-Host "✓ Web UI is healthy!" -ForegroundColor Green
        }
    } catch {
        # Not ready yet
    }
    if (-not $webReady) {
        Start-Sleep -Milliseconds 500
        $attempt++
    }
}

if (-not $webReady) {
    Write-Host "✗ Web UI failed to start within 15 seconds" -ForegroundColor Red
    Stop-Job -Job $serverJob, $webJob -ErrorAction SilentlyContinue
    Remove-Job -Job $serverJob, $webJob -Force -ErrorAction SilentlyContinue
    exit 1
}

Write-Host ""
Write-Host "==================================================================" -ForegroundColor Green
Write-Host "Both services are running and healthy!" -ForegroundColor Green
Write-Host "==================================================================" -ForegroundColor Green
Write-Host "Server API: http://localhost:9090 (Health: http://localhost:9090/health)" -ForegroundColor Cyan
Write-Host "Web UI: http://localhost:8080 (Health: http://localhost:8080/health)" -ForegroundColor Cyan
Write-Host ""
Write-Host "Opening web UI in your browser..." -ForegroundColor Yellow
try {
    Start-Process "http://localhost:8080"
} catch {
    Write-Host "Could not automatically open browser" -ForegroundColor Yellow
}
Write-Host ""
Write-Host "Press Ctrl+C to stop both services" -ForegroundColor Yellow
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
