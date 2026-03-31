@echo off
setlocal

:: Clean up old builds
if exist "node\server.exe" del "node\server.exe"

echo ====================================================
echo [STEP 1] Building Backend (Go)
echo ====================================================
cd node
go build -o server.exe main.go
if %errorlevel% neq 0 (
    echo [ERROR] Backend build failed!
    pause
    exit /b 1
)
cd ..

echo.
echo ====================================================
echo [STEP 2] Building Frontend (Vite/React)
echo ====================================================
cd frontend
call npm install
call npm run build
if %errorlevel% neq 0 (
    echo [ERROR] Frontend build failed!
    pause
    exit /b 1
)
cd ..

echo.
echo ====================================================
echo [STEP 3] Starting Backend node in new window
echo ====================================================
start "Distributed Storage - Backend" cmd /k "cd node && server.exe"

echo.
echo ====================================================
echo [STEP 4] Waiting 20 seconds for initialization
echo ====================================================
timeout /t 20 /nobreak

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

echo [1/2] Shutting down Frontend...
taskkill /F /IM node.exe /T > nul 2>&1
echo [OK] Frontend killed.

echo.
echo [Wait 5 seconds before killing backend...]
timeout /t 5 /nobreak

echo.
echo [2/2] Shutting down Backend...
taskkill /F /IM server.exe /T > nul 2>&1
echo [OK] Backend killed.

echo.
echo ====================================================
echo [SUCCESS] Clean Shutdown Complete.
echo ====================================================
pause

