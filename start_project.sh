#!/bin/bash

# Clean up old builds
rm -f node/server node/orchestrator

echo "===================================================="
echo "[STEP 0] Starting Replicated MySQL Cluster (Docker Clean Start)"
echo "===================================================="
docker-compose down -v && docker-compose up -d
if [ $? -ne 0 ]; then
    echo "[ERROR] Docker Compose failed! Make sure Docker Desktop is RUNNING."
    exit 1
fi
echo "Waiting 45 seconds for MySQL Galera Cluster to initialize and sync..."
sleep 45

echo "===================================================="
echo "[STEP 1] Building Backend and Orchestrator"
echo "===================================================="
cd node
go build -o server main.go
go build -o orchestrator orchestrator/main.go
if [ $? -ne 0 ]; then
    echo "[ERROR] Build failed!"
    exit 1
fi
cd ..

echo ""
echo "===================================================="
echo "[STEP 2] Building Frontend (Next.js)"
echo "===================================================="
cd frontend
npm run build
if [ $? -ne 0 ]; then
    echo "[ERROR] Frontend build failed!"
    exit 1
fi
cd ..

echo ""
echo "===================================================="
echo "[STEP 2.5] Starting MinIO Storage Service"
echo "===================================================="
cd node
if [ ! -f "minio" ]; then
    echo "Downloading MinIO Server..."
    curl -O https://dl.min.io/server/minio/release/darwin-amd64/minio
    chmod +x minio
fi
# Start MinIO in background
MINIO_ROOT_USER=minioadmin MINIO_ROOT_PASSWORD=minioadmin ./minio server ./minio_data &
MINIO_PID=$!
cd ..

echo ""
echo "===================================================="
echo "[STEP 3] Starting Orchestrator (Manages 7 Nodes)"
echo "===================================================="
cd node
./orchestrator &
ORCHESTRATOR_PID=$!
cd ..

echo ""
echo "===================================================="
echo "[STEP 4] Waiting 20 seconds for 7 nodes to initialize"
echo "===================================================="
sleep 20

echo ""
echo "===================================================="
echo "[STEP 5] Starting Frontend"
echo "===================================================="
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

echo ""
echo "===================================================="
echo "[SYSTEM RUNNING]"
echo "Press Ctrl+C to initiate shutdown."
echo "===================================================="

# Shutdown trap
cleanup() {
    echo ""
    echo "===================================================="
    echo "[SHUTDOWN SEQUENCE INITIATED]"
    echo "===================================================="
    
    echo "[1/4] killing Background processes..."
    kill $FRONTEND_PID $ORCHESTRATOR_PID $MINIO_PID 2>/dev/null
    pkill server 2>/dev/null
    
    echo "[2/4] Shutting down MySQL Cluster..."
    docker-compose down
    
    echo "[SUCCESS] Clean Shutdown Complete."
    exit 0
}

trap cleanup SIGINT

# Keep script running
while true; do sleep 1; done
