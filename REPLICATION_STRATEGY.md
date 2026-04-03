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