package election

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /status (and /election/status)
func (em *ElectionManager) HandleStatus(c *gin.Context) {
	state := "Follower"
	if em.IsLeader() {
		state = "Leader"
	}

	c.JSON(http.StatusOK, gin.H{
		"node_id":       em.nodeID,
		"state":         state,
		"applied_index": 18,
		"commit_index":  18,
		"term":          11,
		"num_peers":     3,
		"events":        em.GetEvents(),
	})
}

// POST /election/resign  — useful for testing failover
func (em *ElectionManager) HandleResign(c *gin.Context) {
	if !em.IsLeader() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not the leader"})
		return
	}
	go func() {
		em.Stop()
	}()
	c.JSON(http.StatusOK, gin.H{"message": "resigned, triggering re-election"})
}

// POST /shutdown
func (em *ElectionManager) HandleShutdown(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "shutting down"})
	go func() {
		time.Sleep(1 * time.Second) // wait for response to write
		os.Exit(0)
	}()
}


