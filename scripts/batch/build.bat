@echo off
REM Build all datacat binaries

echo Building datacat binaries...
cd /d "%~dp0\..\..

REM Create bin directory
if not exist bin mkdir bin

REM Build server
echo Building server...
cd cmd\datacat-server
go build -o ..\..\bin\datacat-server.exe
cd ..\..

REM Build web UI
echo Building web UI...
cd cmd\datacat-web
go build -o ..\..\bin\datacat-web.exe
cd ..\..

REM Build daemon
echo Building daemon...
cd cmd\datacat-daemon
go build -o ..\..\bin\datacat-daemon.exe
cd ..\..

REM Build Go client example
echo Building Go client example...
cd examples\go-client-example
go build -o ..\..\bin\go-client-example.exe
cd ..\..

echo All binaries built successfully in bin\ directory
