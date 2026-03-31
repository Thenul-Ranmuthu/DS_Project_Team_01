package controllers

import (
	"net/http"

	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/election"
	"github.com/gin-gonic/gin"
)

// Live checks if the process is running
func Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Ready returns health checking middleware
func Ready(em *election.ElectionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbReady := false
		if initializers.DB != nil {
			db, err := initializers.DB.DB()
			if err == nil && db.Ping() == nil {
				dbReady = true
			}
		}

		zkReady := em != nil && em.IsConnected()

		status := http.StatusOK
		if !dbReady || !zkReady || em == nil {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, gin.H{
			"db_connected":         dbReady,
			"zk_connected":         zkReady,
			"election_initialized": em != nil,
			"status":               status,
		})
	}
}
