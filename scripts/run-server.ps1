#!/usr/bin/env pwsh
# Run the datacat REST API server

$ErrorActionPreference = "Stop"

# Build server binary only
Write-Host "Building datacat server..." -ForegroundColor Yellow
& "$PSScriptRoot\build.ps1" -Components server | Out-Null
Write-Host ""

Write-Host "Starting datacat server..." -ForegroundColor Green
Write-Host ""

# Run the built binary
$repoRoot = Join-Path $PSScriptRoot ".."
$serverBin = Join-Path $repoRoot "bin/datacat-server.exe"

if (-not (Test-Path $serverBin)) {
    Write-Host "Error: Server binary not found at $serverBin" -ForegroundColor Red
    exit 1
}

# Start server in background job to check health
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
        }
    } catch {
        # Not ready yet
    }
    if (-not $serverReady) {
        Start-Sleep -Milliseconds 500
        $attempt++
    }
}

if ($serverReady) {
    Write-Host "[OK] Server is healthy!" -ForegroundColor Green
    Write-Host ""
    Write-Host "==================================================================" -ForegroundColor Green
    Write-Host "DataCat Server is running!" -ForegroundColor Green
    Write-Host "==================================================================" -ForegroundColor Green
    Write-Host "API: http://localhost:9090" -ForegroundColor Cyan
    Write-Host "Health: http://localhost:9090/health" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
    Write-Host ""
} else {
    Write-Host "[ERROR] Server failed to start within 15 seconds" -ForegroundColor Red
    Stop-Job -Job $serverJob -ErrorAction SilentlyContinue
    Receive-Job -Job $serverJob
    Remove-Job -Job $serverJob -Force -ErrorAction SilentlyContinue
    exit 1
}

try {
    # Wait for the job to finish (or Ctrl+C)
    Wait-Job -Job $serverJob | Out-Null
    Receive-Job -Job $serverJob
} finally {
    Write-Host "Stopping server..." -ForegroundColor Yellow
    Stop-Job -Job $serverJob -ErrorAction SilentlyContinue
    Remove-Job -Job $serverJob -Force -ErrorAction SilentlyContinue
}
