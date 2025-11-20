#!/usr/bin/env pwsh
# Build all Go binaries

param(
    [string]$Output = "bin"
)

$ErrorActionPreference = "Stop"

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "Building datacat binaries" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Create output directory
$binDir = Join-Path $PSScriptRoot ".." $Output
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

# Build server
Write-Host "Building datacat-server..." -ForegroundColor Green
Push-Location $PSScriptRoot/../cmd/datacat-server
try {
    $serverBin = Join-Path $binDir "datacat-server.exe"
    go build -o $serverBin
    Write-Host "  Output: $serverBin" -ForegroundColor Gray
} finally {
    Pop-Location
}
Write-Host ""

# Build web UI
Write-Host "Building datacat-web..." -ForegroundColor Green
Push-Location $PSScriptRoot/../cmd/datacat-web
try {
    $webBin = Join-Path $binDir "datacat-web.exe"
    go build -o $webBin
    Write-Host "  Output: $webBin" -ForegroundColor Gray
} finally {
    Pop-Location
}
Write-Host ""

# Build daemon
Write-Host "Building datacat-daemon..." -ForegroundColor Green
Push-Location $PSScriptRoot/../cmd/datacat-daemon
try {
    $daemonBin = Join-Path $binDir "datacat-daemon.exe"
    go build -o $daemonBin
    Write-Host "  Output: $daemonBin" -ForegroundColor Gray
} finally {
    Pop-Location
}
Write-Host ""

# Build Go example
Write-Host "Building go-client-example..." -ForegroundColor Green
Push-Location $PSScriptRoot/../examples/go-client-example
try {
    $exampleBin = Join-Path $binDir "go-client-example.exe"
    go build -o $exampleBin
    Write-Host "  Output: $exampleBin" -ForegroundColor Gray
} finally {
    Pop-Location
}
Write-Host ""

Write-Host "======================================" -ForegroundColor Green
Write-Host "Build complete!" -ForegroundColor Green
Write-Host "Binaries are in: $binDir" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green
