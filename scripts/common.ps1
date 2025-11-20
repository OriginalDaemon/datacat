# Common functions for PowerShell scripts

# Get the repository root directory
function Get-RepoRoot {
    return Split-Path -Parent $PSScriptRoot
}

# Get the venv scripts directory based on platform
function Get-VenvScriptsDir {
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        return "Scripts"
    } else {
        return "bin"
    }
}

# Get Python executable from venv, or system Python if venv doesn't exist
function Get-PythonExe {
    $repoRoot = Get-RepoRoot
    $scriptsDir = Get-VenvScriptsDir

    # Add .exe extension on Windows
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        $venvPython = Join-Path $repoRoot ".venv/$scriptsDir/python.exe"
    } else {
        $venvPython = Join-Path $repoRoot ".venv/$scriptsDir/python"
    }

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
    $scriptsDir = Get-VenvScriptsDir

    # Add .exe extension on Windows
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        $venvPip = Join-Path $repoRoot ".venv/$scriptsDir/pip.exe"
    } else {
        $venvPip = Join-Path $repoRoot ".venv/$scriptsDir/pip"
    }

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
    $scriptsDir = Get-VenvScriptsDir

    # Add .exe extension on Windows
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        $venvPytest = Join-Path $repoRoot ".venv/$scriptsDir/pytest.exe"
    } else {
        $venvPytest = Join-Path $repoRoot ".venv/$scriptsDir/pytest"
    }

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
    $scriptsDir = Get-VenvScriptsDir

    # Add .exe extension on Windows
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        $venvBlack = Join-Path $repoRoot ".venv/$scriptsDir/black.exe"
    } else {
        $venvBlack = Join-Path $repoRoot ".venv/$scriptsDir/black"
    }

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
    $scriptsDir = Get-VenvScriptsDir

    # Add .exe extension on Windows
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        $venvMypy = Join-Path $repoRoot ".venv/$scriptsDir/mypy.exe"
    } else {
        $venvMypy = Join-Path $repoRoot ".venv/$scriptsDir/mypy"
    }

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
