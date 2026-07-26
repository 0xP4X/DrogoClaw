@echo off
setlocal enabledelayedexpansion
cd /d C:\Users\0day\Desktop\drogon

REM Execute the Node.js setup script
echo Running DrogonClaw setup...
node setup-project.js

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Setup completed successfully!
    echo.
) else (
    echo.
    echo Setup failed with error code %ERRORLEVEL%
    echo.
)

endlocal
