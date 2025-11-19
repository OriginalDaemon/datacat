#!/usr/bin/env pwsh
# Format Python code with Black

Write-Host "Formatting Python code with Black..." -ForegroundColor Green
black python/ examples/ tests/

if ($LASTEXITCODE -eq 0) {
    Write-Host "Code formatted successfully!" -ForegroundColor Green
} else {
    Write-Host "Formatting failed!" -ForegroundColor Red
    exit $LASTEXITCODE
}
