#!/usr/bin/env pwsh
# Clean build artifacts and temporary files

param(
    [switch]$All
)

Write-Host "Cleaning build artifacts..." -ForegroundColor Green

# Remove bin directory
if (Test-Path "bin") {
    Write-Host "  Removing bin/" -ForegroundColor Gray
    Remove-Item -Recurse -Force bin
}

# Remove BadgerDB data directory (legacy name)
if (Test-Path "badger_data") {
    Write-Host "  Removing badger_data/" -ForegroundColor Gray
    Remove-Item -Recurse -Force badger_data
}

# Remove datacat data directory (current name)
if (Test-Path "datacat_data") {
    Write-Host "  Removing datacat_data/" -ForegroundColor Gray
    Remove-Item -Recurse -Force datacat_data
}

# Remove config file
if (Test-Path "config.json") {
    Write-Host "  Removing config.json" -ForegroundColor Gray
    Remove-Item -Force config.json
}

# Remove daemon configs directory
if (Test-Path "tmp/daemon_configs") {
    Write-Host "  Removing tmp/daemon_configs/" -ForegroundColor Gray
    Remove-Item -Recurse -Force tmp/daemon_configs
}

# Remove any stray daemon config files in root
Get-ChildItem -Filter "daemon_config_*.json" -ErrorAction SilentlyContinue | ForEach-Object {
    Write-Host "  Removing $($_.Name)" -ForegroundColor Gray
    Remove-Item -Force $_.FullName
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

# Remove virtual environment if -All flag is specified
if ($All -and (Test-Path ".venv")) {
    Write-Host "  Removing .venv/ (virtual environment)" -ForegroundColor Gray
    Remove-Item -Recurse -Force .venv
}

Write-Host "Clean complete!" -ForegroundColor Green

if ($All) {
    Write-Host ""
    Write-Host "Note: Virtual environment was removed. Run '.\scripts\setup.ps1' to recreate it." -ForegroundColor Yellow
}
