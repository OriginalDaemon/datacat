# Common functions for PowerShell scripts

# Get the repository root directory
function Get-RepoRoot {
    return Split-Path -Parent $PSScriptRoot
}

# Get Python executable from venv, or system Python if venv doesn't exist
function Get-PythonExe {
    $repoRoot = Get-RepoRoot
    $venvPython = Join-Path $repoRoot ".venv\Scripts\python.exe"
    
    if (Test-Path $venvPython) {
        return $venvPython
    } else {
        Write-Host "Warning: Virtual environment not found at .venv" -ForegroundColor Yellow
        Write-Host "         Run '.\scripts\setup.ps1' to create it" -ForegroundColor Yellow
        Write-Host "         Falling back to system Python" -ForegroundColor Yellow
        Write-Host ""
        return "python"
    }
}

# Get pip executable from venv, or system pip if venv doesn't exist
function Get-PipExe {
    $repoRoot = Get-RepoRoot
    $venvPip = Join-Path $repoRoot ".venv\Scripts\pip.exe"
    
    if (Test-Path $venvPip) {
        return $venvPip
    } else {
        Write-Host "Warning: Virtual environment not found at .venv" -ForegroundColor Yellow
        Write-Host "         Run '.\scripts\setup.ps1' to create it" -ForegroundColor Yellow
        Write-Host "         Falling back to system pip" -ForegroundColor Yellow
        Write-Host ""
        return "pip"
    }
}

# Get pytest executable from venv, or system pytest if venv doesn't exist
function Get-PytestExe {
    $repoRoot = Get-RepoRoot
    $venvPytest = Join-Path $repoRoot ".venv\Scripts\pytest.exe"
    
    if (Test-Path $venvPytest) {
        return $venvPytest
    } else {
        Write-Host "Warning: Virtual environment not found at .venv" -ForegroundColor Yellow
        Write-Host "         Run '.\scripts\setup.ps1' to create it" -ForegroundColor Yellow
        Write-Host "         Falling back to system pytest" -ForegroundColor Yellow
        Write-Host ""
        return "pytest"
    }
}

# Get black executable from venv, or system black if venv doesn't exist
function Get-BlackExe {
    $repoRoot = Get-RepoRoot
    $venvBlack = Join-Path $repoRoot ".venv\Scripts\black.exe"
    
    if (Test-Path $venvBlack) {
        return $venvBlack
    } else {
        Write-Host "Warning: Virtual environment not found at .venv" -ForegroundColor Yellow
        Write-Host "         Run '.\scripts\setup.ps1' to create it" -ForegroundColor Yellow
        Write-Host "         Falling back to system black" -ForegroundColor Yellow
        Write-Host ""
        return "black"
    }
}

# Get mypy executable from venv, or system mypy if venv doesn't exist
function Get-MypyExe {
    $repoRoot = Get-RepoRoot
    $venvMypy = Join-Path $repoRoot ".venv\Scripts\mypy.exe"
    
    if (Test-Path $venvMypy) {
        return $venvMypy
    } else {
        Write-Host "Warning: Virtual environment not found at .venv" -ForegroundColor Yellow
        Write-Host "         Run '.\scripts\setup.ps1' to create it" -ForegroundColor Yellow
        Write-Host "         Falling back to system mypy" -ForegroundColor Yellow
        Write-Host ""
        return "mypy"
    }
}
