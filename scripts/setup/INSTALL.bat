@echo off
cd /d C:\Users\0day\Desktop\drogon
echo.
echo ========================================
echo DrogonClaw Project Setup
echo ========================================
echo.
echo Step 1: Creating directory structure...
node setup-project.js
if errorlevel 1 (
  echo ERROR: Setup script failed
  pause
  exit /b 1
)
echo.
echo Step 2: Installing npm dependencies...
call npm install
if errorlevel 1 (
  echo ERROR: npm install failed
  pause
  exit /b 1
)
echo.
echo Step 3: Building TypeScript...
call npm run build
if errorlevel 1 (
  echo ERROR: npm run build failed
  pause
  exit /b 1
)
echo.
echo Step 4: Running linter...
call npm run lint
if errorlevel 1 (
  echo ERROR: npm run lint failed
  pause
  exit /b 1
)
echo.
echo ========================================
echo Setup Complete!
echo ========================================
echo.
echo Next steps:
echo   - Copy .env.example to .env
echo   - Update HEXSTRIKE_MCP_PATH in .env
echo   - Run: npm run dev
echo.
pause
