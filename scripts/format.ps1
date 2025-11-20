#!/usr/bin/env pwsh
# Format Python code with Black

# Import common functions
. "$PSScriptRoot\common.ps1"

$black = Get-BlackExe

Write-Host "Formatting Python code with Black..." -ForegroundColor Green
& $black python/ examples/ tests/

if ($LASTEXITCODE -eq 0) {
    Write-Host "Code formatted successfully!" -ForegroundColor Green
} else {
    Write-Host "Formatting failed!" -ForegroundColor Red
    exit $LASTEXITCODE
}
