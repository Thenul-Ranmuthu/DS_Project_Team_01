package election

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /election/status
func (em *ElectionManager) HandleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"node_id":   em.nodeID,
		"is_leader": em.IsLeader(),
		"leader_id": em.LeaderID(),
		"znode":     em.znodePath,
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
