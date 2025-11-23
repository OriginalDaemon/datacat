#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Run the example game with DataCat logging

.DESCRIPTION
    Launches a single instance of the example game. The game simulates
    a typical game loop with events, metrics, state updates, and occasional
    errors and exceptions.

.PARAMETER Mode
    Game mode: normal, hang, or crash (default: normal)

.PARAMETER Duration
    How long to run in seconds (default: 60)

.PARAMETER Player
    Player name/identifier (default: random)

.PARAMETER NoAsync
    Disable async logging mode

.EXAMPLE
    .\scripts\run-example-game.ps1
    Run game normally for 60 seconds

.EXAMPLE
    .\scripts\run-example-game.ps1 -Mode hang -Duration 30
    Run game that will hang after 15 seconds

.EXAMPLE
    .\scripts\run-example-game.ps1 -Mode crash -Duration 20
    Run game that will crash after 15 seconds

.EXAMPLE
    .\scripts\run-example-game.ps1 -NoAsync
    Run game with synchronous logging
#>

param(
    [Parameter()]
    [ValidateSet("normal", "hang", "crash")]
    [string]$Mode = "normal",

    [Parameter()]
    [int]$Duration = 60,

    [Parameter()]
    [string]$Player = "",

    [Parameter()]
    [switch]$NoAsync
)

# Set error action
$ErrorActionPreference = "Stop"

# Get script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir
$ExamplesDir = Join-Path $RootDir "examples"

Write-Host ""
Write-Host "=====================================================================" -ForegroundColor Cyan
Write-Host "Example Game - DataCat Logging Demo" -ForegroundColor Cyan
Write-Host "=====================================================================" -ForegroundColor Cyan
Write-Host ""

# Check if server is running
$serverCheck = $null
try {
    $serverCheck = Invoke-WebRequest -Uri "http://localhost:9090/health" -TimeoutSec 2 -ErrorAction SilentlyContinue
} catch {
    # Server not running
}

if (-not $serverCheck) {
    Write-Host "WARNING: DataCat server doesn't appear to be running!" -ForegroundColor Yellow
    Write-Host "         Start it with: .\scripts\run-server.ps1" -ForegroundColor Yellow
    Write-Host ""
    $response = Read-Host "Continue anyway? (y/n)"
    if ($response -ne 'y' -and $response -ne 'Y') {
        exit 1
    }
    Write-Host ""
}

# Build command
$cmd = @(
    "python",
    (Join-Path $ExamplesDir "example_game.py"),
    "--mode", $Mode,
    "--duration", $Duration
)

if ($Player) {
    $cmd += @("--player", $Player)
}

if ($NoAsync) {
    $cmd += "--no-async"
}

Write-Host "Mode: $Mode" -ForegroundColor Green
Write-Host "Duration: $Duration seconds" -ForegroundColor Green
Write-Host "Async logging: $(-not $NoAsync)" -ForegroundColor Green
Write-Host ""
Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow
Write-Host ""

# Run game
& $cmd[0] $cmd[1..($cmd.Length-1)]

Write-Host ""
Write-Host "Game ended." -ForegroundColor Green
Write-Host ""
Write-Host "View session data at: http://localhost:8080" -ForegroundColor Cyan
Write-Host ""

