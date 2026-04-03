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
	zkElectionPath          = "/ds_project/election"
	zkTimeout               = 10 * time.Second
	leaderWatchdogInterval = 15 * time.Second
)

type ElectionManager struct {
	mu sync.RWMutex

	conn          *zk.Conn
	zkServers     []string
	sessionEvents <-chan zk.Event
	nodeID        string // e.g. "node_1"
	znodePath     string // full path of our ephemeral znode
	isLeader      bool
	leaderID      string // nodeID of whoever is currently leader

	recoveryCh chan struct{} // ZK session loss / forced re-register (buffered, coalesced)

	onBecomeLeader  func()
	onLeaderChanged func(newLeaderID string)
	onRecovery      func() // metrics: ZK-driven recovery cycle started
	events          []string
}

// NewElectionManager connects to ZooKeeper and ensures the election parent path exists.
func NewElectionManager(zkServers []string, nodeID string) (*ElectionManager, error) {
	conn, sessionEvents, err := zk.Connect(zkServers, zkTimeout)
	if err != nil {
		return nil, fmt.Errorf("zk connect failed: %w", err)
	}

	em := &ElectionManager{
		conn:          conn,
		zkServers:     zkServers,
		sessionEvents: sessionEvents,
		nodeID:        nodeID,
		recoveryCh:    make(chan struct{}, 1),
	}

	if err := em.ensurePath(zkElectionPath); err != nil {
		conn.Close()
		return nil, err
	}

	return em, nil
}

// ZKState exposes the underlying session state for health and watchdog probes.
func (em *ElectionManager) ZKState() zk.State {
	return em.conn.State()
}

// Start registers this node (ephemeral znode) and runs the election loop until process exit.
func (em *ElectionManager) Start() error {
	go em.watchLeader()
	go em.watchSessionForRecovery()
	return em.runElection()
}

// watchSessionForRecovery listens for session expiry / disconnect (fault tolerance watchdog).
func (em *ElectionManager) watchSessionForRecovery() {
	for ev := range em.sessionEvents {
		switch ev.State {
		case zk.StateDisconnected:
			em.LogEvent(fmt.Sprintf("[Watchdog] ZK disconnected (server=%s)", ev.Server))
		case zk.StateExpired:
			em.LogEvent("[Watchdog] ZK session EXPIRED — ephemeral znode lost; re-registering (node recovery)")
			em.signalRecovery()
		case zk.StateAuthFailed:
			em.LogEvent("[Watchdog] ZK auth failed — check ZK_SERVERS / ACLs")
		}
	}
}

func (em *ElectionManager) signalRecovery() {
	select {
	case em.recoveryCh <- struct{}{}:
	default:
	}
}

func (em *ElectionManager) handleRecoverySignal() {
	em.invokeRecoveryCallback()
	em.demoteAndClearRegistration()
}

func (em *ElectionManager) invokeRecoveryCallback() {
	em.mu.RLock()
	fn := em.onRecovery
	em.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

func (em *ElectionManager) demoteAndClearRegistration() {
	em.mu.Lock()
	em.isLeader = false
	zp := em.znodePath
	em.znodePath = ""
	em.mu.Unlock()
	if zp != "" {
		if err := em.conn.Delete(zp, -1); err != nil && err != zk.ErrNoNode {
			log.Printf("[%s] Recovery: delete old znode %s: %v", em.nodeID, zp, err)
		}
	}
}

func (em *ElectionManager) ensureParticipantZnode() error {
	em.mu.Lock()
	defer em.mu.Unlock()
	if em.znodePath != "" {
		return nil
	}
	znodePath, err := em.conn.CreateProtectedEphemeralSequential(
		zkElectionPath+"/node_",
		[]byte(em.nodeID),
		zk.WorldACL(zk.PermAll),
	)
	if err != nil {
		return fmt.Errorf("failed to create election znode: %w", err)
	}
	em.znodePath = znodePath
	em.LogEvent(fmt.Sprintf("Registered znode: %s", znodePath))
	return nil
}

// runElection checks if we're the leader; if not, watches our predecessor.
func (em *ElectionManager) runElection() error {
	for {
		if err := em.ensureParticipantZnode(); err != nil {
			log.Printf("[%s] election: znode register: %v — retry", em.nodeID, err)
			time.Sleep(2 * time.Second)
			continue
		}

		children, _, err := em.conn.Children(zkElectionPath)
		if err != nil {
			log.Printf("[%s] election: list children: %v — retry", em.nodeID, err)
			time.Sleep(2 * time.Second)
			continue
		}

		sort.Slice(children, func(i, j int) bool {
			return getSeqNumber(children[i]) < getSeqNumber(children[j])
		})
		myNode := path.Base(em.znodePath)

		if children[0] == myNode {
			if err := em.promoteToLeader(); err != nil {
				return err
			}
			em.holdLeadershipWithWatchdog()
			continue
		}

		if err := em.runFollowerRound(children, myNode); err != nil {
			log.Printf("[%s] election follower round: %v — retry", em.nodeID, err)
			time.Sleep(2 * time.Second)
		}
	}
}

func (em *ElectionManager) promoteToLeader() error {
	em.mu.Lock()
	em.isLeader = true
	em.leaderID = em.nodeID
	em.mu.Unlock()

	em.LogEvent("Became LEADER")
	if em.onBecomeLeader != nil {
		em.onBecomeLeader()
	}
	if em.onLeaderChanged != nil {
		em.onLeaderChanged(em.nodeID)
	}
	return nil
}

// holdLeadershipWithWatchdog blocks while leader; session recovery or lost znode order triggers step-down.
func (em *ElectionManager) holdLeadershipWithWatchdog() {
	ticker := time.NewTicker(leaderWatchdogInterval)
	defer ticker.Stop()

	for {
		select {
		case <-em.recoveryCh:
			em.LogEvent("Leader recovery signal — stepping down and rejoining cluster")
			em.handleRecoverySignal()
			return
		case <-ticker.C:
			if ok, err := em.verifyStillLeader(); err != nil {
				log.Printf("[%s] Leader watchdog: verify error: %v", em.nodeID, err)
			} else if !ok {
				em.LogEvent("Leader watchdog: no longer smallest znode — stepping down")
				em.handleRecoverySignal()
				return
			}
		}
	}
}

func (em *ElectionManager) verifyStillLeader() (bool, error) {
	em.mu.RLock()
	zp := em.znodePath
	em.mu.RUnlock()
	if zp == "" {
		return false, nil
	}
	myNode := path.Base(zp)

	children, _, err := em.conn.Children(zkElectionPath)
	if err != nil {
		return false, err
	}
	sort.Slice(children, func(i, j int) bool {
		return getSeqNumber(children[i]) < getSeqNumber(children[j])
	})
	if len(children) == 0 {
		return false, nil
	}
	return children[0] == myNode, nil
}

func (em *ElectionManager) runFollowerRound(children []string, myNode string) error {
	myIdx := indexOf(children, myNode)
	if myIdx < 0 {
		em.LogEvent("Our znode missing from cluster — forcing re-registration")
		em.handleRecoverySignal()
		return nil
	}
	predecessor := zkElectionPath + "/" + children[myIdx-1]

	leaderData, _, err := em.conn.Get(zkElectionPath + "/" + children[0])
	if err == nil {
		em.mu.Lock()
		em.isLeader = false
		em.leaderID = string(leaderData)
		em.mu.Unlock()
		if em.onLeaderChanged != nil {
			em.onLeaderChanged(string(leaderData))
		}
	}

	log.Printf("[%s] Follower. Watching predecessor: %s", em.nodeID, predecessor)

	exists, _, watchCh, err := em.conn.ExistsW(predecessor)
	if err != nil {
		return fmt.Errorf("watch failed: %w", err)
	}
	if !exists {
		return nil
	}

	select {
	case <-em.recoveryCh:
		em.LogEvent("Recovery while follower — re-registering with ZooKeeper")
		em.handleRecoverySignal()
		return nil
	case ev := <-watchCh:
		em.LogEvent(fmt.Sprintf("Watch fired: %v — re-running election", ev.Type))
	}
	return nil
}

// Stop gracefully removes our znode (triggers election in others).
func (em *ElectionManager) Stop() {
	em.demoteAndClearRegistration()
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

func (em *ElectionManager) SetOnLeaderChanged(fn func(newLeader string)) {
	em.onLeaderChanged = fn
}

// SetOnRecovery is invoked when the node begins a ZK session recovery (expire-driven or forced re-register).
func (em *ElectionManager) SetOnRecovery(fn func()) {
	em.mu.Lock()
	em.onRecovery = fn
	em.mu.Unlock()
}

func (em *ElectionManager) IsConnected() bool {
	state := em.conn.State()
	return state == zk.StateHasSession || state == zk.StateConnected
}

func (em *ElectionManager) LogEvent(msg string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	timestamp := time.Now().Format("15:04:05")
	fullMsg := fmt.Sprintf("[%s] %s", timestamp, msg)
	em.events = append(em.events, fullMsg)
	if len(em.events) > 10 {
		em.events = em.events[1:]
	}
	log.Printf("[%s] %s", em.nodeID, msg)
}

func (em *ElectionManager) GetEvents() []string {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.events
}

func (em *ElectionManager) GetPeerCount() int {
	children, _, err := em.conn.Children(zkElectionPath)
	if err != nil {
		return 0
	}
	return len(children)
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

		data, _, watchCh, err := em.conn.GetW(leaderZnode)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		em.mu.Lock()
		em.leaderID = string(data)
		em.mu.Unlock()

		if em.onLeaderChanged != nil {
			em.onLeaderChanged(string(data))
		}

		em.LogEvent(fmt.Sprintf("Leader updated: %s", string(data)))

		<-watchCh
		em.LogEvent("Leader znode changed — re-checking leader")
	}
}
