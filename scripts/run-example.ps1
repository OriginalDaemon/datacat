#!/usr/bin/env pwsh
# Run example applications

param(
    [Parameter(Mandatory=$true)]
    [ValidateSet(
        "basic",
        "window_tracking",
        "heartbeat",
        "testing",
        "complete",
        "exception_logging",
        "go-client"
    )]
    [string]$Example
)

# Import common functions
. "$PSScriptRoot\common.ps1"

$python = Get-PythonExe

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Running example: $Example" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Check if server is running
Write-Host "Checking if datacat server is running..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:9090/health" -Method GET -UseBasicParsing -ErrorAction Stop -TimeoutSec 2
    Write-Host "✓ Server is running and healthy!" -ForegroundColor Green
} catch {
    Write-Host "✗ Server is not running!" -ForegroundColor Red
    Write-Host "Please start the server first with: .\scripts\run-server.ps1" -ForegroundColor Yellow
    exit 1
}
Write-Host ""

Push-Location $PSScriptRoot/../examples
try {
    switch ($Example) {
        "basic" {
            & $python basic_example.py
        }
        "window_tracking" {
            & $python window_tracking_example.py
        }
        "heartbeat" {
            & $python heartbeat_example.py
        }
        "testing" {
            & $python testing_example.py
        }
        "complete" {
            & $python complete_example.py
        }
        "exception_logging" {
            & $python exception_logging_example.py
        }
        "go-client" {
            Push-Location go-client-example
            try {
                go run .
            } finally {
                Pop-Location
            }
        }
    }
} finally {
    Pop-Location
}
