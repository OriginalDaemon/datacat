@echo off
REM Run Python integration tests

echo Running Python integration tests...
cd /d "%~dp0\..\..

REM Start the server in background
echo Starting test server...
start /b cmd /c "cd cmd\datacat-server && go run main.go config.go"
timeout /t 3 /nobreak >nul

REM Run Python tests
echo Running tests...
pytest tests\ -v

REM Kill the server (note: this is basic, may need taskkill)
echo Stopping test server...
taskkill /f /im go.exe >nul 2>&1

exit /b %ERRORLEVEL%
