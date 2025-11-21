#!/usr/bin/env pwsh
# Build Go binaries selectively or all at once

param(
    [string]$Output = "bin",
    [ValidateSet("all", "server", "web", "daemon", "example")]
    [string[]]$Components = @("all")
)

$ErrorActionPreference = "Stop"

# Create output directory
$repoRoot = Join-Path $PSScriptRoot ".."
$binDir = Join-Path $repoRoot $Output
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

# Determine what to build
$buildAll = $Components -contains "all"
$buildServer = $buildAll -or ($Components -contains "server")
$buildWeb = $buildAll -or ($Components -contains "web")
$buildDaemon = $buildAll -or ($Components -contains "daemon")
$buildExample = $buildAll -or ($Components -contains "example")

if ($buildAll) {
    Write-Host "======================================" -ForegroundColor Cyan
    Write-Host "Building all datacat binaries" -ForegroundColor Cyan
    Write-Host "======================================" -ForegroundColor Cyan
    Write-Host ""
}

# Build server
if ($buildServer) {
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
}

# Build web UI
if ($buildWeb) {
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
}

# Build daemon
if ($buildDaemon) {
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
}

# Build Go example
if ($buildExample) {
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
}

if ($buildAll) {
    Write-Host "======================================" -ForegroundColor Green
    Write-Host "Build complete!" -ForegroundColor Green
    Write-Host "Binaries are in: $binDir" -ForegroundColor Green
    Write-Host "======================================" -ForegroundColor Green
}
