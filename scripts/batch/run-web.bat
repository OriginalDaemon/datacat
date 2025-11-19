@echo off
REM Run the datacat web UI

echo Starting datacat web UI...
echo Web UI will be available at http://localhost:8081
cd /d "%~dp0\..\..
cd cmd\datacat-web
go run main.go
