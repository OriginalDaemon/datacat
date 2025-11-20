#!/usr/bin/env pwsh
# Run Python integration tests

param(
    [switch]$Coverage
)

# Import common functions
. "$PSScriptRoot\common.ps1"

$pytest = Get-PytestExe

Write-Host "Starting datacat server for testing..." -ForegroundColor Green

# Start server in background
$serverJob = Start-Job -ScriptBlock {
    Set-Location $using:PSScriptRoot/../cmd/datacat-server
    go run .
}

# Wait for server to start
Start-Sleep -Seconds 3

try {
    Write-Host "Running Python integration tests..." -ForegroundColor Cyan
    Write-Host ""
    
    Push-Location $PSScriptRoot/..
    try {
        if ($Coverage) {
            & $pytest tests/ -v --cov=python --cov-report=term --cov-report=html
        } else {
            & $pytest tests/ -v
        }
        $exitCode = $LASTEXITCODE
    } finally {
        Pop-Location
    }
    
    exit $exitCode
} finally {
    Write-Host "Stopping test server..." -ForegroundColor Yellow
    Stop-Job -Job $serverJob -ErrorAction SilentlyContinue
    Remove-Job -Job $serverJob -Force -ErrorAction SilentlyContinue
}
