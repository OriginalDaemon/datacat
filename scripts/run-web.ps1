#!/usr/bin/env pwsh
# Run the datacat web UI dashboard

$ErrorActionPreference = "Stop"

# Build web binary only
Write-Host "Building datacat web UI..." -ForegroundColor Yellow
& "$PSScriptRoot\build.ps1" -Components web | Out-Null
Write-Host ""

Write-Host "Checking if datacat server is running..." -ForegroundColor Yellow
$serverCheck = $null
try {
    $serverCheck = Invoke-WebRequest -Uri "http://localhost:9090/health" -TimeoutSec 2 -ErrorAction SilentlyContinue
} catch {
    # Server not running
}

if (-not $serverCheck) {
    Write-Host ""
    Write-Host "WARNING: DataCat server doesn't appear to be running!" -ForegroundColor Yellow
    Write-Host "         The web UI requires the server to be running." -ForegroundColor Yellow
    Write-Host "         Start it with: .\scripts\run-server.ps1" -ForegroundColor Cyan
    Write-Host ""
    $response = Read-Host "Continue anyway? (y/n)"
    if ($response -ne 'y' -and $response -ne 'Y') {
        exit 1
    }
} else {
    Write-Host "✓ Server is running and healthy" -ForegroundColor Green
}

Write-Host ""
Write-Host "Starting datacat web UI..." -ForegroundColor Green
Write-Host "Web UI will be available at http://localhost:8080" -ForegroundColor Cyan
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

# Wait for web UI to be healthy
Write-Host "Waiting for web UI to be ready..." -ForegroundColor Yellow
$maxAttempts = 30
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
    Stop-Job -Job $webJob -ErrorAction SilentlyContinue
    Remove-Job -Job $webJob -Force -ErrorAction SilentlyContinue
    exit 1
}

Write-Host ""

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
