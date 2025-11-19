@echo off
REM Run the datacat server

echo Starting datacat server...
cd /d "%~dp0\..\..
cd cmd\datacat-server
go run main.go config.go
