# Data Replication and Consistency Strategy

## 1. Replication Model
The system uses a Primary-Backup (Leader-Follower) model coordinated by ZooKeeper leader election.

- One node is elected as Leader.
- Other nodes run as Followers.
- Write operations are leader-controlled and then replicated to peers.

## 2. Current Write Flows

### File upload replication
- Client request enters through `POST /upload/:email`.
- The receiving node stores the file with a generated unique name.
- Replication fanout happens through `POST /internal/replicate` to all peers listed in `PEERS`.
- Replicas are stored with the same generated filename so delete propagation is deterministic.

### File delete replication
- Client request enters through `DELETE /files/:id`.
- The node resolves local file path, deletes locally, then broadcasts to peers.
- Replication fanout uses `DELETE /internal/delete/:filename`.

### User create replication (leader-routed)
- Client request enters through `POST /createUser`.
- If request hits a follower, the request is forwarded to the current leader.
- Leader performs create, then replicates to peers through `POST /internal/users`.
- Duplicate prevention is enforced by email check.
- If user already exists, API returns `409` with `{"error":"User Already Exists"}`.

## 3. Internal Replication Endpoints
The cluster currently uses these internal APIs:

- `POST /internal/replicate`
- `DELETE /internal/delete/:filename`
- `POST /internal/users`

## 4. Single-Source Multi-Instance Runtime
All nodes run from one shared codebase (`node/`) with different env presets.

- `.env.leader` (port `5000`)
- `.env.node1` (port `5050`)
- `.env.node2` (port `5051`)

Each instance uses:

- unique `NODE_ID`
- unique `PORT`
- `PEERS` containing the other node base URLs
- unique `UPLOAD_DIR` so local replica folders are visually separated

Example upload directories:

- `uploads/leader`
- `uploads/node1`
- `uploads/node2`

## 5. Consistency and Ordering
- Lamport clock headers are used to preserve logical event ordering across nodes.
- NTP sync is used for stable wall-clock timestamps and operational logs.
- Replication fanout is asynchronous, so behavior is eventual convergence after writes/deletes complete.

## 6. Operational Constraints
- Incorrect or missing `PEERS` values cause partial or failed replication.
- If leader is down, ZooKeeper elects a new leader and follower forwarding targets the new leader.