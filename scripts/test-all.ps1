#!/usr/bin/env pwsh
# Run all tests (Go and Python)

param(
    [switch]$Coverage
)

$ErrorActionPreference = "Stop"
$exitCode = 0

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Running All Tests" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Test Go client library
Write-Host "Testing Go client library..." -ForegroundColor Green
Push-Location $PSScriptRoot/../client
try {
    if ($Coverage) {
        go test -v -coverprofile=coverage.out ./...
        if ($LASTEXITCODE -ne 0) { $exitCode = $LASTEXITCODE }
    } else {
        go test -v ./...
        if ($LASTEXITCODE -ne 0) { $exitCode = $LASTEXITCODE }
    }
} finally {
    Pop-Location
}
Write-Host ""

# Test Python integration
Write-Host "Testing Python integration..." -ForegroundColor Green
Push-Location $PSScriptRoot/..
try {
    if ($Coverage) {
        pytest tests/ -v --cov=python --cov-report=term --cov-report=html
        if ($LASTEXITCODE -ne 0) { $exitCode = $LASTEXITCODE }
    } else {
        pytest tests/ -v
        if ($LASTEXITCODE -ne 0) { $exitCode = $LASTEXITCODE }
    }
} finally {
    Pop-Location
}
Write-Host ""

if ($exitCode -eq 0) {
    Write-Host "======================================" -ForegroundColor Green
    Write-Host "All tests passed!" -ForegroundColor Green
    Write-Host "======================================" -ForegroundColor Green
} else {
    Write-Host "======================================" -ForegroundColor Red
    Write-Host "Some tests failed!" -ForegroundColor Red
    Write-Host "======================================" -ForegroundColor Red
}

exit $exitCode
