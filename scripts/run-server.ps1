#!/usr/bin/env pwsh
# Run the datacat REST API server

Write-Host "Starting datacat server..." -ForegroundColor Green
Write-Host "API will be available at http://localhost:9090" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
Write-Host ""

Push-Location $PSScriptRoot/../cmd/datacat-server
try {
    go run .
} finally {
    Pop-Location
}
