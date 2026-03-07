@echo off
echo Starting 4-node Raft Backend Cluster...

cd backend

:: Force rebuild to apply loopback fixes
if exist server.exe del server.exe

:: Ensure server.exe is built
if not exist server.exe (
    echo [STATUS] server.exe not found! Attempting to build...
    go mod tidy
    go build -o server.exe .
    if errorlevel 1 (
        echo [CRITICAL ERROR] Failed to build Go backend.
        pause
        exit /b 1
    )
    echo [SUCCESS] Backend built successfully.
)

:: Node 1 (Bootstrap)
start "Node 1 - Leader" cmd /k "server.exe -node-id node1 -raft-dir ./data/node1 -http-addr :8000 -raft-addr :9000 -bootstrap true"

:: Wait for Node 1 to stabilize
timeout /t 3

:: Node 2
start "Node 2" cmd /k "server.exe -node-id node2 -raft-dir ./data/node2 -http-addr :8001 -raft-addr :9001 -join :8000"

:: Node 3
start "Node 3" cmd /k "server.exe -node-id node3 -raft-dir ./data/node3 -http-addr :8002 -raft-addr :9002 -join :8000"

:: Node 4
start "Node 4" cmd /k "server.exe -node-id node4 -raft-dir ./data/node4 -http-addr :8003 -raft-addr :9003 -join :8000"

echo Backend cluster starting in separate windows.
pause
