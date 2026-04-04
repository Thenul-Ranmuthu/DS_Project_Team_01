Data Replication and Consistency Strategy
1. Replication Strategy: Primary-Backup (Leader-Follower)
For this distributed file storage system, we have implemented a Primary-Backup replication strategy.

Mechanism: A single "Leader" node (elected via Zookeeper) coordinates all write operations (file uploads and deletions).

Data Flow: When the Leader receives a file, it first persists the data locally and then synchronously or asynchronously propagates the update to all "Follower" nodes.

Justification: This strategy is highly effective for file systems where maintaining a single "source of truth" is critical to prevent data divergence across servers.

2. Consistency Model: Strong Consistency
We have chosen a Strong Consistency model to ensure that any client reading from any node in the system always receives the most recent version of a file.

Implementation: A write operation is only considered "successful" once the Leader has confirmed that the file is safely stored on a majority of nodes or the designated backups.

User Impact: This eliminates the "stale data" problem, ensuring that once a user uploads a file, it is immediately accessible across the entire distributed network.

3. Conflict Resolution: Lamport Logical Clocks
To handle concurrent read/write operations and establish a strict ordering of events, the system utilizes Lamport Clocks.

Mechanism: Each node maintains a local counter that increments with every local event. When messages are exchanged between nodes, the clocks are synchronized to ensure that the "Happened-Before" relationship is preserved.

Conflict Handling: In the event of near-simultaneous updates to the same file, the system uses the Lamport timestamp to determine the definitive order of operations, ensuring all replicas converge to the same state.

4. Time Synchronization
While Lamport Clocks handle logical ordering, the system also integrates NTP (Network Time Protocol) to maintain roughly synchronized physical clocks across servers. This provides human-readable timestamps for file metadata and assists in debugging and logging.

5. Implementation Note (Current Behavior)
The replication workflow is implemented with internal node-to-node APIs and environment-based peer discovery.

Upload replication:
- External client upload enters through /upload/:email on the receiving node.
- The node stores the file locally with a generated unique filename.
- The replication layer forwards that stored filename and file content to each peer using POST /internal/replicate.
- Followers save the replica under the same stored filename for deterministic delete propagation.

Delete replication:
- External delete enters through DELETE /files/:id.
- The node resolves the stored file path, deletes locally, then broadcasts DELETE /internal/delete/:filename to peers.
- Every node exposes internal handlers for both:
	- POST /internal/replicate
	- DELETE /internal/delete/:filename

Configuration requirements:
- Each node must define PEERS in its .env file as a comma-separated list of other node base URLs.
- Example: PEERS=http://localhost:5000,http://localhost:5050,http://localhost:5051 (excluding self for each node).
- Missing or incorrect PEERS values result in partial or no replication.

Consistency note:
- Replication calls are asynchronous, so the implementation provides eventual convergence across replicas after write/delete operations complete.