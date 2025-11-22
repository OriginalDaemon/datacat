#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Launch multiple game instances simultaneously

.DESCRIPTION
    Spawns multiple instances of the example game with different configurations.
    Some will run normally, some will hang, and some will crash - demonstrating
    DataCat's crash detection and hang detection capabilities.

    This creates a realistic simulation of multiple players running the same game,
    making it easy to see DataCat's capabilities in the web UI.

.PARAMETER Count
    Number of game instances to launch (default: 10)

.PARAMETER Duration
    How long each game runs in seconds (default: 60)

.PARAMETER HangRate
    Fraction of games that will hang (default: 0.15)

.PARAMETER CrashRate
    Fraction of games that will crash (default: 0.15)

.PARAMETER Stagger
    Seconds to wait between launching instances (default: 1.0)

.PARAMETER NoAsync
    Disable async logging for all instances

.EXAMPLE
    .\scripts\run-game-swarm.ps1
    Launch 10 games with default settings

.EXAMPLE
    .\scripts\run-game-swarm.ps1 -Count 20 -Duration 120
    Launch 20 games that run for 2 minutes

.EXAMPLE
    .\scripts\run-game-swarm.ps1 -Count 5 -HangRate 0.4 -CrashRate 0.2
    Launch 5 games with 40% hang rate and 20% crash rate

.EXAMPLE
    .\scripts\run-game-swarm.ps1 -Count 50 -Stagger 0.5
    Launch 50 games with 0.5s between each launch
#>

param(
    [Parameter()]
    [int]$Count = 10,

    [Parameter()]
    [int]$Duration = 60,

    [Parameter()]
    [ValidateRange(0, 1)]
    [double]$HangRate = 0.15,

    [Parameter()]
    [ValidateRange(0, 1)]
    [double]$CrashRate = 0.15,

    [Parameter()]
    [double]$Stagger = 1.0,

    [Parameter()]
    [switch]$NoAsync
)

# Set error action
$ErrorActionPreference = "Stop"

# Validate rates
if (($HangRate + $CrashRate) -gt 1.0) {
    Write-Host "ERROR: HangRate + CrashRate cannot exceed 1.0" -ForegroundColor Red
    exit 1
}

# Get script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir
$ExamplesDir = Join-Path $RootDir "examples"

Write-Host ""
Write-Host "=====================================================================" -ForegroundColor Cyan
Write-Host "Game Swarm Launcher - DataCat Demo" -ForegroundColor Cyan
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
    Write-Host "ERROR: DataCat server is not running!" -ForegroundColor Red
    Write-Host "       Start it with: .\scripts\run-both.ps1" -ForegroundColor Yellow
    Write-Host ""
    exit 1
}

# Check if web UI is running
$webCheck = $null
try {
    $webCheck = Invoke-WebRequest -Uri "http://localhost:8080" -TimeoutSec 2 -ErrorAction SilentlyContinue
} catch {
    # Web UI not running
}

if (-not $webCheck) {
    Write-Host "WARNING: Web UI doesn't appear to be running!" -ForegroundColor Yellow
    Write-Host "         Start it with: .\scripts\run-both.ps1" -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "Configuration:" -ForegroundColor Green
Write-Host "  Instances: $Count"
Write-Host "  Duration: $Duration seconds"
Write-Host "  Hang rate: $($HangRate * 100)%"
Write-Host "  Crash rate: $($CrashRate * 100)%"
Write-Host "  Normal rate: $(100 - ($HangRate * 100) - ($CrashRate * 100))%"
Write-Host "  Stagger: $Stagger seconds"
Write-Host "  Async logging: $(-not $NoAsync)"
Write-Host ""

# Calculate expected outcomes
$numHang = [int]($Count * $HangRate)
$numCrash = [int]($Count * $CrashRate)
$numNormal = $Count - $numHang - $numCrash

Write-Host "Expected outcomes:" -ForegroundColor Green
Write-Host "  Normal exits: $numNormal"
Write-Host "  Hangs: $numHang"
Write-Host "  Crashes: $numCrash"
Write-Host ""

Write-Host "This will launch $Count game instances." -ForegroundColor Yellow
Write-Host "Press Ctrl+C at any time to stop all instances." -ForegroundColor Yellow
Write-Host ""

$response = Read-Host "Continue? (y/n)"
if ($response -ne 'y' -and $response -ne 'Y') {
    exit 0
}

Write-Host ""
Write-Host "Launching game swarm using Python script..." -ForegroundColor Cyan
Write-Host ""

# Build command for Python script
$cmd = @(
    "python",
    (Join-Path $ExamplesDir "run_game_swarm.py"),
    "--count", $Count,
    "--duration", $Duration,
    "--hang-rate", $HangRate,
    "--crash-rate", $CrashRate,
    "--stagger", $Stagger
)

if ($NoAsync) {
    $cmd += "--no-async"
}

# Run swarm launcher
try {
    & $cmd[0] $cmd[1..($cmd.Length-1)]
} catch {
    Write-Host ""
    Write-Host "Swarm interrupted or failed: $_" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=====================================================================" -ForegroundColor Cyan
Write-Host "Game Swarm Complete!" -ForegroundColor Cyan
Write-Host "=====================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "View all sessions in the web UI:" -ForegroundColor Green
Write-Host "  http://localhost:8080" -ForegroundColor Cyan
Write-Host ""
Write-Host "You should see:" -ForegroundColor Green
Write-Host "  - All $Count game sessions"
Write-Host "  - Crash detection for crashed games"
Write-Host "  - Hang detection for hung games"
Write-Host "  - Live metrics (FPS, memory, health, score)"
Write-Host "  - Game events (enemies, powerups, achievements)"
Write-Host ""

