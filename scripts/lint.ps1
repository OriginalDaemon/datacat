#!/usr/bin/env pwsh
# Run code quality checks (Black formatter and mypy)

# Import common functions
. "$PSScriptRoot\common.ps1"

$black = Get-BlackExe
$mypy = Get-MypyExe

$ErrorActionPreference = "Stop"
$exitCode = 0

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Running Code Quality Checks" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Check Black formatting
Write-Host "Checking Python code formatting with Black..." -ForegroundColor Green
& $black --check python/ examples/ tests/
if ($LASTEXITCODE -ne 0) { 
    Write-Host "Black formatting check failed! Run '.\scripts\format.ps1' to fix." -ForegroundColor Red
    $exitCode = $LASTEXITCODE 
}
Write-Host ""

# Run mypy type checking
Write-Host "Running mypy type checking..." -ForegroundColor Green
& $mypy python/ --ignore-missing-imports
if ($LASTEXITCODE -ne 0) { 
    Write-Host "mypy type checking failed!" -ForegroundColor Red
    $exitCode = $LASTEXITCODE 
}
Write-Host ""

if ($exitCode -eq 0) {
    Write-Host "======================================" -ForegroundColor Green
    Write-Host "All checks passed!" -ForegroundColor Green
    Write-Host "======================================" -ForegroundColor Green
} else {
    Write-Host "======================================" -ForegroundColor Red
    Write-Host "Some checks failed!" -ForegroundColor Red
    Write-Host "======================================" -ForegroundColor Red
}

exit $exitCode
