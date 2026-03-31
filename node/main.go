package main

import (
	"log/slog"
	"os"
	"time"

	clock "github.com/DS_node/Clock"
	controllers "github.com/DS_node/Controllers"
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/election"
	"github.com/DS_node/middleware"
	"github.com/DS_node/migrate"
	"github.com/DS_node/models"
	"github.com/gin-gonic/gin"
)

var em *election.ElectionManager

func init() {
	initializers.LoadEnvVaribles()
	migrate.MigrateDB()

	// Initial sync at startup
	if err := clock.NTP.Sync("pool.ntp.org"); err != nil {
		slog.Error("Initial sync failed", "error", err, "component", "NTPClock")
	}

	// Re-sync every 10 minutes in the background
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			if err := clock.NTP.Sync("pool.ntp.org"); err != nil {
				slog.Error("Re-sync failed", "error", err, "component", "NTPClock")
			}
		}
	}()

	zkServers := []string{"127.0.0.1:2181"}
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		slog.Error("NODE_ID environment variable is required")
		os.Exit(1)
	}

	var err error
	em, err = election.NewElectionManager(zkServers, nodeID)
	if err != nil {
		slog.Error("Election manager init failed", "error", err)
		os.Exit(1)
	}

	// Set callbacks to integrate with your replication/clock packages

	em.SetOnBecomeLeader(func() {
		slog.Info("This node is now leader — start accepting writes")
		initializers.DB.Create(&models.ElectionEvent{
			NodeID:    nodeID,
			EventType: "became_leader",
		})
	})

	go func() {
		if err := em.Start(); err != nil {
			slog.Error("Election failed", "error", err)
			os.Exit(1)
		}
	}()
}

func main() {
	router := gin.Default()

	// Health and info endpoints (unprotected)
	router.GET("/health/live", controllers.Live)
	router.GET("/health/ready", controllers.Ready(em))

	router.GET("/ping", controllers.PingEndPoint)
	router.GET("/users/files/:email", controllers.GetUserFiles)
	router.GET("/files/:id", controllers.GetFileByID)
	router.GET("/clock", controllers.GetClock)

	// Write endpoints require leader status
	writeGroup := router.Group("/")
	writeGroup.Use(middleware.LeaderOnly(em))
	{
		writeGroup.POST("/createUser", controllers.CreateUser)
		writeGroup.POST("/upload/:email", controllers.UploadMultipleFiles)
		writeGroup.DELETE("/files/:id", controllers.DeleteFile)
	}

	// Lamport clock — lets other nodes (or a monitor) read this node's logical time
	router.GET("/clock", controllers.GetClock)

	router.Run()
}
