@echo off
setlocal EnableDelayedExpansion

cd /d "%~dp0\.."
set "ENV_FILE=%CD%\.env"
set "COMPOSE_FILE=%CD%\docker-compose.desktop.yml"

set "WRITE_ONLY=0"
:parse_args
if "%~1"=="" goto args_done
if /i "%~1"=="--write-env-only" set "WRITE_ONLY=1"
if /i "%~1"=="--help" (
  echo Usage: %~nx0 [--write-env-only]
  echo   --write-env-only  generate .env only ^(for Portainer^)
  echo   Set OKTOPUS_API_HOST / OKTOPUS_API_PORT in the environment to skip prompts.
  exit /b 0
)
shift
goto parse_args
:args_done

where docker >nul 2>&1
if errorlevel 1 (
  echo docker not found.
  exit /b 1
)

echo Using docker-compose.desktop.yml (bridge network). Production on Linux: scripts/stack-up.sh and docker-compose.yml.
echo Note: .env is recreated each run; DB passwords are kept when Docker volumes already exist.

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0stack-up-env.ps1" -EnvFile "%ENV_FILE%"
if errorlevel 1 (
  echo Failed to generate "%ENV_FILE%".
  exit /b 1
)

if not exist "%ENV_FILE%" (
  echo Missing "%ENV_FILE%".
  exit /b 1
)

findstr /B /C:"OKTOPUS_API_PORT=" "%ENV_FILE%" >nul 2>&1
if errorlevel 1 (
  echo Incomplete "%ENV_FILE%": generation was interrupted.
  exit /b 1
)

echo Wrote "%ENV_FILE%"

if "%WRITE_ONLY%"=="1" (
  echo Env only; copy .env to Portainer or run without --write-env-only.
  exit /b 0
)

docker compose --env-file "%ENV_FILE%" -f "%COMPOSE_FILE%" up --build -d
if errorlevel 1 exit /b 1

for /f "usebackq tokens=1,* delims==" %%A in (`findstr /B "OKTOPUS_ADMIN_USERNAME=" "%ENV_FILE%"`) do set "ADMIN_USER=%%B"
for /f "usebackq tokens=1,* delims==" %%A in (`findstr /B "OKTOPUS_ADMIN_PASSWORD=" "%ENV_FILE%"`) do set "ADMIN_PASS=%%B"
for /f "usebackq tokens=1,* delims==" %%A in (`findstr /B "OKTOPUS_API_HOST=" "%ENV_FILE%"`) do set "API_HOST=%%B"
for /f "usebackq tokens=1,* delims==" %%A in (`findstr /B "OKTOPUS_API_PORT=" "%ENV_FILE%"`) do set "API_PORT=%%B"

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0stack-up-banner.ps1"
echo ========== OKTOPUS ==========
echo UI:       http://!API_HOST!:!API_PORT!/
echo Login:    !ADMIN_USER!
echo Password: !ADMIN_PASS!
echo Env file: "%ENV_FILE%"
echo ==============================
echo.

endlocal
