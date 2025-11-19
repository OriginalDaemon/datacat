#!/usr/bin/env pwsh
# Run both the datacat server and web UI in parallel

Write-Host "Starting datacat server and web UI..." -ForegroundColor Green
Write-Host "Server API: http://localhost:8080" -ForegroundColor Cyan
Write-Host "Web UI: http://localhost:8081" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop both services" -ForegroundColor Yellow
Write-Host ""

# Start server in background job
$serverJob = Start-Job -ScriptBlock {
    Set-Location $using:PSScriptRoot/../cmd/datacat-server
    go run main.go
}

# Wait a moment for server to start
Start-Sleep -Seconds 2

# Start web UI in background job
$webJob = Start-Job -ScriptBlock {
    Set-Location $using:PSScriptRoot/../cmd/datacat-web
    go run main.go
}

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
