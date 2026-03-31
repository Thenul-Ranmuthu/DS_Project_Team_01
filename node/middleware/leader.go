package middleware

import (
	"log/slog"
	"net/http"

	"github.com/DS_node/election"
	"github.com/gin-gonic/gin"
)

// LeaderOnly returns a Gin middleware that rejects requests if the node is not the leader.
func LeaderOnly(em *election.ElectionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if em == nil {
			slog.Error("Leader check failed: Election manager is not initialized")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Election manager not initialized"})
			c.Abort()
			return
		}

		if !em.IsLeader() {
			slog.Warn("Rejected write request on follower node",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"client_ip", c.ClientIP(),
				"leader_id", em.GetLeaderID(),
			)

			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":     "This node is not the leader. Writes are only permitted on the leader.",
				"leader_id": em.GetLeaderID(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
