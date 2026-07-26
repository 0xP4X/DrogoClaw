@echo off
REM Create complete DrogonClaw directory structure

cd /d "%~dp0"

echo Creating directory structure...

REM Source directories
mkdir src\types 2>nul
mkdir src\config 2>nul
mkdir src\gateway 2>nul
mkdir src\gateway\routes 2>nul
mkdir src\agent 2>nul
mkdir src\agent\strategies 2>nul
mkdir src\skills 2>nul
mkdir src\skills\recon 2>nul
mkdir src\skills\enumeration 2>nul
mkdir src\skills\exploitation 2>nul
mkdir src\channels 2>nul
mkdir src\channels\cli 2>nul
mkdir src\channels\cli\commands 2>nul
mkdir src\channels\telegram 2>nul
mkdir src\storage 2>nul
mkdir src\utils 2>nul
mkdir src\models 2>nul
mkdir src\reporting 2>nul

REM Test directories
mkdir tests\unit 2>nul
mkdir tests\integration 2>nul
mkdir tests\fixtures 2>nul

REM Docs and config
mkdir docs 2>nul
mkdir config 2>nul
mkdir data 2>nul

echo Directory structure created successfully!
echo.
echo Next steps:
echo 1. npm install
echo 2. npm run build
echo 3. npm run lint
pause
