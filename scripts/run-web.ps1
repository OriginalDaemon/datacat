#!/usr/bin/env pwsh
# Run the datacat web UI dashboard

Write-Host "Starting datacat web UI..." -ForegroundColor Green
Write-Host "Web UI will be available at http://localhost:8080" -ForegroundColor Cyan
Write-Host "Make sure the datacat server is running at http://localhost:9090" -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop the web UI" -ForegroundColor Yellow
Write-Host ""

Push-Location $PSScriptRoot/../cmd/datacat-web
try {
    go run main.go
} finally {
    Pop-Location
}
