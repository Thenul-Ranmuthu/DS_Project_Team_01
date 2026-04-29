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
	"github.com/DS_node/replication"
	"github.com/gin-gonic/gin"
)

var em *election.ElectionManager

func init() {
	initializers.LoadEnvVaribles()
	migrate.MigrateDB()

	// Start the replication retry worker
	replication.StartRetryWorker()

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

	// Local ZooKeeper address
	zkServers := []string{os.Getenv("ZK_SERVER") + ":2181"}
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		log.Fatal("NODE_ID environment variable is required")
	}

	var err error
	em, err = election.NewElectionManager(zkServers, nodeID)
	if err != nil {
		log.Fatalf("Election manager init failed: %v", err)
	}

	// Leader Election Callbacks
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

	// --- AUTOMATED RECOVERY ---
	// When a node joins, it should catch up with the leader
	go replication.TriggerRecovery()
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Lamport-Clock, Idempotency-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {

	router := gin.Default()

	router.Use(CORSMiddleware())

	// User & Health Routes
	router.POST("/createUser", controllers.CreateUser)
	router.GET("/ping", controllers.PingEndPoint)

	// File Management Routes
	router.POST("/upload/:email", controllers.UploadMultipleFiles)
	router.GET("/users/files/:email", controllers.GetUserFiles)
	router.GET("/files/:id", controllers.GetFileByID)
	router.DELETE("/files/:id", controllers.DeleteFile)

	// --- MEMBER 2: INTERNAL REPLICATION ROUTE ---
	router.POST("/internal/replicate", controllers.InternalReplicate)
	router.DELETE("/internal/delete/:filename", controllers.DeleteReplica)
	router.POST("/internal/users", controllers.InternalCreateUser)
	
	// --- RECOVERY ROUTES ---
	router.GET("/internal/users/all", controllers.GetAllUsers)
	router.GET("/internal/files/all", controllers.GetAllFiles)
	router.GET("/internal/files/download/:filename", controllers.InternalDownloadFile)

	// Clock & Election Monitoring
	router.GET("/clock", controllers.GetClock)
	router.GET("/election/status", em.HandleStatus)
	router.POST("/election/resign", em.HandleResign)

	// Start the server
	router.Run()
}
