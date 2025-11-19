@echo off
REM Run linters and code quality checks

echo Running code quality checks...
cd /d "%~dp0\..\..

REM Run Black
echo Checking Python formatting with Black...
black --check python\ examples\ tests\
set BLACK_EXIT=%ERRORLEVEL%

REM Run mypy
echo Running mypy type checking...
mypy python\ --ignore-missing-imports
set MYPY_EXIT=%ERRORLEVEL%

if %BLACK_EXIT%==0 if %MYPY_EXIT%==0 (
    echo All checks passed
    exit /b 0
) else (
    echo Some checks failed
    exit /b 1
)
