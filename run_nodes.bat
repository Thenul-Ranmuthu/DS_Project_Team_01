@echo off
setlocal

:: Start Backend Nodes (node-5051 to node-5054)

echo Starting Node 1 (port 5051)...
start powershell -NoExit -Command "cd node1; $env:NODE_ID='node-5051'; $env:PORT='5051'; CompileDaemon -command='./DS_node'"
timeout /t 15 /nobreak > nul

echo Starting Node 2 (port 5052)...
start powershell -NoExit -Command "cd node2; $env:NODE_ID='node-5052'; $env:PORT='5052'; CompileDaemon -command='./DS_node'"
timeout /t 15 /nobreak > nul

echo Starting Node 3 (port 5053)...
start powershell -NoExit -Command "cd node3; $env:NODE_ID='node-5053'; $env:PORT='5053'; CompileDaemon -command='./DS_node'"
timeout /t 15 /nobreak > nul

echo Starting Node 4 (port 5054)...
start powershell -NoExit -Command "cd node4; $env:NODE_ID='node-5054'; $env:PORT='5054'; CompileDaemon -command='./DS_node'"
timeout /t 30 /nobreak > nul

:: Start Frontend

echo Starting Frontend...
start powershell -NoExit -Command "cd ds_frontend; npm run dev"

echo.
echo All backend nodes and the frontend are starting up in separate terminals!
pause
