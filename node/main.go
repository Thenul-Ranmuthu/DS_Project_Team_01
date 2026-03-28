package main

import (
	"log"
	"os"
	"time"

	clock "github.com/DS_node/Clock"
	controllers "github.com/DS_node/Controllers"
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/election"
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
		log.Printf("[NTPClock] Initial sync failed: %v", err)
	}

	// Re-sync every 10 minutes in the background
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			if err := clock.NTP.Sync("pool.ntp.org"); err != nil {
				log.Printf("[NTPClock] Re-sync failed: %v", err)
			}
		}
	}()

	zkServers := []string{"172.30.112.1:2181"}
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		log.Fatal("NODE_ID environment variable is required")
	}

	var err error
	em, err = election.NewElectionManager(zkServers, nodeID)
	if err != nil {
		log.Fatalf("Election manager init failed: %v", err)
	}

	// Set callbacks to integrate with your replication/clock packages

	em.SetOnBecomeLeader(func() {
		log.Println("This node is now leader — start accepting writes")
		initializers.DB.Create(&models.ElectionEvent{
			NodeID:    nodeID,
			EventType: "became_leader",
		})
	})

	go func() {
		if err := em.Start(); err != nil {
			log.Fatalf("Election failed: %v", err)
		}
	}()
}

func main() {
	router := gin.Default()

	router.POST("/createUser", controllers.CreateUser)

	router.GET("/ping", controllers.PingEndPoint)

	router.POST("/upload/:email", controllers.UploadMultipleFiles)
	router.GET("/users/files/:email", controllers.GetUserFiles)
	router.GET("/files/:id", controllers.GetFileByID)
	router.DELETE("/files/:id", controllers.DeleteFile)

	// Lamport clock — lets other nodes (or a monitor) read this node's logical time
	router.GET("/clock", controllers.GetClock)

	router.Run()
}
