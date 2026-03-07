# DS_Project_Team_01
DS Project Repository (Team-01) - Distrbuted File Storage Web App

A distributed file storage web application built with a Next.js frontend and a Go backend utilizing HashiCorp Raft for distributed consensus and BoltDB for underlying storage. It's designed to run on a 4-node cluster.

## Architecture
- **Frontend**: Next.js (React), Tailwind CSS
- **Backend API**: Go `net/http`
- **Consensus**: HashiCorp Raft
- **Storage**: BoltDB

## Prerequisites
- Node.js (v18+)
- npm
- Go (v1.20+)

## Running the Application

### 1. Compile and Run the Backend (4 Nodes)
Open a terminal and navigate to the `backend` directory.

To build the Go backend executable:
```bash
cd backend
go build -o server main.go
```

You'll need to run four separate instances to form the Raft cluster. Open four separate terminals:

**Node 1 (Leader/Bootstrap):**
```bash
./server -node-id node1 -raft-dir ./data/node1 -http-addr :8000 -raft-addr :9000 -bootstrap true
```

**Node 2:**
```bash
./server -node-id node2 -raft-dir ./data/node2 -http-addr :8001 -raft-addr :9001 -join :8000
```

**Node 3:**
```bash
./server -node-id node3 -raft-dir ./data/node3 -http-addr :8002 -raft-addr :9002 -join :8000
```

**Node 4:**
```bash
./server -node-id node4 -raft-dir ./data/node4 -http-addr :8003 -raft-addr :9003 -join :8000
```

### 2. Run the Frontend
Open a new terminal and navigate to the `frontend` directory.
Install dependencies and start the development server:

```bash
cd frontend
npm install
npm run dev
```

The web application should now be accessible at `http://localhost:3000`.
