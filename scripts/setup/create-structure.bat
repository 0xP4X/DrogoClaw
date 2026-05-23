@echo off
REM DrogonClaw Setup Script
cd /d "C:\Users\0day\Desktop\drogon"

REM Create directory structure
for %%D in (
  src
  src\gateway
  src\agent
  src\agent\strategies
  src\skills
  src\skills\recon
  src\skills\exploitation
  src\channels
  src\channels\cli
  src\channels\telegram
  src\storage
  src\reporting
  src\cli
  src\cli\commands
  src\types
  config
  tests
  tests\unit
  tests\integration
  tests\fixtures
  docs
) do (
  if not exist "%%D" mkdir "%%D"
  echo Created: %%D
)

echo.
echo ✓ Directory structure created successfully!
echo.
echo Now run: npm install
