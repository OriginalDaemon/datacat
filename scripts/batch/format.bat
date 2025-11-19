@echo off
REM Auto-format Python code with Black

echo Formatting Python code with Black...
cd /d "%~dp0\..\..
black python\ examples\ tests\
echo Formatting complete
