@echo off
REM Clean build artifacts and temporary files

echo Cleaning build artifacts...
cd /d "%~dp0\..\..

REM Remove binaries
if exist bin rd /s /q bin
echo Removed bin\

REM Remove Python cache
for /d /r . %%d in (__pycache__) do @if exist "%%d" rd /s /q "%%d"
for /d /r . %%d in (*.egg-info) do @if exist "%%d" rd /s /q "%%d"
del /s /q *.pyc >nul 2>&1
echo Removed Python cache files

REM Remove test coverage
if exist .coverage del .coverage
if exist coverage.out del coverage.out
echo Removed coverage files

REM Remove BadgerDB data
if exist datacat_data rd /s /q datacat_data
echo Removed datacat_data\

echo Cleanup complete
