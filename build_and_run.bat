@echo off
set PROJECT_ROOT=%~dp0
echo.
echo ============================================================
echo   Distributed File Storage System - Build ^& Run 
echo ============================================================
echo.

:: 1. Build Backend
echo [1/4] Building Go Backend (node/)...
cd /d %PROJECT_ROOT%node
go mod tidy
go build -o server.exe main.go
if %errorlevel% neq 0 (
    echo [ERROR] Backend build failed!
    pause
    exit /b %errorlevel%
)
echo [OK] Backend built successfully.
echo.

:: 2. Install Frontend dependencies
echo [2/4] Installing Frontend dependencies (frontend/)...
cd /d %PROJECT_ROOT%frontend
call npm install
if %errorlevel% neq 0 (
    echo [ERROR] npm install failed!
    pause
    exit /b %errorlevel%
)
echo [OK] Frontend dependencies installed.
echo.

:: 3. Start Backend in new window
echo [3/4] Launching Backend...
start "DS Backend" cmd /k "cd /d %PROJECT_ROOT%node && server.exe"

:: 4. Delay for 15 seconds
echo Waiting 15 seconds for the backend node to initialize...
timeout /t 15 /nobreak
echo.

:: 5. Start Frontend dev server in new window
echo [4/4] Launching Frontend...
start "DS Frontend" cmd /k "cd /d %PROJECT_ROOT%frontend && npm run dev"

echo.
echo ============================================================
echo   Both services are starting in separate windows.
echo ============================================================
pause
