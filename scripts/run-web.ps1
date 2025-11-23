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
    Write-Host "[OK] Server is running and healthy" -ForegroundColor Green
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
# Capture both stdout and stderr
$webJob = Start-Job -ScriptBlock {
    param($binPath, $workDir)
    Set-Location $workDir
    & $binPath 2>&1
} -ArgumentList $webBin, $repoRoot

# Show initial output from the job
Write-Host "Job started, checking initial output..." -ForegroundColor Yellow
Start-Sleep -Milliseconds 500
$initialOutput = Receive-Job -Job $webJob -ErrorAction SilentlyContinue
if ($initialOutput) {
    Write-Host "Initial output from web UI:" -ForegroundColor Cyan
    $initialOutput | ForEach-Object { Write-Host $_ }
}

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
            Write-Host "[OK] Web UI is healthy!" -ForegroundColor Green
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
    Write-Host "[ERROR] Web UI failed to start within 15 seconds" -ForegroundColor Red
    Write-Host "Checking job output for errors..." -ForegroundColor Yellow
    $jobOutput = Receive-Job -Job $webJob -ErrorAction SilentlyContinue
    if ($jobOutput) {
        Write-Host "Job output:" -ForegroundColor Yellow
        $jobOutput | ForEach-Object { Write-Host $_ }
    }
    $jobError = Receive-Job -Job $webJob -ErrorAction SilentlyContinue -ErrorVariable jobErr
    if ($jobErr) {
        Write-Host "Job errors:" -ForegroundColor Red
        $jobErr | ForEach-Object { Write-Host $_ }
    }
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
    # Continuously show job output while waiting
    Write-Host "Web UI is running. Logs will appear below:" -ForegroundColor Green
    Write-Host "Press Ctrl+C to stop the web UI" -ForegroundColor Yellow
    Write-Host ""

    # Show output continuously
    while ($true) {
        $output = Receive-Job -Job $webJob -ErrorAction SilentlyContinue
        if ($output) {
            $output | ForEach-Object { Write-Host $_ }
        }

        # Check if job is still running
        if ($webJob.State -ne "Running") {
            break
        }

        Start-Sleep -Milliseconds 200
    }

    # Get any remaining output
    $finalOutput = Receive-Job -Job $webJob -ErrorAction SilentlyContinue
    if ($finalOutput) {
        $finalOutput | ForEach-Object { Write-Host $_ }
    }
} finally {
    Write-Host "Stopping web UI..." -ForegroundColor Yellow
    Stop-Job -Job $webJob -ErrorAction SilentlyContinue
    Remove-Job -Job $webJob -Force -ErrorAction SilentlyContinue
}
