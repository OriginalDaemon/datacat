#!/usr/bin/env pwsh
# Clean build artifacts and temporary files

Write-Host "Cleaning build artifacts..." -ForegroundColor Green

# Remove bin directory
if (Test-Path "bin") {
    Write-Host "  Removing bin/" -ForegroundColor Gray
    Remove-Item -Recurse -Force bin
}

# Remove BadgerDB data directory
if (Test-Path "badger_data") {
    Write-Host "  Removing badger_data/" -ForegroundColor Gray
    Remove-Item -Recurse -Force badger_data
}

# Remove coverage files
Get-ChildItem -Recurse -Filter "coverage.out" | ForEach-Object {
    Write-Host "  Removing $($_.FullName)" -ForegroundColor Gray
    Remove-Item -Force $_.FullName
}

Get-ChildItem -Recurse -Filter ".coverage" | ForEach-Object {
    Write-Host "  Removing $($_.FullName)" -ForegroundColor Gray
    Remove-Item -Force $_.FullName
}

if (Test-Path "htmlcov") {
    Write-Host "  Removing htmlcov/" -ForegroundColor Gray
    Remove-Item -Recurse -Force htmlcov
}

# Remove Python cache
Get-ChildItem -Recurse -Directory -Filter "__pycache__" | ForEach-Object {
    Write-Host "  Removing $($_.FullName)" -ForegroundColor Gray
    Remove-Item -Recurse -Force $_.FullName
}

Get-ChildItem -Recurse -Directory -Filter "*.egg-info" | ForEach-Object {
    Write-Host "  Removing $($_.FullName)" -ForegroundColor Gray
    Remove-Item -Recurse -Force $_.FullName
}

if (Test-Path "python/build") {
    Write-Host "  Removing python/build/" -ForegroundColor Gray
    Remove-Item -Recurse -Force python/build
}

if (Test-Path "python/dist") {
    Write-Host "  Removing python/dist/" -ForegroundColor Gray
    Remove-Item -Recurse -Force python/dist
}

Write-Host "Clean complete!" -ForegroundColor Green
