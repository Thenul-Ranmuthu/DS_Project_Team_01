@echo off
set PROJECT_ROOT=%~dp0
echo Starting the Distributed File Storage System...

:: Run the Backend (Go)
echo [1/2] Launching Backend (Node)...
start "DS Backend" cmd /k "cd /d %PROJECT_ROOT%node && go run main.go"

:: Wait for 15 seconds
echo Waiting 15 seconds for backend to initialize...
timeout /t 15 /nobreak

:: Run the Frontend (Next.js)
echo [2/2] Launching Frontend...
start "DS Frontend" cmd /k "cd /d %PROJECT_ROOT%frontend && npm run dev"

echo.
echo Both components should be starting in separate terminal windows.
pause
