package election

import (
	"fmt"
	"log"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

const (
	zkElectionPath = "/ds_project/election"
	zkTimeout      = 5 * time.Second
)

type ElectionManager struct {
	mu sync.RWMutex

	conn      *zk.Conn
	nodeID    string // e.g. "node-5050"
	znodePath string // full path of our ephemeral znode
	isLeader  bool
	leaderID  string // nodeID of whoever is currently leader

	onBecomeLeader  func()                   // callback: called when this node wins
	onLeaderChanged func(newLeaderID string) // callback: called when leader changes
}

func NewElectionManager(zkServers []string, nodeID string) (*ElectionManager, error) {
	conn, _, err := zk.Connect(zkServers, zkTimeout)
	if err != nil {
		return nil, fmt.Errorf("zk connect failed: %w", err)
	}

	em := &ElectionManager{
		conn:   conn,
		nodeID: nodeID,
	}

	// Ensure persistent parent path exists
	if err := em.ensurePath(zkElectionPath); err != nil {
		return nil, err
	}

	return em, nil
}

// Start registers this node and begins the election watch loop.
func (em *ElectionManager) Start() error {
	znodePath, err := em.conn.CreateProtectedEphemeralSequential(
		zkElectionPath+"/node_",
		[]byte(em.nodeID),
		zk.WorldACL(zk.PermAll),
	)
	if err != nil {
		return fmt.Errorf("failed to create election znode: %w", err)
	}
	em.znodePath = znodePath
	log.Printf("[%s] Registered znode: %s", em.nodeID, znodePath)

	go em.watchLeader()

	return em.runElection()
}

// runElection checks if we're the leader; if not, watches our predecessor.
func (em *ElectionManager) runElection() error {
	for {
		children, _, err := em.conn.Children(zkElectionPath)
		if err != nil {
			return fmt.Errorf("failed to list election children: %w", err)
		}
		// sort.Strings(children) // sequential znodes sort lexicographically
		sort.Slice(children, func(i, j int) bool {
			return getSeqNumber(children[i]) < getSeqNumber(children[j])
		})
		// Our znode's base name (strip the parent path prefix)
		myNode := path.Base(em.znodePath)

		if children[0] == myNode {
			// We have the lowest sequence number — we are the leader
			em.mu.Lock()
			em.isLeader = true
			em.leaderID = em.nodeID
			em.mu.Unlock()

			log.Printf("[%s] Became LEADER", em.nodeID)
			if em.onBecomeLeader != nil {
				em.onBecomeLeader()
			}
			return nil
		}

		// Find our predecessor to watch
		myIdx := indexOf(children, myNode)
		if myIdx < 0 {
			return fmt.Errorf("our znode %s not found in children", myNode)
		}
		predecessor := zkElectionPath + "/" + children[myIdx-1]

		// Update who the current leader is
		leaderData, _, err := em.conn.Get(zkElectionPath + "/" + children[0])
		if err == nil {
			em.mu.Lock()
			em.isLeader = false
			em.leaderID = string(leaderData)
			em.mu.Unlock()
		}

		log.Printf("[%s] Follower. Watching predecessor: %s", em.nodeID, predecessor)

		// Block until predecessor disappears
		exists, _, watchCh, err := em.conn.ExistsW(predecessor)
		if err != nil {
			return fmt.Errorf("watch failed: %w", err)
		}
		if !exists {
			// Predecessor already gone — re-run election immediately
			continue
		}

		// Wait for predecessor deletion event
		event := <-watchCh
		log.Printf("[%s] Watch fired: %v — re-running election", em.nodeID, event.Type)
		// Loop back and re-check
	}
}

// Stop gracefully removes our znode (triggers election in others).
func (em *ElectionManager) Stop() {
	if em.znodePath != "" {
		em.conn.Delete(em.znodePath, -1)
	}
	em.conn.Close()
}

func (em *ElectionManager) IsLeader() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.isLeader
}

func (em *ElectionManager) LeaderID() string {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.leaderID
}

// ensurePath creates all nodes in the path if they don't exist (persistent).
func (em *ElectionManager) ensurePath(p string) error {
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	current := ""
	for _, part := range parts {
		current += "/" + part
		exists, _, err := em.conn.Exists(current)
		if err != nil {
			return err
		}
		if !exists {
			_, err = em.conn.Create(current, []byte{}, 0, zk.WorldACL(zk.PermAll))
			if err != nil && err != zk.ErrNodeExists {
				return err
			}
		}
	}
	return nil
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func (em *ElectionManager) SetOnBecomeLeader(fn func()) {
	em.onBecomeLeader = fn
}

func getSeqNumber(name string) string {
	return name[len(name)-10:]
}

// watchLeader continuously watches the leader znode and updates leaderID whenever it changes.
func (em *ElectionManager) watchLeader() {
	for {
		children, _, err := em.conn.Children(zkElectionPath)
		if err != nil || len(children) == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		sort.Slice(children, func(i, j int) bool {
			return getSeqNumber(children[i]) < getSeqNumber(children[j])
		})

		leaderZnode := zkElectionPath + "/" + children[0]

		// Get leader data AND set a watch on the leader znode
		data, _, watchCh, err := em.conn.GetW(leaderZnode)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		em.mu.Lock()
		em.leaderID = string(data)
		em.mu.Unlock()

		log.Printf("[%s] Leader is: %s", em.nodeID, string(data))

		// Block until the leader znode changes or disappears
		<-watchCh

		log.Printf("[%s] Leader znode changed — re-checking leader", em.nodeID)
		// Loop back and re-read who the new leader is
	}
}
