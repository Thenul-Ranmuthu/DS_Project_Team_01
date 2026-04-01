@echo off
setlocal

:: Clean up old builds
if exist "node\server.exe" del "node\server.exe"

echo ====================================================
echo [STEP 1] Building Backend and Orchestrator
echo ====================================================
cd node
go build -o server.exe main.go
go build -o orchestrator.exe orchestrator\main.go
if %errorlevel% neq 0 (
    echo [ERROR] Build failed!
    pause
    exit /b 1
)
cd ..

echo.
echo ====================================================
echo [STEP 2] Building Frontend (Next.js)
echo ====================================================
cd frontend
call npm run build
if %errorlevel% neq 0 (
    echo [ERROR] Frontend build failed!
    pause
    exit /b 1
)
cd ..

echo.
echo ====================================================
echo [STEP 3] Starting Orchestrator (Manages 7 Nodes)
echo ====================================================
start "Distributed Storage - Orchestrator" cmd /k "cd node && orchestrator.exe"

echo.
echo ====================================================
echo [STEP 4] Waiting 30 seconds for 7 nodes to initialize
echo ====================================================
timeout /t 30 /nobreak

echo.
echo ====================================================
echo [STEP 5] Starting Frontend in new window
echo ====================================================
start "Distributed Storage - Frontend" cmd /k "cd frontend && npm run dev"

echo.
echo ====================================================
echo [SYSTEM RUNNING]
echo Press ANY KEY in this terminal to initiate shutdown.
echo ====================================================
pause > nul

echo.
echo ====================================================
echo [SHUTDOWN SEQUENCE INITIATED]
echo ====================================================

echo [1/3] Shutting down Frontend...
taskkill /F /IM node.exe /T > nul 2>&1
echo [OK] Frontend killed.

echo.
echo [2/3] Shutting down Backend Nodes...
taskkill /F /IM server.exe /T > nul 2>&1
echo [OK] Backend nodes killed.

echo.
echo [3/3] Shutting down Orchestrator...
taskkill /F /IM orchestrator.exe /T > nul 2>&1
echo [OK] Orchestrator killed.

echo.
echo ====================================================
echo [SUCCESS] Clean Shutdown Complete.
echo ====================================================
pause

